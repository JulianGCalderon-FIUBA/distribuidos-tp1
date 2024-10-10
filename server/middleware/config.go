package middleware

import (
	"strings"

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
	ReviewExchange: amqp.ExchangeFanout,
	GamesExchange:  amqp.ExchangeFanout,
}

var DataHandlerQueues = []queueConfig{
	{GamesQueue, GamesExchange, ""},
	{ReviewsQueue, ReviewExchange, ""},
	{GamesPerPlatformQueue, GamesExchange, ""},
}

func Cat(v ...string) string {
	return strings.Join(v, "-")
}
