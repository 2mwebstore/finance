package services

import (
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
)

// New branches get the next BR-number; typed codes like PP-01 don't count. Needs TEST_MYSQL_DSN
// (see platform_integration_test.go).
func TestBranchNextCode(t *testing.T) {
	f := newWLFixture(t)
	f.db.Unscoped().Where("code LIKE ?", "BR-%").Delete(&models.Branch{})
	svc := NewBranchService(f.db)
	first, err := svc.Create(f.ctx, CreateBranchInput{Name: "Test branch one"})
	if err != nil || first.Code != "BR-001" {
		t.Fatalf("first generated code: %v %v", first, err)
	}
	second, err := svc.Create(f.ctx, CreateBranchInput{Name: "Test branch two"})
	if err != nil || second.Code != "BR-002" {
		t.Fatalf("second generated code: %v %v", second, err)
	}
	// A deleted branch's number isn't given again.
	f.db.Delete(second)
	if next, _ := svc.NextCode(f.ctx); next != "BR-003" {
		t.Errorf("after deleting BR-002 the next code is BR-003, got %s", next)
	}
	f.db.Unscoped().Where("code LIKE ?", "BR-%").Delete(&models.Branch{})
}
