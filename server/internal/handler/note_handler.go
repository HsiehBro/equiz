package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type NoteHandler struct {
	noteService *service.NoteService
}

func NewNoteHandler(noteService *service.NoteService) *NoteHandler {
	return &NoteHandler{noteService: noteService}
}

// CreateNote 创建个人笔记或公开评论
func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req model.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	note, err := h.noteService.CreateNote(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(note))
}

// UpdateNote 修改个人笔记
func (h *NoteHandler) UpdateNote(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	noteIDStr := c.Param("id")
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的笔记ID"))
		return
	}

	var req model.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	note, err := h.noteService.UpdateNote(userID, uint(noteID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(note))
}

// DeleteNote 删除个人笔记
func (h *NoteHandler) DeleteNote(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	noteIDStr := c.Param("id")
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的笔记ID"))
		return
	}

	if err := h.noteService.DeleteNote(userID, uint(noteID)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true, "id": noteID}))
}

// ListUserNotes 获取当前登录用户的笔记列表
func (h *NoteHandler) ListUserNotes(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var bankID uint
	if bStr := c.Query("bank_id"); bStr != "" {
		if bid, err := strconv.ParseUint(bStr, 10, 32); err == nil {
			bankID = uint(bid)
		}
	}

	visibility := c.Query("visibility")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	notes, total, err := h.noteService.ListUserNotes(userID, bankID, visibility, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"list":      notes,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// ListQuestionComments 获取指定题目的公开评论（支持游客访问，已登录时附带点赞态）
func (h *NoteHandler) ListQuestionComments(c *gin.Context) {
	qIDStr := c.Param("id")
	questionID, err := strconv.ParseUint(qIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的试题ID"))
		return
	}

	currentUserID, _ := middleware.GetCurrentUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	comments, total, err := h.noteService.ListQuestionComments(uint(questionID), currentUserID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"list":      comments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// ToggleLike 点赞或取消点赞
func (h *NoteHandler) ToggleLike(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	noteIDStr := c.Param("id")
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的笔记ID"))
		return
	}

	liked, newCount, err := h.noteService.ToggleNoteLike(userID, uint(noteID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"note_id":    noteID,
		"is_liked":   liked,
		"like_count": newCount,
	}))
}
