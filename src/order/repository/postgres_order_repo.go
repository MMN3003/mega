package repository

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/order/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var _ domain.OrderRepository = (*OrderRepo)(nil)

// ---------- MARKETS ----------
// gorm.Model includes:
// ID        uint `gorm:"primarykey"`
// CreatedAt time.Time
// UpdatedAt time.Time
// DeletedAt gorm.DeletedAt `gorm:"index"`
type Order struct {
	gorm.Model

	Status                 string          `json:"status" gorm:"index"`
	Volume                 decimal.Decimal `json:"volume"`
	FromNetwork            string          `json:"from_network"`
	ToNetwork              string          `json:"to_network"`
	UserAddress            string          `json:"user_address"`
	MarketID               uint            `json:"market_id"`
	MegaMarketID           uint            `json:"mega_market_id"`
	IsBuy                  bool            `json:"is_buy"`
	ContractAddress        string          `json:"contract_address"`
	Deadline               int64           `json:"deadline"`
	DestinationAddress     *string         `json:"destination_address"`
	TokenAddress           string          `json:"token_address"`
	Signature              *string         `json:"signature"`
	DepositTxHash          *string         `json:"deposit_tx_hash"`
	ReleaseTxHash          *string         `json:"release_tx_hash"`
	UserId                 string          `json:"user_id" gorm:"index"`
	DestinationTokenSymbol string          `json:"destination_token_symbol"`
	SlipagePercentage      decimal.Decimal `json:"slipage_percentage"`
	Price                  decimal.Decimal `json:"price"`
	SourceTokenSymbol      string          `json:"source_token_symbol"`
}

// ---------- REPO ----------

type OrderRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewOrderRepo(db *gorm.DB, log *logger.Logger) *OrderRepo {
	if err := db.AutoMigrate(&Order{}); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}
	return &OrderRepo{db: db, log: log}
}

// ---------- ORDER CRUD ----------

func (r *OrderRepo) SaveOrder(ctx context.Context, o *domain.Order) (*domain.Order, error) {
	// check if signature exist convert it to string use marshal

	model := Order{
		Status:                 string(o.Status),
		Volume:                 o.Volume,
		FromNetwork:            o.FromNetwork,
		ToNetwork:              o.ToNetwork,
		UserAddress:            o.UserAddress,
		MarketID:               o.MarketID,
		DestinationTokenSymbol: o.DestinationTokenSymbol,
		IsBuy:                  o.IsBuy,
		ContractAddress:        o.ContractAddress,
		Deadline:               o.Deadline,
		DestinationAddress:     o.DestinationAddress,
		TokenAddress:           o.TokenAddress,
		Signature:              marshalToString(o.Signature),
		DepositTxHash:          o.DepositTxHash,
		ReleaseTxHash:          o.ReleaseTxHash,
		UserId:                 o.UserId,
		MegaMarketID:           o.MegaMarketID,
		SlipagePercentage:      o.SlipagePercentage,
		Price:                  o.Price,
		SourceTokenSymbol:      o.SourceTokenSymbol,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, err
	}
	return r.GetOrderByID(ctx, model.ID)
}

func (r *OrderRepo) GetOrderByID(ctx context.Context, id uint) (*domain.Order, error) {
	var o Order
	if err := r.db.WithContext(ctx).First(&o, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainOrder(&o), nil
}

func (r *OrderRepo) UpdateOrder(ctx context.Context, o *domain.Order) error {
	return r.db.WithContext(ctx).Model(&Order{}).
		Where("id = ?", o.ID).
		Updates(Order{
			Status:                 string(o.Status),
			Volume:                 o.Volume,
			FromNetwork:            o.FromNetwork,
			ToNetwork:              o.ToNetwork,
			UserAddress:            o.UserAddress,
			MarketID:               o.MarketID,
			IsBuy:                  o.IsBuy,
			ContractAddress:        o.ContractAddress,
			Deadline:               o.Deadline,
			DestinationAddress:     o.DestinationAddress,
			TokenAddress:           o.TokenAddress,
			Signature:              marshalToString(o.Signature),
			DepositTxHash:          o.DepositTxHash,
			ReleaseTxHash:          o.ReleaseTxHash,
			UserId:                 o.UserId,
			MegaMarketID:           o.MegaMarketID,
			DestinationTokenSymbol: o.DestinationTokenSymbol,
			SlipagePercentage:      o.SlipagePercentage,
			Price:                  o.Price,
			SourceTokenSymbol:      o.SourceTokenSymbol,
		}).Error
}

func (r *OrderRepo) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Order{}, id).Error
}
func (r *OrderRepo) SoftDeleteAll(ctx context.Context) error {
	return r.db.
		WithContext(ctx).
		Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&Order{}).Error
}

func (r *OrderRepo) GetOrdersByUserId(ctx context.Context, userId string) ([]domain.Order, error) {
	var models []Order
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomainOrders(models), nil
}

func (r *OrderRepo) GetOrdersByStatus(ctx context.Context, status domain.OrderStatus) ([]domain.Order, error) {
	var models []Order
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomainOrders(models), nil
}

func (r *OrderRepo) ChangeStatusByIds(ctx context.Context, ids []uint, status domain.OrderStatus) error {
	return r.db.WithContext(ctx).Model(&Order{}).
		Where("id in ?", ids).
		Updates(Order{Status: string(status)}).Error
}

// ---------- HELPERS ----------

func (r *OrderRepo) toDomainOrder(o *Order) *domain.Order {
	return &domain.Order{
		ID:                     o.ID,
		Status:                 domain.OrderStatus(o.Status),
		Volume:                 o.Volume,
		FromNetwork:            o.FromNetwork,
		ToNetwork:              o.ToNetwork,
		UserAddress:            o.UserAddress,
		MarketID:               o.MarketID,
		IsBuy:                  o.IsBuy,
		ContractAddress:        o.ContractAddress,
		Deadline:               o.Deadline,
		DestinationAddress:     o.DestinationAddress,
		TokenAddress:           o.TokenAddress,
		Signature:              unmarshalFromJSON[domain.OrderSignature](o.Signature),
		DepositTxHash:          o.DepositTxHash,
		ReleaseTxHash:          o.ReleaseTxHash,
		UserId:                 o.UserId,
		MegaMarketID:           o.MegaMarketID,
		DestinationTokenSymbol: o.DestinationTokenSymbol,
		SlipagePercentage:      o.SlipagePercentage,
		Price:                  o.Price,
		SourceTokenSymbol:      o.SourceTokenSymbol,
	}
}
func (r *OrderRepo) toDomainOrders(os []Order) []domain.Order {
	var dos []domain.Order
	for _, o := range os {
		dos = append(dos, *r.toDomainOrder(&o))
	}
	return dos
}
func marshalToString(v interface{}) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}
func unmarshalFromJSON[T any](data interface{}) T {
	// Get the type of T to determine if it's a pointer
	var zero T
	tType := reflect.TypeOf(zero)

	if data == nil {
		// For pointer types, return nil, for non-pointer return zero value
		if tType != nil && tType.Kind() == reflect.Ptr {
			return zero
		}
		return zero
	}

	// Check if data is already the correct type
	if typedData, ok := data.(T); ok {
		return typedData
	}

	// For pointer types, also check if data is the underlying type
	if tType != nil && tType.Kind() == reflect.Ptr {
		elemType := tType.Elem()
		// Check if data is of the underlying type (e.g., T is *string, data is string)
		if reflect.TypeOf(data) == elemType {
			val := reflect.ValueOf(data)
			result := reflect.New(elemType)
			result.Elem().Set(val)
			return result.Interface().(T)
		}
	}

	// Handle JSON string unmarshaling
	if str, ok := data.(string); ok {
		// For pointer types, unmarshal and take address
		if tType != nil && tType.Kind() == reflect.Ptr {
			elemType := tType.Elem()
			result := reflect.New(elemType).Interface()
			if err := json.Unmarshal([]byte(str), result); err == nil {
				return reflect.ValueOf(result).Convert(tType).Interface().(T)
			}
		} else {
			// For non-pointer types
			var result T
			if err := json.Unmarshal([]byte(str), &result); err == nil {
				return result
			}
		}
	}

	return zero
}
