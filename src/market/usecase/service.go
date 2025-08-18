package usecase

import (
	"context"
	"errors"
	"strconv"

	"github.com/MMN3003/mega/src/Infrastructure/ompfinex"
	"github.com/MMN3003/mega/src/config"
	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/market/domain"
	"github.com/shopspring/decimal"
)

type Service struct {
	marketsRepo    domain.MarketRepository
	logger         *logger.Logger
	ompfinexClient *ompfinex.Client
}

func NewService(m domain.MarketRepository, logg *logger.Logger, cfg *config.Config) *Service {
	ompfinexClient, _ := ompfinex.NewClient(cfg.OMP.BaseURL,
		ompfinex.WithAuthToken(cfg.OMP.Token),
	)
	s := &Service{
		marketsRepo:    m,
		logger:         logg,
		ompfinexClient: ompfinexClient,
	}
	return s
}

func (s *Service) UpsertMarketPairs(ctx context.Context, exchangeName string, markets []string) error {

	var marketList []domain.Market
	for _, market := range markets {
		marketList = append(marketList, domain.Market{
			ExchangeName:             exchangeName,
			MarketName:               market,
			IsActive:                 true,
			ExchangeMarketIdentifier: market,
		})
	}
	return s.marketsRepo.UpsertMarketsForExchange(ctx, marketList)
}

func (s *Service) FetchAndUpdateMarkets(ctx context.Context) ([]domain.Market, error) {

	markets, err := s.ompfinexClient.ListMarkets(ctx)
	if err != nil {
		s.logger.Errorf("failed to fetch markets: %v", err)
		return nil, err
	}

	var marketList []domain.Market
	for _, market := range markets {
		// log object market
		s.logger.Infof("fetched market: %+v", market)
		marketList = append(marketList, domain.Market{
			ExchangeName:             "ompfinex",
			MarketName:               market.BaseCurrency.ID + "/" + market.QuoteCurrency.ID,
			IsActive:                 true,
			ExchangeMarketIdentifier: strconv.FormatInt(market.ID, 10),
		})
	}
	return marketList, s.marketsRepo.UpsertMarketsForExchange(ctx, marketList)
}

func (s *Service) GetBestExchangePriceByVolume(ctx context.Context, marketName string, volume decimal.Decimal) (decimal.Decimal, string, error) {
	markets, err := s.marketsRepo.GetMarketsByMarketName(ctx, marketName)
	if err != nil {
		s.logger.Errorf("failed to get markets by market name: %v", err)
		return decimal.Zero, "", err
	}
	if len(markets) == 0 {
		s.logger.Errorf("no markets found for market name: %s", marketName)
		return decimal.Zero, "", errors.New("no markets found for market name")
	}
	minPrice := decimal.Zero
	exchangeName := ""
	for _, market := range markets {
		if market.ExchangeName == "ompfinex" {
			exchangeName = market.ExchangeName
			depth, err := s.ompfinexClient.GetMarketDepth(ctx, market.ExchangeMarketIdentifier)
			if err != nil {
				s.logger.Errorf("failed to get market depth: %v", err)
				return decimal.Zero, "", err
			}
			price, err := s.calculateOmpfinexPrice(depth, volume)
			if err != nil {
				s.logger.Errorf("failed to calculate price: %v", err)
				return decimal.Zero, "", err
			}
			if minPrice.IsZero() || price.LessThan(minPrice) {
				minPrice = price
			}
		}
	}
	return minPrice, exchangeName, nil
}

// calculateOmpfinexPrice calculates the price to buy the requested volume
func (s *Service) calculateOmpfinexPrice(depth ompfinex.OrderBook, volume decimal.Decimal) (decimal.Decimal, error) {
	if volume.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, errors.New("volume must be positive")
	}

	totalVolume := decimal.Zero
	totalCost := decimal.Zero

	for _, ask := range depth.Asks {
		if len(ask) != 2 {
			continue // skip malformed entry
		}

		price, err := decimal.NewFromString(ask[0])
		if err != nil {
			continue
		}

		availVol, err := decimal.NewFromString(ask[1])
		if err != nil {
			continue
		}

		if availVol.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// If this level covers remaining volume
		if totalVolume.Add(availVol).GreaterThanOrEqual(volume) {
			needed := volume.Sub(totalVolume)
			totalCost = totalCost.Add(price.Mul(needed))
			return totalCost.Div(volume), nil // weighted average price
		}

		// Consume full level
		totalCost = totalCost.Add(price.Mul(availVol))
		totalVolume = totalVolume.Add(availVol)
	}

	return decimal.Zero, errors.New("not enough volume in order book")
}
