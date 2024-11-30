package middleware_test

import (
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

// diskMap.Start() is used to clear the map before get operation because tests don't contemplate leftover info

func TestInsert(t *testing.T) {
	diskMap := middleware.NewDiskMap("map")
	db, err := database.NewDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}
	snapshot, err := db.NewSnapshot()
	if err != nil {
		t.Fatalf("Failed init snapshot: %v", err)
	}

	stats := []middleware.GameStat{
		{
			AppID: 1,
			Stat:  1,
			Name:  "Dungeons & Dragons",
		},
		{
			AppID: 2,
			Stat:  2,
			Name:  "Fortnite",
		},
		{
			AppID: 3,
			Stat:  3,
			Name:  "Rust",
		},
		{
			AppID: 4,
			Stat:  4,
			Name:  "Stardew Valley",
		},
	}

	for _, stat := range stats {
		diskMap.Start()
		stat1, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat1 != nil {
			t.Fatalf("Element should not exist")
		}

		err = diskMap.Insert(snapshot, stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}
	err = snapshot.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	diskMap.Start()
	for _, stat := range stats {

		stat2, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat2 == nil {
			t.Fatalf("Element should exist")
		}

		if stat != *stat2 {
			t.Fatalf("Element should be the same")
		}
	}
}

func TestUpdate(t *testing.T) {
	diskMap := middleware.NewDiskMap("map")
	db, err := database.NewDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}
	snapshot, err := db.NewSnapshot()
	if err != nil {
		t.Fatalf("Failed init snapshot: %v", err)
	}

	stats := []middleware.GameStat{
		{
			AppID: 1,
			Stat:  1,
			Name:  "Dungeons & Dragons",
		},
		{
			AppID: 2,
			Stat:  2,
			Name:  "Fortnite",
		},
		{
			AppID: 3,
			Stat:  3,
			Name:  "Rust",
		},
		{
			AppID: 4,
			Stat:  4,
			Name:  "Stardew Valley",
		},
	}
	for _, stat := range stats {
		err = diskMap.Insert(snapshot, stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	err = snapshot.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	snapshot, err = db.NewSnapshot()
	if err != nil {
		t.Fatalf("Failed init snapshot: %v", err)
	}

	updated := make([]middleware.GameStat, 0)

	for _, stat := range stats {
		diskMap.Start()
		stat1, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat1 == nil {
			t.Fatalf("Element should exist")
		}

		stat1.Stat += 10
		err = diskMap.Insert(snapshot, *stat1)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		updated = append(updated, *stat1)
	}

	err = snapshot.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	for i, stat := range stats {
		diskMap.Start()
		stat2, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat2 == nil {
			t.Fatalf("Element should exist")
		}

		if updated[i] != *stat2 {
			t.Fatalf("Element should be the same")
		}
	}

}

func TestIncrement(t *testing.T) {
	diskMap := middleware.NewDiskMap("map")
	db, err := database.NewDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

	var snapshot *database.Snapshot

	for _, stat := range []middleware.GameStat{
		{
			AppID: 1,
		},
		{
			AppID: 2,
		},
		{
			AppID: 3,
		},
		{
			AppID: 4,
		},
	} {
		expected := 0
		for range 10 {
			snapshot, err = db.NewSnapshot()
			if err != nil {
				t.Fatalf("Failed init snapshot: %v", err)
			}
			increment := (expected + 1) * int(stat.AppID)
			expected += increment
			err := diskMap.Increment(snapshot, stat.AppID, uint64(increment))
			if err != nil {
				t.Fatalf("Failed to increment: %v", err)
			}
			err = snapshot.Commit()
			if err != nil {
				t.Fatalf("Failed to commit: %v", err)
			}

		}
		diskMap.Start()
		finalStat, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if finalStat == nil {
			t.Fatalf("Element should exist")
		}

		if finalStat.Stat != uint64(expected) {
			t.Fatalf("Element should be the same")
		}

	}
}

func TestRename(t *testing.T) {
	diskMap := middleware.NewDiskMap("map")
	db, err := database.NewDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

	var snapshot *database.Snapshot

	for _, stat := range []middleware.GameStat{
		{
			AppID: 1,
			Name:  "Dungeons & Dragons",
		},
		{
			AppID: 2,
			Name:  "Fortnite",
		},
		{
			AppID: 3,
			Name:  "Rust",
		},
		{
			AppID: 4,
			Name:  "Stardew Valley",
		},
	} {
		for i := range 10 {
			snapshot, err = db.NewSnapshot()
			if err != nil {
				t.Fatalf("Failed init snapshot: %v", err)
			}

			expected := fmt.Sprintf("%v-%v", stat.Name, i)
			err := diskMap.Rename(snapshot, stat.AppID, expected)
			if err != nil {
				t.Fatalf("Failed to rename: %v", err)
			}
			err = snapshot.Commit()
			if err != nil {
				t.Fatalf("Failed to commit: %v", err)
			}
			diskMap.Start()
			renamedStat, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
			if err != nil {
				t.Fatalf("Failed to get: %v", err)
			}
			if renamedStat == nil {
				t.Fatalf("Element should exist")
			}
			if renamedStat.Name != expected {
				t.Fatalf("Element should be the same")
			}
		}
	}
}

func TestAll(t *testing.T) {
	diskMap := middleware.NewDiskMap("map")
	db, err := database.NewDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}
	var snapshot *database.Snapshot
	for _, stat := range []middleware.GameStat{
		{
			AppID: 1,
			Stat:  1,
			Name:  "Dungeons & Dragons",
		},
		{
			AppID: 2,
			Stat:  2,
			Name:  "Fortnite",
		},
		{
			AppID: 3,
			Stat:  3,
			Name:  "Rust",
		},
		{
			AppID: 4,
			Stat:  4,
			Name:  "Stardew Valley",
		},
	} {
		snapshot, err = db.NewSnapshot()
		if err != nil {
			t.Fatalf("Failed init snapshot: %v", err)
		}

		err = diskMap.Insert(snapshot, stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		err = snapshot.Commit()
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		snapshot, err = db.NewSnapshot()
		if err != nil {
			t.Fatalf("Failed init snapshot: %v", err)
		}

		err = diskMap.Increment(snapshot, stat.AppID, stat.Stat)
		if err != nil {
			t.Fatalf("Failed to increment: %v", err)
		}

		err = snapshot.Commit()
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		snapshot, err = db.NewSnapshot()
		if err != nil {
			t.Fatalf("Failed init snapshot: %v", err)
		}

		newName := fmt.Sprintf("%v-2", stat.Name)
		err = diskMap.Rename(snapshot, stat.AppID, newName)
		if err != nil {
			t.Fatalf("Failed to rename: %v", err)
		}

		err = snapshot.Commit()
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		diskMap.Start()
		actualStat, err := diskMap.Get(db, strconv.Itoa(int(stat.AppID)))
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if actualStat == nil {
			t.Fatal("Element should exist")
		}

		expectedStat := middleware.GameStat{
			AppID: stat.AppID,
			Name:  newName,
			Stat:  stat.Stat * 2,
		}

		if !reflect.DeepEqual(expectedStat, *actualStat) {
			t.Fatalf("Elements should be equal")
		}
	}
}

func TestGetAll(t *testing.T) {
	diskMap := middleware.NewDiskMap("map")
	db, err := database.NewDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

	snapshot, err := db.NewSnapshot()
	if err != nil {
		t.Fatalf("Failed init snapshot: %v", err)
	}

	expected := []middleware.GameStat{
		{
			AppID: 1,
			Stat:  1,
			Name:  "Dungeons & Dragons",
		},
		{
			AppID: 2,
			Stat:  2,
			Name:  "Fortnite",
		},
		{
			AppID: 3,
			Stat:  3,
			Name:  "Rust",
		},
		{
			AppID: 4,
			Stat:  4,
			Name:  "Stardew Valley",
		},
	}

	for _, stat := range expected {
		err = diskMap.Insert(snapshot, stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	err = snapshot.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	diskMap.Start()
	all, err := diskMap.GetAll(db)
	if err != nil {
		t.Fatalf("Failed to get all: %v", err)
	}

	if !reflect.DeepEqual(expected, all) {
		t.Fatalf("Elements should be equal")
	}
}
