package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Changing a saved sale or purchase. The document keeps its number; its old lines are taken back
// out of (or put back into) stock and the new lines applied, all in one transaction, so stock and
// average cost end up as if the new version had been saved in the first place.

// keepRate: the document's rate when its currency stays the same, else today's rate.
func (s *InventoryService) keepRate(ctx context.Context, oldCurrency string, oldRate decimal.Decimal, newCurrency string) (decimal.Decimal, error) {
	rate, err := currencyRate(ctx, s.db, newCurrency)
	if err != nil {
		return decimal.Zero, err
	}
	if newCurrency == oldCurrency && oldRate.IsPositive() {
		return oldRate, nil
	}
	return rate, nil
}

// UpdateSale changes a completed sale: its goods go back into stock and the new lines go out
// (at the average cost after that), in the sale's branch (or a new one).
func (s *InventoryService) UpdateSale(ctx context.Context, id uuid.UUID, in SaleInput, scope BranchScope, actor uuid.UUID) (*SaleView, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	date, err := s.parseDocDate("sale_date", in.SaleDate)
	if err != nil {
		return nil, err
	}
	due, err := s.parseDueDate(in.DueDate, date)
	if err != nil {
		return nil, err
	}
	paymentID, err := s.paymentMethod(ctx, in.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	lines, err := s.parseLines(ctx, in.Items, "unit_price")
	if err != nil {
		return nil, err
	}
	subtotal, discount, shipping, total, err := docTotals(lines, in.Discount, in.Shipping)
	if err != nil {
		return nil, err
	}
	note, err := cleanNote(in.Note)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sale models.Sale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&sale, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		if !scope.Has(sale.BranchID) {
			return ErrInventoryNotFound
		}
		if sale.Status != models.DocCompleted {
			return fmt.Errorf("a %s sale can't be changed", sale.Status)
		}
		rate, err := s.keepRate(ctx, sale.Currency, sale.ExchangeRate, in.Currency)
		if err != nil {
			return err
		}
		// The old lines come back into stock at the cost they left at.
		for _, i := range lockOrder(len(sale.Items), func(i int) uuid.UUID { return sale.Items[i].ProductID }) {
			it := sale.Items[i]
			if _, err := s.applyMove(tx, move{
				branchID: sale.BranchID, productID: it.ProductID, qty: it.Quantity, unitCost: it.UnitCost,
				kind: models.MoveSaleEdit, refType: "sale", refID: &sale.ID, refNumber: sale.Number, note: "sale changed", actor: actor,
			}); err != nil {
				return err
			}
		}
		// The new lines go out.
		discountShares := shares(lines, discount)
		items := make([]models.SaleItem, len(lines))
		for i, l := range lines {
			net := l.lineTotal.Sub(discountShares[i]).Round(4)
			items[i] = models.SaleItem{
				ID: uuid.New(), SaleID: sale.ID, ProductID: l.product.ID, Quantity: l.qty,
				UnitPrice: l.amount, Discount: l.discount, LineTotal: l.lineTotal, NetTotal: net, NetTotalBase: toBase(net, in.Currency, rate), Note: l.note,
			}
		}
		totalCost := decimal.Zero
		for _, i := range lockOrder(len(items), func(i int) uuid.UUID { return items[i].ProductID }) {
			cost, err := s.applyMove(tx, move{
				branchID: branchID, productID: items[i].ProductID, qty: items[i].Quantity.Neg(),
				kind: models.MoveSaleEdit, refType: "sale", refID: &sale.ID, refNumber: sale.Number, note: "sale changed", actor: actor,
			})
			if err != nil {
				return err
			}
			items[i].UnitCost = cost
			totalCost = totalCost.Add(items[i].Quantity.Mul(cost).Round(4))
		}
		if err := tx.Where("sale_id = ?", sale.ID).Delete(&models.SaleItem{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		return tx.Model(&models.Sale{}).Where("id = ?", sale.ID).Updates(map[string]interface{}{
			"branch_id": branchID, "customer": strings.TrimSpace(in.Customer), "sale_date": date, "due_date": due, "currency": in.Currency,
			"payment_method_id": paymentID, "subtotal": subtotal, "discount": discount, "shipping": shipping, "total": total,
			"exchange_rate": rate, "total_base": toBase(total, in.Currency, rate), "total_cost": totalCost, "note": note,
			"updated_at": time.Now(),
		}).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetSale(ctx, id, scope)
}

// UpdatePurchase changes a pending purchase, or an approved one: the new lines go into stock at
// their landed cost and the old goods come back out (enough must be left). The approval stays.
func (s *InventoryService) UpdatePurchase(ctx context.Context, id uuid.UUID, in PurchaseInput, scope BranchScope, actor uuid.UUID) (*PurchaseView, error) {
	branchID := uuid.MustParse(in.BranchID)
	if !scope.Has(branchID) {
		return nil, ErrBranchAccessDenied
	}
	date, err := s.parseDocDate("purchase_date", in.PurchaseDate)
	if err != nil {
		return nil, err
	}
	lines, err := s.parseLines(ctx, in.Items, "unit_cost")
	if err != nil {
		return nil, err
	}
	subtotal, discount, shipping, total, err := docTotals(lines, in.Discount, in.Shipping)
	if err != nil {
		return nil, err
	}
	paymentID, err := s.paymentMethod(ctx, in.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	note, err := cleanNote(in.Note)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p, err := lockPurchase(tx, id, scope)
		if err != nil {
			return err
		}
		if p.Status != models.DocPending && p.Status != models.DocApproved {
			return fmt.Errorf("a %s purchase can't be changed", p.Status)
		}
		rate, err := s.keepRate(ctx, p.Currency, p.ExchangeRate, in.Currency)
		if err != nil {
			return err
		}
		landed := landedCosts(lines, discount, shipping)
		items := make([]models.PurchaseItem, len(lines))
		for i, l := range lines {
			items[i] = models.PurchaseItem{
				ID: uuid.New(), PurchaseID: p.ID, ProductID: l.product.ID,
				Quantity: l.qty, UnitCost: l.amount, Discount: l.discount, LineTotal: l.lineTotal, LandedCost: toBase(landed[i], in.Currency, rate),
			}
		}
		if p.Status == models.DocApproved {
			// The new goods come in first, then the old ones leave at the cost they came in at: the
			// same stock value either way, but units already sold don't block a change that adds more.
			for _, i := range lockOrder(len(items), func(i int) uuid.UUID { return items[i].ProductID }) {
				if _, err := s.applyMove(tx, move{
					branchID: branchID, productID: items[i].ProductID, qty: items[i].Quantity, unitCost: items[i].LandedCost,
					kind: models.MovePurchaseEdit, refType: "purchase", refID: &p.ID, refNumber: p.Number, note: "purchase changed", actor: actor,
				}); err != nil {
					return err
				}
			}
			for _, i := range lockOrder(len(p.Items), func(i int) uuid.UUID { return p.Items[i].ProductID }) {
				it := p.Items[i]
				if _, err := s.applyMove(tx, move{
					branchID: p.BranchID, productID: it.ProductID, qty: it.Quantity.Neg(), unitCost: stockCost(it), atCost: true,
					kind: models.MovePurchaseEdit, refType: "purchase", refID: &p.ID, refNumber: p.Number, note: "purchase changed", actor: actor,
				}); err != nil {
					var short *stockShortError
					if errors.As(err, &short) {
						return usedUp(tx, it.ProductID, p.BranchID)
					}
					return err
				}
			}
		}
		if err := tx.Where("purchase_id = ?", p.ID).Delete(&models.PurchaseItem{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		return tx.Model(&models.Purchase{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"branch_id": branchID, "supplier": strings.TrimSpace(in.Supplier), "purchase_date": date, "currency": in.Currency,
			"payment_method_id": paymentID, "subtotal": subtotal, "discount": discount, "shipping": shipping, "total": total,
			"exchange_rate": rate, "total_base": toBase(total, in.Currency, rate), "note": note, "updated_at": time.Now(),
		}).Error
	})
	if err != nil {
		return nil, s.describeShort(ctx, err)
	}
	return s.GetPurchase(ctx, id, scope)
}

// usedUp: a purchase's old goods can't all leave stock again because some were sold or moved.
func usedUp(tx *gorm.DB, productID, branchID uuid.UUID) error {
	var p models.Product
	var b models.Branch
	tx.Select("sku", "name").First(&p, "id = ?", productID)
	tx.Select("code").First(&b, "id = ?", branchID)
	return fieldError("items", fmt.Sprintf("some %s (%s) from this purchase is already sold or moved out of %s, so it can't be cut that much", p.Name, p.SKU, b.Code))
}
