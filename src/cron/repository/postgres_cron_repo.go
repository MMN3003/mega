package repository

import (
	"context"
	"errors"
	"time"

	"github.com/MMN3003/mega/src/cron/domain"
	"github.com/MMN3003/mega/src/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ domain.CronRepository = (*CronRepo)(nil)

// ---------- MARKETS ----------
// gorm.Model includes:
// ID        uint `gorm:"primarykey"`
// CreatedAt time.Time
// UpdatedAt time.Time
// DeletedAt gorm.DeletedAt `gorm:"index"`
type Cron struct {
	ID        uuid.UUID `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ---------- REPO ----------

type CronRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewCronRepo(db *gorm.DB, log *logger.Logger) *CronRepo {
	if err := db.AutoMigrate(&Cron{}); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}
	return &CronRepo{db: db, log: log}
}

// ---------- ORDER CRUD ----------

func (r *CronRepo) SaveCron(ctx context.Context, c *domain.Cron) (*domain.Cron, error) {
	model := Cron{
		ID: c.ID,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, err
	}
	return r.GetCronByID(ctx, model.ID)
}

func (r *CronRepo) GetCronByID(ctx context.Context, id uuid.UUID) (*domain.Cron, error) {
	var c Cron
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainCron(&c), nil
}

func (r *CronRepo) DeleteCron(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&Cron{}, id).Error
}

// ---------- HELPERS ----------

func (r *CronRepo) toDomainCron(c *Cron) *domain.Cron {
	return &domain.Cron{
		ID: c.ID,
	}
}
func (r *CronRepo) toDomainCrons(cs []Cron) []domain.Cron {
	var dos []domain.Cron
	for _, o := range cs {
		dos = append(dos, *r.toDomainCron(&o))
	}
	return dos
}
