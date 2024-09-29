package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Game struct {
	AppID                  int
	Name                   string
	ReleaseDate            time.Time
	Windows                bool
	Mac                    bool
	Linux                  bool
	AveragePlaytimeForever int
	Genres                 []string
}

func GameFromFullRecord(record []string) (game Game, err error) {
	if len(record) < 37 {
		err = fmt.Errorf("expected 37 fields, got %v", len(record))
		return
	}
	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}
	releaseDate, err := time.Parse("Jan 2, 2006", record[2])
	if err != nil {
		return
	}
	averagePlaytimeForever, err := strconv.Atoi(record[29])
	if err != nil {
		return
	}

	game.AppID = appId
	game.Name = record[1]
	game.ReleaseDate = releaseDate
	game.Windows = record[17] == "true"
	game.Mac = record[18] == "true"
	game.Linux = record[19] == "true"
	game.AveragePlaytimeForever = averagePlaytimeForever
	game.Genres = strings.Split(record[36], ",")

	return
}

type Review struct {
	AppID int
	Text  string
	Score int
}

func ReviewFromFullRecord(record []string) (review Review, err error) {
	if len(record) < 4 {
		err = fmt.Errorf("expected 4 fields, got %v", len(record))
		return
	}
	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}
	score, err := strconv.Atoi(record[3])
	if err != nil {
		return
	}

	review.AppID = appId
	review.Text = record[2]
	review.Score = score

	return
}
