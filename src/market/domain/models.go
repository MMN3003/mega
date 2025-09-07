package domain

import "github.com/shopspring/decimal"

type Market struct {
	ID                          uint
	ExchangeMarketIdentifier    string
	ExchangeName                string
	MarketName                  string
	MegaMarketID                uint
	IsActive                    bool
	ExchangeMarketFeePercentage decimal.Decimal
}

type MegaMarket struct {
	ID                     uint
	ExchangeMarketNames    string
	IsActive               bool
	FeePercentage          decimal.Decimal
	SourceTokenSymbol      string
	DestinationTokenSymbol string
	SlipagePercentage      decimal.Decimal
}
