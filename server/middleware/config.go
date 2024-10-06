package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// Data Handler
const GamesExchange string = "games"
const ReviewExchange string = "reviews"
const GamesQueue string = "games"
const ReviewsQueue string = "reviews"
const GamesPerPlatformQueue string = "games-per-platform"

// Genre Filter
const IndieExchange string = "indie"
const ActionExchange string = "action"
const DecadeQueue string = "decades"
const TopNAmountReviewsGamesQueue string = "games-top-n-amount-reviews"
const MoreThanNReviewsGamesQueue string = "games-more-than-n-reviews"
const NinetyPercentileGamesQueue string = "games-90-percentile"
const IndieGameKeys string = "indie"
const ActionGameKeys string = "action"

// Review filter
const ReviewsScoreFilterExchange string = "reviews-filter-score"
const NinetyPercentileReviewsQueue string = "reviews-90-percentile"
const LanguageReviewsFilterQueue string = "reviews-language-filter"
const TopNAmountReviewsQueue string = "reviews-top-n-partitioner"
const PositiveReviewKeys string = "positive-reviews"
const NegativeReviewKeys string = "negative-reviews"

// Decade filter
const DecadeExchange string = "decades"
const TopNHistoricAvgQueue string = "top-n-historic-avg"

// language filter
const ReviewsEnglishFilterExchange string = "reviews-filter-english"
const NThousandEnglishReviewsQueue string = "reviews-english-n-thousand-partitioner"

// Results
const ResultsQueue string = "results"

var DataHandlerexchanges = map[string]string{
	ReviewExchange: amqp.ExchangeFanout,
	GamesExchange:  amqp.ExchangeFanout,
}

var DataHandlerQueues = map[string]string{
	GamesPerPlatformQueue: GamesExchange,
	GamesQueue:            GamesExchange,
	ReviewsQueue:          ReviewExchange,
}

var GenreFilterExchanges = map[string]string{
	GamesExchange:  amqp.ExchangeFanout,
	IndieExchange:  amqp.ExchangeFanout,
	ActionExchange: amqp.ExchangeFanout,
}

var GenreFilterQueues = map[string]string{
	GamesQueue:                  GamesExchange,
	DecadeQueue:                 IndieExchange,
	TopNAmountReviewsGamesQueue: IndieExchange,
	MoreThanNReviewsGamesQueue:  ActionExchange,
	NinetyPercentileGamesQueue:  ActionExchange,
}

var DecadeFilterExchanges = map[string]string{
	IndieExchange:  amqp.ExchangeFanout,
	DecadeExchange: amqp.ExchangeDirect,
}

var DecadeFilterQueues = map[string]string{
	DecadeQueue:          IndieExchange,
	TopNHistoricAvgQueue: DecadeExchange,
}
