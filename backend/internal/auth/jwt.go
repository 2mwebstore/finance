package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   string   `json:"uid"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	SessionID string  `json:"sid"`
	jwt.RegisteredClaims
}

type TokenService struct {
	cfg *config.Config
}

func NewTokenService(cfg *config.Config) *TokenService {
	return &TokenService{cfg: cfg}
}

func (s *TokenService) GenerateAccessToken(userID uuid.UUID, username string, sessionID uuid.UUID, roles []string) (string, time.Time, error) {
	if s.cfg.JWTSecret == "" {
		return "", time.Time{}, errors.New("JWT_SECRET is not configured")
	}
	expiresAt := time.Now().Add(s.cfg.JWTAccessTTL)
	claims := Claims{
		UserID:    userID.String(),
		Username:  username,
		Roles:     roles,
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.cfg.JWTIssuer,
			Audience:  []string{s.cfg.JWTAudience},
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *TokenService) ParseAccessToken(tokenStr string) (*Claims, error) {
	if s.cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is not configured")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithIssuer(s.cfg.JWTIssuer), jwt.WithAudience(s.cfg.JWTAudience))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// GenerateRefreshToken returns a cryptographically random opaque token and its hash.
func GenerateRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := randRead(b); err != nil {
		return "", "", err
	}
	plain = base64URL(b)
	hash = hashToken(plain)
	return plain, hash, nil
}

func hashToken(plain string) string {
	// Simple SHA-256 for refresh token storage (not password)
	h := sha256Sum([]byte(plain))
	return base64URL(h)
}

// helpers to avoid importing crypto packages in multiple places
func randRead(b []byte) (int, error) {
	return cryptoRandRead(b)
}
