package usecase

import (
	"context"
	"errors"
	"strconv"

	"github.com/MMN3003/mega/src/Infrastructure/ompfinex"
	"github.com/MMN3003/mega/src/Infrastructure/wallex"
	"github.com/MMN3003/mega/src/config"
	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/order/adapter/market"
	"github.com/MMN3003/mega/src/order/domain"
	"github.com/shopspring/decimal"
)

var _ domain.OrderUsecase = (*Service)(nil)

type Service struct {
	orderRepo      domain.OrderRepository
	logger         *logger.Logger
	ompfinexClient *ompfinex.Client
	wallexClient   *wallex.Client
	marketAdapter  market.MarketAdapter
}

func NewService(o domain.OrderRepository, logg *logger.Logger, cfg *config.Config) *Service {
	ompfinexClient, _ := ompfinex.NewClient(cfg.OMP.BaseURL,
		ompfinex.WithAuthToken(cfg.OMP.Token),
	)
	wallexClient, _ := wallex.NewClient(cfg.Wallex.BaseURL,
		wallex.WithAPIKey(cfg.Wallex.APIKey),
	)
	s := &Service{
		orderRepo:      o,
		logger:         logg,
		ompfinexClient: ompfinexClient,
		wallexClient:   wallexClient,
	}
	return s
}
func (s *Service) SetAdapters(ctx context.Context, marketAdapter market.MarketAdapter) error {
	s.marketAdapter = marketAdapter
	return nil
}
func (s *Service) PlaceMarketOrder(ctx context.Context, marketId uint, volume decimal.Decimal, isBuy bool) (string, error) {
	market, err := s.marketAdapter.GetMarketByID(ctx, marketId)
	if err != nil {
		return "", err
	}
	switch market.ExchangeName {
	case "ompfinex":
		marketId, _ := strconv.ParseInt(market.ExchangeMarketIdentifier, 10, 64)
		side := ompfinex.SideSell
		if isBuy {
			side = ompfinex.SideBuy
		}
		order, err := s.ompfinexClient.PlaceOrder(ctx, ompfinex.PlaceOrderRequest{
			MarketID: marketId,
			Side:     side,
			Type:     ompfinex.OrderMarket,
			Price:    nil,
			Amount:   volume,
		})
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(order.ID, 10), nil
	case "wallex":
		side := wallex.OrderSideSell
		if isBuy {
			side = wallex.OrderSideBuy
		}
		order, err := s.wallexClient.PlaceMarketOrder(ctx, market.ExchangeMarketIdentifier, side, volume)
		if err != nil {
			return "", err
		}
		return order.ClientOrderID, nil
	default:
		return "", errors.New("unsupported exchange")
	}
}
func (s *Service) SubmitOrder(ctx context.Context, o *domain.Order) (*domain.Order, error) {
	o.Status = domain.OrderPending
	order, err := s.orderRepo.SaveOrder(ctx, o)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) FetchPendingOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetOrdersByStatus(ctx, domain.OrderPending)
	if err != nil {
		return err
	}
	ids := make([]uint, len(orders))
	for i, o := range orders {
		s.logger.Infof("Order %d is pending", o.ID)
		ids[i] = o.ID
	}
	err = s.orderRepo.ChangeStatusByIds(ctx, ids, domain.OrderUserDebitInProgress)
	if err != nil {
		return err
	}
	for _, o := range orders {
		order := o
		go func(order domain.Order) {
			s.logger.Infof("Order %d is pending", order.ID)
			reciept := false
			//TODO: run contrant story
			if reciept {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderUserDebitSuccess)
			} else {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderFailedUserDebit)
			}
			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}
func (s *Service) FetchSuccessDebitOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetOrdersByStatus(ctx, domain.OrderUserDebitSuccess)
	if err != nil {
		return err
	}
	ids := make([]uint, len(orders))
	for i, o := range orders {
		s.logger.Infof("Order %d is pending", o.ID)
		ids[i] = o.ID
	}
	err = s.orderRepo.ChangeStatusByIds(ctx, ids, domain.OrderTreasuryCreditInProgress)
	if err != nil {
		return err
	}
	for _, o := range orders {
		order := o
		go func(order domain.Order) {
			s.logger.Infof("Order %d is pending", order.ID)
			reciept := false
			//TODO: send amount to user destination address
			if reciept {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderCompleted)
			} else {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderFailedTreasuryCredit)
			}
			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}
func (s *Service) FetchFailedTreasuryCreditOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetOrdersByStatus(ctx, domain.OrderFailedTreasuryCredit)
	if err != nil {
		return err
	}
	ids := make([]uint, len(orders))
	for i, o := range orders {
		s.logger.Infof("Order %d is pending", o.ID)
		ids[i] = o.ID
	}
	err = s.orderRepo.ChangeStatusByIds(ctx, ids, domain.OrderRefundInProgress)
	if err != nil {
		return err
	}
	for _, o := range orders {
		order := o
		go func(order domain.Order) {
			s.logger.Infof("Order %d is pending", order.ID)
			reciept := false
			//TODO: refund amount to user source
			if reciept {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderRefundSuccess)
			} else {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderFailedTreasuryCredit)
			}
			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}

// func (s *Service) ListPairs(ctx context.Context) ([]map[string]string, error) {
// 	out := []map[string]string{}
// 	for netName, adapter := range s.adapters {
// 		tokens, err := adapter.ListSupportedTokens(ctx)
// 		if err != nil {
// 			s.logger.Errorf("ListSupportedTokens err: %v", err)
// 			continue
// 		}
// 		for _, t := range tokens {
// 			for otherNet, otherAdapter := range s.adapters {
// 				if otherNet == netName {
// 					continue
// 				}
// 				otherTokens, _ := otherAdapter.ListSupportedTokens(ctx)
// 				for _, ot := range otherTokens {
// 					out = append(out, map[string]string{
// 						"from_network": netName,
// 						"from_token":   t.Symbol,
// 						"to_network":   otherNet,
// 						"to_token":     ot.Symbol,
// 					})
// 				}
// 			}
// 		}
// 	}
// 	return out, nil
// }

// type CreateQuoteRequest struct {
// 	FromNetwork string
// 	FromToken   string
// 	ToNetwork   string
// 	ToToken     string
// 	AmountIn    decimal.Decimal
// 	UserAddress string
// }

// func (s *Service) CreateQuote(ctx context.Context, req CreateQuoteRequest) (*domain.Quote, error) {
// 	_, ok := s.adapters[req.FromNetwork]
// 	if !ok {
// 		return nil, errors.New("unsupported from network")
// 	}
// 	toAdapter, ok := s.adapters[req.ToNetwork]
// 	if !ok {
// 		return nil, errors.New("unsupported to network")
// 	}

// 	rateKey := fmt.Sprintf("%s|%s", req.FromToken, req.ToToken)
// 	rate, ok := s.rates[rateKey]
// 	if !ok {
// 		return nil, errors.New("rate not available for pair")
// 	}
// 	amountOut := req.AmountIn.Mul(rate)

// 	treasuryBal, err := toAdapter.GetTreasuryBalance(ctx, req.ToToken)
// 	if err != nil {
// 		return nil, fmt.Errorf("treasury balance error: %w", err)
// 	}
// 	if treasuryBal.LessThan(amountOut) {
// 		return nil, errors.New("insufficient treasury liquidity")
// 	}

// 	now := time.Now().UTC()
// 	q := &domain.Quote{
// 		ID:          uuid.New().String(),
// 		FromNetwork: req.FromNetwork,
// 		FromToken:   req.FromToken,
// 		ToNetwork:   req.ToNetwork,
// 		ToToken:     req.ToToken,
// 		AmountIn:    req.AmountIn,
// 		AmountOut:   amountOut,
// 		CreatedAt:   now,
// 		ExpiresAt:   now.Add(s.quoteTTL),
// 		Used:        false,
// 		UserAddress: req.UserAddress,
// 	}
// 	if err := s.quotes.Save(ctx, q); err != nil {
// 		return nil, err
// 	}
// 	return q, nil
// }

// type ExecuteRequest struct {
// 	QuoteID        string
// 	Permit         string
// 	RequestingUser string
// }

// func (s *Service) ExecuteQuote(ctx context.Context, req ExecuteRequest) (tx1 string, tx2 string, err error) {
// 	q, err := s.quotes.GetByID(ctx, req.QuoteID)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	if q.Used {
// 		return "", "", errors.New("quote already used")
// 	}
// 	if time.Now().After(q.ExpiresAt) {
// 		return "", "", errors.New("quote expired")
// 	}
// 	if req.RequestingUser == "" || req.RequestingUser != q.UserAddress {
// 		return "", "", errors.New("requesting user mismatch")
// 	}

// 	fromAdapter, ok := s.adapters[q.FromNetwork]
// 	if !ok {
// 		return "", "", errors.New("from adapter missing")
// 	}
// 	toAdapter, ok := s.adapters[q.ToNetwork]
// 	if !ok {
// 		return "", "", errors.New("to adapter missing")
// 	}

// 	// Step 1: withdraw user funds to treasury on source chain
// 	tx1, err = fromAdapter.ExecuteTradeWithPermit(ctx, q.UserAddress, q.FromToken, q.AmountIn, req.Permit)
// 	if err != nil {
// 		return "", "", fmt.Errorf("executeTrade failed: %w", err)
// 	}

// 	// Step 2: send asset from treasury on target chain to user
// 	tx2, err = toAdapter.SendFromTreasury(ctx, q.UserAddress, q.ToToken, q.AmountOut)
// 	if err != nil {
// 		// production: implement reconciliation and refunds here
// 		return tx1, "", fmt.Errorf("sendFromTreasury failed: %w", err)
// 	}

// 	// mark used
// 	if err := s.quotes.MarkUsed(ctx, q.ID); err != nil {
// 		s.logger.Errorf("mark used failed: %v", err)
// 		// don't break user flow, but alert
// 	}
// 	return tx1, tx2, nil
// }
