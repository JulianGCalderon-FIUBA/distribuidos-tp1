package database_test

import (
	"distribuidos/tp1/database"
	"os"
	"path"
	"slices"
	"testing"
)

func expect(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func TestCreate(t *testing.T) {
	// SETUP

	database_path, err := os.MkdirTemp("", t.Name())
	expect(t, err)

	database.NewDatabase(database_path)

	// EXECUTION

	snapshot, err := database.NewSnapshot(database_path)
	expect(t, err)

	file, err := snapshot.Create("KEY")
	expect(t, err)

	value := []byte("VALUE")

	_, err = file.Write(value)
	expect(t, err)

	err = file.Close()
	expect(t, err)

	err = snapshot.Commit()
	expect(t, err)

	// VERIFICATION

	commited, err := os.ReadFile(path.Join(database_path, database.DATA_DIR, "KEY"))
	expect(t, err)

	if !slices.Equal(commited, value) {
		t.Fatal("Snapshot should have been commited")
	}

	exists, err := fileExists(path.Join())
	expect(t, err)

	if exists {
		t.Fatal("Snapshot should have been erased")
	}
}

func TestCreateAbort(t *testing.T) {
	// SETUP

	database_path, err := os.MkdirTemp("", t.Name())
	expect(t, err)

	database.NewDatabase(database_path)

	// EXECUTION

	snapshot, err := database.NewSnapshot(database_path)
	expect(t, err)

	file, err := snapshot.Create("KEY")
	expect(t, err)

	value := []byte("VALUE")

	_, err = file.Write(value)
	expect(t, err)

	err = file.Close()
	expect(t, err)

	err = snapshot.Abort()
	expect(t, err)

	// VERIFICATION

	exists, err := fileExists(path.Join(database_path, database.DATA_DIR, "KEY"))
	expect(t, err)
	if exists {
		t.Fatal("Snapshot should not have been commited")
	}

	exists, err = fileExists(path.Join())
	expect(t, err)
	if exists {
		t.Fatal("Snapshot should have been erased")
	}
}
