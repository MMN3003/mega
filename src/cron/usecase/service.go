package usecase

import (
	"context"

	"github.com/MMN3003/mega/src/cron/domain"
	"github.com/MMN3003/mega/src/logger"
	"github.com/google/uuid"
)

var _ domain.CronUseCase = (*Service)(nil)

type Service struct {
	cronRepo domain.CronRepository
	logger   *logger.Logger
}

func NewService(cronRepo domain.CronRepository, logg *logger.Logger) *Service {
	s := &Service{
		cronRepo: cronRepo,
		logger:   logg,
	}
	return s
}
func (s *Service) CreateCron(ctx context.Context, id uuid.UUID) error {
	_, err := s.cronRepo.SaveCron(ctx, &domain.Cron{ID: id})
	return err
}
func (s *Service) DeleteCron(ctx context.Context, id uuid.UUID) error {
	return s.cronRepo.DeleteCron(ctx, id)
}
