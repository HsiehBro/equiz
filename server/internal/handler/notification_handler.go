package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

// ListNotifications 查询通知列表
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}
	role := middleware.GetCurrentUserRole(c)
	isAdmin := role == "admin"

	list, err := h.notificationService.ListNotifications(userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(list))
}

// MarkAsRead 标记通知为已读
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}
	role := middleware.GetCurrentUserRole(c)
	isAdmin := role == "admin"

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的通知ID"))
		return
	}

	if err := h.notificationService.MarkAsRead(uint(id), userID, isAdmin); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse("标记已读成功"))
}

// DeleteNotification 删除通知
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}
	role := middleware.GetCurrentUserRole(c)
	isAdmin := role == "admin"

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的通知ID"))
		return
	}

	if err := h.notificationService.DeleteNotification(uint(id), userID, isAdmin); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true, "id": id}))
}
