package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"sync"
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
			spendHashTime(password)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := s.checkPIN(ctx, &user, password); err != nil {
		return nil, err
	}
	// Only someone who knows the PIN learns that the account is switched off.
	if !user.IsActive {
		return nil, ErrAccountInactive
	}
	s.db.WithContext(ctx).Model(&user).Update("last_login_at", time.Now())

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

const (
	// maxFailedLogins wrong PINs in a row lock the account.
	maxFailedLogins = 5
	firstLockout    = 15 * time.Minute
	longestLockout  = 24 * time.Hour
)

// lockoutAfter is how long the account locks once it reaches `failures` wrong PINs in a row:
// every 5th failure locks it, twice as long each time (15 min, 30 min, 1 h … 24 h), so a
// 4-digit PIN can't be found by waiting out short locks.
func lockoutAfter(failures int) time.Duration {
	if failures < maxFailedLogins || failures%maxFailedLogins != 0 {
		return 0
	}
	d := firstLockout << (failures/maxFailedLogins - 1)
	if d <= 0 || d > longestLockout {
		return longestLockout
	}
	return d
}

// checkPIN verifies a PIN for the account with brute-force protection. The attempt is counted
// before the PIN is checked, in one statement that also sets the lock, so parallel requests can't
// sneak more than maxFailedLogins guesses past it. A correct PIN clears the count.
func (s *Service) checkPIN(ctx context.Context, user *models.User, pin string) error {
	now := time.Now()
	lockSQL := "CASE WHEN MOD(failed_logins + 1, ?) <> 0 THEN locked_until"
	args := []interface{}{maxFailedLogins}
	for failures := maxFailedLogins; lockoutAfter(failures) < longestLockout; failures += maxFailedLogins {
		lockSQL += " WHEN failed_logins + 1 <= ? THEN ?"
		args = append(args, failures, now.Add(lockoutAfter(failures)))
	}
	lockSQL += " ELSE ? END"
	args = append(args, now.Add(longestLockout), user.ID, now)
	// locked_until is assigned first so it reads the count from before this attempt in every database.
	res := s.db.WithContext(ctx).Exec(
		"UPDATE users SET locked_until = "+lockSQL+", failed_logins = failed_logins + 1 "+
			"WHERE id = ? AND (locked_until IS NULL OR locked_until <= ?)",
		args...,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAccountLocked
	}
	ok, err := VerifyPassword(pin, user.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}
	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", user.ID).
		Updates(map[string]interface{}{"failed_logins": 0, "locked_until": nil}).Error
}

var (
	unknownUserHash     string
	unknownUserHashOnce sync.Once
)

// spendHashTime makes a login for an unknown username take as long as a real PIN check, so
// response times don't reveal which usernames exist.
func spendHashTime(pin string) {
	unknownUserHashOnce.Do(func() { unknownUserHash, _ = HashPassword(strings.Repeat("0", MinPINLength)) })
	if unknownUserHash != "" {
		_, _ = VerifyPassword(pin, unknownUserHash)
	}
}

// refreshGracePeriod is how long the token replaced by a rotation is still accepted.
const refreshGracePeriod = 30 * time.Second

// Refresh issues a new access token. The refresh token is rotated; LoginResult.RefreshToken
// is empty when this request lost a race with a parallel refresh, in which case the caller
// must leave the cookie alone (the winning response already set the new one).
func (s *Service) Refresh(ctx context.Context, refreshPlain, userAgent, ip string) (*LoginResult, error) {
	hash := hashToken(refreshPlain)
	var session models.Session
	rotate := true
	err := s.db.WithContext(ctx).Where("refresh_token_hash = ?", hash).First(&session).Error
	if err != nil {
		// A request sent just before another one rotated the token still carries the old one.
		err = s.db.WithContext(ctx).
			Where("previous_token_hash = ? AND rotated_at > ?", hash, time.Now().Add(-refreshGracePeriod)).
			First(&session).Error
		if err != nil {
			return nil, ErrSessionRevoked
		}
		rotate = false
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

	newPlain := ""
	if rotate {
		plain, newHash, err := GenerateRefreshToken()
		if err != nil {
			return nil, err
		}
		now := time.Now()
		// Only rotate if nobody else did since we read the session (compare-and-swap).
		res := s.db.WithContext(ctx).Model(&models.Session{}).
			Where("id = ? AND refresh_token_hash = ?", session.ID, hash).
			Updates(map[string]interface{}{
				"refresh_token_hash":  newHash,
				"previous_token_hash": hash,
				"rotated_at":          now,
				"user_agent":          userAgent,
				"ip_address":          ip,
				"expires_at":          now.Add(s.cfg.RefreshTokenTTL),
			})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 1 {
			newPlain = plain
		}
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

// ValidateSession checks the token's session is live and belongs to the token's user, so a
// token can never borrow someone else's session.
func (s *Service) ValidateSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	var session models.Session
	err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error
	if err != nil || session.UserID != userID {
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
	if subtle.ConstantTimeCompare([]byte(setupToken), []byte(s.cfg.SetupToken)) != 1 {
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

// ChangePassword sets a new PIN after checking the current one (wrong guesses count toward the
// lockout, like logins) and signs out every other session. keepSessionID stays signed in.
// The new PIN has the current length, so any pending PIN upgrade is done.
func (s *Service) ChangePassword(ctx context.Context, userID, keepSessionID uuid.UUID, currentPassword, newPassword string) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	if err := s.checkPIN(ctx, &user, currentPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password_hash", hash).Error; err != nil {
			return err
		}
		return tx.Model(&models.Session{}).
			Where("user_id = ? AND id <> ? AND is_revoked = ?", userID, keepSessionID, false).
			Updates(map[string]interface{}{"is_revoked": true, "revoked_at": time.Now()}).Error
	})
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
