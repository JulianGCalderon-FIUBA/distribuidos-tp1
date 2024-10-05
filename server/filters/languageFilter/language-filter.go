package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"

	"github.com/rylans/getlang"
)

type Batch middleware.Batch[middleware.Review]

const ENGLISH = "English"

type LanguageFilter struct {
	cfg config
	m   middleware.Middleware
}

func newLanguageFilter(cfg config) (*LanguageFilter, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitLanguageFilter()
	if err != nil {
		return nil, err
	}

	return &LanguageFilter{
		cfg: cfg,
		m:   *m,
	}, nil
}

func (lf *LanguageFilter) run() error {
	log.Infof("Starting language filter")
	return lf.receive()
}

// Reads from queue channel and filters batch, keeping only reviews in English, before sending to exchange
func (lf *LanguageFilter) receive() error {
	deliveryCh, err := lf.m.ReceiveFromQueue(middleware.LanguageReviewsFilterQueue)
	if err != nil {
		return err
	}

	for d := range deliveryCh {
		batch, err := middleware.Deserialize[Batch](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch: %v", err)
			err = d.Nack(false, false)
			if err != nil {
				return fmt.Errorf("failed to nack batch: %v", err)
			}
			continue
		}
		filtered, err := lf.filterBatch(batch)
		if err != nil {
			log.Errorf("Failed to filter batch: %v", err)
			err = d.Nack(false, false)
			if err != nil {
				return fmt.Errorf("failed to nack batch: %v", err)
			}
			continue
		}
		err = lf.sendBatch(filtered)
		if err != nil {
			log.Errorf("Failed to send batches: %v", err)
			err = d.Nack(false, false)
			if err != nil {
				return fmt.Errorf("failed to nack batch: %v", err)
			}
			continue
		}
		err = d.Ack(false)
		if err != nil {
			return fmt.Errorf("failed to ack batch: %v", err)
		}
	}
	return nil
}

// Filters to keep only reviews in English
func (lf *LanguageFilter) filterBatch(batch Batch) (Batch, error) {
	var english Batch
	for _, review := range batch {
		if lf.isEnglish(review.Text) {
			new := middleware.Review{AppID: review.AppID}
			english = append(english, new)
		}
	}
	return english, nil
}

// Sends batch to corresponding exchange if it's not empty
func (lf *LanguageFilter) sendBatch(batch Batch) error {
	var err error
	if len(batch) > 0 {
		err = lf.m.Send(batch, middleware.ReviewsEnglishFilterExchange, "")
	}
	return err
}

// Detects if received text is English or not
func (lf *LanguageFilter) isEnglish(text string) bool {
	info := getlang.FromString(text)
	return info.LanguageName() == ENGLISH
}
