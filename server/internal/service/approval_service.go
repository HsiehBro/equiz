package service

import (
	"errors"
	"fmt"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type ApprovalService struct {
	db *gorm.DB
}

func NewApprovalService(db *gorm.DB) *ApprovalService {
	return &ApprovalService{db: db}
}

// ListPendingBanks 获取待审核公开题库列表（仅管理员可调用）
func (s *ApprovalService) ListPendingBanks() ([]model.QuestionBank, error) {
	var banks []model.QuestionBank
	err := s.db.Where("visibility = 'public' AND review_status = 'pending'").
		Order("created_at DESC").
		Find(&banks).Error
	if err != nil {
		return nil, fmt.Errorf("查询待审核题库失败: %w", err)
	}
	return banks, nil
}

// ApproveBank 审核通过：题库正式面向全员公开，并向上传者发送消息中心通知
func (s *ApprovalService) ApproveBank(bankID uint) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return nil, err
	}

	if bank.ReviewStatus == "approved" {
		return &bank, nil
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		bank.ReviewStatus = "approved"
		bank.Visibility = "public"
		bank.UpdatedAt = time.Now()
		if err := tx.Save(&bank).Error; err != nil {
			return err
		}

		// 向创建者发送审批通过通知
		if bank.CreatorID > 0 {
			userNotice := model.SystemNotification{
				UserID:    bank.CreatorID,
				Type:      "approval_pass",
				Title:     "【题库审核通过】公开申请已批准",
				Content:   fmt.Sprintf("恭喜！您上传的公开题库《%s》已通过管理员审核，现已正式面向全员公开！", bank.Title),
				RelatedID: bank.ID,
				IsRead:    false,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&userNotice).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &bank, nil
}

// RejectBank 审核驳回：直接删除题库（不保留为私有，防止普通用户借此绕过私有题库配额限制），并向上传者发送通知
func (s *ApprovalService) RejectBank(bankID uint) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除该题库关联的所有试题
		if err := tx.Where("bank_id = ?", bank.ID).Delete(&model.Question{}).Error; err != nil {
			return err
		}

		// 2. 彻底删除该题库记录
		if err := tx.Delete(&bank).Error; err != nil {
			return err
		}

		// 3. 向创建者发送审核驳回并删除的通知
		if bank.CreatorID > 0 {
			userNotice := model.SystemNotification{
				UserID:    bank.CreatorID,
				Type:      "approval_reject",
				Title:     "【题库审核未通过】公开申请已驳回并删除",
				Content:   fmt.Sprintf("您申请公开的题库《%s》未通过管理员审核。根据配额防绕过规则，该题库已被直接删除（不保留为私有题库）。", bank.Title),
				RelatedID: bank.ID,
				IsRead:    false,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&userNotice).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &bank, nil
}

// CheckSinglePublicPendingLimit 检查用户是否已有处于待审批状态的公开题库
func (s *ApprovalService) CheckSinglePublicPendingLimit(userID uint) error {
	var count int64
	err := s.db.Model(&model.QuestionBank{}).
		Where("creator_id = ? AND visibility = 'public' AND review_status = 'pending'", userID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("公开题库每次只能上传一个，由admin审批通过后才能再次上传")
	}
	return nil
}
