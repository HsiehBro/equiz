package handler

import (
	"net/http"
	"strconv"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
	bankService     *service.BankService
}

func NewCategoryHandler(categoryService *service.CategoryService, bankService *service.BankService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		bankService:     bankService,
	}
}

// ListCategories 获取所有可用分类列表（支持游客与已登录用户，动态统计题库数与VIP状态）
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	var userID uint
	if uid, err := middleware.GetCurrentUserID(c); err == nil {
		userID = uid
	}
	role := middleware.GetCurrentUserRole(c)

	categories, err := h.categoryService.ListCategories(userID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(categories))
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	BgClass     string `json:"bg_class"`
	SortOrder   int    `json:"sort_order"`
}

// CreateCategory 创建分类（仅管理员）
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数错误：请提供有效的分类名称"))
		return
	}

	cat, err := h.categoryService.CreateCategory(req.Name, req.Description, req.Icon, req.BgClass, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(cat))
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	BgClass     string `json:"bg_class"`
	SortOrder   int    `json:"sort_order"`
	IsActive    *bool  `json:"is_active"`
}

// UpdateCategory 修改分类（仅管理员）
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的分类ID"))
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数错误：请提供有效的分类名称"))
		return
	}

	cat, err := h.categoryService.UpdateCategory(uint(id), req.Name, req.Description, req.Icon, req.BgClass, req.SortOrder, req.IsActive)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(cat))
}

// DeleteCategory 删除分类（仅管理员，关联题库自动归入综合分类）
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的分类ID"))
		return
	}

	if err := h.categoryService.DeleteCategory(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "分类删除成功，关联题库已安全归入默认「综合」分类",
	}))
}

type UpdateBankCategoryRequest struct {
	CategoryID uint `json:"category_id" binding:"required"`
}

// UpdateBankCategory 修改题库所属分类
func (h *CategoryHandler) UpdateBankCategory(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "无效的题库ID"))
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "请先登录"))
		return
	}
	userRole := middleware.GetCurrentUserRole(c)

	var req UpdateBankCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.CategoryID == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "请提供有效的目标分类ID (category_id)"))
		return
	}

	bank, err := h.bankService.UpdateBankCategory(uint(bankID), req.CategoryID, userID, userRole)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}
