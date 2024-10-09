package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/utils"

	"github.com/op/go-logging"
	"github.com/rylans/getlang"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

const ENGLISH = "English"

type config struct {
	RabbitIP string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct{}

func (h handler) Filter(r middleware.Review) []filter.RoutingKey {
	if h.isEnglish(r.Text) {
		return []filter.RoutingKey{filter.RoutingKey(middleware.ReviewsEnglishKey)}
	}

	return []filter.RoutingKey{}
}

// Detects if received text is English or not
func (h handler) isEnglish(text string) bool {
	info := getlang.FromString(text)
	return info.LanguageName() == ENGLISH
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	filterCfg := filter.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    middleware.LanguageReviewsFilterQueue,
		Exchange: middleware.ReviewsEnglishFilterExchange,
		Output: map[filter.RoutingKey][]filter.QueueName{
			filter.RoutingKey(middleware.ReviewsEnglishKey): {
				filter.QueueName(middleware.NThousandEnglishReviewsQueue),
			},
		},
	}

	h := handler{}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(context.Background())
	utils.Expect(err, "Failed to run filter")
}
