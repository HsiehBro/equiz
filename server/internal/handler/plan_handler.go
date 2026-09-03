package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type PlanHandler struct {
	planService *service.PlanService
}

func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{planService: planService}
}

// SavePlan 创建或更新学习规划
func (h *PlanHandler) SavePlan(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req model.SavePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.planService.SavePlan(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(resp))
}

// ListPlans 获取当前登录用户的所有学习规划
func (h *PlanHandler) ListPlans(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	plans, err := h.planService.ListPlans(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(plans))
}

// GetActivePlan 获取当前激活主规划（首页和个人中心使用）
func (h *PlanHandler) GetActivePlan(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	plan, err := h.planService.GetActivePlan(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	if plan == nil {
		c.JSON(http.StatusOK, model.SuccessResponse(nil))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(plan))
}

// DeletePlan 删除指定学习规划
func (h *PlanHandler) DeletePlan(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的规划ID"))
		return
	}

	if err := h.planService.DeletePlan(userID, uint(planID)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true, "id": planID}))
}
