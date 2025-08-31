package http

import (
	"encoding/json"
	"net/http"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/order/usecase"
	"github.com/gin-gonic/gin"
)

// Handler binds usecase + logger
type Handler struct {
	swapSvc *usecase.Service
	logger  *logger.Logger
}

func NewHandler(s *usecase.Service, l *logger.Logger) *Handler {
	return &Handler{swapSvc: s, logger: l}
}
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// r.GET("/markets", h.ListPairs)
	// r.PUT("/market/best-price", h.GetBestExchangePriceByVolume)
	// r.GET("/health", func(c *gin.Context) {
	// 	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	// })
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// // ListPairs godoc
// //
// //	@Summary		List available swap pairs
// //	@Description	Get all available trading pairs
// //	@Tags			swap
// //	@Accept			json
// //	@Produce		json
// //	@Success		200	{object}	http.ListPairsResponseBody
// //	@Failure		500	{object}	object{error=string}
// //	@Router			/swap/pairs [get]
// func (h *Handler) ListPairs(w http.ResponseWriter, r *http.Request) {
// 	ctx := r.Context()
// 	pairs, err := h.swapSvc.ListPairs(ctx)
// 	if err != nil {
// 		h.logger.Errorf("ListPairs err: %v", err)
// 		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
// 		return
// 	}

// 	dtoPairs := make([]PairDTO, len(pairs))
// 	for i, p := range pairs {
// 		dtoPairs[i] = PairDTO{
// 			FromNetwork: p["from_network"],
// 			FromToken:   p["from_token"],
// 			ToNetwork:   p["to_network"],
// 			ToToken:     p["to_token"],
// 		}
// 	}
// 	writeJSON(w, http.StatusOK, ListPairsResponseBody{Pairs: dtoPairs})
// }

// // swagger:route POST /swap/quote swap createQuote
// // Create a swap quote
// //
// // Responses:
// //
// //	200: CreateQuoteResponseBody
// //	400: BadRequest
// func (h *Handler) CreateQuote(w http.ResponseWriter, r *http.Request) {
// 	var reqBody CreateQuoteRequestBody
// 	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
// 		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
// 		return
// 	}
// 	amount, err := decimal.NewFromString(reqBody.AmountIn)
// 	if err != nil {
// 		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid amount"})
// 		return
// 	}

// 	q, err := h.swapSvc.CreateQuote(context.Background(), usecase.CreateQuoteRequest{
// 		FromNetwork: reqBody.FromNetwork,
// 		FromToken:   reqBody.FromToken,
// 		ToNetwork:   reqBody.ToNetwork,
// 		ToToken:     reqBody.ToToken,
// 		AmountIn:    amount,
// 		UserAddress: reqBody.UserAddress,
// 	})
// 	if err != nil {
// 		h.logger.Errorf("CreateQuote err: %v", err)
// 		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
// 		return
// 	}

// 	resp := CreateQuoteResponseBody{
// 		QuoteID:     q.ID,
// 		AmountIn:    q.AmountIn,
// 		AmountOut:   q.AmountOut,
// 		ExpiresAt:   q.ExpiresAt,
// 		FromNetwork: q.FromNetwork,
// 		FromToken:   q.FromToken,
// 		ToNetwork:   q.ToNetwork,
// 		ToToken:     q.ToToken,
// 	}
// 	writeJSON(w, http.StatusOK, resp)
// }

// // swagger:route POST /swap/execute swap executeQuote
// // Execute an existing quote
// //
// // Responses:
// //
// //	200: ExecuteQuoteResponseBody
// //	400: BadRequest
// func (h *Handler) ExecuteQuote(w http.ResponseWriter, r *http.Request) {
// 	var reqBody ExecuteQuoteRequestBody
// 	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
// 		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
// 		return
// 	}

// 	tx1, tx2, err := h.swapSvc.ExecuteQuote(context.Background(), usecase.ExecuteRequest{
// 		QuoteID:        reqBody.QuoteID,
// 		Permit:         reqBody.Permit,
// 		RequestingUser: reqBody.RequestingUser,
// 	})
// 	if err != nil {
// 		h.logger.Errorf("ExecuteQuote err: %v", err)
// 		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
// 		return
// 	}

// 	resp := ExecuteQuoteResponseBody{
// 		TxWithdraw: tx1,
// 		TxDeposit:  tx2,
// 		ExecutedAt: time.Now().UTC(),
// 	}
// 	writeJSON(w, http.StatusOK, resp)
// }
