package router

import (
	"net/http"

	"exam-server/internal/config"
	"exam-server/internal/handler"
	"exam-server/internal/middleware"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 全局中间件
	r.Use(middleware.CORSMiddleware())

	// 基础探活接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "exam-server",
		})
	})

	// 初始化服务层与控制层
	authService := service.NewAuthService(db, cfg)
	bankService := service.NewBankService(db)
	questionService := service.NewQuestionService(db)
	practiceService := service.NewPracticeService(db)
	errorService := service.NewErrorService(db)
	favoriteService := service.NewFavoriteService(db)
	importService := service.NewImportService(db)
	mockService := service.NewMockService(db)
	wechatSecService := service.NewWeChatSecurityService(cfg)
	noteService := service.NewNoteService(db, wechatSecService)
	planService := service.NewPlanService(db)
	approvalService := service.NewApprovalService(db)
	notificationService := service.NewNotificationService(db)

	authHandler := handler.NewAuthHandler(authService)
	bankHandler := handler.NewBankHandler(bankService)
	questionHandler := handler.NewQuestionHandler(questionService, bankService)
	practiceHandler := handler.NewPracticeHandler(practiceService)
	errorHandler := handler.NewErrorHandler(errorService)
	favoriteHandler := handler.NewFavoriteHandler(favoriteService)
	importHandler := handler.NewImportHandler(importService)
	mockHandler := handler.NewMockHandler(mockService)
	noteHandler := handler.NewNoteHandler(noteService)
	planHandler := handler.NewPlanHandler(planService)
	adminHandler := handler.NewAdminHandler(approvalService, bankService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	referralService := service.NewReferralService(db)
	referralHandler := handler.NewReferralHandler(referralService)
	categoryService := service.NewCategoryService(db)
	categoryHandler := handler.NewCategoryHandler(categoryService, bankService)

	jwtAuth := middleware.JWTAuthMiddleware(cfg.JWT.Secret)
	optionalAuth := middleware.OptionalJWTAuthMiddleware(cfg.JWT.Secret)
	adminAuth := middleware.AdminRequiredMiddleware()

	// API 路由组
	apiV1 := r.Group("/api/v1")
	{
		// 1. 认证模块
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/mock-login", authHandler.MockLogin)
			authGroup.POST("/wechat-login", authHandler.WeChatLogin)
			authGroup.GET("/profile", jwtAuth, authHandler.GetProfile)
			authGroup.PUT("/profile", jwtAuth, authHandler.UpdateProfile)
		}

		// 2. 题库模块 (可附带 Token 以获取用户进度并隔离私有题库)
		bankGroup := apiV1.Group("/banks")
		{
			bankGroup.GET("", optionalAuth, bankHandler.ListBanks)
			bankGroup.GET("/:id", bankHandler.GetBank)
			bankGroup.DELETE("/:id", jwtAuth, bankHandler.DeleteBank)
			bankGroup.POST("/:id/apply-public", jwtAuth, bankHandler.ApplyPublic)
			bankGroup.PUT("/:id/category", jwtAuth, categoryHandler.UpdateBankCategory)
			bankGroup.GET("/:id/questions", optionalAuth, questionHandler.GetQuestionsByBank)
		}

		// 2.1 分类模块 (公开列表 + 管理员维护)
		categoryGroup := apiV1.Group("/categories")
		{
			categoryGroup.GET("", optionalAuth, categoryHandler.ListCategories)
			categoryGroup.POST("", jwtAuth, adminAuth, categoryHandler.CreateCategory)
			categoryGroup.PUT("/:id", jwtAuth, adminAuth, categoryHandler.UpdateCategory)
			categoryGroup.DELETE("/:id", jwtAuth, adminAuth, categoryHandler.DeleteCategory)
		}

		// 3. 试题详情与公开评论
		apiV1.GET("/questions/:id", questionHandler.GetQuestion)
		apiV1.GET("/questions/:id/comments", optionalAuth, noteHandler.ListQuestionComments)

		// 4. 刷题与判题 (需登录)
		practiceGroup := apiV1.Group("/practice")
		practiceGroup.Use(jwtAuth)
		{
			practiceGroup.POST("/submit-single", practiceHandler.SubmitSingle)
			practiceGroup.POST("/submit-exam", practiceHandler.SubmitExam)
			practiceGroup.GET("/stats", practiceHandler.GetUserStats)
		}

		// 5. 错题集 (需登录)
		errorGroup := apiV1.Group("/errors")
		errorGroup.Use(jwtAuth)
		{
			errorGroup.GET("", errorHandler.ListErrors)
			errorGroup.GET("/export-pdf", errorHandler.ExportErrorsPDF)
			errorGroup.POST("/:questionId/master", errorHandler.MarkMastered)
		}

		// 6. 收藏夹 (需登录)
		favGroup := apiV1.Group("/favorites")
		favGroup.Use(jwtAuth)
		{
			favGroup.GET("", favoriteHandler.ListFavorites)
			favGroup.POST("/toggle", favoriteHandler.ToggleFavorite)
		}

		// 7. Excel/CSV 导入 (通过 jwtAuth 绑定创建者 CreatorID，确保私有题库归属明确)
		importGroup := apiV1.Group("/import")
		{
			importGroup.GET("/template", importHandler.DownloadTemplate)
			importGroup.POST("/preview", optionalAuth, importHandler.PreviewExcel)
			importGroup.POST("/confirm", jwtAuth, importHandler.ConfirmImport)
		}

		// 8. 模拟考场与诊断模块 (需登录)
		mockGroup := apiV1.Group("/mock")
		mockGroup.Use(jwtAuth)
		{
			mockGroup.POST("/start", mockHandler.StartMock)
			mockGroup.POST("/submit", mockHandler.SubmitMock)
			mockGroup.GET("/records", mockHandler.ListRecords)
			mockGroup.GET("/records/:id", mockHandler.GetRecord)
		}

		// 9. 题目笔记与评论模块 (需登录)
		noteGroup := apiV1.Group("/notes")
		noteGroup.Use(jwtAuth)
		{
			noteGroup.GET("", noteHandler.ListUserNotes)
			noteGroup.POST("", noteHandler.CreateNote)
			noteGroup.PUT("/:id", noteHandler.UpdateNote)
			noteGroup.DELETE("/:id", noteHandler.DeleteNote)
			noteGroup.POST("/:id/like", noteHandler.ToggleLike)
		}

		// 10. 学习规划与目标模块 (需登录)
		planGroup := apiV1.Group("/plans")
		planGroup.Use(jwtAuth)
		{
			planGroup.GET("", planHandler.ListPlans)
			planGroup.GET("/active", planHandler.GetActivePlan)
			planGroup.POST("", planHandler.SavePlan)
			planGroup.DELETE("/:id", planHandler.DeletePlan)
		}

		// 11. 管理员专属模块 (手动审批栏、设置VIP题库等)
		adminGroup := apiV1.Group("/admin")
		adminGroup.Use(jwtAuth, adminAuth)
		{
			adminGroup.GET("/approvals", adminHandler.ListPendingApprovals)
			adminGroup.POST("/approvals/:id/approve", adminHandler.ApproveBank)
			adminGroup.POST("/approvals/:id/reject", adminHandler.RejectBank)
			adminGroup.POST("/banks/:id/toggle-vip", adminHandler.ToggleVIPBank)
		}

		// 12. 系统与审批通知模块
		notifyGroup := apiV1.Group("/notifications")
		notifyGroup.Use(jwtAuth)
		{
			notifyGroup.GET("", notificationHandler.ListNotifications)
			notifyGroup.POST("/:id/read", notificationHandler.MarkAsRead)
			notifyGroup.DELETE("/:id", notificationHandler.DeleteNotification)
		}

		// 13. 会员充值开通模块
		vipGroup := apiV1.Group("/vip")
		vipGroup.Use(jwtAuth)
		{
			vipGroup.POST("/purchase", referralHandler.PurchaseVIP)
		}

		// 14. 裂变活动与邀请奖励模块
		activityGroup := apiV1.Group("/activity")
		activityGroup.Use(jwtAuth)
		{
			activityGroup.POST("/referral/bind", referralHandler.BindReferral)
			activityGroup.GET("/referral/rewards", referralHandler.GetReferralRewards)
		}
	}

	return r
}
