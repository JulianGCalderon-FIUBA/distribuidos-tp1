package middleware

type Score int8

const (
	PositiveScore Score = 1
	NegativeScore Score = -1
)

type Date struct {
	Day   uint8
	Month uint8
	Year  uint16
}

type Game struct {
	AppID                  uint64
	Name                   string
	ReleaseDate            Date
	Windows                bool
	Mac                    bool
	Linux                  bool
	AveragePlaytimeForever uint64
	Genres                 []string
}

type Review struct {
	AppID uint64
	Text  string
	Score Score
}
