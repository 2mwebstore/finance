package main

import (
	"log"
	"os"

	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required; set it in the process environment or local development .env file")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	// Development bootstrap only. Production must use versioned migrations.
	if cfg.AppEnv == "development" || cfg.AppEnv == "test" {
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		if err := database.SeedPermissionsAndRoles(db); err != nil {
			log.Fatalf("seed: %v", err)
		}
		if err := database.SeedCategoriesAndPaymentMethods(db); err != nil {
			log.Fatalf("seed finance: %v", err)
		}
		if err := database.SeedDemoData(db); err != nil {
			log.Fatalf("seed demo: %v", err)
		}
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	// Global IP rate limit (e.g. 120 req/min). Login has a tighter limiter in routes.
	r.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(120)))

	// Basic CORS for development
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		for _, o := range cfg.CORSOrigins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	routes.Setup(r, db, cfg)

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}
	log.Printf("starting %s on :%s (env=%s)", cfg.AppName, port, cfg.AppEnv)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
