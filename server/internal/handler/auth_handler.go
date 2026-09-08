package handler

import (
	"net/http"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type MockLoginRequest struct {
	DevUserID string `json:"dev_user_id"`
	Role      string `json:"role"`     // 'admin' | 'vip' | 'user'
	Nickname  string `json:"nickname"`
}

func (h *AuthHandler) MockLogin(c *gin.Context) {
	// 生产环境安全防线：严格禁用 Mock 登录接口
	if gin.Mode() == gin.ReleaseMode {
		c.JSON(http.StatusForbidden, model.ErrorResponse(403, "生产环境已禁用 Mock 调试登录"))
		return
	}

	var req MockLoginRequest
	_ = c.ShouldBindJSON(&req)

	user, token, err := h.authService.MockLogin(req.DevUserID, req.Role, req.Nickname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"token": token,
		"user":  user,
	}))
}

type WeChatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

func (h *AuthHandler) WeChatLogin(c *gin.Context) {
	var req WeChatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "code 不能为空"))
		return
	}

	user, token, err := h.authService.WeChatLogin(req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"token": token,
		"user":  user,
	}))
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, err.Error()))
		return
	}

	user, err := h.authService.GetUserProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(404, "用户不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, err.Error()))
		return
	}

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误"))
		return
	}

	user, err := h.authService.UpdateProfile(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}
