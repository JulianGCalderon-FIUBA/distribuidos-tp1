package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

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
