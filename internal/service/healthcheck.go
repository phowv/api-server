package service

import (
	"context"
	"fmt"
)

type Healthchecker interface {
	Name() string
	Ping(ctx context.Context) error
}

type HealthcheckService struct {
	chekcers []Healthchecker
}

func NewHealthcheckService(checkers []Healthchecker) *HealthcheckService {
	return &HealthcheckService{
		chekcers: checkers,
	}
}

func (s *HealthcheckService) Check(ctx context.Context) error {
	for _, checker := range s.chekcers {
		err := checker.Ping(ctx)
		if err != nil {
			return fmt.Errorf("failed to pass health checker %s: %w", checker.Name(), err)
		}
	}

	return nil
}
