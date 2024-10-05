package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// review filter
const ReviewsScoreFilterExchange string = "reviews-filter-score"
const NinetyPercentileReviewsQueue string = "reviews-90-percentile"
const LanguageReviewsFilterQueue string = "reviews-language-filter"
const Top5ReviewsFilterQueue string = "reviews-top-5-indie"

// keys
const PositiveReviews string = "positive-reviews"
const NegativeReviews string = "negative-reviews"

// language filter
const EnglishReviewsFilterExchange string = "reviews-filter-english"
const FivethEnglishReviewsQueue string = "reviews-english-5-thousand"

const ReviewExchange string = "reviews"
const GamesExchange string = "games"
const GenresExchange string = "genres"
const DecadeExchange string = "decades"

const GamesPartitionerQueue string = "games-partitioner"
const GamesQueue string = "games"
const ReviewsQueue string = "reviews"
const GenresQueue string = "genres"
const DecadeQueue string = "decades"
const IndiePartitionerQueue string = "indie-partitioner"
const ActionPartitionerQueue string = "action-partitioner"
const PercentilQueue string = "percentil"

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
	GenresExchange: amqp.ExchangeDirect,
}

var GenreFilterQueues = map[string]string{
	GamesQueue:             GamesExchange,
	IndiePartitionerQueue:  GenresExchange,
	ActionPartitionerQueue: GenresExchange,
	DecadeQueue:            GenresExchange,
	PercentilQueue:         GenresExchange,
}
