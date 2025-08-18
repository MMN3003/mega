package main

import (
	"net/http"
	"time"

	"github.com/MMN3003/mega/src/config"
	"github.com/MMN3003/mega/src/logger"
	marketHD "github.com/MMN3003/mega/src/market/delivery/http"
	marketRepo "github.com/MMN3003/mega/src/market/repository"
	market "github.com/MMN3003/mega/src/market/usecase"

	_ "github.com/MMN3003/mega/docs" // Swagger docs
	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
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

	// --- Dependencies ---
	marketRepo := marketRepo.NewRepo(gormDB, logg)
	marketSvc := market.NewService(marketRepo, logg, cfg)
	handler := marketHD.NewHandler(marketSvc, logg)

	// --- Router ---
	r := gin.New()

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
	handler.RegisterRoutes(r)

	// --- Start server ---
	logg.Infof("Starting service on %s (env=%s)", cfg.ListenAddr, cfg.Env)
	logg.Infof("Swagger UI available at http://localhost%s/swagger/index.html", cfg.ListenAddr)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logg.Fatalf("Server terminated unexpectedly: %v", err)
	}
}
