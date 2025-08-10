// go-swap-backend — Minimal production-oriented Go backend for the sequence diagram
// -----------------------------------------------------------------------------
// README (top of file):
// Purpose: provide a pragmatic, extendable Go backend implementing the flows in the
// provided sequence diagram: /swap/pairs, /swap/quote, /swap/execute.
//
// Design principles:
// - Clear separation of concerns: handler layer, quote engine, treasury & chain clients.
// - Pluggable "on-chain client" that can be swapped between a mock and a real
//   go-ethereum ethclient-based implementation.
// - Minimal external dependencies (only `github.com/google/uuid` and go-ethereum imports
//   when "real chain" is enabled).
// - Environment-driven configuration; do NOT commit private keys.
//
// Quick start (mock mode):
// 1) go 1.20+
// 2) go mod init example.com/go-swap-backend
// 3) go get github.com/google/uuid
// 4) gorely: go run main.go
// 5) Example calls:
//    GET  http://localhost:8080/swap/pairs
//    POST http://localhost:8080/swap/quote   {"from_chain":"Sepolia","to_chain":"Mumbai","from_token":"USDT","to_token":"MATIC","amount":100}
//    POST http://localhost:8080/swap/execute {"quote_id":"<id>","permit_signature":"0x...","user_address":"0x..."}
//
// When moving to real mode:
// - set USE_REAL_CHAIN=true
// - provide RPC endpoints: SEPOLIA_RPC and MUMBAI_RPC
// - provide TREASURY_PRIVATE_KEY (for the treasury account that will submit transfers)
// - provide contract addresses and ABIs and implement the small TODOs marked in code.
//
// -----------------------------------------------------------------------------
package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NOTE: If you enable real chain interactions, uncomment the imports below
// and implement the TODOs in the code where marked.
/*
import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)
*/

// --- Simple types ---

type Chain string

const (
	ChainSepolia Chain = "Sepolia"
	ChainMumbai  Chain = "Mumbai"
)

type Token string

const (
	TokenUSDT Token = "USDT"
	TokenMATIC Token = "MATIC"
)

// Pair represents a tradable pair
type Pair struct {
	ID        string `json:"id"`
	FromChain string `json:"from_chain"`
	ToChain   string `json:"to_chain"`
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
}

// Quote is returned to the client and stored server-side until execution
type Quote struct {
	ID           string    `json:"id"`
	FromChain    Chain     `json:"from_chain"`
	ToChain      Chain     `json:"to_chain"`
	FromToken    Token     `json:"from_token"`
	ToToken      Token     `json:"to_token"`
	AmountIn     float64   `json:"amount_in"`
	AmountOut    float64   `json:"amount_out"`
	CreatedAt    time.Time `json:"created_at"`
	Expiration   time.Time `json:"expiration"` // short TTL for safety
	ExecutionFee float64   `json:"execution_fee"` // fee taken off amountOut
}

// ExecuteRequest sent by the app to perform the swap.
type ExecuteRequest struct {
	QuoteID         string `json:"quote_id"`
	PermitSignature string `json:"permit_signature"` // raw hex signature (EIP-2612 style in real world)
	UserAddress     string `json:"user_address"`
}

// ExecuteResponse minimal success response
type ExecuteResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	TxHashFrom string `json:"tx_hash_from,omitempty"`
	TxHashTo   string `json:"tx_hash_to,omitempty"`
}

// --- In-memory datastore ---

var (
	quoteStore   = map[string]*Quote{}
	quoteStoreMu sync.RWMutex
)

// --- Mocked treasury balances (would be on-chain in prod) ---
var treasuryBalances = map[Chain]map[Token]float64{
	ChainSepolia: {
		TokenUSDT: 10000.0,
	},
	ChainMumbai: {
		TokenMATIC: 5000.0,
	},
}

// Quote TTL
const quoteTTL = 2 * time.Minute

// --- Configuration flags via env ---

func envBool(key string) bool {
	v := os.Getenv(key)
	b, _ := strconv.ParseBool(v)
	return b
}

var useRealChain = envBool("USE_REAL_CHAIN")

// --- Handlers ---

func handleGetPairs(w http.ResponseWriter, r *http.Request) {
	// Return pairs where both treasuries have non-zero balance for the respective tokens
	pairs := []Pair{}

	// Very small set: USDT (Sepolia) -> MATIC (Mumbai)
	if treasuryBalances[ChainSepolia][TokenUSDT] > 0 && treasuryBalances[ChainMumbai][TokenMATIC] > 0 {
		pairs = append(pairs, Pair{ID: "sepolia-usdt->mumbai-matic", FromChain: string(ChainSepolia), ToChain: string(ChainMumbai), FromToken: string(TokenUSDT), ToToken: string(TokenMATIC)})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pairs)
}

// Quote request payload
type QuoteRequest struct {
	FromChain string  `json:"from_chain"`
	ToChain   string  `json:"to_chain"`
	FromToken string  `json:"from_token"`
	ToToken   string  `json:"to_token"`
	Amount    float64 `json:"amount"`
}

func handlePostQuote(w http.ResponseWriter, r *http.Request) {
	var req QuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	// Validate treasuries
	if treasuryBalances[Chain(req.FromChain)][Token(req.FromToken)] <= 0 {
		http.Error(w, "source treasury has insufficient balance", http.StatusBadRequest)
		return
	}
	if treasuryBalances[Chain(req.ToChain)][Token(req.ToToken)] <= 0 {
		http.Error(w, "destination treasury has insufficient balance", http.StatusBadRequest)
		return
	}

	// Use simple pricing engine: fixed rate + fee. In real world, call an oracle or AMM.
	rate, feePct := getRateAndFee(Chain(req.FromChain), Chain(req.ToChain), Token(req.FromToken), Token(req.ToToken))
	amountOut := req.Amount * rate
	executionFee := amountOut * feePct
	amountOutAfterFee := amountOut - executionFee

	q := &Quote{
		ID:           uuid.New().String(),
		FromChain:    Chain(req.FromChain),
		ToChain:      Chain(req.ToChain),
		FromToken:    Token(req.FromToken),
		ToToken:      Token(req.ToToken),
		AmountIn:     req.Amount,
		AmountOut:    amountOutAfterFee,
		CreatedAt:    time.Now().UTC(),
		Expiration:   time.Now().UTC().Add(quoteTTL),
		ExecutionFee: executionFee,
	}

	quoteStoreMu.Lock()
	quoteStore[q.ID] = q
	quoteStoreMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

// Simple fixed pricing for demo. Replace with real data source in prod.
func getRateAndFee(fromChain, toChain Chain, fromToken, toToken Token) (rate float64, feePct float64) {
	// Example: 1 USDT (Sepolia) => 0.985 MATIC (Mumbai) before fee
	// We'll return the pre-fee rate and a fee percent.
	if fromChain == ChainSepolia && toChain == ChainMumbai && fromToken == TokenUSDT && toToken == TokenMATIC {
		return 0.985, 0.005 // 0.5% fee
	}
	return 1.0, 0.01
}

func handlePostExecute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	quoteStoreMu.RLock()
	q, ok := quoteStore[req.QuoteID]
	quoteStoreMu.RUnlock()
	if !ok {
		http.Error(w, "quote not found", http.StatusNotFound)
		return
	}
	if time.Now().After(q.Expiration) {
		http.Error(w, "quote expired", http.StatusBadRequest)
		return
	}

	// Step 1: check user's on-chain balance (Sepolia) or rely on permit (EIP-2612) verification
	okBalance, err := checkUserBalanceAndPermit(q, req.UserAddress, req.PermitSignature)
	if err != nil || !okBalance {
		http.Error(w, "user balance / permit verification failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Step 2: call contract on Sepolia to transfer from user to treasury using permit
	txHashFrom, err := callExecuteTradeWithPermit(q, req.UserAddress, req.PermitSignature)
	if err != nil {
		log.Printf("error calling executeTradeWithPermit: %v", err)
		http.Error(w, "on-chain debit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 3: send to user on Mumbai from treasury
	txHashTo, err := sendFromTreasury(q, req.UserAddress)
	if err != nil {
		log.Printf("error sending from treasury: %v", err)
		http.Error(w, "treasury credit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Persist changes to in-memory treasury balances (for mock mode)
	if !useRealChain {
		// Subtract from Sepolia treasury
		treasuryBalances[q.FromChain][q.FromToken] -= q.AmountIn
		// Add nothing to treasury Mumbai because treasury pays out. Subtract from Mumbai treasury too.
		treasuryBalances[q.ToChain][q.ToToken] -= q.AmountOut
	}

	// Remove quote (single-use)
	quoteStoreMu.Lock()
	delete(quoteStore, req.QuoteID)
	quoteStoreMu.Unlock()

	resp := ExecuteResponse{Success: true, Message: "swap executed", TxHashFrom: txHashFrom, TxHashTo: txHashTo}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- The "on-chain" helpers ---

func checkUserBalanceAndPermit(q *Quote, userAddr string, permitSig string) (bool, error) {
	// In production: call token contract balanceOf(user) on q.FromChain and also verify the EIP-2612 permit signature.
	// Here, if useRealChain is true, you must implement the real checks (ethclient + contract ABI).
	if useRealChain {
		return false, errors.New("real chain mode not implemented in this template; implement balance and permit checks")
	}
	// mock: assume the user has enough balance and permit is valid when a non-empty signature is provided
	if permitSig == "" {
		return false, errors.New("empty permit signature")
	}
	return true, nil
}

func callExecuteTradeWithPermit(q *Quote, userAddr string, permitSig string) (string, error) {
	// In production you would:
	// 1) Load contract binding for Phoenix_S (on Sepolia) that implements executeTradeWithPermit(...)
	// 2) Build an authorized transactor using the treasury relayer private key
	// 3) Submit the tx and return the tx hash.
	
	if useRealChain {
		// TODO: implement using go-ethereum ethclient, parsed ABI and bound contract code.
		return "", errors.New("real chain transaction not implemented in template; implement contract call and transactor")
	}
	// mock: return a fabricated tx hash
	fake := fmt.Sprintf("0xfrom_tx_%s", uuid.New().String())
	return fake, nil
}

func sendFromTreasury(q *Quote, userAddr string) (string, error) {
	// In production: treasury account constructs and submits a transfer tx on q.ToChain to userAddr for q.AmountOut.
	if useRealChain {
		// TODO: implement using go-ethereum ethclient and the treasury private key.
		return "", errors.New("real chain send not implemented in template; implement treasury transfer")
	}
	// mock: return a fabricated tx hash
	fake := fmt.Sprintf("0xto_tx_%s", uuid.New().String())
	return fake, nil
}

// --- Utilities ---

// parseHexPrivKey parses a hex-encoded private key string (no 0x)
func parseHexPrivKey(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, err
	}
	// placeholder: in real code use crypto.ToECDSA(b)
	_ = b
	return nil, errors.New("not implemented in template")
}

// --- Main & server bootstrap ---

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting go-swap-backend on :%s (useRealChain=%v)", port, useRealChain)

	http.HandleFunc("/swap/pairs", withJSON(handleGetPairs))
	http.HandleFunc("/swap/quote", withJSON(handlePostQuote))
	http.HandleFunc("/swap/execute", withJSON(handlePostExecute))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// withJSON enforces method and JSON content-type ergonomics for brevity
func withJSON(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		// Allow preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet && r.Header.Get("Content-Type") == "" {
			// allow
		}
		fn(w, r)
	}
}
