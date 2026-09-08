package handler

import (
	"net/http"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type PracticeHandler struct {
	practiceService *service.PracticeService
}

func NewPracticeHandler(practiceService *service.PracticeService) *PracticeHandler {
	return &PracticeHandler{practiceService: practiceService}
}

func (h *PracticeHandler) SubmitSingle(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req service.SubmitSingleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	result, err := h.practiceService.SubmitSingleAnswer(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

func (h *PracticeHandler) SubmitExam(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req service.SubmitExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	result, err := h.practiceService.SubmitExam(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

func (h *PracticeHandler) GetUserStats(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	result, err := h.practiceService.GetUserStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

