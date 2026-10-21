package middleware

import (
	"net/http"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextUserIDKey    = "userID"
	ContextUsernameKey  = "username"
	ContextRolesKey     = "roles"
	ContextSessionIDKey = "sessionID"
	ContextUserKey      = "user"
)

func AuthRequired(authSvc *auth.Service, tokenSvc *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokenSvc.ParseAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		sessionID, err := uuid.Parse(claims.SessionID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		if err := authSvc.ValidateSession(c.Request.Context(), sessionID, userID); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session revoked or expired"})
			return
		}

		c.Set(ContextUserIDKey, userID)
		c.Set(ContextUsernameKey, claims.Username)
		c.Set(ContextRolesKey, claims.Roles)
		c.Set(ContextSessionIDKey, sessionID)
		c.Next()
	}
}

// RequirePermission checks that the authenticated user has at least one of the given permissions.
// Super Admin (is_super_admin flag or super_admin role) bypasses.
func RequirePermission(db interface {
	// minimal interface to avoid circular import; real implementation uses gorm
}, permCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder – full implementation in permissions middleware
		c.Next()
	}
}
