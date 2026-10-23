package permissions

// Codes returns every canonical permission code in stable order.
// Use this for validation, API listings, and tests.
func Codes() []string {
	list := AllPermissions()
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, p.Code)
	}
	return out
}

// IsValid reports whether code is a known permission.
func IsValid(code string) bool {
	for _, c := range Codes() {
		if c == code {
			return true
		}
	}
	return false
}

// Count is the number of seeded permission codes.
const Count = 31

// CodeList is the flat list of all permission code strings (same as Codes()).
// Prefer Codes() or constants (DashboardView, etc.) in application logic.
var CodeList = []string{
	DashboardView,
	IncomeView, IncomeCreate, IncomeEdit, IncomeVoid,
	ExpenseView, ExpenseCreate, ExpenseEdit, ExpenseVoid,
	TransactionsView, TransactionsApprove, TransactionsReject, TransactionsReverse,
	ReportsView, ReportsExport,
	BranchesView, BranchesCreate, BranchesEdit, BranchesManage,
	UsersView, UsersCreate, UsersEdit, UsersManage,
	RolesView, RolesManage, PermissionsManage,
	CategoriesManage, CurrencySettingsManage, AuditLogsView, SettingsManage,
	BackupsManage,
}
