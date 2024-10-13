package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/server/middleware/node"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	lingua "github.com/pemistahl/lingua-go"

	"github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
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

type handler struct{ detector lingua.LanguageDetector }

func (h handler) Filter(r middleware.Review) []string {
	lang, _ := h.isEnglish(r.Text)
	if lang == lingua.English {
		return []string{middleware.EnglishKey}
	}

	return nil
}

// Detects if received text is English or not
func (h handler) isEnglish(text string) (lingua.Language, bool) {
	return h.detector.DetectLanguageOf(text)
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	languages := []lingua.Language{
		lingua.English,
		lingua.Spanish,
	}
	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()

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
	h := handler{detector}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
