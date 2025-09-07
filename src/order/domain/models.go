package domain

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderPending                   OrderStatus = "PENDING"
	OrderUserDebitInProgress       OrderStatus = "USER_DEBIT_IN_PROGRESS"
	OrderUserDebitSuccess          OrderStatus = "USER_DEBIT_SUCCESS"
	OrderMarketUserOrderInProgress OrderStatus = "MARKET_USER_ORDER_IN_PROGRESS"
	OrderMarketUserOrderSuccess    OrderStatus = "MARKET_USER_ORDER_SUCCESS"
	OrderMarketUserOrderFailed     OrderStatus = "MARKET_USER_ORDER_FAILED"
	OrderFailedUserDebit           OrderStatus = "FAILED_USER_DEBIT"
	OrderRefundUserOrder           OrderStatus = "REFUND_USER_ORDER"
	OrderRefundUserOrderInProgress OrderStatus = "REFUND_USER_ORDER_IN_PROGRESS"
	OrderRefundUserOrderSuccess    OrderStatus = "REFUND_USER_ORDER_SUCCESS"
	OrderRefundUserOrderFailed     OrderStatus = "REFUND_USER_ORDER_FAILED"
	OrderTreasuryCreditInProgress  OrderStatus = "TREASURY_CREDIT_IN_PROGRESS"
	OrderCompleted                 OrderStatus = "COMPLETED"
)

type OrderSignature struct {
	V uint8       `json:"v"`
	R common.Hash `json:"r"`
	S common.Hash `json:"s"`
}
type Order struct {
	ID                     uint            `json:"id"`
	Status                 OrderStatus     `json:"status"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
	Volume                 decimal.Decimal `json:"volume"`
	Price                  decimal.Decimal `json:"price"`
	FromNetwork            string          `json:"from_network"`
	ToNetwork              string          `json:"to_network"`
	UserAddress            string          `json:"user_address"`
	MarketID               uint            `json:"market_id"`
	MegaMarketID           uint            `json:"mega_market_id"`
	SlipagePercentage      decimal.Decimal `json:"slipage_percentage"`
	IsBuy                  bool            `json:"is_buy"`
	ContractAddress        string          `json:"contract_address"`
	Deadline               int64           `json:"deadline"`
	DestinationAddress     *string         `json:"destination_address"`
	TokenAddress           string          `json:"token_address"`
	Signature              OrderSignature  `json:"signature"`
	DepositTxHash          *string         `json:"deposit_tx_hash"`
	ReleaseTxHash          *string         `json:"release_tx_hash"`
	UserId                 string          `json:"user_id"`
	DestinationTokenSymbol string          `json:"destination_token_symbol"`
	SourceTokenSymbol      string          `json:"source_token_symbol"`
}

// Coin description
type Coin struct {
	Symbol       string `json:"symbol" db:"symbol"`
	Decimals     int    `json:"decimals" db:"decimals"`
	ContractAddr string `json:"contract_addr,omitempty" db:"contract_addr"`
	Network      string `json:"network" db:"network"`
	DisplayName  string `json:"display_name,omitempty" db:"display_name"`
}

// Quote entity
type Quote struct {
	ID          string          `json:"id" db:"id"`
	FromNetwork string          `json:"from_network" db:"from_network"`
	FromToken   string          `json:"from_token" db:"from_token"`
	ToNetwork   string          `json:"to_network" db:"to_network"`
	ToToken     string          `json:"to_token" db:"to_token"`
	AmountIn    decimal.Decimal `json:"amount_in" db:"amount_in"`
	AmountOut   decimal.Decimal `json:"amount_out" db:"amount_out"`
	ExpiresAt   time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	Used        bool            `json:"used" db:"used"`
	UserAddress string          `json:"user_address" db:"user_address"`
}

const (
	NetworkSepolia = "sepolia"
	NetworkMumbai  = "mumbai"
	// add other networks if needed
)
