package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderPending                  OrderStatus = "PENDING"
	OrderUserDebitInProgress      OrderStatus = "USER_DEBIT_IN_PROGRESS"
	OrderUserDebitSuccess         OrderStatus = "USER_DEBIT_SUCCESS"
	OrderFailedUserDebit          OrderStatus = "FAILED_USER_DEBIT"
	OrderTreasuryCreditInProgress OrderStatus = "TREASURY_CREDIT_IN_PROGRESS"
	OrderCompleted                OrderStatus = "COMPLETED"
	OrderFailedTreasuryCredit     OrderStatus = "FAILED_TREASURY_CREDIT"
	OrderRefundInProgress         OrderStatus = "REFUND_IN_PROGRESS"
	OrderRefundSuccess            OrderStatus = "REFUND_SUCCESS"
)

type Order struct {
	ID                 uint              `json:"id"`
	Status             OrderStatus       `json:"status"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Volume             decimal.Decimal   `json:"volume"`
	FromNetwork        string            `json:"from_network"`
	ToNetwork          string            `json:"to_network"`
	UserAddress        string            `json:"user_address"`
	MarketID           uint              `json:"market_id"`
	IsBuy              bool              `json:"is_buy"`
	ContractAddress    string            `json:"contract_address"`
	Deadline           int64             `json:"deadline"`
	DestinationAddress *string           `json:"destination_address"`
	TokenAddress       string            `json:"token_address"`
	Signature          map[string]string `json:"signature"`
	DepositTxHash      *string           `json:"deposit_tx_hash"`
	ReleaseTxHash      *string           `json:"release_tx_hash"`
	UserId             string            `json:"user_id"`
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
