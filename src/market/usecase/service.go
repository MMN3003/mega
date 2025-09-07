package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/MMN3003/mega/src/Infrastructure/ompfinex"
	"github.com/MMN3003/mega/src/Infrastructure/wallex"
	"github.com/MMN3003/mega/src/config"
	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/market/domain"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type MarketService struct {
	marketsRepo    domain.MarketRepository
	megaMarketRepo domain.MegaMarketRepository
	logger         *logger.Logger
	ompfinexClient *ompfinex.Client
	wallexClient   *wallex.Client
}

func NewService(m domain.MarketRepository, megaMarketRepo domain.MegaMarketRepository, logg *logger.Logger, cfg *config.Config) *MarketService {
	ompfinexClient, _ := ompfinex.NewClient(cfg.OMP.BaseURL,
		ompfinex.WithAuthToken(cfg.OMP.Token),
	)
	wallexClient, _ := wallex.NewClient(cfg.Wallex.BaseURL,
		wallex.WithAPIKey(cfg.Wallex.APIKey),
	)
	s := &MarketService{
		marketsRepo:    m,
		megaMarketRepo: megaMarketRepo,
		logger:         logg,
		ompfinexClient: ompfinexClient,
		wallexClient:   wallexClient,
	}
	return s
}

func (s *MarketService) UpsertMarketPairs(ctx context.Context, exchangeName string, markets []string) error {

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

func (s *MarketService) FetchAndUpdateMarkets(ctx context.Context) ([]domain.Market, map[uint]domain.MegaMarket, error) {
	// --- Step 1: Load MegaMarkets
	megaMarkets, err := s.megaMarketRepo.GetAllActiveMegaMarkets(ctx)
	if err != nil {
		s.logger.Errorf("failed to fetch mega markets: %v", err)
		return nil, nil, err
	}
	// create maga market map id => mega market
	megaMarketMap := make(map[uint]domain.MegaMarket, len(megaMarkets))
	for _, megaMarket := range megaMarkets {
		megaMarketMap[megaMarket.ID] = megaMarket
	}

	// Build lookup map: marketName -> MegaMarketID
	marketNamesMap := make(map[string]uint, len(megaMarkets))
	for _, megaMarket := range megaMarkets {
		var marketNames []string
		if err := json.Unmarshal([]byte(megaMarket.ExchangeMarketNames), &marketNames); err != nil {
			s.logger.Errorf("failed to unmarshal market identifiers for megaMarket=%d: %v", megaMarket.ID, err)
			return nil, nil, err
		}
		for _, name := range marketNames {
			marketNamesMap[name] = megaMarket.ID
		}
	}

	// --- Step 2: Fetch markets concurrently
	var (
		allMarketsMu sync.Mutex
		allMarkets   []domain.Market
		wg           sync.WaitGroup
	)

	fetchers := []struct {
		name   string
		fetch  func(context.Context) ([]domain.Market, error)
		mapper func([]domain.Market, map[string]uint) []domain.Market
	}{
		{
			name: "ompfinex",
			fetch: func(ctx context.Context) ([]domain.Market, error) {
				raw, err := s.ompfinexClient.ListMarkets(ctx)
				if err != nil {
					return nil, err
				}
				mapped := make([]domain.Market, 0, len(raw))
				for _, m := range raw {
					if megaMarketID, ok := marketNamesMap[m.BaseCurrency.ID+"/"+m.QuoteCurrency.ID]; ok {
						s.logger.Infof("[ompfinex] fetched market: %+v", m)
						mapped = append(mapped, domain.Market{
							ExchangeName:             "ompfinex",
							MarketName:               m.BaseCurrency.ID + "/" + m.QuoteCurrency.ID,
							IsActive:                 true,
							ExchangeMarketIdentifier: strconv.FormatInt(m.ID, 10),
							MegaMarketID:             megaMarketID,
						})
					}
				}
				return mapped, nil
			},
		},
		{
			name: "wallex",
			fetch: func(ctx context.Context) ([]domain.Market, error) {
				raw, err := s.wallexClient.GetAllMarkets(ctx)
				if err != nil {
					return nil, err
				}
				mapped := make([]domain.Market, 0, len(raw))
				for _, m := range raw {
					if megaMarketID, ok := marketNamesMap[m.EnBaseAsset+"/"+m.EnQuoteAsset]; ok {
						s.logger.Infof("[wallex] fetched market: %+v", m)
						mapped = append(mapped, domain.Market{
							ExchangeName:             "wallex",
							MarketName:               m.EnBaseAsset + "/" + m.EnQuoteAsset,
							IsActive:                 true,
							ExchangeMarketIdentifier: m.Symbol,
							MegaMarketID:             megaMarketID,
						})
					}
				}
				return mapped, nil
			},
		},
	}

	resultsCh := make(chan []domain.Market, len(fetchers))
	errorsCh := make(chan error, len(fetchers))

	for _, f := range fetchers {
		wg.Add(1)
		go func(f func(context.Context) ([]domain.Market, error), name string) {
			defer wg.Done()
			markets, err := f(ctx)
			if err != nil {
				s.logger.Errorf("[%s] failed to fetch markets: %v", name, err)
				errorsCh <- err
				return
			}
			resultsCh <- markets
		}(f.fetch, f.name)
	}

	wg.Wait()
	close(resultsCh)
	close(errorsCh)

	for markets := range resultsCh {
		allMarketsMu.Lock()
		allMarkets = append(allMarkets, markets...)
		allMarketsMu.Unlock()
	}

	// --- Step 3: Decide if we fail or continue
	if len(allMarkets) == 0 {
		return nil, nil, errors.New("failed to fetch markets from all exchanges")
	}
	s.marketsRepo.SoftDeleteAll(ctx)

	// --- Step 4: Persist
	if err := s.marketsRepo.UpsertMarketsForExchange(ctx, allMarkets); err != nil {
		s.logger.Errorf("failed to upsert markets: %v", err)
		return nil, nil, err
	}

	storedMarkets, err := s.marketsRepo.GetAllActiveMarkets(ctx)
	if err != nil {
		s.logger.Errorf("failed to get active markets: %v", err)
		return nil, nil, err
	}

	return storedMarkets, megaMarketMap, nil
}

func (s *MarketService) GetBestExchangePriceByVolume(
	ctx context.Context,
	megaMarketId uint,
	volume decimal.Decimal,
	isBuy bool,
) (decimal.Decimal, *domain.Market, *domain.MegaMarket, error) {
	// TODO: add fee of transaction
	// --- Fetch candidate markets
	megaMarket, err := s.megaMarketRepo.GetActiveMegaMarketByID(ctx, megaMarketId)
	if err != nil {
		s.logger.Errorf("get active mega market by id failed: %v", err)
		return decimal.Zero, nil, nil, err
	}
	if megaMarket == nil {
		return decimal.Zero, nil, nil, errors.New("no active mega market found for id")
	}
	markets, err := s.marketsRepo.GetMarketsByMegaMarketId(ctx, megaMarketId)
	if err != nil {
		s.logger.Errorf("get markets by mega market id failed: %v", err)
		return decimal.Zero, nil, nil, err
	}

	type result struct {
		price        decimal.Decimal
		exchangeName string
		market       domain.Market
	}

	var (
		results []result
		mu      sync.Mutex
	)

	// --- Run each market check concurrently
	g, ctx := errgroup.WithContext(ctx)
	for _, m := range markets {
		m := m // capture range variable

		g.Go(func() error {
			price, err := s.fetchAndCalculatePrice(ctx, m.ExchangeName, m.ExchangeMarketIdentifier, volume, isBuy)
			if err != nil {
				// Log, but don’t fail the whole group
				s.logger.Errorf("[%s] price calculation failed: %v", m.ExchangeName, err)
				return nil
			}

			mu.Lock()
			results = append(results, result{price: price, exchangeName: m.ExchangeName, market: m})
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait() // we ignore returned error since we log & skip per exchange

	// --- Pick the lowest price
	if len(results) == 0 {
		return decimal.Zero, nil, nil, errors.New("could not determine best price")
	}

	best := results[0]
	for _, r := range results[1:] {
		if r.price.LessThan(best.price) {
			best = r
		}
	}

	return best.price, &best.market, megaMarket, nil
}
func (s *MarketService) fetchAndCalculatePrice(
	ctx context.Context,
	exchangeName string,
	exchangeMarketID string,
	volume decimal.Decimal,
	isBuy bool,
) (decimal.Decimal, error) {
	switch exchangeName {
	case "ompfinex":
		depth, err := s.ompfinexClient.GetMarketDepth(ctx, exchangeMarketID)
		if err != nil {
			return decimal.Zero, err
		}
		return s.calculateOmpfinexPrice(depth, volume, isBuy)

	case "wallex":
		depth, err := s.wallexClient.GetMarketDepth(ctx, exchangeMarketID)
		if err != nil {
			return decimal.Zero, err
		}
		return s.calculateWallexPrice(depth, volume, isBuy)

	default:
		return decimal.Zero, errors.New("unsupported exchange: " + exchangeName)
	}
}

// calculateOmpfinexPrice calculates the price to buy the requested volume
func (s *MarketService) calculateOmpfinexPrice(depth ompfinex.OrderBook, volume decimal.Decimal, isBuy bool) (decimal.Decimal, error) {
	if volume.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, errors.New("volume must be positive")
	}

	totalVolume := decimal.Zero
	totalCost := decimal.Zero
	sideOrders := depth.Asks
	if !isBuy {
		sideOrders = depth.Bids
	}
	for _, ask := range sideOrders {
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
func (s *MarketService) GetMarketByID(ctx context.Context, id uint) (*domain.Market, error) {
	return s.marketsRepo.GetMarketByID(ctx, id)
}
func (s *MarketService) GetMegaMarketByID(ctx context.Context, id uint) (*domain.MegaMarket, error) {
	return s.megaMarketRepo.GetActiveMegaMarketByID(ctx, id)
}

// calculateWallexPrice calculates the minimum average price to buy the specified volume
// by consuming asks from the order book starting from the best (lowest) price.
// Returns the weighted average price or error if not enough volume available.
func (s *MarketService) calculateWallexPrice(depth *wallex.OrderBook, volume decimal.Decimal, isBuy bool) (decimal.Decimal, error) {
	if volume.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, errors.New("volume must be positive")
	}

	var (
		totalVolume = decimal.Zero
		totalCost   = decimal.Zero
	)

	sideOrders := depth.Asks
	if !isBuy {
		sideOrders = depth.Bids
	}
	// Iterate through asks (sorted from best/lowest price first)
	for _, ask := range sideOrders {
		if ask.Price.LessThanOrEqual(decimal.Zero) || ask.Quantity.LessThanOrEqual(decimal.Zero) {
			continue // skip invalid entries
		}

		// Calculate how much we need from this level
		remaining := volume.Sub(totalVolume)
		available := ask.Quantity
		consumed := decimal.Min(remaining, available)

		// Add to totals
		totalCost = totalCost.Add(ask.Price.Mul(consumed))
		totalVolume = totalVolume.Add(consumed)

		// If we've reached the target volume
		if totalVolume.GreaterThanOrEqual(volume) {
			return totalCost.Div(volume), nil // return weighted average price
		}
	}

	return decimal.Zero, fmt.Errorf("not enough liquidity in order book (available: %s, requested: %s)",
		totalVolume.String(), volume.String())
}
