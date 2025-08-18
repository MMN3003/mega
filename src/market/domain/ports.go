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
}
