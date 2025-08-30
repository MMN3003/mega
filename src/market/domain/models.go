package domain

type Market struct {
	ID                       uint
	ExchangeMarketIdentifier string
	ExchangeName             string
	MarketName               string
	MegaMarketID             uint
	IsActive                 bool
}

type MegaMarket struct {
	ID                  uint
	ExchangeMarketNames string
	IsActive            bool
}
