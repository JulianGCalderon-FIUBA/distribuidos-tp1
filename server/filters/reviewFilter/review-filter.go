package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
)

type Batch middleware.Batch[middleware.Review]

type ReviewFilter struct {
	cfg config
	m   middleware.Middleware
}

func newReviewFilter(cfg config) (*ReviewFilter, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitReviewFilter()
	if err != nil {
		return nil, err
	}

	return &ReviewFilter{
		cfg: cfg,
		m:   *m}, nil
}

func (rf *ReviewFilter) run() error {
	log.Infof("Starting review filter")
	return rf.receive()
}

// Reads from queue channel and filters read batch before sending it to exchange
func (rf *ReviewFilter) receive() error {
	deliveryCh, err := rf.m.ReceiveFromQueue(middleware.ReviewsQueue)
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

		positive, negative := rf.filterBatch(batch)
		err = rf.sendBatches(positive, negative)
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
	// se va a llamar cuando tengamos algún tipo de finish
	log.Infof("Done sending all filtered reviews")
	return nil
}

// Filters batch of reviews according to score and returns two filtered batches: one with positive reviews and another one with negative reviews
func (rf *ReviewFilter) filterBatch(batch Batch) (Batch, Batch) {
	var positive Batch
	var negative Batch

	for _, review := range batch {
		switch review.Score {
		case middleware.PositiveScore:
			new := middleware.Review{AppID: review.AppID}
			positive = append(positive, new)
		case middleware.NegativeScore:
			new := middleware.Review{AppID: review.AppID, Text: review.Text}
			negative = append(negative, new)
		}
	}
	return positive, negative
}

// Sends batches to corresponding exchanges if they are not empty
func (rf *ReviewFilter) sendBatches(positive Batch, negative Batch) error {
	var err error
	if len(positive) > 0 {
		err = rf.m.Send(positive, middleware.ReviewsScoreFilterExchange, middleware.PositiveReviews)
	}
	if len(negative) > 0 {
		err = rf.m.Send(negative, middleware.ReviewsScoreFilterExchange, middleware.NegativeReviews)
	}
	return err
}
