package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// review filter
const ReviewsScoreFilterExchange string = "review-score-filter"
const NinetyPercentileReviewsQueue string = "90-percentile"
const LanguageReviewsFilterQueue string = "language-filter-reviews"
const FiftyThReviewsQueue string = "50-thousand"

// keys
const PositiveReviews string = "positive-reviews"
const NegativeReviews string = "negative-reviews"
const IndieGames string = "indie"
const ActionGames string = "action"

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
	GenresExchange: amqp.ExchangeDirect,
}

var GenreFilterQueues = map[string]string{
	GamesQueue:             GamesExchange,
	IndiePartitionerQueue:  GenresExchange,
	ActionPartitionerQueue: GenresExchange,
	DecadeQueue:            GenresExchange,
	PercentilQueue:         GenresExchange,
}

var DecadeFilterExchanges = map[string]string{
	DecadeExchange: amqp.ExchangeDirect,
}

var DecadeFilterQueues = map[string]string{
	Top10HistoricAvgQueue: DecadeExchange,
}
