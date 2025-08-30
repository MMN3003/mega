package domain

import (
	"context"
)

// MarketRepository persistence port
type MarketRepository interface {
	SaveMarket(ctx context.Context, m *Market) error
	GetMarketByID(ctx context.Context, id uint) (*Market, error)
	UpdateMarket(ctx context.Context, m *Market) error
	SoftDelete(ctx context.Context, id uint) error

	GetMarketsByExchangeName(ctx context.Context, exchangeName string) ([]*Market, error)
	GetMarketsByMarketName(ctx context.Context, marketName string) ([]*Market, error)
	UpsertMarketsForExchange(ctx context.Context, markets []Market) error
	GetMarketsByMegaMarketId(ctx context.Context, megaMarketId uint) ([]*Market, error)
}

// MegaMarketRepository persistence port
type MegaMarketRepository interface {
	SaveMegaMarket(ctx context.Context, m *MegaMarket) error
	GetMegaMarketByID(ctx context.Context, id uint) (*MegaMarket, error)
	UpdateMegaMarket(ctx context.Context, m *MegaMarket) error
	SoftDeleteMegaMarket(ctx context.Context, id uint) error
	GetActiveMegaMarketByID(ctx context.Context, id uint) (*MegaMarket, error)
	GetAllActiveMegaMarkets(ctx context.Context) ([]*MegaMarket, error)
}
