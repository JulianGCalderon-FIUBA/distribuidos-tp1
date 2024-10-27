package middleware_test

import (
	"distribuidos/tp1/middleware"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestInsert(t *testing.T) {
	os.RemoveAll("tmp_TestInsert")
	diskMap, err := middleware.NewDiskMap("tmp_TestInsert")
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

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
		stat1, err := diskMap.Get(stat.AppID)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat1 != nil {
			t.Fatalf("Element should not exist")
		}

		err = diskMap.Insert(stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		stat2, err := diskMap.Get(stat.AppID)
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
	os.RemoveAll("tmp_TestUpdate")
	diskMap, err := middleware.NewDiskMap("tmp_TestUpdate")
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

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
		err = diskMap.Insert(stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		stat1, err := diskMap.Get(stat.AppID)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat1 == nil {
			t.Fatalf("Element should exist")
		}

		stat1.Stat += 10
		err = diskMap.Insert(*stat1)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		stat2, err := diskMap.Get(stat.AppID)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if stat2 == nil {
			t.Fatalf("Element should exist")
		}

		if *stat1 != *stat2 {
			t.Fatalf("Element should be the same")
		}

	}
}

func TestIncrement(t *testing.T) {
	os.RemoveAll("tmp_TestIncrement")
	diskMap, err := middleware.NewDiskMap("tmp_TestIncrement")
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

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
			increment := (expected + 1) * int(stat.AppID)
			expected += increment
			err := diskMap.Increment(stat.AppID, uint64(increment))
			if err != nil {
				t.Fatalf("Failed to increment: %v", err)
			}

		}

		finalStat, err := diskMap.Get(stat.AppID)
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
	os.RemoveAll("tmp_TestRename")
	diskMap, err := middleware.NewDiskMap("tmp_TestRename")
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

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
			expected := fmt.Sprintf("%v-%v", stat.Name, i)
			err := diskMap.Rename(stat.AppID, expected)
			if err != nil {
				t.Fatalf("Failed to rename: %v", err)
			}

			renamedStat, err := diskMap.Get(stat.AppID)
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
	os.RemoveAll("tmp_TestAll")
	diskMap, err := middleware.NewDiskMap("tmp_TestAll")
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

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

		err = diskMap.Insert(stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		err = diskMap.Increment(stat.AppID, stat.Stat)
		if err != nil {
			t.Fatalf("Failed to increment: %v", err)
		}

		newName := fmt.Sprintf("%v-2", stat.Name)
		err = diskMap.Rename(stat.AppID, newName)
		if err != nil {
			t.Fatalf("Failed to rename: %v", err)
		}

		actualStat, err := diskMap.Get(stat.AppID)
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
	os.RemoveAll("tmp_TestUpdate")
	diskMap, err := middleware.NewDiskMap("tmp_TestUpdate")
	if err != nil {
		t.Fatalf("Failed init map: %v", err)
	}

	expected := []*middleware.GameStat{
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
		err = diskMap.Insert(*stat)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	all, err := diskMap.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all: %v", err)
	}

	if !reflect.DeepEqual(expected, all) {
		t.Fatalf("Elements should be equal")
	}
}
