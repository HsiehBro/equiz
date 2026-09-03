package service

import (
	"errors"
	"fmt"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type BankService struct {
	db *gorm.DB
}

func NewBankService(db *gorm.DB) *BankService {
	return &BankService{db: db}
}

// ListBanks 查询用户可见的题库（官方题库 + 公开题库 + 该用户自己导入的私有题库），并计算做题进度
// 私有题库隔离原则：私有题库只能看到自己上传的，看不到别人上传的；未登录用户无法查看任何私有题库。
func (s *BankService) ListBanks(userID uint) ([]model.QuestionBank, error) {
	var banks []model.QuestionBank

	var query *gorm.DB
	if userID > 0 {
		// 已登录用户：仅能看到官方题库、所有公开题库、以及本人创建的私有题库
		query = s.db.Where("is_official = ? OR visibility = 'public' OR (visibility = 'private' AND creator_id = ?)", true, userID).Order("id ASC")
	} else {
		// 未登录用户：只能看到官方题库和公开题库，绝不展示任何私有题库
		query = s.db.Where("is_official = ? OR visibility = 'public'", true).Order("id ASC")
	}

	if err := query.Find(&banks).Error; err != nil {
		return nil, fmt.Errorf("failed to query banks: %w", err)
	}

	// 统计每个题库下的做题进度
	for i := range banks {
		bank := &banks[i]
		if bank.TotalCount == 0 {
			// 动态统计题目数
			var count int64
			s.db.Model(&model.Question{}).Where("bank_id = ?", bank.ID).Count(&count)
			bank.TotalCount = int(count)
		}

		if userID > 0 && bank.TotalCount > 0 {
			// 查询已作答过的不同题目数
			var answeredCount int64
			s.db.Model(&model.UserRecord{}).
				Where("user_id = ? AND bank_id = ?", userID, bank.ID).
				Distinct("question_id").
				Count(&answeredCount)

			progress := int((float64(answeredCount) / float64(bank.TotalCount)) * 100)
			if progress > 100 {
				progress = 100
			}
			bank.Progress = progress

			// 最近一次作答时间
			var lastRecord model.UserRecord
			err := s.db.Where("user_id = ? AND bank_id = ?", userID, bank.ID).
				Order("created_at DESC").
				First(&lastRecord).Error
			if err == nil {
				bank.LastPractice = formatRelativeTime(lastRecord.CreatedAt)
			} else {
				bank.LastPractice = "未开始"
			}
		}
	}

	return banks, nil
}

func (s *BankService) GetBankByID(bankID uint) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return nil, err
	}
	return &bank, nil
}

func (s *BankService) DeleteBank(bankID uint, userID uint) error {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return err
	}

	if bank.IsOfficial {
		return errors.New("官方题库不允许删除")
	}

	if bank.Visibility == "public" {
		return errors.New("公开题库已面向全员开放，不可删除")
	}

	if bank.CreatorID != userID {
		return errors.New("无权删除非本人创建的题库")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除题库相关的作答记录、错题、收藏和试题
		if err := tx.Where("bank_id = ?", bankID).Delete(&model.Question{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bank_id = ?", bankID).Delete(&model.UserRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bank_id = ?", bankID).Delete(&model.UserError{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bank_id = ?", bankID).Delete(&model.UserFavorite{}).Error; err != nil {
			return err
		}
		return tx.Delete(&bank).Error
	})
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute*5 {
		return "刚刚"
	} else if diff < time.Hour {
		return fmt.Sprintf("%d分钟前", int(diff.Minutes()))
	} else if diff < time.Hour*24 {
		return fmt.Sprintf("%d小时前", int(diff.Hours()))
	} else if diff < time.Hour*48 {
		return "昨天"
	}
	return t.Format("2006-01-02")
}
