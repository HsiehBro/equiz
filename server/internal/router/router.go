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
	noteService := service.NewNoteService(db)
	planService := service.NewPlanService(db)

	authHandler := handler.NewAuthHandler(authService)
	bankHandler := handler.NewBankHandler(bankService)
	questionHandler := handler.NewQuestionHandler(questionService)
	practiceHandler := handler.NewPracticeHandler(practiceService)
	errorHandler := handler.NewErrorHandler(errorService)
	favoriteHandler := handler.NewFavoriteHandler(favoriteService)
	importHandler := handler.NewImportHandler(importService)
	mockHandler := handler.NewMockHandler(mockService)
	noteHandler := handler.NewNoteHandler(noteService)
	planHandler := handler.NewPlanHandler(planService)

	jwtAuth := middleware.JWTAuthMiddleware(cfg.JWT.Secret)
	optionalAuth := middleware.OptionalJWTAuthMiddleware(cfg.JWT.Secret)

	// API 路由组
	apiV1 := r.Group("/api/v1")
	{
		// 1. 认证模块
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/mock-login", authHandler.MockLogin)
			authGroup.POST("/wechat-login", authHandler.WeChatLogin)
			authGroup.GET("/profile", jwtAuth, authHandler.GetProfile)
		}

		// 2. 题库模块 (可附带 Token 以获取用户进度并隔离私有题库)
		bankGroup := apiV1.Group("/banks")
		{
			bankGroup.GET("", optionalAuth, bankHandler.ListBanks)
			bankGroup.GET("/:id", bankHandler.GetBank)
			bankGroup.DELETE("/:id", jwtAuth, bankHandler.DeleteBank)
			bankGroup.GET("/:id/questions", questionHandler.GetQuestionsByBank)
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
		}

		// 5. 错题集 (需登录)
		errorGroup := apiV1.Group("/errors")
		errorGroup.Use(jwtAuth)
		{
			errorGroup.GET("", errorHandler.ListErrors)
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
	}

	return r
}
