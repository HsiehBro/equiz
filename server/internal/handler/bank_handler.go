package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type BankHandler struct {
	bankService *service.BankService
}

func NewBankHandler(bankService *service.BankService) *BankHandler {
	return &BankHandler{bankService: bankService}
}

func (h *BankHandler) ListBanks(c *gin.Context) {
	// 用户可能未登录（可选 Token）
	var userID uint
	if uid, err := middleware.GetCurrentUserID(c); err == nil {
		userID = uid
	}
	role := middleware.GetCurrentUserRole(c)

	banks, err := h.bankService.ListBanks(userID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(banks))
}

func (h *BankHandler) GetBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	bank, err := h.bankService.GetBankByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(404, "题库不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}

func (h *BankHandler) DeleteBank(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}
	role := middleware.GetCurrentUserRole(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	if err := h.bankService.DeleteBank(uint(id), userID, role); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse("删除成功"))
}

func (h *BankHandler) ApplyPublic(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	var req struct {
		UserName string `json:"user_name"`
	}
	_ = c.ShouldBindJSON(&req)

	bank, err := h.bankService.ApplyPublic(uint(id), userID, req.UserName)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}
