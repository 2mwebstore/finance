package middleware

import (
	"net/http"

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
// branches.manage (the owner and admin roles). Branch-level managers are rejected.
func RequireFullAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(ContextUserIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		_, all, err := GetAuthorizedBranchIDs(db, userIDVal.(uuid.UUID))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if !all {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only the owner or an admin can do this"})
			return
		}
		c.Next()
	}
}

// RequireBranchAccess ensures the user is allowed to operate on the given branchID.
// Super Admin and users with branches.manage may access any active branch.
// Otherwise the branch must appear in user_branches.
func RequireBranchAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(ContextUserIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID := userIDVal.(uuid.UUID)

		branchIDStr := c.Param("id")
		if branchIDStr == "" {
			branchIDStr = c.Query("branch_id")
		}
		if branchIDStr == "" {
			// No branch context required for this route
			c.Next()
			return
		}

		branchID, err := uuid.Parse(branchIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
			return
		}

		var user models.User
		if err := db.Preload("Roles.Permissions").Preload("Branches").First(&user, "id = ?", userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if user.IsSuperAdmin {
			c.Next()
			return
		}

		// Check global branch management permission
		for _, role := range user.Roles {
			for _, p := range role.Permissions {
				if p.Code == "branches.manage" {
					c.Next()
					return
				}
			}
		}

		for _, b := range user.Branches {
			if b.ID == branchID {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no access to this branch"})
	}
}

// GetAuthorizedBranchIDs returns the list of branch IDs the user may access.
// Empty slice + true means "all branches" (Super Admin / branches.manage).
func GetAuthorizedBranchIDs(db *gorm.DB, userID uuid.UUID) (ids []uuid.UUID, all bool, err error) {
	var user models.User
	if err := db.Preload("Roles.Permissions").Preload("Branches").First(&user, "id = ?", userID).Error; err != nil {
		return nil, false, err
	}
	if user.IsSuperAdmin {
		return nil, true, nil
	}
	for _, role := range user.Roles {
		for _, p := range role.Permissions {
			if p.Code == "branches.manage" {
				return nil, true, nil
			}
		}
	}
	for _, b := range user.Branches {
		ids = append(ids, b.ID)
	}
	return ids, false, nil
}
