// Package ompfinex implements a strongly-typed HTTP client for the OMPFinex REST API.
//
// Coverage: Implements the major resources exposed in https://docs.ompfinex.com/
// including auth, user, markets, orders, wallets, deposits/withdrawals (rial & crypto),
// verification (KYC/bank card), alerts, sessions, 2FA, currencies, and favorites.
//
// Notes:
//   - API responses generally include a top-level {status, data, message, token, time, time2}.
//   - When status != "OK", this client returns an error enriched with status/message.
//   - Pagination schema is also supported when present.
//   - Some endpoints in docs accept multipart/form-data. Helpers included.
//   - The API docs include both /v1 and /v2+/v3 paths; update constants as needed.
//   - This file is intentionally self-contained for rapid adoption in services/CLI/tests.
//
// Production hardening you may consider next:
//   - Retry/backoff on 429/5xx with idempotency for GET and safe POSTs
//   - Circuit breaker / rate-limiting client side
//   - Structured metrics and tracing hooks
//   - Timeouts per operation and context propagation
//   - Token refresh flow if/when introduced by the API
package ompfinex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// Default HTTP timeouts tuned for server-side usage.
var (
	DefaultHTTPClient = &http.Client{Timeout: 30 * time.Second}
)

// NewClient constructs a new API client. base should be like "https://api.ompfinex.com".
func NewClient(base string, opts ...Option) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}
	c := &Client{
		BaseURL:   u,
		HTTP:      DefaultHTTPClient,
		UserAgent: "ompfinex-go/1.0",
		Logger:    log.Logger,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Option functional options.
type Option func(*Client)

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.HTTP = h } }
func WithAuthToken(token string) Option    { return func(c *Client) { c.AuthToken = token } }
func WithUserAgent(ua string) Option       { return func(c *Client) { c.UserAgent = ua } }

type Client struct {
	BaseURL   *url.URL
	HTTP      *http.Client
	AuthToken string
	UserAgent string
	Logger    zerolog.Logger // structured logger
}

// WithLogger allows plugging in structured logger
func WithLogger(l zerolog.Logger) Option {
	return func(c *Client) { c.Logger = l }
}

// --- Core HTTP execution with logging ---
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
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.setAuth(req)

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
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	// --- Envelope check ---
	switch v := out.(type) {
	case *ResponseEnvelope[json.RawMessage]:
		if err := apiError(v.Status, v.Message, b); err != nil {
			return err
		}
	}
	return nil
}

// --- Error conversion with logging ---
func apiError(status, message string, body []byte) error {
	if status == "OK" {
		return nil
	}
	if message == "" {
		message = http.StatusText(http.StatusBadRequest)
	}
	tail := truncateString(string(body), 512)
	return fmt.Errorf("ompfinex api error: status=%s message=%s body=%s", status, message, tail)
}

// --- Helpers ---
func truncateJSON(b []byte, max int) []byte {
	if len(b) > max {
		return b[:max]
	}
	return b
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// setAuth sets Authorization header if token present.
func (c *Client) setAuth(req *http.Request) {
	if c.AuthToken == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.AuthToken)
}

// --- Common response envelopes & pagination ---

type ResponseEnvelope[T any] struct {
	Status  string          `json:"status"`
	Data    T               `json:"data"`
	Message string          `json:"message,omitempty"`
	Token   string          `json:"token,omitempty"`
	Time    string          `json:"time,omitempty"`
	Time2   string          `json:"time2,omitempty"`
	Errors  json.RawMessage `json:"errors,omitempty"`
	// Optional pagination lives outside data in some responses; the API shows a separate object.
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	TotalRecords int `json:"total_records"`
	PerPage      int `json:"per_page"`
	Page         int `json:"page"`
	TotalPages   int `json:"total_pages"`
}

// doJSON decodes into a typed envelope and returns data.
func doJSON[T any](c *Client, ctx context.Context, method, p string, q url.Values, in any, contentType string) (T, error) {
	var env ResponseEnvelope[T]
	err := c.do(ctx, method, p, q, in, &env, contentType)
	if err != nil {
		var zero T
		return zero, err
	}
	if err := apiError(env.Status, env.Message, nil); err != nil {
		var zero T
		return zero, err
	}
	// surface token when present; caller may set it.
	if env.Token != "" {
		c.AuthToken = env.Token
	}
	return env.Data, nil
}
func doJSONRaw[T any](c *Client, ctx context.Context, method, path string, query url.Values, body any, contentType string) (T, error) {
	var out T
	if err := c.do(ctx, method, path, query, body, &out, contentType); err != nil {
		return out, err
	}
	return out, nil
}

// --- Auth & User ---

type SignUpRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	RecaptchaToken string `json:"recaptcha_token"`
	ReferralCode   string `json:"referral_code,omitempty"`
}

type SignUpData struct {
	UID int64 `json:"uid"`
}

func (c *Client) SignUp(ctx context.Context, in SignUpRequest) (SignUpData, error) {
	return doJSON[SignUpData](c, ctx, http.MethodPost, "/v1/user/sign-up", nil, in, "")
}

type SignInRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	RecaptchaToken string `json:"recaptcha_token,omitempty"`
	Code           string `json:"code,omitempty"` // Google Authenticator code (2FA)
}

type SignInData struct {
	UID                 int64  `json:"uid"`
	Email               string `json:"email"`
	EmailVerified       string `json:"email_verified"`
	PhoneVerified       string `json:"phone_verified"`
	IdentityCardVerfied string `json:"identity_card_verified"`
	BankVerified        string `json:"bank_verified"`
	AddressVerified     string `json:"address_verified"`
	IdentityVerified    string `json:"identity_verified"`
	GoogleAuthEnabled   bool   `json:"google_auth_enabled"`
}

// SignIn returns SignInData and stores bearer token if provided in envelope.
func (c *Client) SignIn(ctx context.Context, in SignInRequest) (SignInData, error) {
	return doJSON[SignInData](c, ctx, http.MethodPost, "/v1/user/sign-in", nil, in, "")
}

type User struct {
	UID                  int64  `json:"uid"`
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	Email                string `json:"email"`
	Birthday             string `json:"birthday"`
	Phone                string `json:"phone"`
	NationalID           string `json:"national_id"`
	Gender               string `json:"gender"`
	EmailVerified        string `json:"email_verified"`
	PhoneVerified        string `json:"phone_verified"`
	IdentityCardVerified string `json:"identity_card_verified"`
	LandlineVerified     string `json:"landline_phone_verified"`
	BankVerified         string `json:"bank_verified"`
}

func (c *Client) GetUser(ctx context.Context) (User, error) {
	return doJSON[User](c, ctx, http.MethodGet, "/v1/user", nil, nil, "")
}

// UpdateUser takes a partial map to avoid falling behind doc changes.
func (c *Client) UpdateUser(ctx context.Context, fields map[string]any) (User, error) {
	return doJSON[User](c, ctx, http.MethodPut, "/v1/user", nil, fields, "")
}

// DeleteUserTokens logs out from all sessions.
func (c *Client) DeleteUserTokens(ctx context.Context) error {
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodDelete, "/v1/user/token", nil, nil, "")
	return err
}

// --- Credit Card (Bank card) verification ---

type CreditCard struct {
	ID       int64  `json:"id"`
	Card     string `json:"card"` // masked
	Name     string `json:"name"`
	Verified string `json:"verified"`
	Created  string `json:"created_at"`
}

type CreditCardCreateRequest struct {
	Card string `json:"card"` // 16 digits
}

func (c *Client) CreditCardCreate(ctx context.Context, cardNumber string) error {
	in := CreditCardCreateRequest{Card: cardNumber}
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/verification/credit-card", nil, in, "")
	return err
}

// CreditCardDelete deletes by card number or id depending on API; docs vary.
// We support both, preferring id if provided.
func (c *Client) CreditCardDelete(ctx context.Context, id *int64, card string) error {
	payload := map[string]any{}
	if id != nil {
		payload["id"] = *id
	} else if card != "" {
		payload["card"] = card
	} else {
		return errors.New("id or card required")
	}
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodDelete, "/v1/user/verification/credit-card", nil, payload, "")
	return err
}

func (c *Client) CreditCardList(ctx context.Context) ([]CreditCard, error) {
	return doJSON[[]CreditCard](c, ctx, http.MethodGet, "/v1/user/verification/credit-card", nil, nil, "")
}

// --- Personal Information (KYC) upload (multipart form) ---

type PersonalInfoResponse struct {
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Gender             string `json:"gender"`
	Birthday           string `json:"birthday"`
	NationalID         string `json:"national_id"`
	IdentityCardStatus string `json:"identity_card_verified"`
}

func (c *Client) SubmitPersonalInformation(ctx context.Context, birthday, nationalID string, nationalIDImage []byte, filename string, address *string) (PersonalInfoResponse, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("birthday", birthday)
	_ = mw.WriteField("national_id", nationalID)
	if address != nil {
		_ = mw.WriteField("address", *address)
	}
	fw, err := mw.CreateFormFile("national_id_image", filename)
	if err != nil {
		return PersonalInfoResponse{}, err
	}
	if _, err := fw.Write(nationalIDImage); err != nil {
		return PersonalInfoResponse{}, err
	}
	if err := mw.Close(); err != nil {
		return PersonalInfoResponse{}, err
	}
	return doJSON[PersonalInfoResponse](c, ctx, http.MethodPost, "/v1/user/verification/personal-information", nil, bytes.NewReader(buf.Bytes()), mw.FormDataContentType())
}

// --- Alerts ---

type PriceAlert struct {
	ID        int64           `json:"id"`
	MarketID  int64           `json:"market_id"`
	Price     decimal.Decimal `json:"price"`
	Direction string          `json:"direction"` // e.g., ABOVE/BELOW
	CreatedAt string          `json:"created_at"`
}

type CreatePriceAlertRequest struct {
	MarketID  int64           `json:"market_id"`
	Price     decimal.Decimal `json:"price"`
	Direction string          `json:"direction"`
}

func (c *Client) CreatePriceAlert(ctx context.Context, in CreatePriceAlertRequest) (PriceAlert, error) {
	return doJSON[PriceAlert](c, ctx, http.MethodPost, "/v1/user/alert", nil, in, "")
}

func (c *Client) ListPriceAlerts(ctx context.Context) ([]PriceAlert, error) {
	return doJSON[[]PriceAlert](c, ctx, http.MethodGet, "/v1/user/alert", nil, nil, "")
}

func (c *Client) DeletePriceAlert(ctx context.Context, id int64) error {
	p := fmt.Sprintf("/v1/user/alert/%d", id)
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodDelete, p, nil, nil, "")
	return err
}

// --- Markets ---

type CurrencyInfo struct {
	ID       string `json:"id"`
	IconPath string `json:"icon_path"`
	Name     string `json:"name"`
}

type Market struct {
	ID                int64           `json:"id"`
	BaseCurrency      CurrencyInfo    `json:"base_currency"`
	QuoteCurrency     CurrencyInfo    `json:"quote_currency"`
	Name              string          `json:"name"`
	MinPrice          decimal.Decimal `json:"min_price"`
	MaxPrice          decimal.Decimal `json:"max_price"`
	LastPrice         decimal.Decimal `json:"last_price"`
	DayChangePercent  decimal.Decimal `json:"day_change_percent"`
	TradingViewSymbol string          `json:"tradingview_symbol"`
	LikedByUser       bool            `json:"liked_by_user"`
}

func (c *Client) ListMarkets(ctx context.Context) ([]Market, error) {
	return doJSON[[]Market](c, ctx, http.MethodGet, "/v1/market", nil, nil, "")
}

func (c *Client) GetMarket(ctx context.Context, id int64) (Market, error) {
	p := fmt.Sprintf("/v1/market/%d", id)
	return doJSON[Market](c, ctx, http.MethodGet, p, nil, nil, "")
}

// Favorites
func (c *Client) AddMarketToFavorites(ctx context.Context, id int64) error {
	p := fmt.Sprintf("/v1/market/%d/favorite", id)
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, p, nil, nil, "")
	return err
}
func (c *Client) ListFavoriteMarkets(ctx context.Context) ([]Market, error) {
	return doJSON[[]Market](c, ctx, http.MethodGet, "/v1/market/favorite", nil, nil, "")
}

// --- Orders ---

type OrderSide string

const (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"
)

type OrderType string

const (
	OrderLimit  OrderType = "limit"
	OrderMarket OrderType = "market"
)

type PlaceOrderRequest struct {
	MarketID int64            `json:"market_id"`
	Side     OrderSide        `json:"side"`
	Type     OrderType        `json:"type"`
	Price    *decimal.Decimal `json:"price,omitempty"`
	Amount   decimal.Decimal  `json:"amount"`
}

type Order struct {
	ID        int64           `json:"id"`
	MarketID  int64           `json:"market_id"`
	Side      OrderSide       `json:"side"`
	Type      OrderType       `json:"type"`
	Price     decimal.Decimal `json:"price"`
	Amount    decimal.Decimal `json:"amount"`
	Filled    decimal.Decimal `json:"filled"`
	Status    string          `json:"status"`
	CreatedAt string          `json:"created_at"`
}
type OrderId struct {
	ID int64 `json:"id"`
}

func (c *Client) PlaceOrder(ctx context.Context, in PlaceOrderRequest) (OrderId, error) {
	p := fmt.Sprintf("/v1/market/%d/order", in.MarketID)
	return doJSON[OrderId](c, ctx, http.MethodPost, p, nil, in, "")
}
func (c *Client) CancelOrder(ctx context.Context, orderId int64) (interface{}, error) {
	p := fmt.Sprintf("/v1/user/order?id=%d", orderId)
	return doJSON[interface{}](c, ctx, http.MethodDelete, p, nil, nil, "")
}

func (c *Client) GetOrder(ctx context.Context, id int64) (Order, error) {
	p := fmt.Sprintf("/v1/order/%d", id)
	return doJSON[Order](c, ctx, http.MethodGet, p, nil, nil, "")
}

// ListUserOrders supports optional market_id and pagination.
func (c *Client) ListUserOrders(ctx context.Context, marketID *int64, page, limit int) ([]Order, *Pagination, error) {
	q := url.Values{}
	if marketID != nil {
		q.Set("market_id", fmt.Sprint(*marketID))
	}
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprint(limit))
	}
	var env ResponseEnvelope[[]Order]
	if err := c.do(ctx, http.MethodGet, "/v1/order", q, nil, &env, ""); err != nil {
		return nil, nil, err
	}
	if err := apiError(env.Status, env.Message, nil); err != nil {
		return nil, nil, err
	}
	return env.Data, env.Pagination, nil
}

// --- Wallets: last-used (recent external wallets) ---

type LastUsedWallet struct {
	ID            int64   `json:"id"`
	CurrencyToken string  `json:"currency_token"`
	Name          string  `json:"name"`
	Memo          *string `json:"memo"`
	Wallet        string  `json:"wallet"`
	CreatedAt     string  `json:"created_at"`
}

func (c *Client) ListLastUsedWallets(ctx context.Context, currencyToken *string) ([]LastUsedWallet, error) {
	p := "/v1/user/wallet/last-used"
	if currencyToken != nil && *currencyToken != "" {
		p += "/" + url.PathEscape(*currencyToken)
	}
	return doJSON[[]LastUsedWallet](c, ctx, http.MethodGet, p, nil, nil, "")
}

type SaveLastUsedWalletRequest struct {
	CurrencyToken string  `json:"currency_token"`
	Wallet        string  `json:"wallet"`
	Name          *string `json:"name,omitempty"`
	Memo          *string `json:"memo,omitempty"`
}

func (c *Client) SaveLastUsedWallet(ctx context.Context, in SaveLastUsedWalletRequest) (LastUsedWallet, error) {
	return doJSON[LastUsedWallet](c, ctx, http.MethodPost, "/v1/user/wallet/last-used", nil, in, "")
}

func (c *Client) UpdateLastUsedWallet(ctx context.Context, id int64, wallet, name string, memo *string) (LastUsedWallet, error) {
	p := fmt.Sprintf("/v1/user/wallet/last-used/%d", id)
	payload := map[string]any{"wallet": wallet}
	if name != "" {
		payload["name"] = name
	}
	if memo != nil {
		payload["memo"] = *memo
	}
	return doJSON[LastUsedWallet](c, ctx, http.MethodPut, p, nil, payload, "")
}

func (c *Client) DeleteLastUsedWallet(ctx context.Context, id int64) error {
	p := fmt.Sprintf("/v1/user/wallet/last-used/%d", id)
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodDelete, p, nil, nil, "")
	return err
}

// --- Currencies ---

type Currency struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	NameFA        string          `json:"name_fa,omitempty"`
	NetworkTag    string          `json:"network_tag,omitempty"`
	WithdrawFee   decimal.Decimal `json:"withdraw_fee,omitempty"`
	MinWithdraw   decimal.Decimal `json:"min_withdraw,omitempty"`
	Confirmations int             `json:"confirmations,omitempty"`
}

func (c *Client) GetCurrency(ctx context.Context, id string) (Currency, error) {
	p := fmt.Sprintf("/v3/currency/%s", url.PathEscape(id))
	return doJSON[Currency](c, ctx, http.MethodGet, p, nil, nil, "")
}

func (c *Client) ListCurrencies(ctx context.Context) ([]Currency, error) {
	return doJSON[[]Currency](c, ctx, http.MethodGet, "/v2/currencies", nil, nil, "")
}

// --- Deposits / Withdrawals (Rial) ---

type RialDepositInitRequest struct {
	CardID *int64  `json:"card_id,omitempty"`
	Amount *int64  `json:"amount,omitempty"`
	Type   *string `json:"type,omitempty"`    // e.g., "iban"
	IBANID *int64  `json:"iban_id,omitempty"` // when Type == iban
}

type RialDepositInitResponse struct {
	URL string `json:"url"`
}

func (c *Client) RialDepositInit(ctx context.Context, in RialDepositInitRequest) (RialDepositInitResponse, error) {
	return doJSON[RialDepositInitResponse](c, ctx, http.MethodPost, "/v1/user/deposit/rial", nil, in, "")
}

// Rial deposit callback confirm (bank redirect). Payload varies per provider; accept a bag.
func (c *Client) RialDepositConfirm(ctx context.Context, payload map[string]any) error {
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/deposit/rial/callback", nil, payload, "")
	return err
}

// --- Withdrawals (Rial) ---

type RialWithdrawRequest struct {
	Amount int64 `json:"amount"`
	IBANID int64 `json:"iban_id"`
}

type FeeResponse struct {
	Fee decimal.Decimal `json:"fee"`
}

func (c *Client) RialWithdraw(ctx context.Context, in RialWithdrawRequest) error {
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/withdraw/rial", nil, in, "")
	return err
}

func (c *Client) RialWithdrawConfirm(ctx context.Context, code string) error {
	payload := map[string]any{"code": code}
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/withdraw/rial/confirm", nil, payload, "")
	return err
}

func (c *Client) RialWithdrawFee(ctx context.Context, amount int64) (FeeResponse, error) {
	q := url.Values{"amount": {fmt.Sprint(amount)}}
	return doJSON[FeeResponse](c, ctx, http.MethodGet, "/v1/user/withdraw/rial/fee", q, nil, "")
}

// --- Crypto deposits ---

type DepositAddress struct {
	CurrencyToken string `json:"currency_token"`
	Address       string `json:"address"`
	Memo          string `json:"memo,omitempty"`
}

func (c *Client) GetDepositWallet(ctx context.Context, currencyToken string) (DepositAddress, error) {
	p := fmt.Sprintf("/v1/user/wallet/%s", url.PathEscape(currencyToken))
	return doJSON[DepositAddress](c, ctx, http.MethodGet, p, nil, nil, "")
}

// Force wallet balance refresh for a crypto currency.
func (c *Client) RefreshDepositBalance(ctx context.Context, currencyToken string) error {
	p := fmt.Sprintf("/v1/user/wallet/%s/refresh", url.PathEscape(currencyToken))
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, p, nil, nil, "")
	return err
}

// --- Crypto withdrawals ---

type CryptoWithdrawRequest struct {
	CurrencyToken string          `json:"currency_token"`
	Amount        decimal.Decimal `json:"amount"`
	Address       string          `json:"address"`
	Memo          string          `json:"memo,omitempty"`
}

func (c *Client) CryptoWithdraw(ctx context.Context, in CryptoWithdrawRequest) error {
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/withdraw/crypto", nil, in, "")
	return err
}

func (c *Client) CryptoWithdrawConfirm(ctx context.Context, code string) error {
	payload := map[string]any{"code": code}
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/withdraw/crypto/confirm", nil, payload, "")
	return err
}

// --- Sessions ---

type Session struct {
	ID        int64  `json:"id"`
	Device    string `json:"device"`
	IP        string `json:"ip"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) ListSessions(ctx context.Context, page, limit int) ([]Session, *Pagination, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprint(limit))
	}
	var env ResponseEnvelope[[]Session]
	if err := c.do(ctx, http.MethodGet, "/v1/user/sessions", q, nil, &env, ""); err != nil {
		return nil, nil, err
	}
	if err := apiError(env.Status, env.Message, nil); err != nil {
		return nil, nil, err
	}
	return env.Data, env.Pagination, nil
}

func (c *Client) DeleteSession(ctx context.Context, id int64) error {
	p := fmt.Sprintf("/v1/user/sessions/%d", id)
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodDelete, p, nil, nil, "")
	return err
}

// --- Two-Factor (Google Auth) ---

type GoogleAuthKey struct {
	Secret string `json:"secret"`
	QR     string `json:"qr_code"`
}

func (c *Client) GetGoogleAuthSecret(ctx context.Context) (GoogleAuthKey, error) {
	return doJSON[GoogleAuthKey](c, ctx, http.MethodGet, "/v1/user/google-auth", nil, nil, "")
}

func (c *Client) ConfirmGoogleAuth(ctx context.Context, code string) error {
	payload := map[string]any{"code": code}
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodPost, "/v1/user/google-auth", nil, payload, "")
	return err
}

func (c *Client) DisableGoogleAuth(ctx context.Context, password string) error {
	payload := map[string]any{"password": password}
	_, err := doJSON[json.RawMessage](c, ctx, http.MethodDelete, "/v1/user/google-auth", nil, payload, "")
	return err
}

// MarketOrderBook represents the order book snapshot for a market.
// It contains separate arrays for asks (sell side) and bids (buy side).
// Each entry is [price, volume].
type MarketOrder struct {
	ID      int64           `json:"id"`
	Amount  decimal.Decimal `json:"amount"`
	Price   decimal.Decimal `json:"price"`
	Type    string          `json:"type"`     // "buy" or "sell"
	MyOrder bool            `json:"my_order"` // whether it belongs to current user
}

// GetMarketOrderBook fetches the current order book for a given market (e.g., "BTCIRT").
//
// Example usage:
//
//	ob, err := cli.GetMarketOrderBook(ctx, "BTCIRT")
//	if err != nil { ... }
//	fmt.Println("Top bid:", ob.Bids[0], "Top ask:", ob.Asks[0])
func (c *Client) GetMarketOrders(ctx context.Context, marketID int64) ([]MarketOrder, error) {
	path := fmt.Sprintf("/v1/market/%d/order", marketID)
	return doJSON[[]MarketOrder](c, ctx, http.MethodGet, path, nil, nil, "")
}

type OrderBookEntry struct {
	Amount decimal.Decimal `json:"amount"`
	Price  decimal.Decimal `json:"price"`
}

type MarketOrderBook struct {
	Volume24h string           `json:"24h_volume"`
	Asks      []OrderBookEntry `json:"asks"`
	Bids      []OrderBookEntry `json:"bids"`
}

func (c *Client) GetMarketOrderBook(ctx context.Context) (map[string]MarketOrderBook, error) {
	return doJSON[map[string]MarketOrderBook](c, ctx, http.MethodGet, "/v1/orderbook", url.Values{"limit": {"100"}}, nil, "")
}

type OrderBook struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Time         int64      `json:"time"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

func (c *Client) GetMarketDepth(ctx context.Context, marketID string) (OrderBook, error) {
	return doJSON[OrderBook](c, ctx, http.MethodGet, fmt.Sprintf("/v1/market/%s/depth", marketID), url.Values{"limit": {"200"}}, nil, "")
}

// --- Utility helpers ---

// Bool returns a pointer to b.
func Bool(b bool) *bool { return &b }

// String returns a pointer to s.
func String(s string) *string { return &s }

// Int64 returns a pointer to i.
func Int64(i int64) *int64 { return &i }
