package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base contains common fields.
type Base struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// User represents an application user.
type User struct {
	Base
	Username       string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email          string     `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash   string     `gorm:"size:255;not null" json:"-"`
	FullName       string     `gorm:"size:128;not null" json:"full_name"`
	IsActive       bool       `gorm:"default:true;not null" json:"is_active"`
	IsSuperAdmin   bool       `gorm:"default:false;not null" json:"is_super_admin"`
	DefaultBranchID *uuid.UUID `gorm:"type:char(36);index" json:"default_branch_id,omitempty"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	FailedLogins   int        `gorm:"default:0;not null" json:"-"`
	LockedUntil    *time.Time `json:"-"`
	// Set when the user signs in with an old 4-digit PIN; cleared once they choose a 6-digit one.

	Roles    []Role   `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	Branches []Branch `gorm:"many2many:user_branches;" json:"branches,omitempty"`
}

// Role groups permissions.
type Role struct {
	Base
	Code        string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	IsSystem    bool   `gorm:"default:false;not null" json:"is_system"` // cannot delete system roles
	// DataScope is always DataScopeAssigned: users see only the branches they are assigned to;
	// data is not shared between branches. Seeing every branch comes from full access (Manage
	// Branches). Older values are converted on start (see database.ownBranchesOnly).
	DataScope string `gorm:"size:16;not null;default:'assigned'" json:"data_scope"`
	// BranchID: a role for one branch (shown and given only there); nil = shared by every branch.
	BranchID *uuid.UUID `gorm:"type:char(36);index" json:"branch_id,omitempty"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// DataScopeAssigned is the only role data scope (the retired "custom" and "all" are converted to it).
const DataScopeAssigned = "assigned"

// Permission is a granular capability.
type Permission struct {
	Base
	Code        string `gorm:"size:128;uniqueIndex;not null" json:"code"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Module      string `gorm:"size:64;index;not null" json:"module"`
	Description string `gorm:"size:512" json:"description"`
}

// Branch is a business location.
type Branch struct {
	Base
	Code      string  `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name      string  `gorm:"size:128;not null" json:"name"`
	Address   string  `gorm:"size:512" json:"address"`
	Phone     string  `gorm:"size:32" json:"phone"`
	Email     string  `gorm:"size:255" json:"email"`
	IsActive  bool    `gorm:"default:true;not null" json:"is_active"`
	ManagerID *uuid.UUID `gorm:"type:char(36);index" json:"manager_id,omitempty"`
}

// UserRole join table.
type UserRole struct {
	UserID uuid.UUID `gorm:"type:char(36);primaryKey"`
	RoleID uuid.UUID `gorm:"type:char(36);primaryKey"`
}

// RolePermission join table.
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:char(36);primaryKey"`
	PermissionID uuid.UUID `gorm:"type:char(36);primaryKey"`
}

// UserBranch join table.
type UserBranch struct {
	UserID   uuid.UUID `gorm:"type:char(36);primaryKey"`
	BranchID uuid.UUID `gorm:"type:char(36);primaryKey"`
}

// UserManager links a user to a manager who approves or rejects that user's pending transactions.
type UserManager struct {
	UserID    uuid.UUID `gorm:"type:char(36);primaryKey"`
	ManagerID uuid.UUID `gorm:"type:char(36);primaryKey;index"`
}

// Session stores refresh token metadata for revocation.
type Session struct {
	Base
	UserID           uuid.UUID `gorm:"type:char(36);index;not null" json:"user_id"`
	RefreshTokenHash string    `gorm:"size:255;uniqueIndex;not null" json:"-"`
	UserAgent        string    `gorm:"size:512" json:"user_agent"`
	IPAddress        string    `gorm:"size:64" json:"ip_address"`
	ExpiresAt        time.Time `gorm:"index;not null" json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	IsRevoked        bool      `gorm:"default:false;not null" json:"is_revoked"`
	// The token replaced by the last rotation stays usable for a few seconds so
	// parallel refreshes (two tabs, several requests) don't log each other out.
	PreviousTokenHash string     `gorm:"size:255;index" json:"-"`
	RotatedAt         *time.Time `json:"-"`
}

// AuditLog records security and important business events.
type AuditLog struct {
	ID          uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	ActorID     *uuid.UUID `gorm:"type:char(36);index" json:"actor_id,omitempty"`
	ActorEmail  string     `gorm:"size:255" json:"actor_email"`
	Action      string     `gorm:"size:64;index;not null" json:"action"`
	EntityType  string     `gorm:"size:64;index" json:"entity_type"`
	EntityID    string     `gorm:"size:64;index" json:"entity_id"`
	BranchID    *uuid.UUID `gorm:"type:char(36);index" json:"branch_id,omitempty"`
	IPAddress   string     `gorm:"size:64" json:"ip_address"`
	UserAgent   string     `gorm:"size:512" json:"user_agent"`
	Outcome     string     `gorm:"size:32;not null" json:"outcome"` // success, failure
	Details     string     `gorm:"type:text" json:"details"`        // JSON of changed fields (no secrets)
	Correlation string     `gorm:"size:64;index" json:"correlation_id"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// Setting stores key-value configuration.
type Setting struct {
	Key       string    `gorm:"size:128;primaryKey" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy *uuid.UUID `gorm:"type:char(36)" json:"updated_by,omitempty"`
}

// AppSetting is a simple key-value application setting.
type AppSetting struct {
	Key       string    `gorm:"size:64;primaryKey" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedBy *uuid.UUID `gorm:"type:char(36)" json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
