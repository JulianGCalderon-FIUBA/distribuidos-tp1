package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// review filter
const ReviewsScoreFilterExchange string = "reviews-filter-score"
const NinetyPercentileReviewsQueue string = "reviews-90-percentile"
const LanguageReviewsFilterQueue string = "reviews-language-filter"
const Top5AmountReviewsQueue string = "reviews-top-5-partitioner"

// keys
const PositiveReviews string = "positive-reviews"
const NegativeReviews string = "negative-reviews"
const IndieGames string = "indie"
const ActionGames string = "action"

// language filter
const EnglishReviewsFilterExchange string = "reviews-filter-english"
const FivethEnglishReviewsQueue string = "reviews-english-5-thousand-partitioner"

const ReviewExchange string = "reviews"
const GamesExchange string = "games"
const IndieExchange string = "indie"
const ActionExchange string = "action"
const DecadeExchange string = "decades"

const GamesPartitionerQueue string = "games-partitioner"
const GamesQueue string = "games"
const ReviewsQueue string = "reviews"
const DecadeQueue string = "decades"
const Top5AmountReviews string = "top5-amount-reviews"
const Top10HistoricAvgQueue string = "top10-historic-avg"

var DataHandlerexchanges = map[string]string{
	ReviewExchange: amqp.ExchangeFanout,
	GamesExchange:  amqp.ExchangeFanout,
}

var DataHandlerQueues = map[string]string{
	GamesPartitionerQueue: GamesExchange,
	GamesQueue:            GamesExchange,
	ReviewsQueue:          ReviewExchange,
}

var GenreFilterExchanges = map[string]string{
	IndieExchange:  amqp.ExchangeFanout,
	ActionExchange: amqp.ExchangeFanout,
}

var GenreFilterQueues = map[string]string{
	GamesQueue:                   GamesExchange,
	DecadeQueue:                  IndieExchange,
	Top5AmountReviews:            IndieExchange,
	FivethEnglishReviewsQueue:    ActionExchange,
	NinetyPercentileReviewsQueue: ActionExchange,
}

var DecadeFilterExchanges = map[string]string{
	DecadeExchange: amqp.ExchangeDirect,
}

var DecadeFilterQueues = map[string]string{
	DecadeQueue:           IndieExchange,
	Top10HistoricAvgQueue: DecadeExchange,
}
