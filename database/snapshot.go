package database

import (
	"distribuidos/tp1/utils"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// directory inside of the database containing the snapshot
const SNAPSHOT_DIR string = "snapshot"

// file inside of the snapshot indicating that it's valid
const COMMIT_FILE string = "commit"

// directory inside of the snapshot containing appends
const APPENDS_DIR string = "appends"

type Snapshot struct {
	db    *Database
	root  string
	files []*os.File
}

// Accesses the original value of the key
func (s *Snapshot) Get(k string) (*os.File, error) {
	file, err := s.db.Get(k)
	if err != nil {
		return nil, err
	}

	s.files = append(s.files, file)

	return file, nil
}

// Creates a new entry for the given key. It will replace the old entry if it exists
func (s *Snapshot) Create(k string) (*os.File, error) {
	file, err := utils.OpenFileAll(s.KeyPath(k), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)

	if err != nil {
		return nil, err
	}

	s.files = append(s.files, file)

	return file, err
}

// Copies the given entry to the snapshot and returns it's file descriptor.
//
// Fails if the entry has already been copied
func (s *Snapshot) Update(k string) (*os.File, error) {
	src, err := os.Open(s.db.KeyPath(k))
	if errors.Is(err, fs.ErrNotExist) {
		return s.Create(k)
	}
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// O_EXCL asserts that file must not exist
	dst, err := utils.OpenFileAll(s.KeyPath(k), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)

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
	info, err := os.Stat(s.db.KeyPath(k))
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

	file, err := utils.OpenFileAll(s.AppendKeyPath(k), os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_EXCL, 0666)
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

func (s *Snapshot) Delete(k string) (*os.File, error) {
	panic("unimplemented")
}

func (s *Snapshot) Exists(k string) (bool, error) {
	return s.db.Exists(k)
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

func (s *Snapshot) Sync() error {
	for _, f := range s.files {
		err := f.Sync()
		if err != nil {
			return err
		}
	}
	return nil
}

// Commits all changes to the actual database.
//
// This operation is fault tolerant. If it's interrupted, it can be
// completed afterwards by calling `Restore`
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

	return os.RemoveAll(s.root)
}

// Depending on the state of the snapshot, it aborts it or commits it.
func (s *Snapshot) Restore() error {
	exists, err := utils.PathExists(s.CommitPath())
	if err != nil {
		return err
	}

	if exists {
		return s.ApplyCommit()
	} else {
		return os.RemoveAll(s.root)
	}
}

// Register the commit, but do not apply it.
// This is unsafe and should only be used for testing
func (s *Snapshot) RegisterCommit() error {
	err := os.WriteFile(path.Join(s.root, COMMIT_FILE), []byte{}, 0666)
	if err != nil {
		return err
	}

	return nil
}

func (s *Snapshot) ApplyCommit() error {
	err := filepath.WalkDir(path.Join(s.root, DATA_DIR), s.commitFile)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(path.Join(s.root, APPENDS_DIR), s.commitAppendFile)
	if err != nil {
		return err
	}

	utils.MaybeExit(0.001)

	err = os.RemoveAll(s.root)
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

	rel_path, err := filepath.Rel(s.root, modified_path)
	if err != nil {
		return err
	}

	original_path := path.Join(s.db.root, rel_path)

	err = os.Rename(modified_path, original_path)
	if errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(path.Dir(original_path), 0750)
		if err != nil {
			return err
		}
		return os.Rename(modified_path, original_path)
	}

	utils.MaybeExit(0.001)

	return nil
}

// Commits the changes of an append file to the actual database
//
// This operation is NOT atomic on UNIX.
// It should be reexecuted it interrupted and the append file was not erased.
func (s *Snapshot) commitAppendFile(modified_path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	rel_path, err := filepath.Rel(path.Join(s.root, APPENDS_DIR), modified_path)
	if err != nil {
		return err
	}
	original_path := path.Join(s.db.DataDir(), rel_path)

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

	utils.MaybeExit(0.001)

	return os.Remove(modified_path)
}

// auxiliary path functions

func (s *Snapshot) DataDir() string {
	return path.Join(s.root, DATA_DIR)
}
func (s *Snapshot) AppendsDir() string {
	return path.Join(s.root, APPENDS_DIR)
}
func (s *Snapshot) KeyPath(k string) string {
	return path.Join(s.DataDir(), k)
}
func (s *Snapshot) AppendKeyPath(k string) string {
	return path.Join(s.AppendsDir(), k)
}
func (s *Snapshot) CommitPath() string {
	return path.Join(s.root, COMMIT_FILE)
}
