package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===== Customers and suppliers =====
//
// Two lists with the same columns (models.Party); each person is shared by every branch or one
// branch's own (see catalog_owner.go). Sales, quotations and the POS pick a customer; purchases
// pick a supplier, from those shared or the document branch's own. A document keeps the name too: renaming someone renames them on their documents, and a document saved without the list
// (typed by an API client, or from before the lists) keeps the name it was given.

// PartyKind is one of the two lists.
type PartyKind struct {
	table string
	label string // "customer" or "supplier", for messages
	field string // the documents' input field: customer_id or supplier_id
	// docs are the documents that name someone from this list (table → name column).
	docs []partyDocs
}

type partyDocs struct{ table, idColumn, nameColumn string }

var (
	Customers = PartyKind{table: "customers", label: "customer", field: "customer_id", docs: []partyDocs{
		{"sales", "customer_id", "customer"}, {"quotations", "customer_id", "customer"},
	}}
	Suppliers = PartyKind{table: "suppliers", label: "supplier", field: "supplier_id", docs: []partyDocs{
		{"purchases", "supplier_id", "supplier"},
	}}
)

// Label is "customer" or "supplier".
func (k PartyKind) Label() string { return k.label }

type PartyInput struct {
	Name     string `json:"name" binding:"required,min=2,max=191"`
	Phone    string `json:"phone" binding:"max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=191"`
	Address  string `json:"address" binding:"max=512"`
	Note     string `json:"note" binding:"max=512"`
	IsActive *bool  `json:"is_active"` // changing someone only; new ones are on
	// BranchID: "" shared (owner/admin only) or one branch; left out when changing keeps it.
	BranchID *string `json:"branch_id"`
}

type PartyFilter struct {
	Search   string
	Active   string // "true", "false" or "" (everyone)
	Owner    OwnerFilter
	Page     int
	PageSize int
}

type PartyView struct {
	models.Party
	// Documents is how many documents name them (sales and quotations, or purchases).
	Documents int64 `json:"documents"`
}

func (s *InventoryService) ListParties(ctx context.Context, k PartyKind, f PartyFilter, scope BranchScope) ([]PartyView, int64, error) {
	q, err := ownerWhere(s.db.WithContext(ctx).Table(k.table).Where("deleted_at IS NULL"), "branch_id", scope, f.Owner)
	if err != nil {
		return nil, 0, err
	}
	switch f.Active {
	case "true":
		q = q.Where("is_active = ?", true)
	case "false":
		q = q.Where("is_active = ?", false)
	}
	q = docSearch(q, f.Search, "name", "phone", "email", "address")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := pageArgs(f.Page, f.PageSize)
	var list []models.Party
	if err := q.Order("name ASC").Limit(size).Offset((page - 1) * size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	out := make([]PartyView, len(list))
	ids := make([]uuid.UUID, len(list))
	names := BranchNames(ctx, s.db)
	for i, p := range list {
		p.BranchName = ownerName(names, p.BranchID)
		out[i].Party = p
		ids[i] = p.ID
	}
	if len(ids) > 0 {
		counts := map[uuid.UUID]int64{}
		for _, d := range k.docs {
			var rows []struct {
				ID uuid.UUID
				N  int64
			}
			if err := s.db.WithContext(ctx).Table(d.table).Select(d.idColumn+" AS id, COUNT(*) AS n").
				Where(d.idColumn+" IN ? AND deleted_at IS NULL", ids).Group(d.idColumn).Scan(&rows).Error; err != nil {
				return nil, 0, err
			}
			for _, r := range rows {
				counts[r.ID] += r.N
			}
		}
		for i := range out {
			out[i].Documents = counts[out[i].ID]
		}
	}
	return out, total, nil
}

func (s *InventoryService) getParty(db *gorm.DB, k PartyKind, id uuid.UUID) (*models.Party, error) {
	var p models.Party
	if err := db.Table(k.table).Where("id = ? AND deleted_at IS NULL", id).Take(&p).Error; err != nil {
		return nil, notFound(err)
	}
	return &p, nil
}

// SaveParty adds someone to the list (id nil) or changes them. A name is only once in what a
// branch sees (shared people and its own); two branches can each have their own of one name.
// Renaming someone renames them on their documents too.
func (s *InventoryService) SaveParty(ctx context.Context, k PartyKind, id *uuid.UUID, in PartyInput, scope CatalogScope) (*models.Party, error) {
	name := strings.Join(strings.Fields(in.Name), " ")
	if len([]rune(name)) < 2 {
		return nil, fieldError("name", "enter a name of at least 2 characters")
	}
	var saved models.Party
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old *models.Party
		if id != nil {
			var err error
			if old, err = s.getParty(tx, k, *id); err != nil {
				return err
			}
			if err := MayChange(old.BranchID, scope); err != nil {
				return err
			}
		}
		var owner *uuid.UUID
		if old != nil {
			owner = old.BranchID
		}
		if old == nil || in.BranchID != nil {
			var err error
			if owner, err = ParseOwner(ctx, tx, strOr(in.BranchID), scope); err != nil {
				return err
			}
			if old != nil && owner != nil && !sameOwner(owner, old.BranchID) {
				queries := make([]string, len(k.docs))
				for i, d := range k.docs {
					queries[i] = "SELECT COUNT(*) FROM " + d.table + " WHERE " + d.idColumn + " = ? AND branch_id <> ? AND deleted_at IS NULL"
				}
				used, err := countAny(tx, queries, old.ID, *owner)
				if err != nil {
					return err
				}
				if used {
					return fieldError("branch_id", fmt.Sprintf("%s is on another branch's documents, so they can't be %s's own", old.Name, BranchName(ctx, tx, *owner)))
				}
			}
		}
		var same models.Party
		q := tx.Table(k.table).Where("name = ? AND deleted_at IS NULL", name)
		if owner != nil {
			q = q.Where("(branch_id IS NULL OR branch_id = ?)", *owner)
		}
		if id != nil {
			q = q.Where("id <> ?", *id)
		}
		if err := q.Take(&same).Error; err == nil {
			where := "the list"
			if same.BranchID != nil {
				where = BranchName(ctx, tx, *same.BranchID) + "'s list"
			} else if owner != nil {
				where = "the shared list"
			}
			if !same.IsActive {
				return fieldError("name", fmt.Sprintf("a %s named %q is in %s but turned off; turn them on instead", k.label, same.Name, where))
			}
			return fieldError("name", fmt.Sprintf("a %s named %q is already in %s", k.label, same.Name, where))
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		fields := map[string]interface{}{
			"name": name, "phone": strings.TrimSpace(in.Phone), "email": strings.TrimSpace(in.Email),
			"address": strings.TrimSpace(in.Address), "note": strings.TrimSpace(in.Note),
		}
		if old == nil {
			p := models.Party{Base: models.Base{ID: uuid.New()}, Name: name, Phone: fields["phone"].(string), Email: fields["email"].(string),
				Address: fields["address"].(string), Note: fields["note"].(string), IsActive: true, BranchID: owner}
			if err := tx.Table(k.table).Create(&p).Error; err != nil {
				return err
			}
			saved = p
			return nil
		}
		fields["branch_id"] = owner
		if in.IsActive != nil {
			fields["is_active"] = *in.IsActive
		}
		fields["updated_at"] = time.Now()
		if err := tx.Table(k.table).Where("id = ?", old.ID).Updates(fields).Error; err != nil {
			return err
		}
		if old.Name != name {
			for _, d := range k.docs {
				if err := tx.Table(d.table).Where(d.idColumn+" = ?", old.ID).Update(d.nameColumn, name).Error; err != nil {
					return err
				}
			}
		}
		p, err := s.getParty(tx, k, old.ID)
		if err != nil {
			return err
		}
		saved = *p
		return nil
	})
	if err != nil {
		return nil, err
	}
	if saved.BranchID != nil {
		saved.BranchName = BranchName(ctx, s.db, *saved.BranchID)
	}
	return &saved, nil
}

// pickParty works out a document's customer or supplier: someone from the list (by id; the
// document takes their name), shared or the document branch's own, or, with no id, the name
// typed. A turned-off one is refused unless the document already has them (current).
func (s *InventoryService) pickParty(ctx context.Context, k PartyKind, rawID, name string, current *uuid.UUID, branch uuid.UUID) (*uuid.UUID, string, error) {
	rawID = strings.TrimSpace(rawID)
	if rawID == "" {
		return nil, strings.TrimSpace(name), nil
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, "", fieldError(k.field, fmt.Sprintf("choose a %s from the list", k.label))
	}
	p, err := s.getParty(s.db.WithContext(ctx), k, id)
	if err != nil {
		if errors.Is(err, ErrInventoryNotFound) {
			return nil, "", fieldError(k.field, fmt.Sprintf("that %s isn't in the list any more", k.label))
		}
		return nil, "", err
	}
	if !usableIn(p.BranchID, branch) {
		return nil, "", fieldError(k.field, fmt.Sprintf("%s is in %s's list only; choose another %s", p.Name, BranchName(ctx, s.db, *p.BranchID), k.label))
	}
	if !p.IsActive && (current == nil || *current != id) {
		return nil, "", fieldError(k.field, fmt.Sprintf("%s is turned off; choose another %s or turn them on", p.Name, k.label))
	}
	return &p.ID, p.Name, nil
}
