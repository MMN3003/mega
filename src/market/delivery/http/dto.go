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
type GetBestExchangePriceByVolumeRequestBody struct {
	MarketName string `json:"market_name" example:"BTC/USDT"`
	Volume     string `json:"volume" example:"100.0"` // decimal string
}

// CreateQuoteResponseBody returns a quote
// swagger:model CreateQuoteResponseBody
type GetBestExchangePriceByVolumeResponse struct {
	Price        decimal.Decimal `json:"price" example:"100.0"`
	ExchangeName string          `json:"exchange_name" example:"ompfinex"`
}
