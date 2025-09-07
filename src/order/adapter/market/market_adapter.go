package market

import (
	"context"

	"github.com/MMN3003/mega/src/market/domain"
	"github.com/shopspring/decimal"
)

type MarketAdapter interface {
	GetMarketByID(ctx context.Context, id uint) (*domain.Market, error)
	GetMegaMarketByID(ctx context.Context, id uint) (*domain.MegaMarket, error)
	GetBestExchangePriceByVolume(ctx context.Context, megaMarketId uint, volume decimal.Decimal, isBuy bool) (decimal.Decimal, *domain.Market, *domain.MegaMarket, error)
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

func (m *MarketPort) GetMegaMarketByID(ctx context.Context, id uint) (*domain.MegaMarket, error) {
	return m.marketService.GetMegaMarketByID(ctx, id)
}

func (m *MarketPort) GetBestExchangePriceByVolume(ctx context.Context, megaMarketId uint, volume decimal.Decimal, isBuy bool) (decimal.Decimal, *domain.Market, *domain.MegaMarket, error) {
	return m.marketService.GetBestExchangePriceByVolume(ctx, megaMarketId, volume, isBuy)
}
