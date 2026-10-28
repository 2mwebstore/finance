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
	// PermissionsView shows the Permission Codes page (the list of every permission).
	PermissionsView = "permissions.view"

	CategoriesManage       = "categories.manage"
	CurrencySettingsManage = "currency-settings.manage"
	AuditLogsView          = "audit-logs.view"
	SettingsManage         = "settings.manage"
	// BackupsManage: back up, download, upload and restore the whole database.
	BackupsManage = "backups.manage"

	// Inventory: products and stock per branch, sales, purchases, transfers.
	InventoryView     = "inventory.view"
	InventoryManage   = "inventory.manage"
	InventoryAdjust   = "inventory.adjust"
	InventoryTransfer = "inventory.transfer"
	SalesView         = "sales.view"
	SalesCreate       = "sales.create"
	SalesVoid         = "sales.void"
	SalesEdit         = "sales.edit"
	PurchasesView     = "purchases.view"
	PurchasesCreate   = "purchases.create"
	PurchasesApprove  = "purchases.approve"
	PurchasesVoid     = "purchases.void"
	PurchasesEdit     = "purchases.edit"

	InventoryExpensesView   = "inventory-expenses.view"
	InventoryExpensesCreate = "inventory-expenses.create"
	InventoryExpensesVoid   = "inventory-expenses.void"

	// Platforms and weekly win/lose (an HQ is a branch).
	PlatformsView              = "platforms.view"
	PlatformsManage            = "platforms.manage"
	PlatformAssignmentsManage  = "platform-assignments.manage"
	WinLoseView                = "winlose.view"
	WinLoseCreate              = "winlose.create"
	WinLoseEdit                = "winlose.edit"
	WinLoseReportsView         = "winlose-reports.view"
	WinLoseReportsConsolidated = "winlose-reports.consolidated"
	WinLoseSettlementsFinalize = "winlose-settlements.finalize"
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
		{PermissionsView, "View Permission Codes", "Roles", "See the Permission Codes page with every permission in the system"},

		{CategoriesManage, "Manage Categories", "Settings", "Create, edit, and deactivate income/expense categories"},
		{CurrencySettingsManage, "Manage Currency Settings", "Settings", "Configure currency options"},
		{AuditLogsView, "View Audit Logs", "Audit", "View system audit logs"},
		{SettingsManage, "Manage Settings", "Settings", "Change application settings"},
		{BackupsManage, "Manage Backups", "Backups", "Back up the database, send backups to Telegram, and restore from a backup"},

		{InventoryView, "View Inventory", "Inventory", "See products, stock levels and stock history of your branches"},
		{InventoryManage, "Manage Products", "Inventory", "Create and edit products and product categories (shared by every branch)"},
		{InventoryAdjust, "Adjust Stock", "Inventory", "Add, remove or count stock in your branches, with a reason"},
		{InventoryTransfer, "Transfer Stock", "Inventory", "Move stock between branches"},
		{SalesView, "View Sales", "Sales", "See sales of your branches"},
		{SalesCreate, "Create Sales", "Sales", "Record sales; their stock goes out right away"},
		{SalesVoid, "Void Sales", "Sales", "Void a sale and put its stock back"},
		{SalesEdit, "Edit Sales", "Sales", "Change a sale; its goods are put back and the new lines taken out of stock"},
		{PurchasesView, "View Purchases", "Purchases", "See purchases of your branches"},
		{PurchasesCreate, "Create Purchases", "Purchases", "Record purchases from suppliers; stock goes in once approved"},
		{PurchasesApprove, "Approve Purchases", "Purchases", "Approve or reject purchases; approved purchases add stock"},
		{PurchasesVoid, "Void Purchases", "Purchases", "Void a purchase and take its stock back out"},
		{PurchasesEdit, "Edit Purchases", "Purchases", "Change a pending purchase (an approved one also needs Approve Purchases)"},
		{InventoryExpensesView, "View Inventory Expenses", "Inventory Expenses", "See inventory expenses of your branches"},
		{InventoryExpensesCreate, "Create Inventory Expenses", "Inventory Expenses", "Record inventory expenses (shipping, packaging, storage…)"},
		{InventoryExpensesVoid, "Void Inventory Expenses", "Inventory Expenses", "Void an inventory expense"},

		{PlatformsView, "View Platforms", "Platforms", "See platforms, partners and the branches they are assigned to"},
		{PlatformsManage, "Manage Platforms", "Platforms", "Add and edit platforms and partners, with their default partner % and currency"},
		{PlatformAssignmentsManage, "Manage Branch Platform Assignments", "Platforms", "Assign platforms to branches with their partners; change partners and their %, end or move assignments"},
		{WinLoseView, "View Win/Lose", "Win/Lose", "See the weekly win/lose entered for your branches"},
		{WinLoseCreate, "Enter Win/Lose", "Win/Lose", "Enter a week's win/lose and commission for your branches (once a week)"},
		{WinLoseEdit, "Correct Win/Lose", "Win/Lose", "Change a week already saved, with a reason (kept in the audit log)"},
		{WinLoseReportsView, "View Win/Lose Reports", "Win/Lose Reports", "The weekly report and the branch summary for your branches, one branch at a time"},
		{WinLoseReportsConsolidated, "View Consolidated Win/Lose", "Win/Lose Reports", "Combined totals across all your branches"},
		{WinLoseSettlementsFinalize, "Finalize Weekly Settlements", "Win/Lose Reports", "Finalize or reopen a branch's week; a finalized week can't be changed"},
	}
}

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
				RolesView, RolesManage, PermissionsView,
				CategoriesManage, AuditLogsView, SettingsManage,
				InventoryView, InventoryManage, InventoryAdjust, InventoryTransfer,
				SalesView, SalesCreate, SalesVoid, SalesEdit,
				PurchasesView, PurchasesCreate, PurchasesApprove, PurchasesVoid, PurchasesEdit,
				InventoryExpensesView, InventoryExpensesCreate, InventoryExpensesVoid,
				PlatformsView, PlatformsManage, PlatformAssignmentsManage,
				WinLoseView, WinLoseCreate, WinLoseEdit,
				WinLoseReportsView, WinLoseReportsConsolidated, WinLoseSettlementsFinalize,
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
				InventoryView, InventoryAdjust,
				SalesView, SalesCreate, SalesVoid, SalesEdit,
				PurchasesView, PurchasesCreate, PurchasesApprove, PurchasesEdit,
				InventoryExpensesView, InventoryExpensesCreate,
				PlatformsView, WinLoseView, WinLoseCreate, WinLoseEdit, WinLoseReportsView,
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
				InventoryView, SalesView, PurchasesView, PurchasesCreate, PurchasesEdit,
				InventoryExpensesView, InventoryExpensesCreate,
				PlatformsView, WinLoseView, WinLoseCreate, WinLoseReportsView,
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
				InventoryView, SalesView, SalesCreate,
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
				InventoryView, SalesView, PurchasesView, InventoryExpensesView,
				PlatformsView, WinLoseView, WinLoseReportsView,
			},
		},
	}
}
