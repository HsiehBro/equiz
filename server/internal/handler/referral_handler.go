package handler

import (
	"net/http"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	referralService *service.ReferralService
}

func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralService: referralService}
}

type BindReferralRequest struct {
	InviterID uint `json:"inviter_id" binding:"required"`
}

// BindReferral 锁定学员邀请归属关系
func (h *ReferralHandler) BindReferral(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req BindReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "邀请人ID不能为空"))
		return
	}

	if err := h.referralService.BindReferral(userID, req.InviterID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "邀请关系锁定成功",
	}))
}

// GetReferralRewards 获取当前学员的活动累计获赠时长与成功邀请好友明细
func (h *ReferralHandler) GetReferralRewards(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	stats, err := h.referralService.GetReferralStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

type VIPPurchaseRequest struct {
	PlanID string `json:"plan_id" binding:"required"` // 'monthly' | 'quarterly' | 'yearly' | 'lifetime'
}

// PurchaseVIP 开通 VIP 并触发邀请发奖与系统通知
func (h *ReferralHandler) PurchaseVIP(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req VIPPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "套餐类型不能为空"))
		return
	}

	res, err := h.referralService.ProcessVIPPurchase(userID, req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(res))
}
