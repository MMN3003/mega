// Package wallex implements a strongly-typed HTTP client for the Wallex REST API.
//
// Coverage: Implements market data endpoints including:
// - All markets listing
// - Order book depth
//
// Notes:
// - API responses follow a {result, message, success} envelope pattern
// - When success != true, this client returns an error enriched with the message
// - Requires x-api-key header for authentication
// - Base URL is fixed to https://api.wallex.ir
package wallex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// Default HTTP timeouts tuned for server-side usage
var (
	DefaultHTTPClient = &http.Client{Timeout: 30 * time.Second}
)

// NewClient constructs a new API client with the provided API key
func NewClient(baseUrl string, opts ...Option) (*Client, error) {
	if baseUrl == "" {
		return nil, errors.New("base url is required")
	}

	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	c := &Client{
		BaseURL:   u,
		HTTP:      DefaultHTTPClient,
		UserAgent: "wallex-go/1.0",
		Logger:    log.Logger,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Option functional options
type Option func(*Client)

func WithAPIKey(key string) Option         { return func(c *Client) { c.APIKey = key } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.HTTP = h } }
func WithUserAgent(ua string) Option       { return func(c *Client) { c.UserAgent = ua } }
func WithLogger(l zerolog.Logger) Option   { return func(c *Client) { c.Logger = l } }

type Client struct {
	BaseURL   *url.URL
	HTTP      *http.Client
	APIKey    string
	UserAgent string
	Logger    zerolog.Logger
}

// ResponseEnvelope is the standard response structure from Wallex API
type ResponseEnvelope[T any] struct {
	Result  T      `json:"result"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type Market struct {
	Symbol             string          `json:"symbol"`
	BaseAsset          string          `json:"base_asset"`
	QuoteAsset         string          `json:"quote_asset"`
	FaBaseAsset        string          `json:"fa_base_asset"`
	FaQuoteAsset       string          `json:"fa_quote_asset"`
	EnBaseAsset        string          `json:"en_base_asset"`
	EnQuoteAsset       string          `json:"en_quote_asset"`
	Categories         []int           `json:"categories"` // Changed from []string to []int based on response
	Price              decimal.Decimal `json:"price"`
	Change24h          float64         `json:"change_24h"`
	Volume24h          decimal.Decimal `json:"volume_24h"`
	Change7D           float64         `json:"change_7D"`
	QuoteVolume24h     decimal.Decimal `json:"quote_volume_24h"`
	SpotIsNew          bool            `json:"spot_is_new"`
	OtcIsNew           bool            `json:"otc_is_new"`
	IsNew              bool            `json:"is_new"`
	IsSpot             bool            `json:"is_spot"`
	IsOtc              bool            `json:"is_otc"`
	IsMargin           bool            `json:"is_margin"`
	IsTmnBased         bool            `json:"is_tmn_based"`
	IsUsdtBased        bool            `json:"is_usdt_based"`
	IsZeroFee          bool            `json:"is_zero_fee"`
	LeverageStep       float64         `json:"leverage_step"`
	MaxLeverage        float64         `json:"max_leverage"`
	CreatedAt          string          `json:"created_at"`
	AmountPrecision    int             `json:"amount_precision"`
	PricePrecision     int             `json:"price_precision"`
	Flags              []interface{}   `json:"flags"` // Could be empty array or contain mixed types
	IsMarketTypeEnable bool            `json:"is_market_type_enable"`
}

// MarketsResponse contains the list of all markets
type MarketsResponse struct {
	Markets []Market `json:"markets"`
}

// OrderBookEntry represents a single price level in the order book
type OrderBookEntry struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
	Sum      decimal.Decimal `json:"sum"`
}

// OrderBook represents the depth of a market
type OrderBook struct {
	Asks []OrderBookEntry `json:"ask"`
	Bids []OrderBookEntry `json:"bid"`
}

// --- Core HTTP execution with logging ---
func (c *Client) do(
	ctx context.Context,
	method, path string,
	query url.Values,
	out any,
) error {
	u := *c.BaseURL
	u.Path = path
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	// Set required headers
	req.Header.Set("x-api-key", c.APIKey)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	// Execute request
	start := time.Now()
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// Log response
	c.Logger.Info().
		Str("method", method).
		Str("url", u.String()).
		Int("status", resp.StatusCode).
		Str("duration", time.Since(start).String()).
		RawJSON("response", truncateJSON(body, 2048)). // safe logging
		Msg("http response")

	// Status check
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	// Decode into envelope first to check success status
	var env ResponseEnvelope[json.RawMessage]
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	if !env.Success {
		return fmt.Errorf("wallex api error: %s", env.Message)
	}

	// If no output type requested, return now
	if out == nil {
		return nil
	}

	// Decode the result into the requested type
	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("unmarshal result: %w", err)
	}

	return nil
}

// truncateJSON safely truncates JSON for logging
func truncateJSON(b []byte, max int) []byte {
	if len(b) > max {
		return b[:max]
	}
	return b
}

// --- Market Data Endpoints ---

// GetAllMarkets retrieves the list of all available markets
func (c *Client) GetAllMarkets(ctx context.Context) ([]Market, error) {
	var response struct {
		Markets []Market `json:"markets"`
	}

	if err := c.do(ctx, http.MethodGet, "/hector/web/v1/markets", nil, &response); err != nil {
		return nil, err
	}

	return response.Markets, nil
}

// GetMarketDepth retrieves the order book depth for a specific market
// symbol: The market symbol (e.g., "USDCUSDT")
func (c *Client) GetMarketDepth(ctx context.Context, symbol string) (*OrderBook, error) {
	var result OrderBook

	query := url.Values{}
	query.Set("symbol", symbol)

	if err := c.do(ctx, http.MethodGet, "/v1/depth", query, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
