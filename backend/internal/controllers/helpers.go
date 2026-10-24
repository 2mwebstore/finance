package controllers

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"log"
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/access"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// writeInternalError answers database and driver failures with a plain 500 and logs the details.
// Their text (SQL, table names) helps whoever reads the server log, not people using the app;
// sent as a 400 it also looked like the request was wrong.
func writeInternalError(c *gin.Context, err error) bool {
	var myErr *mysql.MySQLError
	if !errors.As(err, &myErr) && !errors.Is(err, driver.ErrBadConn) && !errors.Is(err, sql.ErrConnDone) {
		return false
	}
	log.Printf("internal error on %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong on the server; please try again"})
	return true
}

func getActorID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func actorIsSuperAdmin(c *gin.Context, db *gorm.DB) bool {
	id, ok := getActorID(c)
	if !ok {
		return false
	}
	var user models.User
	if err := db.Select("is_super_admin").First(&user, "id = ?", id).Error; err != nil {
		return false
	}
	return user.IsSuperAdmin
}

func writeServiceError(c *gin.Context, err error) {
	if err != nil && writeInternalError(c, err) {
		return
	}
	switch err {
	case nil:
		return
	default:
		msg := err.Error()
		switch msg {
		case "branch not found", "user not found", "role not found", "category not found", "payment method not found":
			c.JSON(http.StatusNotFound, gin.H{"error": msg})
		case "branch code already exists", "username already taken", "email already taken", "role code already exists", "role is assigned to one or more users", "category code already exists for this type", "payment method code already exists":
			c.JSON(http.StatusConflict, gin.H{"error": msg})
		case "forbidden", "cannot modify or delete system role", "cannot escalate your own privileges", "cannot demote or deactivate the last super admin", "cannot remove your own super admin status":
			c.JSON(http.StatusForbidden, gin.H{"error": msg})
		default:
			if len(msg) > 0 {
				body := gin.H{"error": msg}
				if fields := errorFields(err); fields != nil {
					body["fields"] = fields
				}
				c.JSON(http.StatusBadRequest, body)
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
		}
	}
}

// dataScopeJSON: whose data the user sees (needs Roles.Permissions and Branches loaded): every
// branch, or these branch IDs (their own).
func dataScopeJSON(u *models.User) gin.H {
	ids, all := access.BranchScope(u)
	if ids == nil {
		ids = []uuid.UUID{}
	}
	return gin.H{"all": all, "branch_ids": ids}
}

func roleSummary(roles []models.Role) []gin.H {
	out := make([]gin.H, 0, len(roles))
	for _, r := range roles {
		codes := make([]string, 0, len(r.Permissions))
		seen := map[string]bool{}
		for _, p := range r.Permissions {
			if p.Code == "" || seen[p.Code] {
				continue
			}
			seen[p.Code] = true
			codes = append(codes, p.Code)
		}
		out = append(out, gin.H{
			"id":               r.ID,
			"code":             r.Code,
			"name":             r.Name,
			"permission_codes": codes,
			"data_scope":       models.DataScopeAssigned,
		})
	}
	return out
}

func branchSummary(branches []models.Branch) []gin.H {
	out := make([]gin.H, 0, len(branches))
	for _, b := range branches {
		out = append(out, gin.H{"id": b.ID, "code": b.Code, "name": b.Name, "is_active": b.IsActive})
	}
	return out
}

func userJSON(u *models.User) gin.H {
	return gin.H{
		"id":                u.ID,
		"username":          u.Username,
		"email":             u.Email,
		"full_name":         u.FullName,
		"is_active":         u.IsActive,
		"is_super_admin":    u.IsSuperAdmin,
		"default_branch_id": u.DefaultBranchID,
		"last_login_at":     u.LastLoginAt,
		"roles":             roleSummary(u.Roles),
		"branches":          branchSummary(u.Branches),
		"data_scope":        dataScopeJSON(u),
		"permission_codes":  permissionCodesFromRoles(u),
		"created_at":        u.CreatedAt,
		"updated_at":        u.UpdatedAt,
	}
}

// permissionCodesFromRoles returns unique permission codes from the user's roles.
// Super admin is indicated separately via is_super_admin (all permissions).
func permissionCodesFromRoles(u *models.User) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, r := range u.Roles {
		for _, p := range r.Permissions {
			if p.Code == "" || seen[p.Code] {
				continue
			}
			seen[p.Code] = true
			out = append(out, p.Code)
		}
	}
	return out
}

func branchJSON(b *models.Branch) gin.H {
	return gin.H{
		"id":         b.ID,
		"code":       b.Code,
		"name":       b.Name,
		"address":    b.Address,
		"phone":      b.Phone,
		"email":      b.Email,
		"is_active":  b.IsActive,
		"manager_id": b.ManagerID,
		"created_at": b.CreatedAt,
		"updated_at": b.UpdatedAt,
	}
}

func roleJSON(r *models.Role) gin.H {
	perms := make([]gin.H, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, gin.H{
			"id":     p.ID,
			"code":   p.Code,
			"name":   p.Name,
			"module": p.Module,
		})
	}
	return gin.H{
		"id":          r.ID,
		"code":        r.Code,
		"name":        r.Name,
		"description": r.Description,
		"is_system":   r.IsSystem,
		"permissions": perms,
		"data_scope":  models.DataScopeAssigned,
		"created_at":  r.CreatedAt,
		"updated_at":  r.UpdatedAt,
	}
}
