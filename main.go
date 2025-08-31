package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MMN3003/mega/src/config"
	cron_repo "github.com/MMN3003/mega/src/cron/repository"
	cron_usecase "github.com/MMN3003/mega/src/cron/usecase"
	"github.com/MMN3003/mega/src/logger"
	market_http_delivery "github.com/MMN3003/mega/src/market/delivery/http"
	market_repo "github.com/MMN3003/mega/src/market/repository"
	market "github.com/MMN3003/mega/src/market/usecase"
	order_cron_adapter "github.com/MMN3003/mega/src/order/adapter/cron"
	order_market_adapter "github.com/MMN3003/mega/src/order/adapter/market"
	order_http_delivery "github.com/MMN3003/mega/src/order/delivery/http"
	order_repo "github.com/MMN3003/mega/src/order/repository"
	order_usecase "github.com/MMN3003/mega/src/order/usecase"

	_ "github.com/MMN3003/mega/docs" // Swagger docs
	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadFromEnv()
	logg := logger.New(cfg.Env)

	// --- Database connection ---
	logg.Infof("Connecting to database: %s", cfg.DatabaseURL)

	dsn := cfg.DatabaseURL
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info), // SQL logs
	})
	if err != nil {
		logg.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		logg.Fatalf("Failed to get generic DB handle: %v", err)
	}
	defer sqlDB.Close()

	// Connection pool tuning
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	c := cron.New(cron.WithSeconds())
	// --- repos ---
	marketRepo := market_repo.NewRepo(gormDB, logg)
	megaMarketRepo := market_repo.NewMegaMarketRepo(gormDB, logg)
	orderRepo := order_repo.NewOrderRepo(gormDB, logg)
	cronRepo := cron_repo.NewCronRepo(gormDB, logg)
	// --- services ---
	marketSvc := market.NewService(marketRepo, megaMarketRepo, logg, cfg)
	cronSvc := cron_usecase.NewService(cronRepo, logg)
	orderSvc := order_usecase.NewService(orderRepo, logg, cfg)
	// --- adapters ---
	marketAdapter := order_market_adapter.NewMarketPort(marketSvc)
	cronAdapter := order_cron_adapter.NewCronPort(cronSvc)
	orderSvc.SetAdapters(context.Background(), marketAdapter)
	// --- handlers ---
	market_handler := market_http_delivery.NewHandler(marketSvc, logg)
	order_handler := order_http_delivery.NewHandler(orderSvc, logg)
	// --- cron ---
	order_usecase.NewCronService(c, orderSvc, cronAdapter)

	// --- Router ---
	r := gin.New()
	c.Start()
	defer c.Stop()
	// Core middleware
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logg.Infof("%s %s status:%d duration:%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	})

	// --- Healthcheck ---
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --- Swagger ---
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- API routes ---
	market_handler.RegisterRoutes(r)
	order_handler.RegisterRoutes(r)

	// --- Start server ---

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// Channel to listen for errors from server
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		logg.Infof("Starting service on %s (env=%s)", cfg.ListenAddr, cfg.Env)
		logg.Infof("Swagger UI available at http://localhost%s/swagger/index.html", cfg.ListenAddr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Channel to listen for interrupt or terminate signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErr:
		logg.Fatalf("Server error: %v", err)
	case sig := <-quit:
		logg.Infof("Received signal: %s. Shutting down gracefully...", sig)

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Stop accepting new connections
		if err := srv.Shutdown(ctx); err != nil {
			logg.Errorf("Server forced to shutdown: %v", err)
		}

		// Stop cron jobs
		c.Stop()
		logg.Infof("Cron jobs stopped")

		// Close database connection
		if err := sqlDB.Close(); err != nil {
			logg.Errorf("Error closing database connection: %v", err)
		}
		logg.Infof("Database connection closed")

		logg.Infof("Server exited gracefully")
	}
}
