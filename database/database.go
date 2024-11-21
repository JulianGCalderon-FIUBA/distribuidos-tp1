package database

import (
	"os"
	"path"
)

// directory inside of the database/snapshot containing the actual data
const DATA_DIR string = "data"

func NewDatabase(database_path string) error {
	return os.MkdirAll(path.Join(database_path, DATA_DIR), 0750)
}
