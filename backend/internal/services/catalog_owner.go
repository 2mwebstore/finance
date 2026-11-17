package services

import (
	"context"
	"errors"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===== Shared or one branch's own =====
//
// Products, product categories, customers, suppliers and payment methods are either shared
// (BranchID nil: every branch sees and uses them, and only the owner or an admin adds or changes
// them) or belong to one branch (only that branch sees and uses them, and its people manage
// them). A document only uses what is shared or its own branch's.

// ErrSharedOnlyFull: someone without full access tried to add or change a shared item.
var ErrSharedOnlyFull = errors.New("only the owner or an admin can add or change what every branch shares")

// CatalogScope is who is changing one of these lists.
type CatalogScope struct {
	Branches BranchScope
	Full     bool // owner/admin (branches.manage): adds and changes shared items
}

// OwnerFilter narrows one of these lists.
type OwnerFilter struct {
	// For: shared items and this branch's own (the branch picked in the top bar, or a document's).
	For *uuid.UUID
	// Only: "shared" for shared items only, or a branch id for that branch's own only.
	Only string
}

// SharedOnly is OwnerFilter.Only for shared items (e.g. what can move between branches).
const SharedOnly = "shared"

// ownerWhere limits q (column is the owner column) to shared items plus those of the scope's
// branches, then to the filter.
func ownerWhere(q *gorm.DB, column string, scope BranchScope, f OwnerFilter) (*gorm.DB, error) {
	switch {
	case f.For != nil:
		if !scope.Has(*f.For) {
			return nil, ErrBranchAccessDenied
		}
		q = q.Where("("+column+" IS NULL OR "+column+" = ?)", *f.For)
	case scope.All:
	case len(scope.IDs) > 0:
		q = q.Where("("+column+" IS NULL OR "+column+" IN ?)", scope.IDs)
	default:
		q = q.Where(column + " IS NULL")
	}
	switch only := strings.TrimSpace(f.Only); only {
	case "":
	case SharedOnly:
		q = q.Where(column + " IS NULL")
	default:
		id, err := uuid.Parse(only)
		if err != nil {
			return nil, fieldError("owner", "choose shared or one of your branches")
		}
		if !scope.Has(id) {
			return nil, ErrBranchAccessDenied
		}
		q = q.Where(column+" = ?", id)
	}
	return q, nil
}

// ParseOwner reads the owner chosen on a form: "" is shared (owner/admin only), else one of the
// actor's branches.
func ParseOwner(ctx context.Context, db *gorm.DB, raw string, scope CatalogScope) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if !scope.Full {
			return nil, fieldError("branch_id", "only the owner or an admin can add shared ones; choose your branch")
		}
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fieldError("branch_id", "choose a valid branch")
	}
	if !scope.Branches.Has(id) {
		return nil, ErrBranchAccessDenied
	}
	var n int64
	if err := db.WithContext(ctx).Model(&models.Branch{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fieldError("branch_id", "this branch was not found")
	}
	return &id, nil
}

// MayChange: the actor may change an item with this owner (a shared one: owner/admin only; another
// branch's: not found).
func MayChange(owner *uuid.UUID, scope CatalogScope) error {
	if owner == nil {
		if !scope.Full {
			return ErrSharedOnlyFull
		}
		return nil
	}
	if !scope.Branches.Has(*owner) {
		return ErrInventoryNotFound
	}
	return nil
}

// Visible: the scope sees an item with this owner.
func Visible(owner *uuid.UUID, scope BranchScope) bool {
	return owner == nil || scope.Has(*owner)
}

// usableIn: an item with this owner can go on a document of the branch.
func usableIn(owner *uuid.UUID, branch uuid.UUID) bool {
	return owner == nil || *owner == branch
}

func sameOwner(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// BranchName is a branch's name, for messages.
func BranchName(ctx context.Context, db *gorm.DB, id uuid.UUID) string {
	var b models.Branch
	if err := db.WithContext(ctx).Select("id", "name").Take(&b, "id = ?", id).Error; err != nil || b.Name == "" {
		return "another branch"
	}
	return b.Name
}

// BranchNames maps every branch to its name (for lists).
func BranchNames(ctx context.Context, db *gorm.DB) map[uuid.UUID]string {
	var list []models.Branch
	names := map[uuid.UUID]string{}
	if err := db.WithContext(ctx).Unscoped().Select("id", "name").Find(&list).Error; err == nil {
		for _, b := range list {
			names[b.ID] = b.Name
		}
	}
	return names
}

func ownerName(names map[uuid.UUID]string, owner *uuid.UUID) string {
	if owner == nil {
		return ""
	}
	return names[*owner]
}

// countAny runs COUNT(*) queries (each taking args) and reports whether any found a row.
func countAny(db *gorm.DB, queries []string, args ...interface{}) (bool, error) {
	for _, q := range queries {
		var n int64
		if err := db.Raw(q, args...).Scan(&n).Error; err != nil {
			return false, err
		}
		if n > 0 {
			return true, nil
		}
	}
	return false, nil
}
