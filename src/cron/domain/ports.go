package domain

import (
	"context"

	"github.com/google/uuid"
)

type CronRepository interface {
	SaveCron(ctx context.Context, c *Cron) (*Cron, error)
	DeleteCron(ctx context.Context, id uuid.UUID) error
}

type CronUseCase interface {
	CreateCron(ctx context.Context, id uuid.UUID) error
	DeleteCron(ctx context.Context, id uuid.UUID) error
}
