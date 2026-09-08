package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	approvalService *service.ApprovalService
	bankService     *service.BankService
}

func NewAdminHandler(approvalService *service.ApprovalService, bankService *service.BankService) *AdminHandler {
	return &AdminHandler{
		approvalService: approvalService,
		bankService:     bankService,
	}
}

// ListPendingApprovals 获取待审批公开题库列表
func (h *AdminHandler) ListPendingApprovals(c *gin.Context) {
	banks, err := h.approvalService.ListPendingBanks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, model.SuccessResponse(banks))
}

// ApproveBank 同意公开题库
func (h *AdminHandler) ApproveBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	bank, err := h.approvalService.ApproveBank(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}

// RejectBank 驳回公开题库申请：直接删除题库（防止绕过私有配额限制）
func (h *AdminHandler) RejectBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	bank, err := h.approvalService.RejectBank(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}

// ToggleVIPBank 切换题库 VIP 标识
func (h *AdminHandler) ToggleVIPBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	bank, err := h.bankService.ToggleVIPBank(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}
