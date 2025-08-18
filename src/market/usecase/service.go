package usecase

import (
	"context"
	"strconv"

	"github.com/MMN3003/mega/src/Infrastructure/ompfinex"
	"github.com/MMN3003/mega/src/config"
	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/market/domain"
)

type Service struct {
	markets        domain.MarketRepository
	logger         *logger.Logger
	ompfinexClient *ompfinex.Client
}

func NewService(m domain.MarketRepository, logg *logger.Logger, cfg *config.Config) *Service {
	ompfinexClient, _ := ompfinex.NewClient(cfg.OMP.BaseURL,
		ompfinex.WithAuthToken(cfg.OMP.Token),
	)
	s := &Service{
		markets:        m,
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
	return s.markets.UpsertMarketsForExchange(ctx, marketList)
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
	return marketList, s.markets.UpsertMarketsForExchange(ctx, marketList)
}
