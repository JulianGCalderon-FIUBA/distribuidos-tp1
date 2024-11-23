package database_test

import (
	"distribuidos/tp1/database"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
)

func setupDatabase(t *testing.T, data map[string]string) string {
	database_path := t.TempDir()

	err := database.NewDatabase(database_path)
	expect(t, err)

	for k, v := range data {
		err = os.WriteFile(path.Join(database_path, database.DATA_DIR, k), []byte(v), 0666)
		if err != nil {
			t.Fatalf("failed to write %v: %v", k, err)
		}
	}

	return database_path
}

func assertDatabaseContent(t *testing.T, database_path string, data map[string]string) {
	t.Helper()

	keys := make(map[string]struct{})

	// find all files
	err := filepath.WalkDir(path.Join(database_path, database.DATA_DIR), func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		expect(t, err)

		p, err = filepath.Rel(path.Join(database_path, database.DATA_DIR), p)
		expect(t, err)

		keys[p] = struct{}{}

		return nil
	})
	expect(t, err)

	// compare data
	for k, expected_value := range data {
		actual_value, err := os.ReadFile(path.Join(database_path, database.DATA_DIR, k))
		if err != nil {
			t.Fatalf("missing database value: %v", k)
		}

		if expected_value != string(actual_value) {
			t.Fatalf("difference at key %v, expected %v, got %v", k, expected_value, string(actual_value))
		}

		delete(keys, k)
	}

	// assert there are no other file
	if len(keys) > 0 {
		t.Fatalf("database contains extra values %v", slices.Collect(maps.Keys(keys)))
	}
}

func expect(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func Test(t *testing.T) {
	type TestCase struct {
		name        string
		data        map[string]string
		transaction func(t *testing.T, s *database.Snapshot) map[string]string
	}

	cases := []TestCase{
		{
			name: "Create",
			data: map[string]string{},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Create("KEY")
				expect(t, err)

				_, err = file.WriteString("VALUE")
				expect(t, err)

				return map[string]string{
					"KEY": "VALUE",
				}
			},
		},

		{
			name: "Update",
			data: map[string]string{
				"KEY": "VALUE",
			},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Update("KEY")
				expect(t, err)

				_, err = file.WriteString("NEW_VALUE")
				expect(t, err)

				return map[string]string{
					"KEY": "NEW_VALUE",
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("TestAbort %v", c.name), func(t *testing.T) {
			db_path := setupDatabase(t, c.data)

			snapshot, err := database.NewSnapshot(db_path)
			expect(t, err)

			c.transaction(t, snapshot)

			err = snapshot.Abort()
			expect(t, err)

			assertDatabaseContent(t, db_path, c.data)

			exists, err := pathExists(path.Join(db_path, database.SNAPSHOT_DIR))
			expect(t, err)
			if exists {
				t.Fatalf("Snapshot should have been erased")
			}
		})

		t.Run(fmt.Sprintf("TestCommit %v", c.name), func(t *testing.T) {
			db_path := setupDatabase(t, c.data)

			snapshot, err := database.NewSnapshot(db_path)
			expect(t, err)

			transaction_data := c.transaction(t, snapshot)

			err = snapshot.Commit()
			expect(t, err)

			expected_data := maps.Clone(c.data)
			maps.Copy(expected_data, transaction_data)
			assertDatabaseContent(t, db_path, transaction_data)

			exists, err := pathExists(path.Join(db_path, database.SNAPSHOT_DIR))
			expect(t, err)
			if exists {
				t.Fatalf("Snapshot should have been erased")
			}
		})
	}
}
