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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
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

// --- Market Data Endpoints ---

// GetAllMarkets retrieves the list of all available markets
func (c *Client) GetAllMarkets(ctx context.Context) ([]Market, error) {
	result, err := doJSON[struct{ Markets []Market }](c, ctx, http.MethodGet, "/hector/web/v1/markets", nil, nil, "")
	if err != nil {
		return nil, err
	}

	return result.Markets, nil
}

// GetMarketDepth retrieves the order book depth for a specific market
// symbol: The market symbol (e.g., "USDCUSDT")
func (c *Client) GetMarketDepth(ctx context.Context, symbol string) (*OrderBook, error) {
	var result OrderBook

	query := url.Values{}
	query.Set("symbol", symbol)

	result, err := doJSON[OrderBook](c, ctx, http.MethodGet, "/v1/depth", query, nil, "")
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) do(
	ctx context.Context,
	method, p string,
	q url.Values,
	body any,
	out any,
	contentType string,
) error {
	u := *c.BaseURL
	u.Path = path.Join(u.Path, p)
	u.RawQuery = q.Encode()

	// --- Build request body ---
	var r io.Reader
	if body != nil {
		switch b := body.(type) {
		case io.Reader:
			r = b
		case []byte:
			r = bytes.NewReader(b)
		default:
			buf, err := json.Marshal(b)
			if err != nil {
				return fmt.Errorf("marshal body: %w", err)
			}
			r = bytes.NewReader(buf)
			if contentType == "" {
				contentType = "application/json"
			}
		}
	}

	// --- Build request ---
	req, err := http.NewRequestWithContext(ctx, method, u.String(), r)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	// Set required headers
	if c.APIKey != "" {
		req.Header.Set("x-api-key", c.APIKey)
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// --- Execute request ---
	start := time.Now()
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// --- Logging response ---
	c.Logger.Info().
		Str("method", method).
		Str("url", u.String()).
		Int("status", resp.StatusCode).
		Str("duration", time.Since(start).String()).
		RawJSON("response", truncateJSON(b, 2048)). // safe logging
		Msg("http response")

	// --- Status check ---
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http error %d: %s", resp.StatusCode, string(b))
	}

	// --- Decode output ---
	if out == nil {
		return nil
	}

	// Decode into envelope first to check success status
	var env ResponseEnvelope[json.RawMessage]
	if err := json.Unmarshal(b, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	if !env.Success {
		return fmt.Errorf("wallex api error: %s", env.Message)
	}

	// Decode the result into the requested type
	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("unmarshal result: %w", err)
	}

	return nil
}

// doJSON decodes into a typed envelope and returns data
func doJSON[T any](
	c *Client,
	ctx context.Context,
	method, path string,
	query url.Values,
	body any,
	contentType string,
) (T, error) {
	var out T
	err := c.do(ctx, method, path, query, body, &out, contentType)
	return out, err
}

// --- Helpers ---
func truncateJSON(b []byte, max int) []byte {
	if len(b) > max {
		return b[:max]
	}
	return b
}

type OrderResponse struct {
	Symbol            string `json:"symbol"`
	SourceMarket      string `json:"sourceMarket"`
	DestinationMarket string `json:"destinationMarket"`
	Type              string `json:"type"`
	Side              string `json:"side"`
	ClientOrderID     string `json:"clientOrderId"`
	TransactTime      int64  `json:"transactTime"`
	Price             string `json:"price"`
	OrigQty           string `json:"origQty"`
	ExecutedSum       string `json:"executedSum"`
	ExecutedQty       string `json:"executedQty"`
	ExecutedPrice     string `json:"executedPrice"`
	Sum               string `json:"sum"`
	ExecutedPercent   int    `json:"executedPercent"`
	Status            string `json:"status"`
	Active            bool   `json:"active"`
	Fills             []Fill `json:"fills"`
}

type Fill struct {
	Price               string                 `json:"price"`
	Quantity            string                 `json:"quantity"`
	Fee                 string                 `json:"fee"`
	FeeCoefficient      string                 `json:"feeCoefficient"`
	FeeAsset            map[string]interface{} `json:"feeAsset"` // left generic because JSON has {}
	Timestamp           time.Time              `json:"timestamp"`
	Symbol              string                 `json:"symbol"`
	Sum                 string                 `json:"sum"`
	MakerFeeCoefficient string                 `json:"makerFeeCoefficient"`
	TakerFeeCoefficient string                 `json:"takerFeeCoefficient"`
	IsBuyer             bool                   `json:"isBuyer"`
}
type PlaceMarketOrderRequest struct {
	Symbol   string          `json:"symbol"`             // Market symbol (e.g., "BTCUSDT")
	Side     OrderSide       `json:"side"`               // "buy" or "sell"
	Quantity decimal.Decimal `json:"quantity,omitempty"` // Amount to buy/sell (for market orders)
	From     string          `json:"from"`               // "otc"
}
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

func (c *Client) PlaceMarketOrder(ctx context.Context, symbol string, side OrderSide, quantity decimal.Decimal) (*OrderResponse, error) {
	// Validate inputs
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if side != OrderSideBuy && side != OrderSideSell {
		return nil, errors.New("side must be 'buy' or 'sell'")
	}
	if quantity == decimal.Zero {
		return nil, errors.New("quantity is required for market orders")
	}

	// Prepare request payload
	orderRequest := PlaceMarketOrderRequest{
		Symbol:   symbol,
		Side:     side,
		Quantity: quantity,
		From:     "otc",
	}

	// Execute POST request to OTC order endpoint
	response, err := doJSON[OrderResponse](c, ctx, http.MethodPost, "/v1/account/easy-trade/orders", nil, orderRequest, "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to place market order: %w", err)
	}

	return &response, nil
}
