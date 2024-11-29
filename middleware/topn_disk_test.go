package middleware_test

import (
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"reflect"
	"testing"
)

func expect(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func TestDisk(t *testing.T) {
	database_path := t.TempDir()

	db, err := database.NewDatabase(database_path)
	expect(t, err)

	topn := middleware.NewTopNDisk("topn", 5)

	games := []middleware.GameStat{
		{
			AppID: 1,
			Name:  "game1",
			Stat:  1,
		},
		{
			AppID: 2,
			Name:  "game2",
			Stat:  2,
		},
		{
			AppID: 3,
			Name:  "game3",
			Stat:  3,
		},
	}
	for _, g := range games {
		topn.Put(g)
	}

	snapshot, err := db.NewSnapshot()
	expect(t, err)
	err = topn.Save(snapshot)
	expect(t, err)
	err = snapshot.Commit()
	expect(t, err)

	topn2 := middleware.NewTopNDisk("topn", 5)
	err = topn2.LoadDisk(db)
	expect(t, err)

	if !reflect.DeepEqual(topn, topn2) {
		t.Fatalf("Expected %v, but got %v", topn, topn2)
	}
}

func TestOrder(t *testing.T) {
	topn := middleware.NewTopNDisk("topn", 5)

	games := []middleware.GameStat{
		{
			AppID: 1,
			Name:  "game1",
			Stat:  1,
		},
		{
			AppID: 2,
			Name:  "game2",
			Stat:  3,
		},
		{
			AppID: 3,
			Name:  "game3",
			Stat:  2,
		},
		{
			AppID: 4,
			Name:  "game4",
			Stat:  6,
		},
		{
			AppID: 5,
			Name:  "game5",
			Stat:  4,
		},
		{
			AppID: 6,
			Name:  "game6",
			Stat:  5,
		},
	}
	for _, g := range games {
		topn.Put(g)
	}

	top := topn.Get()

	expectedTop := []middleware.GameStat{
		{
			AppID: 4,
			Name:  "game4",
			Stat:  6,
		},
		{
			AppID: 6,
			Name:  "game6",
			Stat:  5,
		},
		{
			AppID: 5,
			Name:  "game5",
			Stat:  4,
		},
		{
			AppID: 2,
			Name:  "game2",
			Stat:  3,
		},
		{
			AppID: 3,
			Name:  "game3",
			Stat:  2,
		},
	}

	if !reflect.DeepEqual(expectedTop, top) {
		t.Fatalf("Expected %v, but got %v", expectedTop, top)
	}
}
