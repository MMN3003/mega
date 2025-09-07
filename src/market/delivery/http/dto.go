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
	ID                          uint   `json:"id"`
	ExchangeName                string `json:"exchange_name" example:"ompfinex"`
	MarketName                  string `json:"market_name" example:"BTC/USDT"`
	IsActive                    bool   `json:"is_active" example:"true"`
	ExchangeMarketIdentifier    string `json:"exchange_market_identifier" example:"BTC/USDT"`
	MegaMarketID                uint   `json:"mega_market_id" example:"1"`
	ExchangeMarketFeePercentage string `json:"exchange_market_fee_percentage" example:"0.01"`
}
type MegaMarketDto struct {
	ID                     uint            `json:"id"`
	IsActive               bool            `json:"is_active" example:"true"`
	ExchangeMarketNames    string          `json:"exchange_market_names" example:"BTC/USDT"`
	FeePercentage          decimal.Decimal `json:"fee_percentage" example:"0.01"`
	SourceTokenSymbol      string          `json:"source_token_symbol" example:"BTC"`
	DestinationTokenSymbol string          `json:"destination_token_symbol" example:"USDT"`
}

type MarketAndMegaMarketDto struct {
	MarketDto
	MegaMarket MegaMarketDto `json:"mega_market"`
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
		ID:                          m.ID,
		ExchangeName:                m.ExchangeName,
		MarketName:                  m.MarketName,
		IsActive:                    m.IsActive,
		ExchangeMarketIdentifier:    m.ExchangeMarketIdentifier,
		MegaMarketID:                m.MegaMarketID,
		ExchangeMarketFeePercentage: m.ExchangeMarketFeePercentage.String(),
	}
}
func MegaMarketDtoFromDomain(m domain.MegaMarket) MegaMarketDto {
	return MegaMarketDto{
		ID:                     m.ID,
		IsActive:               m.IsActive,
		ExchangeMarketNames:    m.ExchangeMarketNames,
		FeePercentage:          m.FeePercentage,
		SourceTokenSymbol:      m.SourceTokenSymbol,
		DestinationTokenSymbol: m.DestinationTokenSymbol,
	}
}
func MarketAndMegaMarketDtoFromDomain(m domain.Market, megaMarket domain.MegaMarket) MarketAndMegaMarketDto {
	return MarketAndMegaMarketDto{
		MarketDto:  MarketDtoFromDomain(m),
		MegaMarket: MegaMarketDtoFromDomain(megaMarket),
	}
}

// fromDomain converts a slice of domain.Market into FetchAndUpdateMarketsResponse
func FetchAndUpdateMarketsResponseFromDomain(markets []domain.Market, megaMarketMap map[uint]domain.MegaMarket) FetchAndUpdateMarketsResponse {
	dtos := make([]MarketAndMegaMarketDto, len(markets))
	for i, m := range markets {
		dtos[i] = MarketAndMegaMarketDtoFromDomain(m, megaMarketMap[m.MegaMarketID])
	}
	return FetchAndUpdateMarketsResponse{Markets: dtos}
}

// swagger:model ListPairsResponseBody
type FetchAndUpdateMarketsResponse struct {
	Markets []MarketAndMegaMarketDto `json:"markets"`
}

// CreateQuoteRequestBody is the payload to request a quote
// swagger:model CreateQuoteRequestBody
type GetBestExchangePriceByVolumeRequestBody struct {
	MegaMarketID uint   `json:"mega_market_id" example:"4"`
	Volume       string `json:"volume" example:"100.0"` // decimal string
	IsBuy        bool   `json:"is_buy" example:"true"`
}

// CreateQuoteResponseBody returns a quote
// swagger:model CreateQuoteResponseBody
type GetBestExchangePriceByVolumeResponse struct {
	Price  decimal.Decimal        `json:"price" example:"100.0"`
	Market MarketAndMegaMarketDto `json:"market"`
}

func GetBestExchangePriceByVolumeResponseFromDomain(m *domain.Market, mm *domain.MegaMarket, price decimal.Decimal) GetBestExchangePriceByVolumeResponse {
	return GetBestExchangePriceByVolumeResponse{
		Price:  price,
		Market: MarketAndMegaMarketDtoFromDomain(*m, *mm),
	}
}
