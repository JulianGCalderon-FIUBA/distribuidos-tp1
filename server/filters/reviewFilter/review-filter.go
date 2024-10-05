package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
)

type Batch middleware.Batch[middleware.Review]

type ReviewFilter struct {
	cfg config
	m   middleware.Middleware
	p   int
	n   int
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

// Reads from queue channel and filters batch in positive and negative reviews before sending it to exchange
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
		if batch.EOF {
			log.Infof("Sent %v positive reviews from client %v", rf.p, batch.ClientID)
			log.Infof("Sent %v negative reviews from client %v", rf.n, batch.ClientID)
			rf.p = 0
			rf.n = 0
		}
	}
	return nil
}

// Filters batch of reviews according to score and returns two filtered batches: one with positive reviews and another one with negative reviews
func (rf *ReviewFilter) filterBatch(batch Batch) (Batch, Batch) {
	positive := Batch{
		Data:     []middleware.Review{},
		ClientID: batch.ClientID,
		BatchID:  batch.BatchID,
		EOF:      batch.EOF,
	}
	negative := Batch{
		Data:     []middleware.Review{},
		ClientID: batch.ClientID,
		BatchID:  batch.BatchID,
		EOF:      batch.EOF,
	}

	for _, review := range batch.Data {
		switch review.Score {
		case middleware.PositiveScore:
			new := middleware.Review{AppID: review.AppID}
			positive.Data = append(positive.Data, new)
		case middleware.NegativeScore:
			new := middleware.Review{AppID: review.AppID, Text: review.Text}
			negative.Data = append(negative.Data, new)
		}
	}

	rf.p += len(positive.Data)
	rf.n += len(negative.Data)

	return positive, negative
}

// Sends batches to corresponding exchanges if they are not empty
func (rf *ReviewFilter) sendBatches(positive Batch, negative Batch) error {
	var err error
	err = rf.m.Send(positive, middleware.ReviewsScoreFilterExchange, middleware.PositiveReviewKeys)
	if err != nil {
		return err
	}
	err = rf.m.Send(negative, middleware.ReviewsScoreFilterExchange, middleware.NegativeReviewKeys)
	if err != nil {
		return err
	}

	return nil
}
