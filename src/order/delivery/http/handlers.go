package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/order/usecase"
	"github.com/gin-gonic/gin"
)

// Handler binds usecase + logger
type Handler struct {
	service *usecase.Service
	logger  *logger.Logger
}

func NewHandler(s *usecase.Service, l *logger.Logger) *Handler {
	return &Handler{service: s, logger: l}
}
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/:id", h.GetOrderById)
	r.POST("/submit", h.SubmitOrder)
	// r.GET("/health", func(c *gin.Context) {
	// 	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	// })
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// GetOrderById godoc
//
//	@Summary		Get order by id
//	@Description	Get order by id
//	@Tags			order
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	http.FetchAndUpdateMarketsResponse
//	@Failure		500	{object}	object{error=string}
//	@Router			/order/:id [get]
func (h *Handler) GetOrderById(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Errorf("GetOrderById err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	order, err := h.service.GetOrderById(ctx, uint(id))
	if err != nil {
		h.logger.Errorf("GetOrderById err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, fromOrderDomain(order))
}

// SubmitOrder godoc
//
//	@Summary		Submit order
//	@Description	Submit a new order
//	@Tags			order
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SubmitOrderRequestBody	true	"Request body"
//	@Success		200	{object}	SubmitOrderResponse
//	@Failure		400	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/order/submit [post]
func (h *Handler) SubmitOrder(c *gin.Context) {
	ctx := c.Request.Context()
	// get data from body
	var req SubmitOrderRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("SubmitOrder err: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	order, err := h.service.SubmitOrder(ctx, req.ToOrder())
	if err != nil {
		h.logger.Errorf("SubmitOrder err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, fromOrderDomain(order))
}

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
