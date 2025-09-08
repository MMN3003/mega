package repository

import (
	"context"
	"errors"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/market/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var _ domain.MegaMarketRepository = (*MegaMarketRepo)(nil)

// ---------- MARKETS ----------
// gorm.Model includes:
// ID        uint `gorm:"primarykey"`
// CreatedAt time.Time
// UpdatedAt time.Time
// DeletedAt gorm.DeletedAt `gorm:"index"`
type MegaMarket struct {
	gorm.Model
	ExchangeMarketNames    string
	IsActive               bool `gorm:"not null;default:true"`
	FeePercentage          decimal.Decimal
	SourceTokenSymbol      string
	DestinationTokenSymbol string
	SlipagePercentage      decimal.Decimal
}

// ---------- REPO ----------

type MegaMarketRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func (r *MegaMarketRepo) Seed(ctx context.Context) error {
	// Check if the table already has data
	var count int64
	if err := r.db.WithContext(ctx).Model(&MegaMarket{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		r.log.Infof("MegaMarket table already seeded with %d records", count)
		return nil
	}

	// Define your seed data
	seedData := []MegaMarket{
		{
			ExchangeMarketNames:    `["BTC/USDT","Bitcoin/Tether"]`,
			IsActive:               true,
			FeePercentage:          decimal.NewFromFloat(0.01), // TODO: change this
			SourceTokenSymbol:      "BTC",
			DestinationTokenSymbol: "USDT",
			SlipagePercentage:      decimal.NewFromFloat(0.02),
		},
		{
			ExchangeMarketNames:    `["DOGE/USDT","Dogecoin/Tether"]`,
			IsActive:               true,
			FeePercentage:          decimal.NewFromFloat(0.01), // TODO: change this
			SourceTokenSymbol:      "DOGE",
			DestinationTokenSymbol: "USDT",
			SlipagePercentage:      decimal.NewFromFloat(0.02),
		},
		{
			ExchangeMarketNames:    `["ETH/USDT","Ethereum/Tether"]`,
			IsActive:               true,
			FeePercentage:          decimal.NewFromFloat(0.01), // TODO: change this
			SourceTokenSymbol:      "ETH",
			DestinationTokenSymbol: "USDT",
			SlipagePercentage:      decimal.NewFromFloat(0.02),
		},
	}

	// Insert seed data
	if err := r.db.WithContext(ctx).Create(&seedData).Error; err != nil {
		return err
	}

	r.log.Infof("Seeded MegaMarket table with %d records", len(seedData))
	return nil
}
func NewMegaMarketRepo(db *gorm.DB, log *logger.Logger) *MegaMarketRepo {
	if err := db.AutoMigrate(&MegaMarket{}); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}

	repo := &MegaMarketRepo{db: db, log: log}
	repo.Seed(context.Background())
	return repo
}

// ---------- MARKET CRUD ----------

func (r *MegaMarketRepo) SaveMegaMarket(ctx context.Context, m *domain.MegaMarket) error {
	model := MegaMarket{
		ExchangeMarketNames:    m.ExchangeMarketNames,
		IsActive:               m.IsActive,
		FeePercentage:          m.FeePercentage,
		SourceTokenSymbol:      m.SourceTokenSymbol,
		DestinationTokenSymbol: m.DestinationTokenSymbol,
		SlipagePercentage:      m.SlipagePercentage,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *MegaMarketRepo) GetMegaMarketByID(ctx context.Context, id uint) (*domain.MegaMarket, error) {
	var m MegaMarket
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainMegaMarket(&m), nil
}
func (r *MegaMarketRepo) GetActiveMegaMarketByID(ctx context.Context, id uint) (*domain.MegaMarket, error) {
	var m MegaMarket
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainMegaMarket(&m), nil
}
func (r *MegaMarketRepo) SoftDeleteMegaMarket(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&MegaMarket{}, id).Error
}
func (r *MegaMarketRepo) UpdateMegaMarket(ctx context.Context, m *domain.MegaMarket) error {
	return r.db.WithContext(ctx).Model(&MegaMarket{}).
		Where("id = ?", m.ID).
		Updates(MegaMarket{
			ExchangeMarketNames:    m.ExchangeMarketNames,
			IsActive:               m.IsActive,
			FeePercentage:          m.FeePercentage,
			SourceTokenSymbol:      m.SourceTokenSymbol,
			DestinationTokenSymbol: m.DestinationTokenSymbol,
			SlipagePercentage:      m.SlipagePercentage,
		}).Error
}

func (r *MegaMarketRepo) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&MegaMarket{}, id).Error
}
func (r *MegaMarketRepo) GetAllActiveMegaMarkets(ctx context.Context) ([]domain.MegaMarket, error) {
	var ms []MegaMarket
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&ms).Error; err != nil {
		return nil, err
	}
	return r.toDomainMegaMarkets(ms), nil
}

// ---------- HELPERS ----------

func (r *MegaMarketRepo) toDomainMegaMarkets(ms []MegaMarket) []domain.MegaMarket {
	var dms []domain.MegaMarket
	for _, m := range ms {
		dms = append(dms, *r.toDomainMegaMarket(&m))
	}
	return dms
}
func (r *MegaMarketRepo) toDomainMegaMarket(m *MegaMarket) *domain.MegaMarket {
	return &domain.MegaMarket{
		ID:                     m.ID,
		ExchangeMarketNames:    m.ExchangeMarketNames,
		IsActive:               m.IsActive,
		FeePercentage:          m.FeePercentage,
		SourceTokenSymbol:      m.SourceTokenSymbol,
		DestinationTokenSymbol: m.DestinationTokenSymbol,
		SlipagePercentage:      m.SlipagePercentage,
	}
}
