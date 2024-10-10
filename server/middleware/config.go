package middleware

import (
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

// El nombrado de las colas y exchanges sigue las siguientes reglas:
// - Si es un exchange, termina en 'x'
// - Si es un cola, el formato es `Tipo-Destino`

// data Handler
const (
	ExchangeGames   string = "games-x"
	ExchangeReviews string = "reviews-x"
	GamesGenre      string = "games-genre"
	ReviewsScore    string = "reviews-score"
	GamesQ1         string = "games-Q1"
)

// genre Filter
const (
	ExchangeGenre string = "genre-x"
	GamesDecade   string = "games-decade"
	GamesQ3       string = "games-q3"
	GamesQ4       string = "games-q4"
	GamesQ5       string = "games-q5"
	IndieKey      string = "indie"
	ActionKey     string = "action"
)

// review filter
const (
	ExchangeScore   string = "score-x"
	ReviewsQ5       string = "reviews-q5"
	ReviewsLanguage string = "reviews-language"
	ReviewsQ3       string = "reviews-q3"
	PositiveKey     string = "positive"
	NegativeKey     string = "negative"
)

// decade filter
const DecadeExchange string = "decades"
const DecadeKey string = "decade"

// language filter
const ReviewsEnglishFilterExchange string = "reviews-filter-english"
const NThousandEnglishReviewsQueue string = "reviews-english-n-thousand-partitioner"
const ReviewsEnglishKey string = "english"

// topNHistoricAvg aggregator
const TopNHistoricAvgPQueue string = "top-n-historic-avg-partitioner"
const TopNHistoricAvgJQueue string = "top-n-historic-avg-joiner"

// results
const ResultsQueue string = "results"

// games per platform
const GamesPerPlatformJoin string = "games-per-platform-join"

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
