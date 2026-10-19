package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrAccountLocked      = errors.New("account is temporarily locked")
	ErrSessionRevoked     = errors.New("session has been revoked")
	ErrSessionExpired     = errors.New("session has expired")
	ErrSetupDisabled      = errors.New("first-run setup is disabled")
	ErrSetupTokenInvalid  = errors.New("invalid setup token")
)

type Service struct {
	db    *gorm.DB
	cfg   *config.Config
	tokens *TokenService
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		cfg:    cfg,
		tokens: NewTokenService(cfg),
	}
}

type LoginResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string // only returned to set cookie; not in JSON body
	User         *models.User
	SessionID    uuid.UUID
}

func (s *Service) Login(ctx context.Context, identifier, password, userAgent, ip string) (*LoginResult, error) {
	var user models.User
	err := s.db.WithContext(ctx).
		Preload("Roles").
		Where("username = ? OR email = ?", identifier, identifier).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		_ = s.recordFailedLogin(ctx, &user)
		return nil, ErrInvalidCredentials
	}

	// Reset failed logins
	s.db.WithContext(ctx).Model(&user).Updates(map[string]interface{}{
		"failed_logins": 0,
		"locked_until":  nil,
		"last_login_at": time.Now(),
	})

	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, r.Code)
	}

	sessionID := uuid.New()
	refreshPlain, refreshHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	session := models.Session{
		Base:             models.Base{ID: sessionID},
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		UserAgent:        userAgent,
		IPAddress:        ip,
		ExpiresAt:        time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}

	accessToken, expiresAt, err := s.tokens.GenerateAccessToken(user.ID, user.Username, sessionID, roles)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshPlain,
		User:         &user,
		SessionID:    sessionID,
	}, nil
}

func (s *Service) recordFailedLogin(ctx context.Context, user *models.User) error {
	failed := user.FailedLogins + 1
	updates := map[string]interface{}{"failed_logins": failed}
	if failed >= 5 {
		lockUntil := time.Now().Add(15 * time.Minute)
		updates["locked_until"] = lockUntil
	}
	return s.db.WithContext(ctx).Model(user).Updates(updates).Error
}

func (s *Service) Refresh(ctx context.Context, refreshPlain, userAgent, ip string) (*LoginResult, error) {
	hash := hashToken(refreshPlain)
	var session models.Session
	err := s.db.WithContext(ctx).Where("refresh_token_hash = ?", hash).First(&session).Error
	if err != nil {
		return nil, ErrSessionRevoked
	}
	if session.IsRevoked || (session.RevokedAt != nil) {
		return nil, ErrSessionRevoked
	}
	if session.ExpiresAt.Before(time.Now()) {
		return nil, ErrSessionExpired
	}

	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// Rotate refresh token
	newPlain, newHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	session.RefreshTokenHash = newHash
	session.UserAgent = userAgent
	session.IPAddress = ip
	session.ExpiresAt = time.Now().Add(s.cfg.RefreshTokenTTL)
	if err := s.db.WithContext(ctx).Save(&session).Error; err != nil {
		return nil, err
	}

	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, r.Code)
	}

	accessToken, expiresAt, err := s.tokens.GenerateAccessToken(user.ID, user.Username, session.ID, roles)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: newPlain,
		User:         &user,
		SessionID:    session.ID,
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.Session{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		}).Error
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.Session{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		}).Error
}

func (s *Service) ValidateSession(ctx context.Context, sessionID uuid.UUID) error {
	var session models.Session
	err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error
	if err != nil {
		return ErrSessionRevoked
	}
	if session.IsRevoked {
		return ErrSessionRevoked
	}
	if session.ExpiresAt.Before(time.Now()) {
		return ErrSessionExpired
	}
	return nil
}

// SetupSuperAdmin creates the first Super Admin. Requires valid SETUP_TOKEN.
func (s *Service) SetupSuperAdmin(ctx context.Context, setupToken, username, email, password, fullName string) (*models.User, error) {
	if s.cfg.SetupToken == "" {
		return nil, ErrSetupDisabled
	}
	if setupToken != s.cfg.SetupToken {
		return nil, ErrSetupTokenInvalid
	}

	// Check if any super admin already exists
	var count int64
	s.db.WithContext(ctx).Model(&models.User{}).Where("is_super_admin = ?", true).Count(&count)
	if count > 0 {
		return nil, ErrSetupDisabled
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		FullName:     fullName,
		IsActive:     true,
		IsSuperAdmin: true,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		// Assign Super Admin role if it exists
		var role models.Role
		if err := tx.Where("code = ?", "super_admin").First(&role).Error; err == nil {
			tx.Model(&user).Association("Roles").Append(&role)
		}
		// Mark setup complete
		tx.Save(&models.Setting{Key: "setup_completed", Value: "true", UpdatedAt: time.Now()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	if currentPassword != "" {
		ok, err := VerifyPassword(currentPassword, user.PasswordHash)
		if err != nil || !ok {
			return ErrInvalidCredentials
		}
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&user).Update("password_hash", hash).Error
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).
		Preload("Roles.Permissions").
		Preload("Branches").
		First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
