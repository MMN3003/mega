package market

import (
	"context"

	"github.com/MMN3003/mega/src/cron/domain"
	"github.com/google/uuid"
)

type CronAdapter interface {
	CreateCron(ctx context.Context, id uuid.UUID) error
	DeleteCron(ctx context.Context, id uuid.UUID) error
}

var _ CronAdapter = (*CronPort)(nil)

// init market port
func NewCronPort(cronService domain.CronUseCase) CronAdapter {
	return &CronPort{cronService: cronService}
}

type CronPort struct {
	cronService domain.CronUseCase
}

func (m *CronPort) CreateCron(ctx context.Context, id uuid.UUID) error {
	return m.cronService.CreateCron(ctx, id)
}

func (m *CronPort) DeleteCron(ctx context.Context, id uuid.UUID) error {
	return m.cronService.DeleteCron(ctx, id)
}
