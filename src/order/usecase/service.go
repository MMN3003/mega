package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/MMN3003/mega/src/Infrastructure/ethereum"
	"github.com/MMN3003/mega/src/Infrastructure/ompfinex"
	"github.com/MMN3003/mega/src/Infrastructure/wallex"
	"github.com/MMN3003/mega/src/config"
	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/order/adapter/market"
	"github.com/MMN3003/mega/src/order/domain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

var _ domain.OrderUsecase = (*Service)(nil)

type Service struct {
	orderRepo      domain.OrderRepository
	logger         *logger.Logger
	ompfinexClient *ompfinex.Client
	wallexClient   *wallex.Client
	ethereumClient *ethereum.EthereumClient
	marketAdapter  market.MarketAdapter
}

func NewService(o domain.OrderRepository, logg *logger.Logger, cfg *config.Config, ethereumClient *ethereum.EthereumClient) *Service {
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
		ethereumClient: ethereumClient,
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
	market, err := s.marketAdapter.GetMarketByID(ctx, o.MarketID)
	if err != nil {
		return nil, err
	}
	megaMarket, err := s.marketAdapter.GetMegaMarketByID(ctx, market.MegaMarketID)
	if err != nil {
		return nil, err
	}

	o.Status = domain.OrderPending
	o.MegaMarketID = market.MegaMarketID
	o.SlipagePercentage = megaMarket.SlipagePercentage
	if o.IsBuy {
		o.SourceTokenSymbol, o.DestinationTokenSymbol =
			megaMarket.SourceTokenSymbol, megaMarket.DestinationTokenSymbol
	} else {
		o.SourceTokenSymbol, o.DestinationTokenSymbol =
			megaMarket.DestinationTokenSymbol, megaMarket.SourceTokenSymbol
	}

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
			receipt, err := s.ethereumClient.ExecuteTradeWithPermit(ctx, ethereum.Params{
				TokenAddress: common.HexToAddress(order.TokenAddress),
				Amount:       order.Volume.BigInt(),
				Deadline:     big.NewInt(order.Deadline),
				QuoteID:      fmt.Sprintf("%d", order.ID),
				UserAddress:  common.HexToAddress(order.UserAddress),
				Signature: struct {
					V uint8
					R common.Hash
					S common.Hash
				}{
					V: order.Signature.V,
					R: order.Signature.R,
					S: order.Signature.S,
				},
			})
			if err != nil {
				s.logger.Errorf("ExecuteTradeWithPermit err: %v", err)
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderFailedUserDebit)
			}

			if receipt.Status == 1 {
				// TODO: store receipt
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderUserDebitSuccess)
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
	err = s.orderRepo.ChangeStatusByIds(ctx, ids, domain.OrderMarketUserOrderInProgress)
	if err != nil {
		return err
	}
	for _, o := range orders {
		order := o
		go func(order domain.Order) {
			s.logger.Infof("Order %d is pending", order.ID)
			exchangeOrderId, err := s.PlaceMarketOrder(ctx, order.MarketID, order.Volume, order.IsBuy)
			if err != nil {
				s.logger.Errorf("PlaceMarketOrder err: %v", err)
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderMarketUserOrderFailed)
			}
			if exchangeOrderId != "" {
				// store exchange order id
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderMarketUserOrderSuccess)
			}
			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}
func (s *Service) FetchMarketUserOrderSuccessOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetOrdersByStatus(ctx, domain.OrderMarketUserOrderSuccess)
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
			//TODO: minus our fee from destination price
			receipt, err := s.ethereumClient.WithdrawTreasury(ctx, ethereum.WithdrawTreasuryParams{
				RecipientAddress: *order.DestinationAddress,
				Amount:           order.Price.String(),
				TokenSymbol:      order.DestinationTokenSymbol,
			})
			if err != nil {
				// store reciept log
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderRefundUserOrder)
			}
			if receipt.Status == 1 {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderCompleted)
			}
			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}
func (s *Service) FetchFailedMarketUserOrderOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetOrdersByStatus(ctx, domain.OrderMarketUserOrderFailed)
	if err != nil {
		return err
	}
	ids := make([]uint, len(orders))
	for i, o := range orders {
		s.logger.Infof("Order %d is pending", o.ID)
		ids[i] = o.ID
	}
	err = s.orderRepo.ChangeStatusByIds(ctx, ids, domain.OrderMarketUserOrderInProgress)
	if err != nil {
		return err
	}
	for _, o := range orders {
		order := o
		go func(order domain.Order) {
			s.logger.Infof("Order %d is pending", order.ID)
			price, _, _, err := s.marketAdapter.GetBestExchangePriceByVolume(ctx, order.MegaMarketID, order.Volume, order.IsBuy)

			if err != nil {
				s.logger.Errorf("GetBestExchangePriceByVolume err: %v", err)
				return
			}
			//  check slipage if slipage fail return the user money
			if price.GreaterThan(order.Price.Add(order.Price.Mul(order.SlipagePercentage))) {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderRefundUserOrder)
			} else {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderUserDebitSuccess) // try again
			}

			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}

func (s *Service) FetchReturnUserOrders(ctx context.Context) error {
	orders, err := s.orderRepo.GetOrdersByStatus(ctx, domain.OrderRefundUserOrder)
	if err != nil {
		return err
	}
	ids := make([]uint, len(orders))
	for i, o := range orders {
		s.logger.Infof("Order %d is pending", o.ID)
		ids[i] = o.ID
	}
	err = s.orderRepo.ChangeStatusByIds(ctx, ids, domain.OrderRefundUserOrderInProgress)
	if err != nil {
		return err
	}
	for _, o := range orders {
		order := o
		go func(order domain.Order) {
			s.logger.Infof("Order %d is pending", order.ID)
			receipt, err := s.ethereumClient.WithdrawTreasury(ctx, ethereum.WithdrawTreasuryParams{
				RecipientAddress: order.UserAddress,
				Amount:           order.Volume.String(),
				TokenSymbol:      order.SourceTokenSymbol,
			})

			if err != nil {
				s.logger.Errorf("GetBestExchangePriceByVolume err: %v", err)
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderRefundUserOrder) // try again
			}

			//TODO:  market user order
			if receipt.Status == 1 {
				err = s.orderRepo.ChangeStatusByIds(ctx, []uint{order.ID}, domain.OrderRefundUserOrderSuccess) // canceled completly
			}
			if err != nil {
				s.logger.Errorf("ChangeStatusByIds err: %v", err)
			}
		}(order)
	}

	return nil
}
func (s *Service) GetOrderById(ctx context.Context, id uint) (*domain.Order, error) {
	return s.orderRepo.GetOrderByID(ctx, id)
}
