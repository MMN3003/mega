package domain

type Market struct {
	ID                       uint
	ExchangeMarketIdentifier string
	ExchangeName             string
	MarketName               string
	IsActive                 bool
}
