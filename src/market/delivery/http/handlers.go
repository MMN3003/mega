package http

import (
	"net/http"

	"github.com/MMN3003/mega/src/logger"
	"github.com/MMN3003/mega/src/market/usecase"

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
	r.GET("/markets", h.ListPairs)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// ListPairs godoc
//
//	@Summary		List available market
//	@Description	Get all available market
//	@Tags			market
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	http.FetchAndUpdateMarketsResponse
//	@Failure		500	{object}	object{error=string}
//	@Router			/markets [get]
func (h *Handler) ListPairs(c *gin.Context) {
	ctx := c.Request.Context()
	markets, err := h.service.FetchAndUpdateMarkets(ctx)
	if err != nil {
		h.logger.Errorf("ListPairs err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, FetchAndUpdateMarketsResponseFromDomain(markets))
}
