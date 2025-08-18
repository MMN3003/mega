// Package http provides HTTP handlers for swap operations
//
// Schemes: http
// Host: localhost:8080
// BasePath: /
// Version: 1.0.0
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// swagger:meta
package http

import (
	"time"

	"github.com/MMN3003/mega/src/market/domain"
	"github.com/shopspring/decimal"
)

// MarketDto describes a tradable pair
// swagger:model MarketDto
type MarketDto struct {
	ExchangeName             string `json:"exchange_name" example:"ompfinex"`
	MarketName               string `json:"market_name" example:"BTC/USDT"`
	IsActive                 bool   `json:"is_active" example:"true"`
	ExchangeMarketIdentifier string `json:"exchange_market_identifier" example:"BTC/USDT"`
}

// ListPairsResponse lists pairs
// swagger:response ListPairsResponse
type ListPairsResponse struct {
	// in: body
	Body struct {
		Markets []MarketDto `json:"markets"`
	}
}

func MarketDtoFromDomain(m domain.Market) MarketDto {
	return MarketDto{
		ExchangeName:             m.ExchangeName,
		MarketName:               m.MarketName,
		IsActive:                 m.IsActive,
		ExchangeMarketIdentifier: m.ExchangeMarketIdentifier,
	}
}

// fromDomain converts a slice of domain.Market into FetchAndUpdateMarketsResponse
func FetchAndUpdateMarketsResponseFromDomain(markets []domain.Market) FetchAndUpdateMarketsResponse {
	dtos := make([]MarketDto, len(markets))
	for i, m := range markets {
		dtos[i] = MarketDtoFromDomain(m)
	}
	return FetchAndUpdateMarketsResponse{Markets: dtos}
}

// swagger:model ListPairsResponseBody
type FetchAndUpdateMarketsResponse struct {
	Markets []MarketDto `json:"markets"`
}

// CreateQuoteRequestBody is the payload to request a quote
// swagger:model CreateQuoteRequestBody
type CreateQuoteRequestBody struct {
	FromNetwork string `json:"from_network" example:"sepolia"`
	FromToken   string `json:"from_token" example:"USDT"`
	ToNetwork   string `json:"to_network" example:"mumbai"`
	ToToken     string `json:"to_token" example:"MATIC"`
	AmountIn    string `json:"amount_in" example:"100.0"` // decimal string
	UserAddress string `json:"user_address" example:"0xabc..."`
}

// CreateQuoteRequest wrapper for swagger param
// swagger:parameters CreateQuote
type CreateQuoteRequest struct {
	// in: body
	Body CreateQuoteRequestBody
}

// CreateQuoteResponseBody returns a quote
// swagger:model CreateQuoteResponseBody
type CreateQuoteResponseBody struct {
	QuoteID     string          `json:"quote_id" example:"b9f..."`
	AmountIn    decimal.Decimal `json:"amount_in" example:"100.0"`
	AmountOut   decimal.Decimal `json:"amount_out" example:"98.5"`
	ExpiresAt   time.Time       `json:"expires_at"`
	FromNetwork string          `json:"from_network"`
	FromToken   string          `json:"from_token"`
	ToNetwork   string          `json:"to_network"`
	ToToken     string          `json:"to_token"`
}

// CreateQuoteResponse wrapper for swagger response
// swagger:response CreateQuoteResponse
type CreateQuoteResponse struct {
	// in: body
	Body CreateQuoteResponseBody
}

// ExecuteQuoteRequestBody payload to execute a quote
// swagger:model ExecuteQuoteRequestBody
type ExecuteQuoteRequestBody struct {
	QuoteID        string `json:"quote_id" example:"b9f..."`
	Permit         string `json:"permit" example:"0xpermit..."`
	RequestingUser string `json:"requesting_user" example:"user1"`
}

// ExecuteQuoteRequest wrapper for swagger param
// swagger:parameters ExecuteQuote
type ExecuteQuoteRequest struct {
	// in: body
	Body ExecuteQuoteRequestBody
}

// ExecuteQuoteResponseBody returns execution result
// swagger:model ExecuteQuoteResponseBody
type ExecuteQuoteResponseBody struct {
	TxWithdraw string    `json:"tx_withdraw" example:"0xabc..."`
	TxDeposit  string    `json:"tx_deposit" example:"0xdef..."`
	ExecutedAt time.Time `json:"executed_at"`
}

// ExecuteQuoteResponse wrapper for swagger response
// swagger:response ExecuteQuoteResponse
type ExecuteQuoteResponse struct {
	// in: body
	Body ExecuteQuoteResponseBody
}
