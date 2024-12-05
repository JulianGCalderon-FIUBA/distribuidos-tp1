package middleware

import (
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

// El nombrado de las colas y exchanges sigue las siguientes reglas:
// - Si es un exchange, termina en 'x'
// - Si es un cola, el formato es `TipoDato-Query-Destino`

// Data Handler
const (
	ExchangeGames   string = "games-x"
	ExchangeReviews string = "reviews-x"
)

// Genre Filter
const (
	GamesGenre    string = "games-genre"
	ExchangeGenre string = "genre-x"
	IndieKey      string = "indie"
	ActionKey     string = "action"
)

// Score filter
const (
	ReviewsScore  string = "reviews-score"
	ExchangeScore string = "score-x"
	PositiveKey   string = "positive"
	NegativeKey   string = "negative"
)

// Decade filter
const (
	GamesDecade     string = "games-decade"
	ExchangeDecade  string = "decade-x"
	DecadeKeyPrefix string = "decade"
)

// Language filter
const (
	ReviewsLanguage  string = "reviews-language"
	ExchangeLanguage string = "language-x"
	EnglishKey       string = "english"
)

// Q1
const (
	GamesQ1   string = "games-Q1"
	PartialQ1 string = "partial-Q1-joiner"
)

// Q2
const (
	GamesQ2   string = "games-Q2"
	PartialQ2 string = "partial-Q2-joiner"
)

// Q3
const (
	GamesQ3   string = "games-Q3"
	ReviewsQ3 string = "reviews-Q3"
	GroupedQ3 string = "grouped-Q3-top"
	PartialQ3 string = "partial-Q3-joiner"
)

// Q4
const (
	GamesQ4         string = "games-Q4"
	ReviewsQ4       string = "reviews-Q4"
	GroupedQ4Joiner string = "grouped-Q4-joiner"
	GroupedQ4Filter string = "grouped-Q4-filter"
	KeyQ4           string = "Q4key"
	ExchangeQ4      string = "Q4-x"
)

// Q5
const (
	GamesQ5             string = "games-Q5"
	ReviewsQ5           string = "reviews-Q5"
	GroupedQ5Joiner     string = "grouped-Q5-joiner"
	GroupedQ5Percentile string = "grouped-Q5-percentil"
)

// Results
const (
	Results   string = "results"
	ResultsQ4 string = "results-Q4"
)

func Cat(v ...any) string {
	vs := make([]string, len(v))
	for i, v := range v {
		vs[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(vs, "-")
}

func Dial(ip string) (*amqp.Connection, *amqp.Channel, error) {
	addr := fmt.Sprintf("amqp://guest:guest@%v:5672/", ip)
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}
