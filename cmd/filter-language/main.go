package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	lingua "github.com/pemistahl/lingua-go"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

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

type handler struct {
	detector lingua.LanguageDetector
}

func (h handler) Filter(r middleware.Review) []string {
	if h.isEnglish(r.Text) {
		return []string{middleware.EnglishKey}
	}

	return nil
}

// Detects if received text is English or not
func (h handler) isEnglish(text string) bool {
	lang, _ := h.detector.DetectLanguageOf(text)
	return lang == lingua.English
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
	h := handler{
		detector: detector,
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
	p, err := middleware.NewFilter(filterCfg, h.Filter)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
