package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

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

func Filter(r middleware.Review) []string {
	if isEnglish(r.Text) {
		return []string{middleware.EnglishKey}
	}

	return nil
}

// Detects if received text is English or not
func isEnglish(text string) bool {
	info := getlang.FromString(text)
	return info.LanguageName() == ENGLISH
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	filterCfg := middleware.FilterConfig{
		RabbitIP: cfg.RabbitIP,
		Queue:    middleware.ReviewsLanguage,
		Exchange: middleware.ExchangeLanguage,
		QueuesByKey: map[string][]string{
			middleware.EnglishKey: {
				middleware.ReviewsQ4,
			},
		},
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	p, err := middleware.NewFilter(filterCfg, Filter)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
