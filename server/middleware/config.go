package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// data Handler
const GamesExchange string = "games"
const ReviewExchange string = "reviews"
const GamesQueue string = "games"
const ReviewsQueue string = "reviews"
const GamesPerPlatformQueue string = "games-per-platform"

// genre Filter
const GenresExchange string = "genres"
const DecadeQueue string = "decades"
const TopNAmountReviewsGamesQueue string = "games-top-n-amount-reviews"
const MoreThanNReviewsGamesQueue string = "games-more-than-n-reviews"
const NinetyPercentileGamesQueue string = "games-90-percentile"
const IndieGameKeys string = "indie"
const ActionGameKeys string = "action"

// review filter
const ReviewsScoreFilterExchange string = "reviews-filter-score"
const NinetyPercentileReviewsQueue string = "reviews-90-percentile"
const LanguageReviewsFilterQueue string = "reviews-language-filter"
const TopNAmountReviewsQueue string = "reviews-top-n-partitioner"
const PositiveReviewKey string = "positive-review"
const NegativeReviewKey string = "negative-review"

// decade filter
const DecadeExchange string = "decades"
const TopNHistoricAvgQueue string = "top-n-historic-avg"

// language filter
const ReviewsEnglishFilterExchange string = "reviews-filter-english"
const NThousandEnglishReviewsQueue string = "reviews-english-n-thousand-partitioner"

type queueConfig struct {
	name       string
	exchange   string
	routingKey string
}
// games per platform
const GamesPerPlatformJoin string = "games-per-platform-join"

// Results
const ResultsQueue string = "results"

var DataHandlerexchanges = map[string]string{
	ReviewExchange: amqp.ExchangeFanout,
	GamesExchange:  amqp.ExchangeFanout,
}

var DataHandlerQueues = []queueConfig{
	{GamesQueue, GamesExchange, ""},
	{ReviewsQueue, ReviewExchange, ""},
	{GamesPerPlatformQueue, GamesExchange, ""},
}

var GenreFilterExchanges = map[string]string{
	GamesExchange:  amqp.ExchangeFanout,
	GenresExchange: amqp.ExchangeDirect,
}

var GenreFilterQueues = []queueConfig{
	{GamesQueue, GamesExchange, ""},
	{DecadeQueue, GenresExchange, IndieGameKeys},
	{TopNAmountReviewsGamesQueue, GenresExchange, IndieGameKeys},
	{MoreThanNReviewsGamesQueue, GenresExchange, ActionGameKeys},
	{NinetyPercentileGamesQueue, GenresExchange, ActionGameKeys},
}

var DecadeFilterExchanges = map[string]string{
	GenresExchange: amqp.ExchangeDirect,
	DecadeExchange: amqp.ExchangeDirect,
}

var DecadeFilterQueues = []queueConfig{
	{DecadeQueue, GenresExchange, ""},
	{TopNHistoricAvgQueue, DecadeExchange, ""},
}
