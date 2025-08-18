package network

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/swap/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MockAdapter simulates on-chain behavior; use for local testing.
type MockAdapter struct {
	network      string
	treasuryAddr string
	mu           sync.Mutex
	balances     map[string]map[string]decimal.Decimal // addr -> token -> balance
	logger       *logger.Logger
}

func NewMockAdapter(network string, logger *logger.Logger) *MockAdapter {
	m := &MockAdapter{
		network:      network,
		treasuryAddr: "treasury-" + network,
		balances:     make(map[string]map[string]decimal.Decimal),
		logger:       logger,
	}
	// seed treasury balances
	m.setBalance(m.treasuryAddr, "USDT", decimal.NewFromInt(10000))
	m.setBalance(m.treasuryAddr, "MATIC", decimal.NewFromInt(10000))
	return m
}

func (m *MockAdapter) setBalance(address, token string, amount decimal.Decimal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.balances[address]; !ok {
		m.balances[address] = make(map[string]decimal.Decimal)
	}
	m.balances[address][token] = amount
}

func (m *MockAdapter) getBalance(address, token string) decimal.Decimal {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.balances[address]; !ok {
		return decimal.Zero
	}
	b, ok := m.balances[address][token]
	if !ok {
		return decimal.Zero
	}
	return b
}

func (m *MockAdapter) ExecuteTradeWithPermit(ctx context.Context, userAddress string, token string, amount decimal.Decimal, permit string) (string, error) {
	if permit == "" {
		return "", errors.New("permit required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	userBal := m.balances[userAddress][token]
	if userBal.LessThan(amount) {
		return "", errors.New("insufficient funds on user")
	}
	m.balances[userAddress][token] = userBal.Sub(amount)
	m.balances[m.treasuryAddr][token] = m.balances[m.treasuryAddr][token].Add(amount)
	tx := fmt.Sprintf("%s-exec-%s", m.network, uuid.New().String())
	m.logger.Infof("mock ExecTradeWithPermit tx=%s user=%s token=%s amount=%s", tx, userAddress, token, amount.String())
	return tx, nil
}

func (m *MockAdapter) SendFromTreasury(ctx context.Context, toAddress string, token string, amount decimal.Decimal) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	treasuryBal := m.balances[m.treasuryAddr][token]
	if treasuryBal.LessThan(amount) {
		return "", errors.New("treasury insufficient")
	}
	m.balances[m.treasuryAddr][token] = treasuryBal.Sub(amount)
	if _, ok := m.balances[toAddress]; !ok {
		m.balances[toAddress] = make(map[string]decimal.Decimal)
	}
	m.balances[toAddress][token] = m.balances[toAddress][token].Add(amount)
	tx := fmt.Sprintf("%s-send-%s", m.network, uuid.New().String())
	m.logger.Infof("mock SendFromTreasury tx=%s to=%s token=%s amount=%s", tx, toAddress, token, amount.String())
	return tx, nil
}

func (m *MockAdapter) GetTreasuryBalance(ctx context.Context, token string) (decimal.Decimal, error) {
	b := m.getBalance(m.treasuryAddr, token)
	return b, nil
}

func (m *MockAdapter) ListSupportedTokens(ctx context.Context) ([]domain.Coin, error) {
	return []domain.Coin{
		{Symbol: "USDT", Decimals: 6, Network: m.network},
		{Symbol: "MATIC", Decimals: 18, Network: m.network},
	}, nil
}

// NewMockAdapters returns adapters for Sepolia and Mumbai pre-seeded with demo user.
func NewMockAdapters(logg *logger.Logger) map[string]domain.OnChainAdapter {
	out := map[string]domain.OnChainAdapter{}
	sep := NewMockAdapter(domain.NetworkSepolia, logg)
	mum := NewMockAdapter(domain.NetworkMumbai, logg)

	// seed a demo user on Sepolia
	sep.setBalance("user1", "USDT", decimal.NewFromInt(1000))

	out[domain.NetworkSepolia] = sep
	out[domain.NetworkMumbai] = mum
	return out
}
