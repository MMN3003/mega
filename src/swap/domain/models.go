package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

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
