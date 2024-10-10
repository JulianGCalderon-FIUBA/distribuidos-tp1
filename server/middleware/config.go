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
const GenresExchange string = "genres"
const DecadeQueue string = "decades"
const TopNAmountReviewsGamesQueue string = "games-top-n-amount-reviews"
const MoreThanNReviewsGamesQueue string = "games-more-than-n-reviews"
const NinetyPercentileGamesQueue string = "games-90-percentile"
const IndieGameKey string = "indie"
const ActionGameKey string = "action"
const EmptyKey string = "empty"

// review filter
const ReviewsScoreFilterExchange string = "reviews-filter-score"
const NinetyPercentileReviewsQueue string = "reviews-90-percentile"
const LanguageReviewsFilterQueue string = "reviews-language-filter"
const TopNAmountReviewsQueue string = "reviews-top-n-partitioner"
const PositiveReviewKey string = "positive-review"
const NegativeReviewKey string = "negative-review"

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
