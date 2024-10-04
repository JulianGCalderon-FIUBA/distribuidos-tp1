package main

import (
	"distribuidos/tp1/server/middleware"
)

type Batch middleware.Batch[middleware.Review]

type ReviewFilter struct {
	cfg config
	m   middleware.Middleware
	ch  chan Batch
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

	ch := make(chan Batch)

	return &ReviewFilter{
		cfg: cfg,
		m:   *m,
		ch:  ch}, nil
}

func (rf *ReviewFilter) run() error {

	err := rf.receive()
	if err != nil {
		log.Errorf("Failed to receive batches: %v", err)
	}

	return nil
}

func (rf *ReviewFilter) receive() error {
	deliveryCh, err := rf.m.ReceiveFromQueue(middleware.ReviewsFilterQueue)
	if err != nil {
		return err
	}

	for d := range deliveryCh {
		batch, err := middleware.Deserialize[Batch](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch: %v", err)
			_ = d.Nack(false, false)
			continue
		}

		positive, negative := rf.filterBatch(batch)
		if len(positive) > 0 {
			err = rf.m.Send(positive, middleware.ReviewsScoreFilterExchange, middleware.PositiveReviews)
			if err != nil {
				log.Errorf("Failed to send positive reviews batch: %v", err)
				_ = d.Nack(false, false)
				continue
			}
		}
		if len(negative) > 0 {
			err = rf.m.Send(negative, middleware.ReviewsScoreFilterExchange, middleware.NegativeReviews)
			if err != nil {
				log.Errorf("Failed to send negative reviews batch: %v", err)
				_ = d.Nack(false, false)
				continue
			}
		}
		_ = d.Ack(false)
	}
	return nil
}

func (rf *ReviewFilter) filterBatch(batch Batch) (Batch, Batch) {
	var positive Batch
	var negative Batch

	for _, review := range batch {

		switch review.Score {
		case middleware.PositiveScore:
			positive = append(positive, review)
		case middleware.NegativeScore:
			negative = append(negative, review)
		}
	}
	return positive, negative
}
