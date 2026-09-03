package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type MockHandler struct {
	mockService *service.MockService
}

func NewMockHandler(mockService *service.MockService) *MockHandler {
	return &MockHandler{mockService: mockService}
}

// StartMock 生成模考试卷 (按考纲百分比随机抽取试题，强校验拦截题量不足)
func (h *MockHandler) StartMock(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req service.StartMockExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	paper, err := h.mockService.StartMockExam(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(paper))
}

// SubmitMock 提交全卷作答并完成大纲维度多维诊断与成绩记录
func (h *MockHandler) SubmitMock(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req service.SubmitMockExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	record, err := h.mockService.SubmitMockExam(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(record))
}

// ListRecords 查询当前用户的模考历史记录列表
func (h *MockHandler) ListRecords(c *gin.Context) {
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

	records, err := h.mockService.ListMockRecords(userID, bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(records))
}

// GetRecord 获取指定模考记录的完整成绩单与逐题解析快照
func (h *MockHandler) GetRecord(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的记录ID"))
		return
	}

	record, err := h.mockService.GetMockRecordByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(404, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(record))
}
