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

	"github.com/MMN3003/mega/src/order/domain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type OrderSignaturePayload struct {
	V uint8  `json:"v"`
	R string `json:"r"`
	S string `json:"s"`
}

// SubmitOrderRequestBody is the payload to submit a new order
// swagger:model SubmitOrderRequestBody
type SubmitOrderRequestBody struct {
	Volume             decimal.Decimal       `json:"volume"`
	Price              decimal.Decimal       `json:"price"`
	FromNetwork        string                `json:"from_network"`
	ToNetwork          string                `json:"to_network"`
	UserAddress        string                `json:"user_address"`
	MarketID           uint                  `json:"market_id"`
	IsBuy              bool                  `json:"is_buy"`
	Deadline           int64                 `json:"deadline"`
	DestinationAddress *string               `json:"destination_address"`
	TokenAddress       string                `json:"token_address"`
	Signature          OrderSignaturePayload `json:"signature"`
	UserId             string                `json:"user_id"`
}

func (c SubmitOrderRequestBody) ToOrder() *domain.Order {
	return &domain.Order{
		Volume:             c.Volume,
		Price:              c.Price,
		FromNetwork:        c.FromNetwork,
		ToNetwork:          c.ToNetwork,
		UserAddress:        c.UserAddress,
		MarketID:           c.MarketID,
		IsBuy:              c.IsBuy,
		Deadline:           c.Deadline,
		DestinationAddress: c.DestinationAddress,
		TokenAddress:       c.TokenAddress,
		Signature: domain.OrderSignature{
			V: c.Signature.V,
			R: common.HexToHash(c.Signature.R),
			S: common.HexToHash(c.Signature.S),
		},
		UserId: c.UserId,
	}
}

// SubmitOrderResponse is the response to submit a new order
// swagger:model SubmitOrderResponse
type SubmitOrderResponse struct {
	ID                     uint                  `json:"id"`
	Status                 domain.OrderStatus    `json:"status"`
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`
	Volume                 decimal.Decimal       `json:"volume"`
	Price                  decimal.Decimal       `json:"price"`
	FromNetwork            string                `json:"from_network"`
	ToNetwork              string                `json:"to_network"`
	UserAddress            string                `json:"user_address"`
	MarketID               uint                  `json:"market_id"`
	MegaMarketID           uint                  `json:"mega_market_id"`
	SlipagePercentage      decimal.Decimal       `json:"slipage_percentage"`
	IsBuy                  bool                  `json:"is_buy"`
	ContractAddress        string                `json:"contract_address"`
	Deadline               int64                 `json:"deadline"`
	DestinationAddress     *string               `json:"destination_address"`
	TokenAddress           string                `json:"token_address"`
	Signature              OrderSignaturePayload `json:"signature"`
	DepositTxHash          *string               `json:"deposit_tx_hash"`
	ReleaseTxHash          *string               `json:"release_tx_hash"`
	UserId                 string                `json:"user_id"`
	DestinationTokenSymbol string                `json:"destination_token_symbol"`
	SourceTokenSymbol      string                `json:"source_token_symbol"`
}

func fromOrderDomain(order *domain.Order) SubmitOrderResponse {
	return SubmitOrderResponse{
		ID:                 order.ID,
		Status:             order.Status,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
		Volume:             order.Volume,
		Price:              order.Price,
		FromNetwork:        order.FromNetwork,
		ToNetwork:          order.ToNetwork,
		UserAddress:        order.UserAddress,
		MarketID:           order.MarketID,
		MegaMarketID:       order.MegaMarketID,
		SlipagePercentage:  order.SlipagePercentage,
		IsBuy:              order.IsBuy,
		ContractAddress:    order.ContractAddress,
		Deadline:           order.Deadline,
		DestinationAddress: order.DestinationAddress,
		TokenAddress:       order.TokenAddress,
		Signature: OrderSignaturePayload{
			V: order.Signature.V,
			R: order.Signature.R.Hex(),
			S: order.Signature.S.Hex(),
		},
		DepositTxHash:          order.DepositTxHash,
		ReleaseTxHash:          order.ReleaseTxHash,
		UserId:                 order.UserId,
		DestinationTokenSymbol: order.DestinationTokenSymbol,
		SourceTokenSymbol:      order.SourceTokenSymbol,
	}
}

// PairDTO describes a tradable pair
// swagger:model PairDTO
type PairDTO struct {
	FromNetwork string `json:"from_network" example:"sepolia"`
	FromToken   string `json:"from_token" example:"USDT"`
	ToNetwork   string `json:"to_network" example:"mumbai"`
	ToToken     string `json:"to_token" example:"MATIC"`
}

// ListPairsResponse lists pairs
// swagger:response ListPairsResponse
type ListPairsResponse struct {
	// in: body
	Body struct {
		Pairs []PairDTO `json:"pairs"`
	}
}

// swagger:model ListPairsResponseBody
type ListPairsResponseBody struct {
	Pairs []PairDTO `json:"pairs"`
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
