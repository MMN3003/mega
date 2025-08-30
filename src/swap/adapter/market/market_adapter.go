package market

import (
	"context"

	"github.com/MMN3003/mega/src/market/domain"
)

type MarketAdapter interface {
	GetMarketByID(ctx context.Context, id uint) (*domain.Market, error)
}

var _ MarketAdapter = (*MarketPort)(nil)

// init market port
func NewMarketPort(marketService domain.MarketUseCase) MarketAdapter {
	return &MarketPort{marketService: marketService}
}

type MarketPort struct {
	marketService domain.MarketUseCase
}

func (m *MarketPort) GetMarketByID(ctx context.Context, id uint) (*domain.Market, error) {
	return m.marketService.GetMarketByID(ctx, id)
}
