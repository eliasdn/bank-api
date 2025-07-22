package main

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/handlers"
	"bank-api/internal/middleware"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database connection
	dbInstance, err := db.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create Gin router with middleware
	router := gin.Default()

	// Add rate limiting
	rateLimiter := middleware.NewRateLimiter(cfg)
	defer rateLimiter.StopCleanup()

	// Add logging middleware
	logger := middleware.NewSimpleLogger()
	requestLogger := middleware.NewRequestLogger(logger)
	router.Use(requestLogger.LoggingMiddleware())

	// Add error handling middleware
	errorHandler := middleware.NewErrorHandler(logger)
	router.Use(errorHandler.ErrorHandlerMiddleware())
	router.Use(errorHandler.RecoveryMiddleware())

	// Global middleware
	router.Use(func(c *gin.Context) {
		c.Set("db", dbInstance)
		c.Next()
	})
	router.Use(rateLimiter.RateLimit())

	// Expose metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Rate limit status endpoint
	router.GET("/api/v1/rate-limit-status", func(c *gin.Context) {
		status := rateLimiter.GetRateLimitStatus(c)
		c.JSON(http.StatusOK, status)
	})

	// Health check endpoints
	healthHandler := handlers.NewHealthHandler(dbInstance.DB)
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/health/ready", healthHandler.ReadinessCheck)
	router.GET("/health/live", healthHandler.LivenessCheck)
	router.GET("/health/detailed", healthHandler.DetailedHealthCheck)

	// Set up routes
	api := router.Group("/api/v1")
	{
		// Initialize handlers and middleware
		h := handlers.NewHandler(dbInstance, cfg)
		authMiddleware := middleware.NewAuthMiddleware(cfg)

		// Authentication routes
		auth := api.Group("/auth")
		auth.Use(authMiddleware.Authenticate())
		{
			auth.POST("/register", h.RegisterUser)
			auth.POST("/login", h.LoginUser)
		}

		// Account routes (protected)
		accounts := api.Group("/accounts")
		accounts.Use(authMiddleware.Authenticate())
		{
			accounts.GET("/", h.GetAccounts)
			accounts.POST("/", h.CreateAccount)
			accounts.GET("/:id", h.GetAccount)
		}

		// Transaction routes (protected)
		transactions := api.Group("/accounts/:id/transactions")
		transactions.Use(authMiddleware.Authenticate())
		{
			transactions.GET("/", h.GetTransactions)
			transactions.POST("/deposit", h.Deposit)
			transactions.POST("/withdraw", h.Withdraw)
			transactions.POST("/transfer", h.Transfer)
		}
	}

	// Configure server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown handling
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("Server running on port %s", cfg.Server.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
