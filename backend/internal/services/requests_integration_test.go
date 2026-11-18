package services

// Requests (expenses waiting for approval) against a real MySQL database (only with TEST_MYSQL_DSN
// naming a scratch database whose name contains "test"; see platform_integration_test.go). The
// test branches are TRQ1 and TRQ2; everyone it adds is named "trq_…".

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cambodia-finance/income-expense/internal/database"
	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRequestsApproveRejectAndPermissions(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil || !strings.Contains(cfg.DBName, "test") {
		t.Fatalf("TEST_MYSQL_DSN must name a scratch database with \"test\" in its name, got %q", cfg.DBName)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	svc := NewTransactionService(db, "Asia/Phnom_Penh")

	branch := models.Branch{Code: "TRQ1", Name: "Requests test TRQ1", IsActive: true}
	other := models.Branch{Code: "TRQ2", Name: "Requests test TRQ2", IsActive: true}
	for _, b := range []*models.Branch{&branch, &other} {
		if err := db.Where(models.Branch{Code: b.Code}).FirstOrCreate(b).Error; err != nil {
			t.Fatal(err)
		}
	}
	clean := func() {
		var ids []uuid.UUID
		db.Model(&models.Transaction{}).Unscoped().Where("branch_id IN ?", []uuid.UUID{branch.ID, other.ID}).Pluck("id", &ids)
		if len(ids) > 0 {
			db.Where("transaction_id IN ?", ids).Delete(&models.TransactionApproval{})
			db.Where("transaction_id IN ?", ids).Delete(&models.TransactionAttachment{})
			db.Unscoped().Where("id IN ?", ids).Delete(&models.Transaction{})
		}
		var users []models.User
		db.Unscoped().Where("username LIKE ?", "trq_%").Find(&users)
		for i := range users {
			db.Exec("DELETE FROM user_roles WHERE user_id = ?", users[i].ID)
			db.Exec("DELETE FROM user_branches WHERE user_id = ?", users[i].ID)
			db.Unscoped().Delete(&users[i])
		}
		var roles []models.Role
		db.Unscoped().Where("code LIKE ?", "trq_%").Find(&roles)
		for i := range roles {
			db.Exec("DELETE FROM role_permissions WHERE role_id = ?", roles[i].ID)
			db.Unscoped().Delete(&roles[i])
		}
		db.Unscoped().Where("code LIKE ?", "trq_%").Delete(&models.Category{})
		db.Unscoped().Where("code LIKE ?", "TRQ_%").Delete(&models.PaymentMethod{})
	}
	clean()
	t.Cleanup(clean)

	perm := func(code string) models.Permission {
		t.Helper()
		for _, p := range permissions.AllPermissions() {
			if p.Code == code {
				var row models.Permission
				if err := db.Where(models.Permission{Code: code}).Attrs(models.Permission{Name: p.Name, Module: p.Module}).FirstOrCreate(&row).Error; err != nil {
					t.Fatal(err)
				}
				return row
			}
		}
		t.Fatalf("unknown permission %s", code)
		return models.Permission{}
	}
	role := func(code string, codes ...string) models.Role {
		t.Helper()
		r := models.Role{Code: code, Name: code}
		for _, c := range codes {
			r.Permissions = append(r.Permissions, perm(c))
		}
		if err := db.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
		return r
	}
	person := func(name string, r models.Role, in ...models.Branch) uuid.UUID {
		t.Helper()
		u := models.User{Username: name, Email: name + "@example.test", PasswordHash: "x", FullName: name, IsActive: true, Roles: []models.Role{r}, Branches: in}
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		return u.ID
	}

	// When the request permissions first appear, roles keep what they could do before.
	t.Run("start-up grant", func(t *testing.T) {
		entering := role("trq_entering", permissions.ExpenseCreate)
		approving := role("trq_approving", permissions.TransactionsApprove)
		for _, code := range []string{permissions.RequestsView, permissions.RequestsCreate, permissions.RequestsApprove} {
			var p models.Permission
			if db.Unscoped().Where("code = ?", code).First(&p).Error == nil {
				db.Exec("DELETE FROM role_permissions WHERE permission_id = ?", p.ID)
				db.Unscoped().Delete(&p)
			}
		}
		if err := database.EnsureRequiredPermissions(db); err != nil {
			t.Fatal(err)
		}
		has := func(r models.Role) map[string]bool {
			var got models.Role
			db.Preload("Permissions").First(&got, "id = ?", r.ID)
			out := map[string]bool{}
			for _, p := range got.Permissions {
				out[p.Code] = true
			}
			return out
		}
		e, a := has(entering), has(approving)
		if !e[permissions.RequestsView] || !e[permissions.RequestsCreate] || e[permissions.RequestsApprove] {
			t.Errorf("Create Expenses should bring View and Make Requests only: %v", e)
		}
		if !a[permissions.RequestsView] || !a[permissions.RequestsApprove] || a[permissions.RequestsCreate] {
			t.Errorf("Approve Transactions should bring View and Approve Requests only: %v", a)
		}
	})

	cat := models.Category{Code: "trq_supplies", Name: "TRQ supplies", Type: models.TxnTypeExpense, IsActive: true, BranchID: &branch.ID}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	pm := models.PaymentMethod{Code: "TRQ_CASH", Name: "TRQ cash", IsActive: true, BranchID: &branch.ID}
	if err := db.Create(&pm).Error; err != nil {
		t.Fatal(err)
	}

	requester := person("trq_requester", role("trq_requester", permissions.RequestsView, permissions.RequestsCreate), branch)
	makerOnly := person("trq_maker", role("trq_maker", permissions.RequestsCreate), branch)
	approver := person("trq_approver", role("trq_approver", permissions.RequestsView, permissions.RequestsCreate, permissions.RequestsApprove, permissions.ExpenseCreate), branch)
	// Approve Transactions alone no longer approves expenses (they are requests).
	oldApprover := person("trq_old_approver", role("trq_old_approver", permissions.TransactionsApprove, permissions.TransactionsReject, permissions.ExpenseCreate, permissions.ExpenseView), branch)
	elsewhere := person("trq_elsewhere", role("trq_elsewhere", permissions.RequestsView, permissions.RequestsApprove), other)

	today := time.Now().Format("2006-01-02")
	ask := func(actor uuid.UUID, amount string) (*models.Transaction, error) {
		return svc.CreateRequest(ctx, CreateRequestInput{
			BranchID: branch.ID.String(), CategoryID: cat.ID.String(), Currency: "USD", Amount: amount,
			TransactionDate: today, Description: "TRQ request " + amount,
		}, actor)
	}
	receipt := func(txnID, by uuid.UUID) {
		t.Helper()
		if err := db.Create(&models.TransactionAttachment{TransactionID: txnID, StorageKey: "trq/" + uuid.NewString(), Name: "receipt.jpg", ContentType: "image/jpeg", Size: 10, CreatedByID: by}).Error; err != nil {
			t.Fatal(err)
		}
	}
	listed := func(items []models.Transaction, id uuid.UUID) bool {
		for _, it := range items {
			if it.ID == id {
				return true
			}
		}
		return false
	}
	requests := func(actor uuid.UUID, status string) []models.Transaction {
		t.Helper()
		items, _, _, err := svc.ListRequests(ctx, RequestListFilter{Status: status, BranchID: branch.ID.String(), PageSize: 100}, actor)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}
	expenses := func() []models.Transaction {
		t.Helper()
		items, _, err := svc.List(ctx, TransactionListFilter{Type: models.TxnTypeExpense, BranchID: branch.ID.String(), ExcludeRequests: true, PageSize: 100}, approver)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}

	// Making a request needs Make Requests; Create Expenses isn't enough.
	if _, err := ask(oldApprover, "5"); !errors.Is(err, ErrTypeNotAllowed) {
		t.Fatalf("request without Make Requests: %v", err)
	}
	mine, err := ask(makerOnly, "7")
	if err != nil {
		t.Fatal(err)
	}
	r1, err := ask(requester, "25.50")
	if err != nil || r1.Status != models.TxnStatusPending || r1.Type != models.TxnTypeExpense {
		t.Fatalf("request: %+v %v", r1, err)
	}
	// Even someone who could approve it waits: a request always goes to someone else.
	own, err := ask(approver, "3")
	if err != nil || own.Status != models.TxnStatusPending {
		t.Fatalf("approver's own request should be pending: %+v %v", own, err)
	}

	// Pending: on Requests, not on Expense.
	if !listed(requests(approver, RequestStatusPending), r1.ID) {
		t.Error("pending request missing from Requests")
	}
	if listed(expenses(), r1.ID) {
		t.Error("pending request should not be on Expense")
	}
	// Someone who may only make requests sees their own.
	if got := requests(makerOnly, ""); len(got) != 1 || got[0].ID != mine.ID {
		t.Errorf("maker should see only their own request, got %d", len(got))
	}

	// Who may decide: the buttons and the API agree.
	decisions, err := svc.Decisions(ctx, approver, []models.Transaction{*r1})
	if err != nil || !decisions[r1.ID].CanApprove || !decisions[r1.ID].CanReject {
		t.Errorf("approver should be offered approve and reject: %+v %v", decisions[r1.ID], err)
	}
	if d, _ := svc.Decisions(ctx, oldApprover, []models.Transaction{*r1}); d[r1.ID].CanApprove || d[r1.ID].CanReject {
		t.Errorf("Approve Transactions alone should not decide requests: %+v", d[r1.ID])
	}
	if _, err := svc.ApproveRequest(ctx, r1.ID, "", oldApprover); !errors.Is(err, ErrTypeNotAllowed) {
		t.Errorf("approve without Approve Requests: %v", err)
	}
	if _, err := svc.Approve(ctx, r1.ID, "", oldApprover); !errors.Is(err, ErrTypeNotAllowed) {
		t.Errorf("approve on the transactions route without Approve Requests: %v", err)
	}
	if _, err := svc.ApproveRequest(ctx, own.ID, "", approver); err == nil || !strings.Contains(err.Error(), "own") {
		t.Errorf("approving your own request: %v", err)
	}
	if _, err := svc.ApproveRequest(ctx, r1.ID, "", approver); !errors.Is(err, ErrReceiptRequired) {
		t.Errorf("approve without a receipt: %v", err)
	}
	receipt(r1.ID, requester)
	if _, err := svc.ApproveRequest(ctx, r1.ID, "", elsewhere); !errors.Is(err, ErrBranchAccessDenied) {
		t.Errorf("approver of another branch: %v", err)
	}
	// The requester adds files to their own pending request only.
	if !svc.CanChangeAttachments(ctx, makerOnly, mine) || svc.CanChangeAttachments(ctx, makerOnly, r1) {
		t.Error("Make Requests should change the files of your own pending request only")
	}

	// Approve: two people at once, only one wins; the request is now an expense.
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = svc.ApproveRequest(ctx, r1.ID, "ok", approver)
		}(i)
	}
	wg.Wait()
	wins := 0
	for _, err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("second approval: %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("exactly one approval should win, got %d (%v)", wins, results)
	}
	approved, err := svc.GetByID(ctx, r1.ID, approver)
	if err != nil || approved.Status != models.TxnStatusApproved || approved.ApprovedByID == nil || *approved.ApprovedByID != approver {
		t.Fatalf("approved request: %+v %v", approved, err)
	}
	var records int64
	db.Model(&models.TransactionApproval{}).Where("transaction_id = ? AND action = ?", r1.ID, "approve").Count(&records)
	if records != 1 {
		t.Errorf("one approval record expected, got %d", records)
	}
	if _, err := svc.ApproveRequest(ctx, r1.ID, "", approver); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("approving twice: %v", err)
	}
	if !listed(expenses(), r1.ID) {
		t.Error("approved request should be on Expense")
	}
	if !listed(requests(approver, RequestStatusApproved), r1.ID) || listed(requests(approver, RequestStatusPending), r1.ID) {
		t.Error("approved request should be under Approved, not Pending")
	}
	totals, err := svc.TotalsByCurrency(ctx, approver, &branch.ID, nil, nil, models.TxnTypeExpense, "USD", "")
	if err != nil || totals["expense_usd"].String() != "25.5" {
		t.Errorf("approved request should count once: %v %v", totals["expense_usd"], err)
	}

	// Reject: a reason is required; it stays on Requests and never counts.
	r2, err := ask(requester, "40")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RejectRequest(ctx, r2.ID, "  ", approver); !errors.Is(err, ErrReasonRequired) {
		t.Errorf("reject without a reason: %v", err)
	}
	if _, err := svc.RejectRequest(ctx, r2.ID, "no receipt", oldApprover); !errors.Is(err, ErrTypeNotAllowed) {
		t.Errorf("reject without Approve Requests: %v", err)
	}
	rejected, err := svc.RejectRequest(ctx, r2.ID, "no receipt", approver)
	if err != nil || rejected.Status != models.TxnStatusRejected {
		t.Fatalf("reject: %+v %v", rejected, err)
	}
	if out := svc.People(ctx, []models.Transaction{*rejected})[r2.ID].Outcome; out == nil || out.Action != "rejected" || out.Notes != "no receipt" {
		t.Errorf("rejection reason: %+v", out)
	}
	if !listed(requests(approver, RequestStatusRejected), r2.ID) || listed(expenses(), r2.ID) {
		t.Error("rejected request stays on Requests only")
	}
	if _, err := svc.ApproveRequest(ctx, r2.ID, "", approver); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("approving a rejected request: %v", err)
	}
	if totals, _ := svc.TotalsByCurrency(ctx, approver, &branch.ID, nil, nil, models.TxnTypeExpense, "USD", ""); totals["expense_usd"].String() != "25.5" {
		t.Errorf("rejected request should not count: %v", totals["expense_usd"])
	}

	// Expenses entered on Expense: approved at once with Approve Requests (not a request), pending
	// without it (Approve Transactions doesn't approve expenses any more).
	direct, err := svc.Create(ctx, CreateTransactionInput{Type: models.TxnTypeExpense, BranchID: branch.ID.String(), CategoryID: cat.ID.String(), Currency: "USD", Amount: "9", TransactionDate: today}, approver)
	if err != nil || direct.Status != models.TxnStatusApproved {
		t.Fatalf("expense by an approver: %+v %v", direct, err)
	}
	if listed(requests(approver, ""), direct.ID) {
		t.Error("an expense approved when entered is not a request")
	}
	entered, err := svc.Create(ctx, CreateTransactionInput{Type: models.TxnTypeExpense, BranchID: branch.ID.String(), CategoryID: cat.ID.String(), Currency: "USD", Amount: "11", TransactionDate: today}, oldApprover)
	if err != nil || entered.Status != models.TxnStatusPending {
		t.Fatalf("expense without Approve Requests should wait: %+v %v", entered, err)
	}
	if !listed(requests(approver, RequestStatusPending), entered.ID) {
		t.Error("a pending expense entered on Expense is a request too")
	}

	// The chips' numbers.
	_, _, counts, err := svc.ListRequests(ctx, RequestListFilter{BranchID: branch.ID.String()}, approver)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Pending != 3 || counts.Approved != 1 || counts.Rejected != 1 || counts.All != 5 {
		t.Errorf("counts: %+v", counts)
	}
	// The sidebar's pending chip: the same as the Pending chip, for what each person sees.
	if n, err := svc.PendingRequestCount(ctx, approver, branch.ID.String()); err != nil || n != 3 {
		t.Errorf("approver's pending count: %d %v", n, err)
	}
	if n, err := svc.PendingRequestCount(ctx, makerOnly, ""); err != nil || n != 1 {
		t.Errorf("Make Requests only counts your own pending request: %d %v", n, err)
	}
	if _, err := svc.PendingRequestCount(ctx, elsewhere, branch.ID.String()); !errors.Is(err, ErrBranchAccessDenied) {
		t.Errorf("pending count of another branch: %v", err)
	}
	if _, _, _, err := svc.ListRequests(ctx, RequestListFilter{Status: "maybe"}, approver); err == nil {
		t.Error("unknown status should be refused")
	}

	// Only a Super Administrator decides on their own request (the receipt rule still applies).
	superUser := models.User{Username: "trq_super", Email: "trq_super@example.test", PasswordHash: "x", FullName: "trq_super", IsActive: true, IsSuperAdmin: true}
	if err := db.Create(&superUser).Error; err != nil {
		t.Fatal(err)
	}
	super := superUser.ID
	s1, err := ask(super, "15")
	if err != nil || s1.Status != models.TxnStatusPending {
		t.Fatalf("super admin's request should wait like any other: %+v %v", s1, err)
	}
	if d, _ := svc.Decisions(ctx, super, []models.Transaction{*s1}); !d[s1.ID].CanApprove || !d[s1.ID].CanReject {
		t.Errorf("super admin should be offered approve and reject on their own request: %+v", d[s1.ID])
	}
	if !svc.CanChangeApprovalFiles(ctx, super, s1) {
		t.Error("super admin should add approval files to their own request")
	}
	if _, err := svc.ApproveRequest(ctx, s1.ID, "", super); !errors.Is(err, ErrReceiptRequired) {
		t.Errorf("super admin approving without a receipt: %v", err)
	}
	receipt(s1.ID, super)
	if got, err := svc.ApproveRequest(ctx, s1.ID, "", super); err != nil || got.Status != models.TxnStatusApproved {
		t.Errorf("super admin approving their own request: %+v %v", got, err)
	}
	s2, err := ask(super, "16")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := svc.RejectRequest(ctx, s2.ID, "not needed", super); err != nil || got.Status != models.TxnStatusRejected {
		t.Errorf("super admin rejecting their own request: %+v %v", got, err)
	}
	// Everyone else still can't, owner/admin included.
	if d, _ := svc.Decisions(ctx, approver, []models.Transaction{*own}); d[own.ID].CanApprove || d[own.ID].CanReject {
		t.Errorf("approver should not be offered their own request: %+v", d[own.ID])
	}
}
