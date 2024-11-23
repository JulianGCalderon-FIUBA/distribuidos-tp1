package database_test

import (
	"distribuidos/tp1/database"
	"os"
	"path"
	"slices"
	"testing"
)

func expect(t *testing.T, err error) {
	t.Helper()
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

func TestCommitCreate(t *testing.T) {
	// SETUP

	database_path := t.TempDir()

	err := database.NewDatabase(database_path)
	expect(t, err)

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

func TestCommitUpdate(t *testing.T) {
	// SETUP

	database_path := t.TempDir()

	err := database.NewDatabase(database_path)
	expect(t, err)

	old_value := []byte("OLD_VALUE")
	err = os.WriteFile(path.Join(database_path, database.DATA_DIR, "KEY"), old_value, 0666)
	expect(t, err)

	// EXECUTION

	snapshot, err := database.NewSnapshot(database_path)
	expect(t, err)

	file, err := snapshot.Update("KEY")
	expect(t, err)

	new_value := []byte("NEW_VALUE")

	_, err = file.Write(new_value)
	expect(t, err)

	err = file.Close()
	expect(t, err)

	err = snapshot.Commit()
	expect(t, err)

	// VERIFICATION

	commited, err := os.ReadFile(path.Join(database_path, database.DATA_DIR, "KEY"))
	expect(t, err)

	if !slices.Equal(commited, new_value) {
		t.Fatal("Snapshot should have been commited")
	}

	exists, err := fileExists(path.Join())
	expect(t, err)

	if exists {
		t.Fatal("Snapshot should have been erased")
	}
}

func TestAbort(t *testing.T) {
	// SETUP

	database_path := t.TempDir()

	err := database.NewDatabase(database_path)
	expect(t, err)

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
