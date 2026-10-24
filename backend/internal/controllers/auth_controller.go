package controllers

import (
	"log"
	"net/http"
	"time"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/auth"
	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthController struct {
	svc   *auth.Service
	cfg   *config.Config
	audit *audit.Service
}

func NewAuthController(svc *auth.Service, cfg *config.Config, auditSvc *audit.Service) *AuthController {
	return &AuthController{svc: svc, cfg: cfg, audit: auditSvc}
}

type loginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type setupRequest struct {
	SetupToken string `json:"setup_token" binding:"required"`
	Username   string `json:"username" binding:"required,min=3,max=64"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=4,max=6,numeric"`
	FullName   string `json:"full_name" binding:"required,min=2"`
}

type changePasswordRequest struct {
	// Required so a stolen session can't quietly take over the account.
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=4,max=6,numeric"`
}

func (ac *AuthController) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := ac.svc.Login(c.Request.Context(), req.Identifier, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if ac.audit != nil {
			ac.audit.Record(c.Request.Context(), audit.Event{
				Action: audit.ActionLoginFailed, EntityType: "user", Outcome: "failure",
				IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
				Details: map[string]interface{}{"identifier": req.Identifier},
			})
		}
		switch err {
		case auth.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case auth.ErrAccountInactive:
			c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		case auth.ErrAccountLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "account is temporarily locked"})
		default:
			log.Printf("login failed due to an internal error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	ac.setRefreshCookie(c, result.RefreshToken)

	if ac.audit != nil {
		uid := result.User.ID
		ac.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &uid, ActorEmail: result.User.Email, Action: audit.ActionLogin,
			EntityType: "user", EntityID: uid.String(), Outcome: "success",
			IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": result.AccessToken,
		"expires_at":   result.ExpiresAt,
		"user": gin.H{
			"id":             result.User.ID,
			"username":       result.User.Username,
			"email":          result.User.Email,
			"full_name":      result.User.FullName,
			"is_super_admin": result.User.IsSuperAdmin,
			"roles":          roleCodes(result.User),
		},
	})
}

func (ac *AuthController) Refresh(c *gin.Context) {
	refresh, err := c.Cookie("refresh_token")
	if err != nil || refresh == "" {
		// No session cookie (first visit or signed out): nothing to refresh, and not an error.
		c.Status(http.StatusNoContent)
		return
	}

	result, err := ac.svc.Refresh(c.Request.Context(), refresh, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		ac.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
		return
	}

	// Empty when a parallel refresh rotated first; its response already set the new cookie.
	if result.RefreshToken != "" {
		ac.setRefreshCookie(c, result.RefreshToken)
	}

	if ac.audit != nil {
		uid := result.User.ID
		ac.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &uid, ActorEmail: result.User.Email, Action: audit.ActionLogin,
			EntityType: "user", EntityID: uid.String(), Outcome: "success",
			IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": result.AccessToken,
		"expires_at":   result.ExpiresAt,
	})
}

func (ac *AuthController) Logout(c *gin.Context) {
	sessionIDVal, exists := c.Get(middleware.ContextSessionIDKey)
	if exists {
		sessionID := sessionIDVal.(uuid.UUID)
		_ = ac.svc.Logout(c.Request.Context(), sessionID)
	}
	if ac.audit != nil {
		if uid, ok := getActorID(c); ok {
			ac.audit.Record(c.Request.Context(), audit.Event{
				ActorID: &uid, Action: audit.ActionLogout, EntityType: "session",
				Outcome: "success", IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
			})
		}
	}
	ac.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (ac *AuthController) LogoutAll(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(uuid.UUID)
	_ = ac.svc.LogoutAll(c.Request.Context(), userID)
	ac.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

func (ac *AuthController) Me(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(uuid.UUID)
	user, err := ac.svc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	perms := make([]string, 0)
	seen := make(map[string]bool)
	for _, role := range user.Roles {
		for _, p := range role.Permissions {
			if !seen[p.Code] {
				seen[p.Code] = true
				perms = append(perms, p.Code)
			}
		}
	}

	branches := make([]gin.H, 0, len(user.Branches))
	for _, b := range user.Branches {
		branches = append(branches, gin.H{
			"id":   b.ID,
			"code": b.Code,
			"name": b.Name,
		})
	}

	// Data scope: every branch, or the branches the user's roles let them see (the app's branch
	// pickers offer exactly these; "branches" stays the ones the user is assigned to).
	scopeBranches := []gin.H{}
	inScope, allBranches := ac.svc.ScopeBranches(c.Request.Context(), user)
	for _, b := range inScope {
		scopeBranches = append(scopeBranches, gin.H{"id": b.ID, "code": b.Code, "name": b.Name})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"full_name":      user.FullName,
		"is_active":      user.IsActive,
		"is_super_admin": user.IsSuperAdmin,
		"roles":          roleCodes(user),
		"permissions":    perms,
		"branches":       branches,
		"all_branches":   allBranches,
		"scope_branches": scopeBranches,
		"last_login_at":  user.LastLoginAt,
	})
}

func (ac *AuthController) Setup(c *gin.Context) {
	var req setupRequest
	if !bindJSON(c, &req) {
		return
	}

	user, err := ac.svc.SetupSuperAdmin(c.Request.Context(), req.SetupToken, req.Username, req.Email, req.Password, req.FullName)
	if err != nil {
		switch err {
		case auth.ErrSetupDisabled:
			c.JSON(http.StatusForbidden, gin.H{"error": "setup is disabled"})
		case auth.ErrSetupTokenInvalid:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid setup token"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	if ac.audit != nil {
		uid := user.ID
		ac.audit.Record(c.Request.Context(), audit.Event{
			ActorID: &uid, ActorEmail: user.Email, Action: audit.ActionSetupAdmin,
			EntityType: "user", EntityID: uid.String(), Outcome: "success",
			IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "super admin created",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func (ac *AuthController) ChangePassword(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var req changePasswordRequest
	if !bindJSON(c, &req) {
		return
	}

	sessionID, _ := c.Get(middleware.ContextSessionIDKey)
	keep, _ := sessionID.(uuid.UUID)
	if err := ac.svc.ChangePassword(c.Request.Context(), userID, keep, req.CurrentPassword, req.NewPassword); err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			// 400, not 401: the session is fine, only the PIN typed was wrong.
			msg := "current PIN is incorrect"
			c.JSON(http.StatusBadRequest, gin.H{"error": msg, "fields": gin.H{"current_password": msg}})
		case auth.ErrAccountLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "too many wrong PINs; the account is temporarily locked"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func (ac *AuthController) setRefreshCookie(c *gin.Context, token string) {
	maxAge := int(ac.cfg.RefreshTokenTTL.Seconds())
	c.SetSameSite(sameSiteMode(ac.cfg.CookieSameSite))
	c.SetCookie(
		"refresh_token",
		token,
		maxAge,
		"/api/v1/auth",
		ac.cfg.CookieDomain,
		ac.cfg.CookieSecure,
		true,
	)
}

func (ac *AuthController) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(sameSiteMode(ac.cfg.CookieSameSite))
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth", ac.cfg.CookieDomain, ac.cfg.CookieSecure, true)
}

func sameSiteMode(s string) http.SameSite {
	switch s {
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func roleCodes(user *models.User) []string {
	codes := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		codes = append(codes, r.Code)
	}
	return codes
}

// Silence unused import if time is not used in some builds
var _ = time.Now
