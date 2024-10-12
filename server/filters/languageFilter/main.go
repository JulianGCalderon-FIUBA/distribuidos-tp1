package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/middleware/filter"
	"distribuidos/tp1/middleware/node"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
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

func (h handler) Filter(r middleware.Review) []string {
	if h.isEnglish(r.Text) {
		return []string{middleware.EnglishKey}
	}

	return nil
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
		Queue:    middleware.ReviewsLanguage,
		Exchange: node.ExchangeConfig{
			Name: middleware.ExchangeLanguage,
			Type: amqp.ExchangeDirect,
			QueuesByKey: map[string][]string{
				middleware.EnglishKey: {
					middleware.ReviewsQ4,
				},
			},
		},
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	h := handler{}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
