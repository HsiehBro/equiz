package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type ErrorHandler struct {
	errorService *service.ErrorService
}

func NewErrorHandler(errorService *service.ErrorService) *ErrorHandler {
	return &ErrorHandler{errorService: errorService}
}

func (h *ErrorHandler) ListErrors(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	bankIDStr := c.Query("bank_id")
	var bankID uint
	if bankIDStr != "" {
		if bid, err := strconv.ParseUint(bankIDStr, 10, 32); err == nil {
			bankID = uint(bid)
		}
	}

	includeMastered := c.Query("include_mastered") == "true"

	errorsList, err := h.errorService.ListUserErrors(userID, bankID, includeMastered)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(errorsList))
}

func (h *ErrorHandler) MarkMastered(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	qidStr := c.Param("questionId")
	qid, err := strconv.ParseUint(qidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的试题ID"))
		return
	}

	if err := h.errorService.MarkMastered(userID, uint(qid)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse("已移出错题集"))
}
