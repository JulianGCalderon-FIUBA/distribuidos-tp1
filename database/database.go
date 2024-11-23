package database

import (
	"distribuidos/tp1/utils"
	"os"
	"path"
)

// directory inside of the database/snapshot containing the actual data
const DATA_DIR string = "data"
const APPENDS_DIR string = "appends"

// creates database at path if it doesn't exist
func NewDatabase(database_path string) error {
	return os.MkdirAll(path.Join(database_path, DATA_DIR), 0750)
}

// loads the database from path, restoring any commited snapshot
func LoadDatabase(database_path string) error {
	err := NewDatabase(database_path)
	if err != nil {
		return err
	}

	exists, err := utils.PathExists(path.Join(database_path, SNAPSHOT_DIR, COMMIT_FILE))
	if err != nil {
		return err
	}

	// if there is no commit file, discard the snapshot
	if !exists {
		err := os.RemoveAll(path.Join(database_path, SNAPSHOT_DIR))
		return err
	}

	// if there is a commit file, we apply the snapshot
	snapshot, err := LoadSnapshot(database_path)
	if err != nil {
		return err
	}

	return snapshot.ApplyCommit()
}
