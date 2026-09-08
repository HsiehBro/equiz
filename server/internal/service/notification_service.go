package service

import (
	"fmt"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// ListNotifications 查询用户通知。若为管理员，同时获取针对管理员的审核提醒通知 (user_id = 0)
func (s *NotificationService) ListNotifications(userID uint, isAdmin bool) ([]model.SystemNotification, error) {
	var list []model.SystemNotification

	query := s.db.Model(&model.SystemNotification{})
	if isAdmin {
		// 管理员可看到发给自己以及系统全局/管理员待审批的通知
		query = query.Where("user_id = ? OR user_id = 0", userID)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Order("created_at DESC").Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("查询系统通知失败: %w", err)
	}

	// 动态关联题目与题库最新信息，确保引用的题干与题库标题始终与实际试题一致
	questionIDs := []uint{}
	bankIDs := []uint{}
	for _, n := range list {
		if n.QuestionID > 0 {
			questionIDs = append(questionIDs, n.QuestionID)
		}
		if n.BankID > 0 {
			bankIDs = append(bankIDs, n.BankID)
		}
	}

	qTitleMap := map[uint]string{}
	qBankMap := map[uint]uint{}
	if len(questionIDs) > 0 {
		var questions []model.Question
		s.db.Where("id IN ?", questionIDs).Find(&questions)
		for _, q := range questions {
			qTitleMap[q.ID] = q.Title
			qBankMap[q.ID] = q.BankID
			if q.BankID > 0 {
				bankIDs = append(bankIDs, q.BankID)
			}
		}
	}

	bTitleMap := map[uint]string{}
	if len(bankIDs) > 0 {
		var banks []model.QuestionBank
		s.db.Where("id IN ?", bankIDs).Find(&banks)
		for _, b := range banks {
			bTitleMap[b.ID] = b.Title
		}
	}

	for i := range list {
		// 动态校准真实题干
		if latestTitle, ok := qTitleMap[list[i].QuestionID]; ok && latestTitle != "" {
			list[i].QuestionTitle = latestTitle
		}
		// 动态校准所属题库 ID 与题库名称
		if realBankID, ok := qBankMap[list[i].QuestionID]; ok && realBankID > 0 {
			list[i].BankID = realBankID
		}
		if bankTitle, ok := bTitleMap[list[i].BankID]; ok && bankTitle != "" {
			list[i].BankTitle = bankTitle
		}
	}

	return list, nil
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(id uint, userID uint, isAdmin bool) error {
	query := s.db.Model(&model.SystemNotification{}).Where("id = ?", id)
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}
	return query.Update("is_read", true).Error
}

// DeleteNotification 删除系统通知 (幂等设计，如已删除也视为成功)
func (s *NotificationService) DeleteNotification(id uint, userID uint, isAdmin bool) error {
	query := s.db.Where("id = ?", id)
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}
	res := query.Delete(&model.SystemNotification{})
	if res.Error != nil {
		return res.Error
	}
	return nil
}
