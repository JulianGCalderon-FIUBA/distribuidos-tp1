package database_test

import (
	"distribuidos/tp1/database"
	"distribuidos/tp1/utils"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
)

func setupDatabase(t *testing.T, data map[string]string) *database.Database {
	database_path := t.TempDir()

	db, err := database.NewDatabase(database_path)
	expect(t, err)

	for k, v := range data {
		err = os.WriteFile(path.Join(database_path, database.DATA_DIR, k), []byte(v), 0666)
		if err != nil {
			t.Fatalf("failed to write %v: %v", k, err)
		}
	}

	return db
}

func assertDatabaseContent(t *testing.T, db *database.Database, data map[string]string) {
	t.Helper()

	keys := make(map[string]struct{})

	// find all files
	err := filepath.WalkDir(db.DataDir(), func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		expect(t, err)

		p, err = filepath.Rel(db.DataDir(), p)
		expect(t, err)

		keys[p] = struct{}{}

		return nil
	})
	expect(t, err)

	// compare data
	for k, expected_value := range data {
		actual_value, err := os.ReadFile(db.KeyPath(k))
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

func TestTransaction(t *testing.T) {
	type TestCase struct {
		name        string
		data        map[string]string
		transaction func(t *testing.T, s *database.Snapshot) map[string]string
	}

	cases := []TestCase{
		{
			name: "Create",
			data: map[string]string{
				"OTHER_KEY": "OTHER_VALUE",
			},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Create("KEY")
				expect(t, err)

				_, err = file.Write([]byte("VALUE"))
				expect(t, err)

				return map[string]string{
					"KEY": "VALUE",
				}
			},
		},

		{
			name: "Update",
			data: map[string]string{
				"KEY":       "VALUE",
				"OTHER_KEY": "OTHER_VALUE",
			},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Update("KEY")
				expect(t, err)

				_, err = file.Write([]byte("NEW_VALUE"))
				expect(t, err)

				return map[string]string{
					"KEY": "NEW_VALUE",
				}
			},
		},

		{
			name: "Append",
			data: map[string]string{
				"KEY":       "VALUE",
				"OTHER_KEY": "OTHER_VALUE",
			},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Append("KEY")
				expect(t, err)

				_, err = file.WriteString("_NEW")
				expect(t, err)

				return map[string]string{
					"KEY": "VALUE_NEW",
				}
			},
		},

		{
			name: "AppendNoFile",
			data: map[string]string{
				"OTHER_KEY": "OTHER_VALUE",
			},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Append("KEY")
				expect(t, err)

				_, err = file.WriteString("_NEW")
				expect(t, err)

				return map[string]string{
					"KEY": "_NEW",
				}
			},
		},

		{
			name: "AppendEmptyFile",
			data: map[string]string{
				"KEY":       "",
				"OTHER_KEY": "OTHER_VALUE",
			},
			transaction: func(t *testing.T, s *database.Snapshot) map[string]string {
				file, err := s.Append("KEY")
				expect(t, err)

				_, err = file.WriteString("_NEW")
				expect(t, err)

				return map[string]string{
					"KEY": "_NEW",
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			t.Run("Abort", func(t *testing.T) {
				db := setupDatabase(t, c.data)

				snapshot, err := db.NewSnapshot()
				expect(t, err)

				c.transaction(t, snapshot)

				err = snapshot.Abort()
				expect(t, err)

				assertDatabaseContent(t, db, c.data)

				assertSnapshotErased(t, db)
			})

			t.Run("Commit", func(t *testing.T) {
				db := setupDatabase(t, c.data)

				snapshot, err := db.NewSnapshot()
				expect(t, err)

				transaction_data := c.transaction(t, snapshot)

				err = snapshot.Commit()
				expect(t, err)

				expected_data := maps.Clone(c.data)
				maps.Copy(expected_data, transaction_data)
				assertDatabaseContent(t, db, expected_data)

				assertSnapshotErased(t, db)
			})

			t.Run("FailureBeforeCommit", func(t *testing.T) {
				db := setupDatabase(t, c.data)

				snapshot, err := db.NewSnapshot()
				expect(t, err)

				c.transaction(t, snapshot)

				// we load the database to simulate that we have been killed
				err = db.Restore()
				expect(t, err)

				assertDatabaseContent(t, db, c.data)

				assertSnapshotErased(t, db)

				err = snapshot.Close()
				expect(t, err)
			})

			t.Run("FailureAfterCommit", func(t *testing.T) {
				db := setupDatabase(t, c.data)

				snapshot, err := db.NewSnapshot()
				expect(t, err)

				transaction_data := c.transaction(t, snapshot)

				// we simulate an after commit interrupt by only registering the commit
				snapshot.Close()
				expect(t, err)
				err = snapshot.RegisterCommit()
				expect(t, err)

				// we load the database to simulate that we have been killed
				err = db.Restore()
				expect(t, err)

				expected_data := maps.Clone(c.data)
				maps.Copy(expected_data, transaction_data)
				assertDatabaseContent(t, db, expected_data)

				assertSnapshotErased(t, db)
			})
		})
	}
}

func assertSnapshotErased(t *testing.T, db *database.Database) {
	t.Helper()

	exists, err := utils.PathExists(db.SnapshotPath())
	expect(t, err)

	if exists {
		t.Fatalf("Snapshot should have been erased")
	}
}
