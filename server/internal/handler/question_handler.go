package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type QuestionHandler struct {
	questionService *service.QuestionService
}

func NewQuestionHandler(questionService *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionService: questionService}
}

func (h *QuestionHandler) GetQuestionsByBank(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	var userID uint
	if uid, err := middleware.GetCurrentUserID(c); err == nil {
		userID = uid
	}

	questions, err := h.questionService.GetQuestionsByBankID(uint(bankID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(questions))
}

func (h *QuestionHandler) GetQuestion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题目ID"))
		return
	}

	var userID uint
	if uid, err := middleware.GetCurrentUserID(c); err == nil {
		userID = uid
	}

	q, err := h.questionService.GetQuestionByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(404, "试题不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(q))
}
