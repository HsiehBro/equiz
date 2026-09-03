package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type FavoriteHandler struct {
	favService *service.FavoriteService
}

func NewFavoriteHandler(favService *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{favService: favService}
}

type ToggleFavoriteRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
}

func (h *FavoriteHandler) ToggleFavorite(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}

	var req ToggleFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "试题ID不能为空"))
		return
	}

	isBookmarked, err := h.favService.ToggleFavorite(userID, req.QuestionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"question_id":   req.QuestionID,
		"is_bookmarked": isBookmarked,
	}))
}

func (h *FavoriteHandler) ListFavorites(c *gin.Context) {
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

	favorites, err := h.favService.ListFavorites(userID, bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(favorites))
}
