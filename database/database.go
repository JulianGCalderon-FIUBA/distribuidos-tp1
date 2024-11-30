package database

import (
	"distribuidos/tp1/utils"
	"os"
	"path"
)

// directory inside of the database/snapshot containing the actual data
const DATA_DIR string = "data"

type Database struct {
	root string
}

// creates database at path if it doesn't exist
// restores any in progress snapshot
func NewDatabase(root string) (*Database, error) {
	err := os.MkdirAll(path.Join(root, DATA_DIR), 0750)
	if err != nil {
		return nil, err
	}

	db := &Database{
		root: root,
	}

	err = db.Restore()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Database) Restore() error {
	snapshot := Snapshot{
		db:   db,
		root: db.SnapshotPath(),
	}
	return snapshot.Restore()
}

func (db *Database) NewSnapshot() (*Snapshot, error) {
	err := os.MkdirAll(db.SnapshotDataDir(), 0750)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(db.SnapshotAppendDir(), 0750)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		db:    db,
		root:  db.SnapshotPath(),
		files: []*os.File{},
	}, nil
}

// Accesses value of the key
//
// File should be manually closed
func (db *Database) Get(k string) (*os.File, error) {
	return os.Open(db.KeyPath(k))
}

func (db *Database) Exists(k string) (bool, error) {
	return utils.PathExists(db.KeyPath(k))
}

func (db *Database) GetAll(k string) ([]string, error) {
	files := make([]string, 0)
	entries, err := os.ReadDir(db.KeyPath(k))
	if err != nil {
		return nil, nil
	}

	for _, e := range entries {
		n := path.Join(k, e.Name())
		files = append(files, n)
	}

	return files, nil
}

func (db *Database) Delete() error {
	return os.RemoveAll(db.root)
}

// auxiliary path functions

func (db *Database) KeyPath(k string) string {
	return path.Join(db.DataDir(), k)
}

func (db *Database) CommitPath() string {
	return path.Join(db.root, SNAPSHOT_DIR, COMMIT_FILE)
}

func (db *Database) DataDir() string {
	return path.Join(db.root, DATA_DIR)
}

func (db *Database) SnapshotPath() string {
	return path.Join(db.root, SNAPSHOT_DIR)
}

func (db *Database) SnapshotDataDir() string {
	return path.Join(db.SnapshotPath(), DATA_DIR)
}

func (db *Database) SnapshotAppendDir() string {
	return path.Join(db.SnapshotPath(), APPENDS_DIR)
}
