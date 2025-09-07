package ethereum

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const phoenixProtocol = "PHOENIX"
const erc20ABI = `[
	{
		"constant": false,
		"inputs": [
			{"name": "to", "type": "address"},
			{"name": "amount", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [{"name": "", "type": "uint8"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "symbol",
		"outputs": [{"name": "", "type": "string"}],
		"type": "function"
	}
]`

// Errors
var (
	ErrMissingEnvVars    = errors.New("missing required environment variables")
	ErrConnectNetwork    = errors.New("failed to connect to network")
	ErrInvalidPrivateKey = errors.New("failed to parse private key")
	ErrReadABI           = errors.New("failed to read ABI file")
	ErrParseABI          = errors.New("failed to parse ABI")
	ErrCreateTransactor  = errors.New("failed to create transactor")
	ErrContractCall      = errors.New("failed to call contract function")
	ErrSendTransaction   = errors.New("failed to send transaction")
	ErrMineTransaction   = errors.New("failed to mine transaction")
	ErrInvalidAmount     = errors.New("failed to parse amount")
	ErrUnsupportedToken  = errors.New("unsupported token symbol")
)

// Config holds Ethereum client config
type Config struct {
	RPCURL          string
	PrivateKey      string
	PhoenixContract string
	ChainID         *big.Int
	abiFiles        map[string]string // Optional: contract-specific ABIs
	SupportedTokens map[string]string // Symbol → contract address (e.g. "USDT": "0x...", "DAI": "0x...")
}

// Params for executeTradeWithPermit
type Params struct {
	TokenAddress common.Address
	UserAddress  common.Address
	Amount       *big.Int
	Deadline     *big.Int
	QuoteID      string
	Signature    struct {
		V uint8
		R common.Hash
		S common.Hash
	}
}

// WithdrawTreasuryParams for WithdrawTreasury
type WithdrawTreasuryParams struct {
	RecipientAddress string
	Amount           string
	TokenSymbol      string
}

// EthereumClient encapsulates everything
type EthereumClient struct {
	client     *ethclient.Client
	wallet     common.Address
	privateKey *ecdsa.PrivateKey
	contracts  map[string]*bind.BoundContract // phoenix + tokens
	abi        map[string]abi.ABI
	config     Config
}

func phoenixABIPath() string {
	_, filename, _, _ := runtime.Caller(0) // this file: ethereum.go
	dir := filepath.Dir(filename)          // src/infrastructure/ethereum
	return filepath.Join(dir, "phoenixAbi.json")
}

// NewEthereumClient initializes the client
func NewEthereumClient(ctx context.Context, config Config) (*EthereumClient, error) {
	if config.RPCURL == "" || config.PrivateKey == "" {
		return nil, fmt.Errorf("%w: RPC_URL or PRIVATE_KEY", ErrMissingEnvVars)
	}
	config.abiFiles = map[string]string{
		"PHOENIX": phoenixABIPath(),
	}
	client, err := ethclient.DialContext(ctx, config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectNetwork, err)
	}
	key := strings.TrimPrefix(config.PrivateKey, "0x")

	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}
	wallet := crypto.PubkeyToAddress(privateKey.PublicKey)

	contracts := make(map[string]*bind.BoundContract)
	abis := make(map[string]abi.ABI)

	// Load custom ABIs if provided
	for name, path := range config.abiFiles {
		abiData, err := os.ReadFile(path)
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("%w: %s: %v", ErrReadABI, path, err)
		}
		parsedABI, err := abi.JSON(io.NopCloser(bytes.NewReader(abiData)))
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("%w: %s: %v", ErrParseABI, path, err)
		}
		abis[name] = parsedABI
	}

	// Load universal ERC20 ABI once
	erc20Parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("%w: ERC20 ABI: %v", ErrParseABI, err)
	}
	abis["erc20"] = erc20Parsed

	// Register supported tokens
	for symbol, addr := range config.SupportedTokens {
		contracts[strings.ToUpper(symbol)] = bind.NewBoundContract(common.HexToAddress(addr), erc20Parsed, client, client, client)
	}

	// Phoenix contract (if required)
	if config.PhoenixContract != "" {
		phoenixABI, ok := abis[phoenixProtocol]
		if !ok {
			return nil, fmt.Errorf("%w: phoenix ABI missing", ErrMissingEnvVars)
		}
		contracts[phoenixProtocol] = bind.NewBoundContract(common.HexToAddress(config.PhoenixContract), phoenixABI, client, client, client)
	}

	return &EthereumClient{
		client:     client,
		wallet:     wallet,
		privateKey: privateKey,
		contracts:  contracts,
		abi:        abis,
		config:     config,
	}, nil
}

func (ec *EthereumClient) Close() { ec.client.Close() }

func (ec *EthereumClient) WalletAddress() common.Address { return ec.wallet }

// ExecuteTradeWithPermit remains phoenix-specific
func (ec *EthereumClient) ExecuteTradeWithPermit(ctx context.Context, params Params) (*types.Receipt, error) {
	fmt.Printf("Admin Wallet: %s\n", ec.wallet.Hex())

	quoteIDBytes32 := common.BytesToHash([]byte(params.QuoteID))

	auth, err := bind.NewKeyedTransactorWithChainID(ec.privateKey, ec.config.ChainID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateTransactor, err)
	}

	contract, exists := ec.contracts[phoenixProtocol]
	if !exists {
		return nil, fmt.Errorf("%w: phoenix contract not initialized", ErrMissingEnvVars)
	}

	// Call (dry run)
	var result []interface{}
	if err := contract.Call(nil, &result, "executeTradeWithPermit",
		params.TokenAddress, params.UserAddress, params.Amount, params.Deadline,
		quoteIDBytes32, params.Signature.V, params.Signature.R, params.Signature.S,
	); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContractCall, err)
	}

	// Send TX
	tx, err := contract.Transact(auth, "executeTradeWithPermit",
		params.TokenAddress, params.UserAddress, params.Amount, params.Deadline,
		quoteIDBytes32, params.Signature.V, params.Signature.R, params.Signature.S,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSendTransaction, err)
	}

	fmt.Printf("TX sent: %s\n", tx.Hash().Hex())
	receipt, err := bind.WaitMined(ctx, ec.client, tx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMineTransaction, err)
	}
	if receipt.Status != 1 {
		return receipt, fmt.Errorf("%w: trade failed", ErrMineTransaction)
	}
	return receipt, nil
}

// WithdrawTreasury is now general
func (ec *EthereumClient) WithdrawTreasury(ctx context.Context, params WithdrawTreasuryParams) (*types.Receipt, error) {
	symbol := strings.ToUpper(params.TokenSymbol)

	if symbol == "ETH" {
		amountWei, ok := new(big.Int).SetString(params.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("%w: %s to wei", ErrInvalidAmount, params.Amount)
		}
		auth, err := bind.NewKeyedTransactorWithChainID(ec.privateKey, ec.config.ChainID)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCreateTransactor, err)
		}
		auth.Value = amountWei

		tx := types.NewTransaction(
			uint64(auth.Nonce.Int64()),
			common.HexToAddress(params.RecipientAddress),
			amountWei, auth.GasLimit, auth.GasPrice, nil,
		)
		signedTx, err := auth.Signer(auth.From, tx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSendTransaction, err)
		}
		if err := ec.client.SendTransaction(ctx, signedTx); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSendTransaction, err)
		}
		return bind.WaitMined(ctx, ec.client, signedTx)
	}

	// ERC20 withdrawal
	contract, ok := ec.contracts[symbol]
	if !ok {
		return nil, fmt.Errorf("%w: %s not supported", ErrUnsupportedToken, symbol)
	}

	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAmount, params.Amount)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(ec.privateKey, ec.config.ChainID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateTransactor, err)
	}

	tx, err := contract.Transact(auth, "transfer",
		common.HexToAddress(params.RecipientAddress),
		amount,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSendTransaction, err)
	}
	return bind.WaitMined(ctx, ec.client, tx)
}
