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

// this map has the exchange name as key and the exchange type as value
var DataHandlerexchanges = map[string]string{
	ReviewExchange: amqp.ExchangeFanout,
	GamesExchange:  amqp.ExchangeFanout,
}

// this map has the queue name as key and the exchange name as value
var DataHandlerQueues = map[string]string{
	GamesPartitionerQueue: GamesExchange,
	GamesQueue:            GamesExchange,
	ReviewsQueue:          ReviewExchange,
}

var GenreFilterExchanges = map[string]string{
	GenresExchange: amqp.ExchangeDirect,
}

var GenreFilterQueues = map[string]string{
	GenresQueue: GenresExchange,
}
