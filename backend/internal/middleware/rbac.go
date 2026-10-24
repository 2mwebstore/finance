package middleware

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/access"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequireAnyPermission aborts with 403 if the user lacks all of the listed permissions.
// Super Admin always passes.
func RequireAnyPermission(db *gorm.DB, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(ContextUserIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID := userIDVal.(uuid.UUID)

		var user models.User
		if err := db.Preload("Roles.Permissions").First(&user, "id = ?", userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if user.IsSuperAdmin {
			c.Next()
			return
		}

		has := make(map[string]bool)
		for _, role := range user.Roles {
			for _, p := range role.Permissions {
				has[p.Code] = true
			}
		}

		for _, code := range codes {
			if has[code] {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

// RequireFullAccess allows only users who run the whole system: Super Admin or anyone with
// branches.manage (the owner and admin roles). Branch-level managers are rejected, and so is
// someone whose role only lets them see every branch (that is data scope, not authority).
func RequireFullAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(ContextUserIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		full, err := HasFullAccess(db, userIDVal.(uuid.UUID))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if !full {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only the owner or an admin can do this"})
			return
		}
		c.Next()
	}
}

// HasFullAccess reports owner/admin authority: Super Admin, or a role with branches.manage.
func HasFullAccess(db *gorm.DB, userID uuid.UUID) (bool, error) {
	var user models.User
	if err := db.Preload("Roles.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}
	return access.IsFullAccess(&user), nil
}

// LoadForScope loads a user with what access.BranchScope and access.IsFullAccess need.
func LoadForScope(db *gorm.DB, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := db.Preload("Roles.Permissions").Preload("Branches").First(&user, "id = ?", userID).Error
	return &user, err
}

// RequireBranchAccess ensures the user may see the branch in :id or ?branch_id (see access.BranchScope).
func RequireBranchAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(ContextUserIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		branchIDStr := c.Param("id")
		if branchIDStr == "" {
			branchIDStr = c.Query("branch_id")
		}
		if branchIDStr == "" {
			c.Next()
			return
		}
		branchID, err := uuid.Parse(branchIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
			return
		}
		ids, all, err := GetAuthorizedBranchIDs(db, userIDVal.(uuid.UUID))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if !all {
			for _, id := range ids {
				if id == branchID {
					c.Next()
					return
				}
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no access to this branch"})
			return
		}
		c.Next()
	}
}

// GetAuthorizedBranchIDs returns the branches whose data the user may see (see access.BranchScope).
// all = true means every branch.
func GetAuthorizedBranchIDs(db *gorm.DB, userID uuid.UUID) (ids []uuid.UUID, all bool, err error) {
	user, err := LoadForScope(db, userID)
	if err != nil {
		return nil, false, err
	}
	ids, all = access.BranchScope(user)
	return ids, all, nil
}
