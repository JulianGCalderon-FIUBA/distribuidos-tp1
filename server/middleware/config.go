package middleware

import (
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

// El nombrado de las colas y exchanges sigue las siguientes reglas:
// - Si es un exchange, termina en 'x'
// - Si es un cola, el formato es `TipoDato-Query-Destino`

// Data Handler
const (
	ExchangeGames   string = "games-x"
	ExchangeReviews string = "reviews-x"
)

// Genre Filter
const (
	GamesGenre    string = "games-genre"
	ExchangeGenre string = "genre-x"
	IndieKey      string = "indie"
	ActionKey     string = "action"
)

// Score filter
const (
	ReviewsScore  string = "reviews-score"
	ExchangeScore string = "score-x"
	PositiveKey   string = "positive"
	NegativeKey   string = "negative"
)

// Decade filter
const (
	GamesDecade     string = "games-decade"
	ExchangeDecade  string = "decade-x"
	DecadeKeyPrefix string = "decade"
)

// Language filter
const (
	ReviewsLanguage  string = "reviews-language"
	ExchangeLanguage string = "language-x"
	EnglishKey       string = "english"
)

// Q1
const (
	GamesQ1   string = "games-Q1"
	PartialQ1 string = "partial-Q1-joiner"
)

// Q2
const (
	GamesQ2   string = "games-Q2"
	PartialQ2 string = "partial-Q2-joiner"
)

// Q3
const (
	GamesQ3   string = "games-Q3"
	ReviewsQ3 string = "reviews-Q3"
	GroupedQ3 string = "grouped-Q3-top"
	PartialQ3 string = "partial-Q3-joiner"
)

// Q4
const (
	GamesQ4             string = "games-Q4"
	ReviewsQ4           string = "reviews-Q4"
	GroupedQ4Joiner     string = "grouped-Q4-joiner"
	GroupedQ4Percentile string = "grouped-Q4-percentile"
)

// Q5
const (
	GamesQ5         string = "games-Q5"
	ReviewsQ5       string = "reviews-Q5"
	GroupedQ5Joiner string = "grouped-Q5-joiner"
	GroupedQ5Filter string = "grouped-Q5-filter"
)

// Results
const (
	Results string = "results"
)

type queueConfig struct {
	name       string
	exchange   string
	routingKey string
}

var DataHandlerexchanges = map[string]string{
	ExchangeReviews: amqp.ExchangeFanout,
	ExchangeGames:   amqp.ExchangeFanout,
}

var DataHandlerQueues = []queueConfig{
	{GamesGenre, ExchangeGames, ""},
	{ReviewsScore, ExchangeReviews, ""},
	{GamesQ1, ExchangeGames, ""},
}

func Cat(v ...string) string {
	return strings.Join(v, "-")
}
