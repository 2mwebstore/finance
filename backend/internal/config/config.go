package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv   string
	AppPort  string
	AppName  string
	Timezone string
	// AutoMigrate creates/updates tables and seeds roles, permissions and categories on start.
	// Always on in development/test; set AUTO_MIGRATE=true on hosts without a migration step (Railway).
	AutoMigrate bool

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBParams   string
	// LogSQL prints every SQL statement (DB_LOG_SQL=true). Off by default: a single start-up
	// migration is hundreds of statements, more than hosts like Railway accept per second.
	LogSQL bool

	JWTSecret       string
	JWTAccessTTL    time.Duration
	JWTIssuer       string
	JWTAudience     string
	RefreshTokenTTL time.Duration
	CookieSecure    bool
	CookieDomain    string
	CookieSameSite  string

	SetupToken string

	LoginRateLimit  int
	LoginRateWindow time.Duration
	// RateLimitPerMinute is the per-IP limit for all API requests. A whole office behind one
	// public IP shares it, so it is generous.
	RateLimitPerMinute int
	// TrustedProxies may set X-Forwarded-For (the client IP used for rate limits and audit logs).
	// Defaults to private networks: nginx, the Nuxt server and the host's edge proxy.
	TrustedProxies []string

	CORSOrigins []string
}

// privateNetworks are the proxies in front of the API in every supported deployment.
const privateNetworks = "127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,100.64.0.0/10,fc00::/7"

func Load() *Config {
	appEnv := getEnv("APP_ENV", "development")
	if appEnv == "development" || appEnv == "test" {
		loadLocalEnv()
	}

	return &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		// Hosts like Railway assign the port through PORT.
		AppPort:     getEnv("APP_PORT", getEnv("PORT", "8080")),
		AppName:     getEnv("APP_NAME", "income-expense-api"),
		Timezone:    getEnv("TZ", "Asia/Phnom_Penh"),
		AutoMigrate: appEnv == "development" || appEnv == "test" || getEnvBool("AUTO_MIGRATE", false),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "finance"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "income_expense"),
		DBParams:   getEnv("DB_PARAMS", "charset=utf8mb4&parseTime=True&loc=Asia%2FPhnom_Penh"),
		LogSQL:     getEnvBool("DB_LOG_SQL", false),

		JWTSecret:       getEnv("JWT_SECRET", ""),
		JWTAccessTTL:    time.Duration(getEnvInt("JWT_ACCESS_TTL_MINUTES", 15)) * time.Minute,
		JWTIssuer:       getEnv("JWT_ISSUER", "income-expense-api"),
		JWTAudience:     getEnv("JWT_AUDIENCE", "income-expense-web"),
		RefreshTokenTTL: time.Duration(getEnvInt("REFRESH_TOKEN_TTL_HOURS", 168)) * time.Hour,
		CookieSecure:    getEnvBool("COOKIE_SECURE", false),
		CookieDomain:    getEnv("COOKIE_DOMAIN", ""),
		CookieSameSite:  getEnv("COOKIE_SAME_SITE", "Strict"),

		SetupToken: getEnv("SETUP_TOKEN", ""),

		LoginRateLimit:  getEnvInt("LOGIN_RATE_LIMIT", 10),
		LoginRateWindow: time.Duration(getEnvInt("LOGIN_RATE_WINDOW_MINUTES", 15)) * time.Minute,

		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 300),
		TrustedProxies:     splitList(getEnv("TRUSTED_PROXIES", privateNetworks)),

		CORSOrigins: splitList(getEnv("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
	}
}

// loadLocalEnv reads a local .env for development only. Variables already
// supplied by the process always take precedence over values in the file.
func loadLocalEnv() {
	for _, path := range []string{".env", filepath.Join("backend", ".env")} {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "" {
				continue
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			_ = os.Setenv(key, value)
		}
		return
	}
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?" + c.DBParams
}

// splitList splits a comma-separated setting, ignoring spaces and empty items.
func splitList(v string) []string {
	out := []string{}
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}
