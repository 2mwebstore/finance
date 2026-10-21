package routes

import (
	"log"
	"net/http"

	"github.com/cambodia-finance/income-expense/internal/audit"
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

func Setup(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
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

	branchCtrl := controllers.NewBranchController(branchSvc, db)
	userCtrl := controllers.NewUserController(userSvc, db)
	roleCtrl := controllers.NewRoleController(roleSvc)
	catCtrl := controllers.NewCategoryController(catSvc, db)
	pmCtrl := controllers.NewPaymentMethodController(pmSvc)
	store, err := storage.NewFromEnv()
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	log.Printf("attachments: using %s storage", store.Kind())
	attachmentSvc := services.NewAttachmentService(db, txnSvc, store)
	txnCtrl := controllers.NewTransactionController(txnSvc, attachmentSvc, auditSvc)
	reportCtrl := controllers.NewReportController(reportSvc, auditSvc)
	auditCtrl := controllers.NewAuditController(auditSvc, db, cfg.Timezone)
	notifyCtrl := controllers.NewNotificationController(tg)
	settingsCtrl := controllers.NewSettingsController(settingsSvc, auditSvc)
	cashCtrl := controllers.NewCashController(cashSvc)

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
				branches.GET("/:id", middleware.RequireAnyPermission(db, permissions.BranchesView, permissions.BranchesManage), branchCtrl.Get)
				branches.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.BranchesEdit, permissions.BranchesManage), branchCtrl.Update)
				branches.DELETE("/:id", middleware.RequireAnyPermission(db, permissions.BranchesManage), branchCtrl.Delete)
				branches.POST("/:id/users", middleware.RequireAnyPermission(db, permissions.BranchesManage), branchCtrl.AssignUsers)
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
				// Roles and permissions are system-wide: only the owner/admin (full access) may change them.
				roles.POST("", middleware.RequireAnyPermission(db, permissions.RolesManage), middleware.RequireFullAccess(db), roleCtrl.Create)
				roles.GET("/:id", middleware.RequireAnyPermission(db, permissions.RolesView, permissions.RolesManage), roleCtrl.Get)
				roles.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.RolesManage), middleware.RequireFullAccess(db), roleCtrl.Update)
				roles.PUT("/:id/permissions", middleware.RequireAnyPermission(db, permissions.PermissionsManage), middleware.RequireFullAccess(db), roleCtrl.SetPermissions)
				roles.DELETE("/:id", middleware.RequireAnyPermission(db, permissions.RolesManage), middleware.RequireFullAccess(db), roleCtrl.Delete)
			}
			api.GET("/permissions", middleware.RequireAnyPermission(db, permissions.RolesView, permissions.RolesManage), roleCtrl.ListPermissions)

			cats := api.Group("/categories")
			{
				cats.GET("", middleware.RequireAnyPermission(db, permissions.DashboardView, permissions.IncomeView, permissions.ExpenseView, permissions.IncomeCreate, permissions.ExpenseCreate, permissions.CategoriesManage), catCtrl.List)
				cats.POST("", middleware.RequireAnyPermission(db, permissions.CategoriesManage), catCtrl.Create)
				cats.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.CategoriesManage), catCtrl.Update)
			}
			pms := api.Group("/payment-methods")
			{
				pms.GET("", middleware.RequireAnyPermission(db, permissions.DashboardView, permissions.IncomeView, permissions.ExpenseView, permissions.CategoriesManage), pmCtrl.List)
				pms.POST("", middleware.RequireAnyPermission(db, permissions.CategoriesManage), pmCtrl.Create)
				pms.PATCH("/:id", middleware.RequireAnyPermission(db, permissions.CategoriesManage), pmCtrl.Update)
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
				// One image/PDF per transaction (stored in R2, or local disk in development).
				txns.GET("/:id/attachment", middleware.RequireAnyPermission(db, permissions.TransactionsView, permissions.IncomeView, permissions.ExpenseView), txnCtrl.Attachment)
				txns.POST("/:id/attachment", middleware.RequireAnyPermission(db, permissions.IncomeCreate, permissions.ExpenseCreate, permissions.IncomeEdit, permissions.ExpenseEdit), txnCtrl.UploadAttachment)
				txns.DELETE("/:id/attachment", middleware.RequireAnyPermission(db, permissions.IncomeCreate, permissions.ExpenseCreate, permissions.IncomeEdit, permissions.ExpenseEdit), txnCtrl.DeleteAttachment)
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

			api.GET("/audit-logs", middleware.RequireAnyPermission(db, permissions.AuditLogsView), auditCtrl.List)
			// Retention cleanup is destructive: require audit access, system settings and full access (owner/admin/super admin).
			api.POST("/audit-logs/cleanup",
				middleware.RequireAnyPermission(db, permissions.AuditLogsView),
				middleware.RequireAnyPermission(db, permissions.SettingsManage),
				middleware.RequireFullAccess(db),
				auditCtrl.Cleanup)
			api.GET("/notifications/status", middleware.RequireAnyPermission(db, permissions.SettingsManage), notifyCtrl.Status)
			api.POST("/notifications/test", middleware.RequireAnyPermission(db, permissions.SettingsManage), notifyCtrl.Test)
			api.GET("/settings", middleware.RequireAnyPermission(db, permissions.SettingsManage, permissions.DashboardView), settingsCtrl.Get)
			api.PUT("/settings", middleware.RequireAnyPermission(db, permissions.SettingsManage), settingsCtrl.Update)
		}
	}
}
