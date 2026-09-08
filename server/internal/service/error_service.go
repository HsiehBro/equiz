package service

import (
	"errors"
	"fmt"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type ErrorService struct {
	db *gorm.DB
}

func NewErrorService(db *gorm.DB) *ErrorService {
	return &ErrorService{db: db}
}

// ListUserErrors 获取用户的错题列表，默认只返回未掌握的 (is_mastered = false)
func (s *ErrorService) ListUserErrors(userID uint, bankID uint, includeMastered bool) ([]model.UserError, error) {
	var userErrors []model.UserError

	query := s.db.Where("user_id = ?", userID)
	if bankID > 0 {
		query = query.Where("bank_id = ?", bankID)
	}
	if !includeMastered {
		query = query.Where("is_mastered = ?", false)
	}

	// 预加载试题详情并按最后答错时间倒序
	err := query.Preload("Question").Order("last_wrong_at DESC").Find(&userErrors).Error
	if err != nil {
		return nil, err
	}

	return userErrors, nil
}

// MarkMastered 将错题标记为已掌握（移出错题集）
func (s *ErrorService) MarkMastered(userID uint, questionID uint) error {
	res := s.db.Model(&model.UserError{}).
		Where("user_id = ? AND question_id = ?", userID, questionID).
		Update("is_mastered", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("错题记录不存在")
	}
	return nil
}

// ExportErrorsPDF 导出指定科目错题集的 PDF（背题模式）
func (s *ErrorService) ExportErrorsPDF(userID uint, bankID uint) ([]byte, string, error) {
	var bank model.QuestionBank
	bankName := "错题集"
	if bankID > 0 {
		if err := s.db.First(&bank, bankID).Error; err != nil {
			return nil, "", fmt.Errorf("题库不存在: %w", err)
		}
		if bank.Title != "" {
			bankName = bank.Title
		}
	}

	// 查出该科目该用户未掌握的错题
	errorsList, err := s.ListUserErrors(userID, bankID, false)
	if err != nil {
		return nil, "", fmt.Errorf("获取错题列表失败: %w", err)
	}

	if len(errorsList) == 0 {
		return nil, "", errors.New("当前科目暂无未掌握的错题，无需导出")
	}

	pdfBytes, err := GenerateErrorsPDF(bankName, errorsList)
	if err != nil {
		return nil, "", fmt.Errorf("生成 PDF 失败: %w", err)
	}

	return pdfBytes, bankName, nil
}

