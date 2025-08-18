package repository

import (
	"context"
	"errors"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/market/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ domain.MarketRepository = (*Repo)(nil)

// ---------- MARKETS ----------
// gorm.Model includes:
// ID        uint `gorm:"primarykey"`
// CreatedAt time.Time
// UpdatedAt time.Time
// DeletedAt gorm.DeletedAt `gorm:"index"`
type Market struct {
	gorm.Model
	ExchangeMarketIdentifier string `gorm:"not null;uniqueIndex:uidx_exchange_market"`
	ExchangeName             string `gorm:"not null;uniqueIndex:uidx_exchange_market"`
	MarketName               string `gorm:"not null;index:idx_market"`
	IsActive                 bool   `gorm:"not null;default:true"`
}

// ---------- REPO ----------

type Repo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewRepo(db *gorm.DB, log *logger.Logger) *Repo {
	if err := db.AutoMigrate(&Market{}); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}
	return &Repo{db: db, log: log}
}

// ---------- MARKET CRUD ----------

func (r *Repo) SaveMarket(ctx context.Context, m *domain.Market) error {
	model := Market{
		ExchangeMarketIdentifier: m.ExchangeMarketIdentifier,
		ExchangeName:             m.ExchangeName,
		MarketName:               m.MarketName,
		IsActive:                 m.IsActive,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *Repo) GetMarketByID(ctx context.Context, id uint) (*domain.Market, error) {
	var m Market
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainMarket(&m), nil
}

func (r *Repo) UpdateMarket(ctx context.Context, m *domain.Market) error {
	return r.db.WithContext(ctx).Model(&Market{}).
		Where("id = ?", m.ID).
		Updates(Market{
			ExchangeMarketIdentifier: m.ExchangeMarketIdentifier,
			ExchangeName:             m.ExchangeName,
			MarketName:               m.MarketName,
			IsActive:                 m.IsActive,
		}).Error
}

func (r *Repo) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Market{}, id).Error
}

// Indexed fetch: by ExchangeName
func (r *Repo) GetMarketsByExchangeName(ctx context.Context, exchangeName string) ([]*domain.Market, error) {
	var models []Market
	if err := r.db.WithContext(ctx).
		Where("exchange_name = ?", exchangeName).
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Market, 0, len(models))
	for _, m := range models {
		out = append(out, r.toDomainMarket(&m))
	}
	return out, nil
}

// Indexed fetch: by MarketName
func (r *Repo) GetMarketsByMarketName(ctx context.Context, marketName string) ([]*domain.Market, error) {
	var models []Market
	if err := r.db.WithContext(ctx).
		Where("market_name = ?", marketName).
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Market, 0, len(models))
	for _, m := range models {
		out = append(out, r.toDomainMarket(&m))
	}
	return out, nil
}

// UpsertMarketsForExchange inserts or updates a batch of markets for an exchange.
func (r *Repo) UpsertMarketsForExchange(ctx context.Context, markets []domain.Market) error {
	var models []Market
	for _, m := range markets {
		models = append(models, Market{
			ExchangeMarketIdentifier: m.ExchangeMarketIdentifier,
			ExchangeName:             m.ExchangeName,
			MarketName:               m.MarketName,
			IsActive:                 m.IsActive,
		})
	}

	// Use GORM upsert with PostgreSQL ON CONFLICT
	// conflict target: exchange_identifier + market_name (you should define a unique index on these two columns!)
	if err := r.db.WithContext(ctx).
		Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "exchange_market_identifier"}, {Name: "exchange_name"}},
				DoUpdates: clause.AssignmentColumns([]string{"exchange_name", "is_active", "updated_at"}),
			},
		).
		Create(&models).Error; err != nil {
		r.log.Errorf("failed to upsert markets for exchange=%s: %v", markets[0].ExchangeName, err)
		return err
	}

	return nil
}

// ---------- HELPERS ----------

func (r *Repo) toDomainMarket(m *Market) *domain.Market {
	return &domain.Market{
		ID:                       m.ID,
		ExchangeMarketIdentifier: m.ExchangeMarketIdentifier,
		ExchangeName:             m.ExchangeName,
		MarketName:               m.MarketName,
		IsActive:                 m.IsActive,
	}
}
