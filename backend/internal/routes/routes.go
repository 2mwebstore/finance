package routes

import (
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/audit"
	"github.com/cambodia-finance/income-expense/internal/backup"
	"github.com/cambodia-finance/income-expense/internal/auth"
	"github.com/cambodia-finance/income-expense/internal/config"
	"github.com/cambodia-finance/income-expense/internal/controllers"
	"github.com/cambodia-finance/income-expense/internal/middleware"
	"github.com/cambodia-finance/income-expense/internal/notifications"
	"github.com/cambodia-finance/income-expense/internal/permissions"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/cambodia-finance/income-expense/internal/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(r *gin.Engine, db *gorm.DB, cfg *config.Config, store storage.Store, backups *backup.Service) {
	authSvc := auth.NewService(db, cfg)
	tokenSvc := auth.NewTokenService(cfg)
	auditSvc := audit.NewService(db)
	tg := notifications.NewTelegramNotifierFromEnv()
	authCtrl := controllers.NewAuthController(authSvc, cfg, auditSvc)

	branchSvc := services.NewBranchService(db)
	userSvc := services.NewUserService(db)
	roleSvc := services.NewRoleService(db)
	catSvc := services.NewCategoryService(db)
	pmSvc := services.NewPaymentMethodService(db)
	txnSvc := services.NewTransactionService(db, cfg.Timezone)
	reportSvc := services.NewReportService(db, cfg.Timezone)
	cashSvc := services.NewCashService(db)
	settingsSvc := services.NewSettingsService(db)

	branchTelegram := services.NewBranchTelegramService(db, cfg.Timezone)
	branchCtrl := controllers.NewBranchController(branchSvc, db, branchTelegram)
	branchTelegramCtrl := controllers.NewBranchTelegramController(branchTelegram, db, auditSvc)
	userCtrl := controllers.NewUserController(userSvc, db)
	roleCtrl := controllers.NewRoleController(roleSvc, db)
	catCtrl := controllers.NewCategoryController(catSvc, db)
	pmCtrl := controllers.NewPaymentMethodController(pmSvc)
	attachmentSvc := services.NewAttachmentService(db, txnSvc, store)
	txnCtrl := controllers.NewTransactionController(txnSvc, attachmentSvc, auditSvc, branchTelegram)
	invCtrl := controllers.NewInventoryController(services.NewInventoryService(db, cfg.Timezone, store), db, auditSvc)
	platformCtrl := controllers.NewPlatformController(services.NewPlatformService(db, cfg.Timezone), db, auditSvc)
	reportCtrl := controllers.NewReportController(reportSvc, auditSvc)
	auditCtrl := controllers.NewAuditController(auditSvc, db, cfg.Timezone)
	notifyCtrl := controllers.NewNotificationController(tg)
	settingsCtrl := controllers.NewSettingsController(settingsSvc, auditSvc)
	currencyCtrl := controllers.NewCurrencyController(services.NewCurrencyService(db), auditSvc)
	cashCtrl := controllers.NewCashController(cashSvc)
	backupCtrl := controllers.NewBackupController(backups, db, auditSvc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/setup", authCtrl.Setup)
			loginLimiter := middleware.NewRateLimiter(cfg.LoginRateLimit)
			authGroup.POST("/login", middleware.RateLimitLogin(loginLimiter), authCtrl.Login)
			authGroup.POST("/refresh", authCtrl.Refresh)
			protected := authGroup.Group("")
			protected.Use(middleware.AuthRequired(authSvc, tokenSvc))
			{
				protected.POST("/logout", authCtrl.Logout)
				protected.POST("/logout-all", authCtrl.LogoutAll)
				protected.GET("/me", authCtrl.Me)
				protected.POST("/change-password", authCtrl.ChangePassword)
			}
		}

		api := v1.Group("")
		api.Use(middleware.AuthRequired(authSvc, tokenSvc))
		{
			branches := api.Group("/branches")
			{
				branches.GET("", middleware.RequireAnyPermission(db, permissions.BranchesView, permissions.BranchesManage, permissions.UsersCreate, permissions.UsersManage), branchCtrl.List)
				branches.POST("", middleware.RequireAnyPermission(db, permissions.BranchesCreate, permissions.BranchesManage), branchCtrl.Create)
				branches.GET("/next-code", middleware.RequireAnyPermission(db, permissions.BranchesCreate, permissions.BranchesManage), branchCtrl.NextCode)
				branches.GET("/:id", middleware.RequireAnyPermission(db, permissions.BranchesView, permissions.BranchesManage), branchCtrl.Get)
				branches.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.BranchesEdit, permissions.BranchesManage), branchCtrl.Update)
				branches.DELETE("/:id", middleware.RequireAnyPermission(db, permissions.BranchesManage), branchCtrl.Delete)
				branches.POST("/:id/users", middleware.RequireAnyPermission(db, permissions.BranchesManage), branchCtrl.AssignUsers)
				// The branch's Telegram chat for transaction updates (only branches the user sees).
				branches.GET("/:id/telegram", middleware.RequireAnyPermission(db, permissions.BranchesEdit, permissions.BranchesManage), branchTelegramCtrl.Get)
				branches.PUT("/:id/telegram", middleware.RequireAnyPermission(db, permissions.BranchesEdit, permissions.BranchesManage), branchTelegramCtrl.Save)
				branches.POST("/:id/telegram/test", middleware.RequireAnyPermission(db, permissions.BranchesEdit, permissions.BranchesManage), branchTelegramCtrl.Test)
			}

			users := api.Group("/users")
			{
				users.GET("", middleware.RequireAnyPermission(db, permissions.UsersView, permissions.UsersManage), userCtrl.List)
				users.POST("", middleware.RequireAnyPermission(db, permissions.UsersCreate), userCtrl.Create)
				users.GET("/:id", middleware.RequireAnyPermission(db, permissions.UsersView, permissions.UsersManage), userCtrl.Get)
				users.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.UsersEdit, permissions.UsersManage), userCtrl.Update)
				users.PATCH("/:id/status", middleware.RequireAnyPermission(db, permissions.UsersManage), userCtrl.SetActive)
				users.POST("/:id/password", middleware.RequireAnyPermission(db, permissions.UsersManage), userCtrl.SetPassword)
				users.POST("/:id/roles", middleware.RequireAnyPermission(db, permissions.UsersManage), userCtrl.AssignRoles)
				users.POST("/:id/branches", middleware.RequireAnyPermission(db, permissions.UsersManage), userCtrl.AssignBranches)
				users.POST("/:id/managers", middleware.RequireAnyPermission(db, permissions.UsersManage), userCtrl.AssignManagers)
			}

			roles := api.Group("/roles")
			{
				roles.GET("", middleware.RequireAnyPermission(db, permissions.RolesView, permissions.RolesManage, permissions.UsersCreate, permissions.UsersManage), roleCtrl.List)
				// Roles are system-wide: without full access (owner/admin) the controller limits changes
				// to roles within the actor's own permissions and branches.
				roles.POST("", middleware.RequireAnyPermission(db, permissions.RolesManage), roleCtrl.Create)
				roles.GET("/:id", middleware.RequireAnyPermission(db, permissions.RolesView, permissions.RolesManage), roleCtrl.Get)
				roles.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.RolesManage), roleCtrl.Update)
				roles.PUT("/:id/permissions", middleware.RequireAnyPermission(db, permissions.PermissionsManage), roleCtrl.SetPermissions)
				roles.DELETE("/:id", middleware.RequireAnyPermission(db, permissions.RolesManage), roleCtrl.Delete)
			}
			// The catalog: the Permission Codes page (permissions.view) and the role editor (roles.*).
			api.GET("/permissions", middleware.RequireAnyPermission(db, permissions.PermissionsView, permissions.RolesView, permissions.RolesManage), roleCtrl.ListPermissions)

			cats := api.Group("/categories")
			{
				cats.GET("", middleware.RequireAnyPermission(db, permissions.DashboardView, permissions.IncomeView, permissions.ExpenseView, permissions.IncomeCreate, permissions.ExpenseCreate, permissions.CategoriesManage, permissions.InventoryManage, permissions.InventoryExpensesView, permissions.InventoryExpensesCreate), catCtrl.List)
				// Financial expense categories need categories.manage, inventory expense categories
				// inventory.manage (checked by type in the controller).
				cats.POST("", middleware.RequireAnyPermission(db, permissions.CategoriesManage, permissions.InventoryManage), catCtrl.Create)
				cats.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.CategoriesManage, permissions.InventoryManage), catCtrl.Update)
			}
			pms := api.Group("/payment-methods")
			{
				pms.GET("", middleware.RequireAnyPermission(db, permissions.DashboardView, permissions.IncomeView, permissions.ExpenseView, permissions.CategoriesManage, permissions.InventoryManage, permissions.SalesCreate, permissions.PurchasesCreate, permissions.InventoryExpensesCreate), pmCtrl.List)
				// Payment methods are shared by transactions, sales, purchases and expenses (managed
				// under Inventory → Payment Methods).
				pms.POST("", middleware.RequireAnyPermission(db, permissions.CategoriesManage, permissions.InventoryManage), pmCtrl.Create)
				pms.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.CategoriesManage, permissions.InventoryManage), pmCtrl.Update)
			}

			txns := api.Group("/transactions")
			{
				txns.GET("", middleware.RequireAnyPermission(db, permissions.TransactionsView, permissions.IncomeView, permissions.ExpenseView), txnCtrl.List)
				txns.POST("", middleware.RequireAnyPermission(db, permissions.IncomeCreate, permissions.ExpenseCreate), txnCtrl.Create)
				txns.GET("/totals", middleware.RequireAnyPermission(db, permissions.DashboardView, permissions.ReportsView, permissions.TransactionsView, permissions.IncomeView, permissions.ExpenseView), txnCtrl.Totals)
				txns.GET("/:id", middleware.RequireAnyPermission(db, permissions.TransactionsView, permissions.IncomeView, permissions.ExpenseView), txnCtrl.Get)
				txns.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.IncomeEdit, permissions.ExpenseEdit), txnCtrl.Update)
				txns.POST("/:id/void", middleware.RequireAnyPermission(db, permissions.IncomeVoid, permissions.ExpenseVoid), txnCtrl.Void)
				txns.POST("/:id/approve", middleware.RequireAnyPermission(db, permissions.TransactionsApprove), txnCtrl.Approve)
				txns.POST("/:id/reject", middleware.RequireAnyPermission(db, permissions.TransactionsReject), txnCtrl.Reject)
				txns.POST("/:id/reverse", middleware.RequireAnyPermission(db, permissions.TransactionsReverse), txnCtrl.Reverse)
				// Up to 2 images/PDFs per transaction (stored in R2, or local disk in development).
				txns.POST("/:id/attachments", middleware.RequireAnyPermission(db, permissions.IncomeCreate, permissions.ExpenseCreate, permissions.IncomeEdit, permissions.ExpenseEdit), txnCtrl.UploadAttachment)
				txns.GET("/:id/attachments/:attachmentId", middleware.RequireAnyPermission(db, permissions.TransactionsView, permissions.IncomeView, permissions.ExpenseView), txnCtrl.Attachment)
				txns.DELETE("/:id/attachments/:attachmentId", middleware.RequireAnyPermission(db, permissions.IncomeCreate, permissions.ExpenseCreate, permissions.IncomeEdit, permissions.ExpenseEdit), txnCtrl.DeleteAttachment)
			}

			// Phase 6: cash control
			ob := api.Group("/opening-balances")
			{
				ob.GET("", middleware.RequireAnyPermission(db, permissions.ReportsView, permissions.DashboardView), cashCtrl.ListOpeningBalances)
				ob.POST("", middleware.RequireAnyPermission(db, permissions.TransactionsApprove, permissions.SettingsManage), cashCtrl.CreateOpeningBalance)
			}
			cr := api.Group("/cash-reconciliations")
			{
				cr.GET("", middleware.RequireAnyPermission(db, permissions.ReportsView, permissions.DashboardView), cashCtrl.ListReconciliations)
				cr.POST("", middleware.RequireAnyPermission(db, permissions.TransactionsApprove, permissions.IncomeCreate), cashCtrl.CreateReconciliation)
			}

			// Phase 5: dashboard & reports
			api.GET("/dashboard/summary", middleware.RequireAnyPermission(db, permissions.DashboardView), reportCtrl.Dashboard)
			reports := api.Group("/reports")
			{
				reports.GET("/income", middleware.RequireAnyPermission(db, permissions.ReportsView), reportCtrl.Income)
				reports.GET("/expenses", middleware.RequireAnyPermission(db, permissions.ReportsView), reportCtrl.Expenses)
				reports.GET("/transactions", middleware.RequireAnyPermission(db, permissions.ReportsView), reportCtrl.Transactions)
				reports.GET("/daily", middleware.RequireAnyPermission(db, permissions.ReportsView), reportCtrl.Daily)
				reports.GET("/branches", middleware.RequireAnyPermission(db, permissions.ReportsView), reportCtrl.Branches)
				reports.GET("/export", middleware.RequireAnyPermission(db, permissions.ReportsExport), reportCtrl.Export)
			}

			// Inventory: shared products, stock per branch, sales, purchases, transfers (own branches only).
			inv := api.Group("/inventory")
			{
				need := func(codes ...string) gin.HandlerFunc { return middleware.RequireAnyPermission(db, codes...) }
				// Pickers on the sale/purchase/transfer forms need the catalog too.
				catalogReaders := need(permissions.InventoryView, permissions.InventoryManage, permissions.SalesCreate, permissions.PurchasesCreate, permissions.InventoryTransfer, permissions.InventoryAdjust)
				inv.GET("/categories", catalogReaders, invCtrl.ListCategories)
				inv.POST("/categories", need(permissions.InventoryManage), invCtrl.SaveCategory)
				inv.PATCH("/categories/:id", need(permissions.InventoryManage), invCtrl.SaveCategory)
				inv.GET("/products", catalogReaders, invCtrl.ListProducts)
				inv.POST("/products", need(permissions.InventoryManage), invCtrl.SaveProduct)
				inv.PATCH("/products/:id", need(permissions.InventoryManage), invCtrl.SaveProduct)
				inv.GET("/products/:id/image", catalogReaders, invCtrl.ProductImage)
				inv.POST("/products/:id/image", need(permissions.InventoryManage), invCtrl.UploadProductImage)
				inv.DELETE("/products/:id/image", need(permissions.InventoryManage), invCtrl.DeleteProductImage)

				inv.GET("/stock", need(permissions.InventoryView), invCtrl.ListStock)
				inv.GET("/movements", need(permissions.InventoryView), invCtrl.ListMovements)
				inv.POST("/adjustments", need(permissions.InventoryAdjust), invCtrl.Adjust)

				inv.GET("/sales", need(permissions.SalesView), invCtrl.ListSales)
				inv.GET("/sales/:id", need(permissions.SalesView), invCtrl.GetSale)
				inv.POST("/sales", need(permissions.SalesCreate), invCtrl.CreateSale)
				inv.POST("/sales/:id/void", need(permissions.SalesVoid), invCtrl.VoidSale)
				inv.PATCH("/sales/:id", need(permissions.SalesEdit), invCtrl.UpdateSale)

				inv.GET("/purchases", need(permissions.PurchasesView), invCtrl.ListPurchases)
				inv.GET("/purchases/:id", need(permissions.PurchasesView), invCtrl.GetPurchase)
				inv.POST("/purchases", need(permissions.PurchasesCreate), invCtrl.CreatePurchase)
				inv.POST("/purchases/:id/approve", need(permissions.PurchasesApprove), invCtrl.ApprovePurchase)
				inv.POST("/purchases/:id/reject", need(permissions.PurchasesApprove), invCtrl.RejectPurchase)
				inv.POST("/purchases/:id/void", need(permissions.PurchasesVoid), invCtrl.VoidPurchase)
				inv.PATCH("/purchases/:id", need(permissions.PurchasesEdit), invCtrl.UpdatePurchase)

				inv.GET("/transfers", need(permissions.InventoryView, permissions.InventoryTransfer), invCtrl.ListTransfers)
				inv.POST("/transfers", need(permissions.InventoryTransfer), invCtrl.CreateTransfer)
				inv.POST("/transfers/:id/void", need(permissions.InventoryTransfer), invCtrl.VoidTransfer)

				inv.GET("/expenses", need(permissions.InventoryExpensesView), invCtrl.ListExpenses)
				inv.POST("/expenses", need(permissions.InventoryExpensesCreate), invCtrl.CreateExpense)
				inv.POST("/expenses/:id/void", need(permissions.InventoryExpensesVoid), invCtrl.VoidExpense)

				// Reports show stock value and profit: they need reports.view and inventory.view.
				rep := inv.Group("/reports", need(permissions.ReportsView), need(permissions.InventoryView))
				rep.GET("/summary", invCtrl.ReportSummary)
				rep.GET("/stock", invCtrl.ReportStock)
				rep.GET("/sales", invCtrl.ReportSales)
				rep.GET("/purchases", invCtrl.ReportPurchases)
				rep.GET("/low-stock", invCtrl.ReportLowStock)
				rep.GET("/expenses", invCtrl.ReportExpenses)
			}

			// Platforms and weekly win/lose. An HQ is a branch: the service limits everything to the
			// actor's own branches; the permissions below decide what they may do there.
			{
				need := func(codes ...string) gin.HandlerFunc { return middleware.RequireAnyPermission(db, codes...) }
				api.GET("/platforms", need(permissions.PlatformsView, permissions.PlatformsManage, permissions.PlatformAssignmentsManage), platformCtrl.ListPlatforms)
				api.POST("/platforms", need(permissions.PlatformsManage), platformCtrl.SavePlatform)
				api.PATCH("/platforms/:id", need(permissions.PlatformsManage), platformCtrl.SavePlatform)

				// Partners are managed with the platforms; reports and the weekly entry read their names.
				api.GET("/partners", need(permissions.PlatformsView, permissions.PlatformsManage, permissions.PlatformAssignmentsManage, permissions.WinLoseReportsView), platformCtrl.ListPartners)
				api.POST("/partners", need(permissions.PlatformsManage), platformCtrl.SavePartner)
				api.PATCH("/partners/:id", need(permissions.PlatformsManage), platformCtrl.SavePartner)

				// Groups: titled sets of agents in the person's own branch (the Group Agent Assignments page).
				// Report viewers list them too (the reports filter by group).
				api.GET("/platform-groups", need(permissions.PlatformsView, permissions.PlatformAssignmentsManage, permissions.WinLoseReportsView), platformCtrl.ListGroups)
				api.GET("/platform-groups/:id", need(permissions.PlatformsView, permissions.PlatformAssignmentsManage), platformCtrl.GetGroup)
				api.POST("/platform-groups", need(permissions.PlatformAssignmentsManage), platformCtrl.CreateGroup)
				api.PATCH("/platform-groups/:id", need(permissions.PlatformAssignmentsManage), platformCtrl.UpdateGroup)
				api.POST("/platform-groups/:id/active", need(permissions.PlatformAssignmentsManage), platformCtrl.SetGroupActive)
				api.DELETE("/platform-groups/:id", need(permissions.PlatformAssignmentsManage), platformCtrl.DeleteGroup)
				api.GET("/platform-assignments", need(permissions.PlatformsView, permissions.PlatformAssignmentsManage), platformCtrl.ListAssignments)
				api.POST("/platform-assignments", need(permissions.PlatformAssignmentsManage), platformCtrl.Assign)
				api.PATCH("/platform-assignments/:id", need(permissions.PlatformAssignmentsManage), platformCtrl.UpdateAssignment)
				api.POST("/platform-assignments/:id/end", need(permissions.PlatformAssignmentsManage), platformCtrl.EndAssignment)
				api.POST("/platform-assignments/:id/move", need(permissions.PlatformAssignmentsManage), platformCtrl.MoveAssignment)

				wl := api.Group("/winlose")
				wl.GET("/formula", need(permissions.WinLoseView, permissions.WinLoseCreate, permissions.WinLoseEdit, permissions.WinLoseReportsView), platformCtrl.Formula)
				// New rows need winlose.create and changed saved rows winlose.edit (checked per row).
				wl.GET("/entries", need(permissions.WinLoseView, permissions.WinLoseCreate, permissions.WinLoseEdit), platformCtrl.ListEntries)
				// Deleting an entry removes saved amounts: Correct Win/Lose, with a reason.
				wl.DELETE("/entries/:id", need(permissions.WinLoseEdit), platformCtrl.DeleteEntry)
				// No branch_id: the person's own group (their default branch).
				wl.GET("/entry", need(permissions.WinLoseView, permissions.WinLoseCreate, permissions.WinLoseEdit), platformCtrl.WeekSheet)
				wl.POST("/entry/preview", need(permissions.WinLoseCreate, permissions.WinLoseEdit), platformCtrl.Preview)
				wl.PUT("/entry", need(permissions.WinLoseCreate, permissions.WinLoseEdit), platformCtrl.SaveWeek)
				// Totals across HQs also need winlose-reports.consolidated (checked in the service).
				wl.GET("/reports", need(permissions.WinLoseReportsView), platformCtrl.Report)
				wl.GET("/settlements", need(permissions.WinLoseReportsView, permissions.WinLoseSettlementsFinalize), platformCtrl.ListSettlements)
				wl.POST("/settlements", need(permissions.WinLoseSettlementsFinalize), platformCtrl.FinalizeWeek)
				wl.POST("/settlements/:id/reopen", need(permissions.WinLoseSettlementsFinalize), platformCtrl.ReopenWeek)
			}

			api.GET("/audit-logs", middleware.RequireAnyPermission(db, permissions.AuditLogsView), auditCtrl.List)
			// Retention cleanup is destructive: require audit access, system settings and full access (owner/admin/super admin).
			api.POST("/audit-logs/cleanup",
				middleware.RequireAnyPermission(db, permissions.AuditLogsView),
				middleware.RequireAnyPermission(db, permissions.SettingsManage),
				middleware.RequireFullAccess(db),
				auditCtrl.Cleanup)
			api.GET("/notifications/status", middleware.RequireAnyPermission(db, permissions.SettingsManage), notifyCtrl.Status)
			api.POST("/notifications/test", middleware.RequireAnyPermission(db, permissions.SettingsManage), notifyCtrl.Test)
			// Database backups: the whole database, so owner / Super Admin only (full access + backups.manage).
			bk := api.Group("/backups", middleware.RequireAnyPermission(db, permissions.BackupsManage), middleware.RequireFullAccess(db))
			{
				bk.GET("", backupCtrl.List)
				bk.POST("", backupCtrl.Create)
				bk.PUT("/settings", backupCtrl.UpdateSettings)
				bk.POST("/telegram-test", backupCtrl.TestTelegram)
				bk.POST("/upload", backupCtrl.Upload)
				bk.GET("/:name/download", backupCtrl.Download)
				bk.POST("/:name/telegram", backupCtrl.Resend)
				bk.POST("/:name/restore", backupCtrl.Restore)
				bk.DELETE("/:name", backupCtrl.Delete)
			}
			api.GET("/settings", middleware.RequireAnyPermission(db, permissions.SettingsManage, permissions.DashboardView), settingsCtrl.Get)
			// Exchange rate and default currency: every signed-in user reads them (forms convert
			// prices); changing them needs currency-settings.manage.
			api.GET("/currency", currencyCtrl.Get)
			api.PUT("/currency", middleware.RequireAnyPermission(db, permissions.CurrencySettingsManage), currencyCtrl.Update)
			api.PUT("/settings", middleware.RequireAnyPermission(db, permissions.SettingsManage), settingsCtrl.Update)
		}
	}
}
