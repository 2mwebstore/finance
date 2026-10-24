package permissions

// Canonical permission codes.
const (
	DashboardView = "dashboard.view"

	IncomeView   = "income.view"
	IncomeCreate = "income.create"
	IncomeEdit   = "income.edit"
	IncomeVoid   = "income.void"

	ExpenseView   = "expense.view"
	ExpenseCreate = "expense.create"
	ExpenseEdit   = "expense.edit"
	ExpenseVoid   = "expense.void"

	TransactionsView    = "transactions.view"
	TransactionsApprove = "transactions.approve"
	TransactionsReject  = "transactions.reject"
	TransactionsReverse = "transactions.reverse"

	ReportsView   = "reports.view"
	ReportsExport = "reports.export"

	BranchesView   = "branches.view"
	BranchesCreate = "branches.create"
	BranchesEdit   = "branches.edit"
	BranchesManage = "branches.manage"

	UsersView   = "users.view"
	UsersCreate = "users.create"
	UsersEdit   = "users.edit"
	UsersManage = "users.manage"

	RolesView         = "roles.view"
	RolesManage       = "roles.manage"
	PermissionsManage = "permissions.manage"

	CategoriesManage       = "categories.manage"
	CurrencySettingsManage = "currency-settings.manage"
	AuditLogsView          = "audit-logs.view"
	SettingsManage         = "settings.manage"
	// BackupsManage: back up, download, upload and restore the whole database.
	BackupsManage = "backups.manage"
)

// AllPermissions returns the full list for seeding.
func AllPermissions() []struct {
	Code, Name, Module, Description string
} {
	return []struct {
		Code, Name, Module, Description string
	}{
		{DashboardView, "View Dashboard", "Dashboard", "Access the main dashboard"},

		{IncomeView, "View Income", "Income", "View income transactions"},
		{IncomeCreate, "Create Income", "Income", "Create income transactions"},
		{IncomeEdit, "Edit Income", "Income", "Edit income transactions"},
		{IncomeVoid, "Void Income", "Income", "Void income transactions"},

		{ExpenseView, "View Expenses", "Expenses", "View expense transactions"},
		{ExpenseCreate, "Create Expenses", "Expenses", "Create expense transactions"},
		{ExpenseEdit, "Edit Expenses", "Expenses", "Edit expense transactions"},
		{ExpenseVoid, "Void Expenses", "Expenses", "Void expense transactions"},

		{TransactionsView, "View Transactions", "Transactions", "View all transactions"},
		{TransactionsApprove, "Approve Transactions", "Transactions", "Approve pending transactions"},
		{TransactionsReject, "Reject Transactions", "Transactions", "Reject pending transactions"},
		{TransactionsReverse, "Reverse Transactions", "Transactions", "Reverse finalized transactions"},

		{ReportsView, "View Reports", "Reports", "View financial reports"},
		{ReportsExport, "Export Reports", "Reports", "Export reports to CSV/Excel/PDF"},

		{BranchesView, "View Branches", "Branches", "View branch list and details"},
		{BranchesCreate, "Create Branches", "Branches", "Create new branches"},
		{BranchesEdit, "Edit Branches", "Branches", "Edit branch details"},
		{BranchesManage, "Manage Branches", "Branches", "Full branch management including assignments"},

		{UsersView, "View Users", "Users", "View user list and profiles"},
		{UsersCreate, "Create Users", "Users", "Create new users"},
		{UsersEdit, "Edit Users", "Users", "Edit user profiles"},
		{UsersManage, "Manage Users", "Users", "Manage user roles, passwords, and account activation"},

		{RolesView, "View Roles", "Roles", "View roles and their permissions"},
		{RolesManage, "Manage Roles", "Roles", "Create, edit, and delete non-system roles"},
		{PermissionsManage, "Manage Permissions", "Roles", "Assign permissions to roles"},

		{CategoriesManage, "Manage Categories", "Settings", "Create, edit, and deactivate income/expense categories"},
		{CurrencySettingsManage, "Manage Currency Settings", "Settings", "Configure currency options"},
		{AuditLogsView, "View Audit Logs", "Audit", "View system audit logs"},
		{SettingsManage, "Manage Settings", "Settings", "Change application settings"},
		{BackupsManage, "Manage Backups", "Backups", "Back up the database, send backups to Telegram, and restore from a backup"},
	}
}

// DefaultRoleDataScopes: system roles that see every branch. The rest see the branches their
// users are assigned to ("assigned", the default).
var DefaultRoleDataScopes = map[string]string{"super_admin": "all", "owner": "all", "admin": "all"}

// DefaultRoles returns system roles and their permission codes.
func DefaultRoles() map[string]struct {
	Name        string
	Description string
	Permissions []string
} {
	all := make([]string, 0)
	for _, p := range AllPermissions() {
		all = append(all, p.Code)
	}

	return map[string]struct {
		Name        string
		Description string
		Permissions []string
	}{
		"super_admin": {
			Name:        "Super Admin",
			Description: "Full system access. Cannot be demoted without replacement.",
			Permissions: all,
		},
		"owner": {
			Name:        "Owner",
			Description: "Full system permissions for the seeded demo owner.",
			Permissions: all,
		},
		"admin": {
			Name:        "Admin",
			Description: "Administrative access excluding critical system settings.",
			Permissions: []string{
				DashboardView,
				IncomeView, IncomeCreate, IncomeEdit, IncomeVoid,
				ExpenseView, ExpenseCreate, ExpenseEdit, ExpenseVoid,
				TransactionsView, TransactionsApprove, TransactionsReject, TransactionsReverse,
				ReportsView, ReportsExport,
				BranchesView, BranchesCreate, BranchesEdit, BranchesManage,
				UsersView, UsersCreate, UsersEdit, UsersManage,
				RolesView, RolesManage,
				CategoriesManage, AuditLogsView, SettingsManage,
			},
		},
		"manager": {
			Name:        "Manager",
			Description: "Branch manager with reporting and approval rights.",
			Permissions: []string{
				DashboardView,
				IncomeView, IncomeCreate, IncomeEdit, IncomeVoid,
				ExpenseView, ExpenseCreate, ExpenseEdit, ExpenseVoid,
				TransactionsView, TransactionsApprove, TransactionsReject,
				ReportsView, ReportsExport,
				BranchesView,
				UsersView,
				CategoriesManage,
			},
		},
		"accountant": {
			Name:        "Accountant",
			Description: "Financial data entry and reporting.",
			Permissions: []string{
				DashboardView,
				IncomeView, IncomeCreate, IncomeEdit,
				ExpenseView, ExpenseCreate, ExpenseEdit,
				TransactionsView,
				ReportsView, ReportsExport,
				BranchesView,
			},
		},
		"cashier": {
			Name:        "Cashier",
			Description: "Create income and expense transactions only.",
			Permissions: []string{
				DashboardView,
				IncomeView, IncomeCreate,
				ExpenseView, ExpenseCreate,
				TransactionsView,
			},
		},
		"viewer": {
			Name:        "Viewer",
			Description: "Read-only access to dashboard and reports.",
			Permissions: []string{
				DashboardView,
				IncomeView,
				ExpenseView,
				TransactionsView,
				ReportsView,
				BranchesView,
			},
		},
	}
}
