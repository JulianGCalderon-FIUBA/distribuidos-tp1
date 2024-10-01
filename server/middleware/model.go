package middleware

import "time"

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

type Review struct {
	AppID int
	Text  string
	Score int
}
