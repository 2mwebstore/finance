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

	// Schema + base data (roles, permissions, categories). Development/test always; other
	// environments opt in with AUTO_MIGRATE=true (e.g. Railway, which has no migration step).
	if cfg.AutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		if err := database.SeedPermissionsAndRoles(db); err != nil {
			log.Fatalf("seed: %v", err)
		}
		if err := database.SeedCategoriesAndPaymentMethods(db); err != nil {
			log.Fatalf("seed finance: %v", err)
		}
		// Demo branches/users (PIN 246810) only in development, or when explicitly requested.
		seedDemo := os.Getenv("SEED_DEMO") == "true" || os.Getenv("SEED_DEMO") == "1"
		if cfg.AppEnv == "development" || cfg.AppEnv == "test" || seedDemo {
			if err := database.SeedDemoData(db); err != nil {
				log.Fatalf("seed demo: %v", err)
			}
		}
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// Without this gin believes any X-Forwarded-For header, so anyone could pick a fresh "IP"
	// per request and walk past the login rate limit.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("TRUSTED_PROXIES: %v", err)
	}
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	// Per-IP limit for every request. Login has a tighter limiter in routes.
	r.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(cfg.RateLimitPerMinute)))

	// CORS, for setups where the web app runs on another origin (the default proxies /api/v1 same-origin).
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed, wildcard := false, false
		for _, o := range cfg.CORSOrigins {
			if o == origin {
				allowed = true
				break
			}
			wildcard = wildcard || o == "*"
		}
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else if wildcard && origin != "" {
			// "*" opens the API to any site, but never with cookies: that would let any page act as the signed-in user.
			c.Header("Access-Control-Allow-Origin", origin)
		}
		if allowed || (wildcard && origin != "") {
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
