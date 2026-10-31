package main

import (
	"context"
	"log"
	"os"

	"github.com/cambodia-finance/income-expense/internal/backup"
	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/routes"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/cambodia-finance/income-expense/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if os.Getenv("APP_ENV") == "" {
		log.Println("warning: APP_ENV is not set, so the API runs with development settings (auto-migrate, demo data); set APP_ENV=production on servers")
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required; set it in the process environment or local development .env file")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	// Tables the running code can't work without, created even when AUTO_MIGRATE is off,
	// so a deploy never serves code that reads a table the database doesn't have yet.
	if err := database.EnsureRequiredTables(db); err != nil {
		log.Fatalf("required tables: %v", err)
	}
	if err := database.EnsureRequiredPermissions(db); err != nil {
		log.Fatalf("required permissions: %v", err)
	}

	// Schema (AUTO_MIGRATE) and base data (AUTO_SEED) are separate switches, so a host can update
	// tables without touching data, or the other way round. Development/test always do both.
	if cfg.AutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Println("auto-migrate: tables are up to date")
	}
	if cfg.AutoSeed {
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

	// Saved win/lose weeks that aren't finalized follow the current formula. A failure is logged and
	// the server still starts (the weeks keep what they were worked out with).
	if n, err := services.NewPlatformService(db, cfg.Timezone).RecalculateOpenWeeks(context.Background()); err != nil {
		log.Printf("win/lose: could not work saved weeks out with the current formula: %v", err)
	} else if n > 0 {
		log.Printf("win/lose: %d saved results worked out again with the current formula", n)
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
	// Health checks arrive every few seconds during deploys; they would drown out real requests.
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/health"}}), gin.Recovery())
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

	// Attachment files (R2, or local disk in development).
	store, err := storage.NewFromEnv()
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	log.Printf("attachments: using %s storage", store.Kind())

	// Daily backups of the database and attachment files (kept in BACKUP_DIR, sent to Telegram);
	// settings live in the admin page.
	backups, err := backup.NewService(db, store, cfg.BackupDir, cfg.Timezone, cfg.AppName)
	if err != nil {
		log.Fatalf("backups: %v", err)
	}
	backups.StartScheduler(context.Background())

	routes.Setup(r, db, cfg, store, backups)

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
