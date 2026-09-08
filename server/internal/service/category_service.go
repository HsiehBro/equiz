package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type CategoryService struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

// ListCategories 获取所有可用分类，并动态关联题库数量与 VIP 标识
func (s *CategoryService) ListCategories(userID uint, userRole string) ([]model.Category, error) {
	var categories []model.Category
	query := s.db.Order("sort_order ASC, id ASC")
	if userRole != "admin" {
		query = query.Where("is_active = ?", true)
	}
	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}

	// 查询当前用户可见的所有题库，统计各分类的题库数量与 VIP 状态
	var banks []model.QuestionBank
	var bankQuery *gorm.DB

	if userRole == "admin" {
		bankQuery = s.db.Where("is_official = ? OR visibility = 'public' OR creator_id = ?", true, userID)
	} else if userID > 0 {
		bankQuery = s.db.Where(
			"is_official = ? OR (visibility = 'public' AND review_status = 'approved') OR creator_id = ?",
			true, userID,
		)
	} else {
		bankQuery = s.db.Where("is_official = ? OR (visibility = 'public' AND review_status = 'approved')", true)
	}

	_ = bankQuery.Select("id, category_id, category, is_vip").Find(&banks).Error

	// 统计题库数与 VIP
	countMap := make(map[uint]int)
	vipMap := make(map[uint]bool)

	for _, b := range banks {
		catID := b.CategoryID
		if catID == 0 {
			catID = 1 // 兜底归入综合
		}
		countMap[catID]++
		if b.IsVIP {
			vipMap[catID] = true
		}
	}

	for i := range categories {
		c := &categories[i]
		c.BankCount = countMap[c.ID]
		c.HasVIP = vipMap[c.ID]
		c.IsVip = vipMap[c.ID]
	}

	return categories, nil
}

// CreateCategory 创建新分类（仅管理员）
func (s *CategoryService) CreateCategory(name, desc, icon, bgClass string, sortOrder int) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("分类名称不能为空")
	}

	var count int64
	if err := s.db.Model(&model.Category{}).Where("LOWER(TRIM(name)) = LOWER(TRIM(?))", name).Count(&count).Error; err == nil && count > 0 {
		return nil, fmt.Errorf("分类名称「%s」已存在，请勿重复添加", name)
	}

	if icon == "" {
		icon = "/assets/icons/menu_book_primary.svg"
	}
	if bgClass == "" {
		bgClass = "bg-blue-light"
	}

	category := model.Category{
		Name:        name,
		Description: desc,
		Icon:        icon,
		BgClass:     bgClass,
		SortOrder:   sortOrder,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(&category).Error; err != nil {
		return nil, fmt.Errorf("创建分类失败: %w", err)
	}

	return &category, nil
}

// UpdateCategory 更新分类（仅管理员）
func (s *CategoryService) UpdateCategory(id uint, name, desc, icon, bgClass string, sortOrder int, isActive *bool) (*model.Category, error) {
	var category model.Category
	if err := s.db.First(&category, id).Error; err != nil {
		return nil, errors.New("分类不存在")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("分类名称不能为空")
	}

	// 默认「综合」分类不允许修改名称
	if category.ID == 1 && name != category.Name {
		return nil, errors.New("默认「综合」分类名称不可修改")
	}

	// 检查重名（排查非自身的同名分类）
	var count int64
	if err := s.db.Model(&model.Category{}).
		Where("id != ? AND LOWER(TRIM(name)) = LOWER(TRIM(?))", id, name).
		Count(&count).Error; err == nil && count > 0 {
		return nil, fmt.Errorf("分类名称「%s」已被其他分类使用", name)
	}

	oldName := category.Name
	category.Name = name
	category.Description = desc
	if icon != "" {
		category.Icon = icon
	}
	if bgClass != "" {
		category.BgClass = bgClass
	}
	category.SortOrder = sortOrder
	if isActive != nil {
		category.IsActive = *isActive
	}
	category.UpdatedAt = time.Now()

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&category).Error; err != nil {
			return err
		}
		// 若名称发生变化，同步更新关联题库的 category 冗余字段
		if oldName != name {
			if err := tx.Model(&model.QuestionBank{}).
				Where("category_id = ?", id).
				Update("category", name).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("更新分类失败: %w", err)
	}

	return &category, nil
}

// DeleteCategory 删除分类（仅管理员）：关联的题库自动归入系统默认「综合」分类
func (s *CategoryService) DeleteCategory(id uint) error {
	if id == 1 {
		return errors.New("默认「综合」分类为系统核心基础分类，不允许删除")
	}

	var category model.Category
	if err := s.db.First(&category, id).Error; err != nil {
		return errors.New("分类不存在")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 将关联题库安全迁移至「综合」分类 (ID: 1)
		if err := tx.Model(&model.QuestionBank{}).
			Where("category_id = ?", id).
			Updates(map[string]interface{}{
				"category_id": 1,
				"category":    "综合",
			}).Error; err != nil {
			return fmt.Errorf("迁移题库到综合分类失败: %w", err)
		}

		// 2. 删除该分类
		if err := tx.Delete(&category).Error; err != nil {
			return fmt.Errorf("删除分类失败: %w", err)
		}

		return nil
	})
}
