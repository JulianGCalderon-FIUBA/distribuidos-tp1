package main

import "distribuidos/tp1/server/middleware"

type LanguageFilter struct {
	cfg config
	m   middleware.Middleware
}

func newLanguageFilter(cfg config) (*LanguageFilter, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitReviewFilter()
	if err != nil {
		return nil, err
	}

	return &LanguageFilter{
		cfg: cfg,
		m:   *m}, nil
}

func (lf *LanguageFilter) run() {

}
