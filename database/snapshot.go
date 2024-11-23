package database

import (
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

type Snapshot struct {
	database_path string
	snapshot_path string
	files         []*os.File
}

// directory inside of the database containing the snapshot
const SNAPSHOT_DIR string = "snapshot"

// file inside of the snapshot indicating that it's valid
const COMMIT_FILE string = "commit"

func NewSnapshot(database_path string) (*Snapshot, error) {
	snapshot_path := path.Join(database_path, SNAPSHOT_DIR)

	err := os.MkdirAll(path.Join(snapshot_path, DATA_DIR), 0750)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(path.Join(snapshot_path, APPENDS_DIR), 0750)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		database_path: database_path,
		snapshot_path: snapshot_path,
	}, nil
}

// loads an existing snapshot
func LoadSnapshot(database_path string) (*Snapshot, error) {
	snapshot_path := path.Join(database_path, SNAPSHOT_DIR)

	return &Snapshot{
		database_path: database_path,
		snapshot_path: snapshot_path,
	}, nil
}

// Creates a new entry for the given key. It will replace the old entry if it exists
func (s *Snapshot) Create(k string) (*os.File, error) {
	file, err := os.Create(path.Join(s.snapshot_path, DATA_DIR, k))
	if err != nil {
		return nil, err
	}

	s.files = append(s.files, file)

	return file, err
}

// Copies the given entry to the snapshot and returns it's file descriptor.
// The file descriptor must be manually closed.
//
// Fails if the entry has already been copied
func (s *Snapshot) Update(k string) (*os.File, error) {
	src, err := os.Open(path.Join(s.database_path, DATA_DIR, k))
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// O_EXCL asserts that file must not exist
	dst, err := os.OpenFile(
		path.Join(s.snapshot_path, DATA_DIR, k),
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0666,
	)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		dst.Close()
		return nil, err
	}

	_, err = dst.Seek(0, io.SeekStart)
	if err != nil {
		dst.Close()
		return nil, err
	}

	s.files = append(s.files, dst)

	return dst, nil
}

// Append data to a given value. The file cursor's position should not be manually modified
func (s *Snapshot) Append(k string) (*os.File, error) {
	var size int64
	info, err := os.Stat(path.Join(s.database_path, DATA_DIR, k))
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			size = 0
		} else {
			return nil, err
		}
	} else {
		size = info.Size()
	}

	// if original size is empty, or doesn't exist, we reuse the Create operation
	if size == 0 {
		return s.Create(k)
	}

	file, err := os.Create(path.Join(s.snapshot_path, APPENDS_DIR, k))
	if err != nil {
		return nil, err
	}

	err = binary.Write(file, binary.LittleEndian, &size)
	if err != nil {
		file.Close()
		return nil, err
	}

	s.files = append(s.files, file)

	return file, err
}

// Commits all changes to the actual database.
//
// This operation is fault tolerant. If it's interrupted, it can be
// completed afterwards
func (s *Snapshot) Commit() error {
	err := s.Close()
	if err != nil {
		return err
	}

	err = s.RegisterCommit()
	if err != nil {
		return err
	}

	return s.ApplyCommit()
}

// Aborts the changes of the snapshot
func (s *Snapshot) Abort() error {
	err := s.Close()
	if err != nil {
		return err
	}

	return os.RemoveAll(s.snapshot_path)
}

func (s *Snapshot) Close() error {
	for _, f := range s.files {
		err := f.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// Register the commit, but do not apply it.
// This is unsafe and should only be used for testing
func (s *Snapshot) RegisterCommit() error {
	file, err := os.Create(path.Join(s.snapshot_path, COMMIT_FILE))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *Snapshot) ApplyCommit() error {
	err := filepath.WalkDir(path.Join(s.snapshot_path, DATA_DIR), s.commitFile)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(path.Join(s.snapshot_path, APPENDS_DIR), s.commitAppendFile)
	if err != nil {
		return err
	}

	err = os.RemoveAll(s.snapshot_path)
	if err != nil {
		return err
	}
	return nil
}

// Commits the changes of a single file to the actual database
//
// This operation is atomic on UNIX
func (s *Snapshot) commitFile(modified_path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	rel_path, err := filepath.Rel(s.snapshot_path, modified_path)
	if err != nil {
		return err
	}

	original_path := path.Join(s.database_path, rel_path)

	return os.Rename(modified_path, original_path)
}

func (s *Snapshot) commitAppendFile(modified_path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	rel_path, err := filepath.Rel(path.Join(s.snapshot_path, APPENDS_DIR), modified_path)
	if err != nil {
		return err
	}
	original_path := path.Join(s.database_path, DATA_DIR, rel_path)

	append_file, err := os.Open(modified_path)
	if err != nil {
		return err
	}
	defer append_file.Close()

	var original_end_offset int64
	err = binary.Read(append_file, binary.LittleEndian, &original_end_offset)
	if err != nil {
		return err
	}

	original_file, err := os.OpenFile(original_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer original_file.Close()

	_, err = original_file.Seek(original_end_offset, io.SeekStart)
	if err != nil {
		return err
	}

	_, err = io.Copy(original_file, append_file)
	if err != nil {
		return err
	}

	return os.Remove(modified_path)
}
