package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// QuoteRepository persistence port
type QuoteRepository interface {
	Save(ctx context.Context, q *Quote) error
	GetByID(ctx context.Context, id string) (*Quote, error)
	MarkUsed(ctx context.Context, id string) error
	ListActive(ctx context.Context) ([]*Quote, error)
}

// OnChainAdapter port for network adapter
type OnChainAdapter interface {
	// ExecuteTradeWithPermit withdraws token from user's account to treasury using permit (EIP-2612 style).
	ExecuteTradeWithPermit(ctx context.Context, userAddress string, token string, amount decimal.Decimal, permit string) (txHash string, err error)

	// SendFromTreasury: transfer token from treasury to target address (on same network)
	SendFromTreasury(ctx context.Context, toAddress string, token string, amount decimal.Decimal) (txHash string, err error)

	// GetTreasuryBalance returns treasury balance for token
	GetTreasuryBalance(ctx context.Context, token string) (decimal.Decimal, error)

	// ListSupportedTokens returns tokens supported on this network
	ListSupportedTokens(ctx context.Context) ([]Coin, error)
}
