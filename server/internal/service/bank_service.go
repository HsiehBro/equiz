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

// ListBanks 查询用户可见的题库（官方题库 + 审核通过的公开题库 + 该用户自己导入的私有/待审核题库），并计算做题进度
// 私有题库与未过审题库隔离原则：非公开或待审批题库仅创建者本人可见；未登录用户无法查看任何私有/未过审题库。
func (s *BankService) ListBanks(userID uint, userRole string) ([]model.QuestionBank, error) {
	var banks []model.QuestionBank

	var query *gorm.DB
	if userRole == "admin" {
		// 管理员：可查看官方题库、所有已公开题库、所有待审核题库、以及本人创建的私有题库
		query = s.db.Where("is_official = ? OR visibility = 'public' OR creator_id = ?", true, userID).Order("id ASC")
	} else if userID > 0 {
		// 已登录普通/VIP用户：
		// 1. 官方题库 (is_official = true)
		// 2. 审核通过已正式公开的题库 (visibility = 'public' AND review_status = 'approved')
		// 3. 本人创建的题库 (creator_id = userID，无论私有还是待审核)
		query = s.db.Where(
			"is_official = ? OR (visibility = 'public' AND review_status = 'approved') OR creator_id = ?",
			true, userID,
		).Order("id ASC")
	} else {
		// 未登录用户：只能看到官方题库和已通过审核的公开题库
		query = s.db.Where("is_official = ? OR (visibility = 'public' AND review_status = 'approved')", true).Order("id ASC")
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

// DeleteBank 删除题库
// 管理员权限：管理员可以删除任何公开题库；
// 普通用户权限：仅可删除本人创建的私有题库，不可删除公开题库与官方题库。
func (s *BankService) DeleteBank(bankID uint, userID uint, userRole string) error {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return err
	}

	if bank.IsOfficial {
		return errors.New("官方题库不允许删除")
	}

	if userRole == "admin" {
		// 管理员可以删除任何非官方公开题库与私有题库
	} else {
		// 非管理员检查
		if bank.Visibility == "public" && bank.ReviewStatus == "approved" {
			return errors.New("公开题库已面向全员开放，不可删除（仅管理员拥有删除公开题库权限）")
		}
		if bank.CreatorID != userID {
			return errors.New("无权删除非本人创建的题库")
		}
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

// ToggleVIPBank 切换公开题库的 VIP 标识（仅 Admin 可调用）
func (s *BankService) ToggleVIPBank(bankID uint) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return nil, err
	}

	bank.IsVIP = !bank.IsVIP
	if err := s.db.Save(&bank).Error; err != nil {
		return nil, err
	}
	return &bank, nil
}

// CheckBankVIPAccess 检查题库访问权限：VIP题库所有用户可见，但只有VIP用户或Admin可使用
func (s *BankService) CheckBankVIPAccess(bankID uint, userID uint, userRole string, isVIP bool) error {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return err
	}

	if bank.IsVIP {
		if userRole != "admin" && !isVIP {
			// 若 Token Claims 尚不是 VIP（例如刚刚在应用内购买），查询数据库校验最新状态
			if userID > 0 {
				var u model.User
				if err := s.db.First(&u, userID).Error; err == nil {
					if u.Role == "admin" || u.Role == "vip" || u.IsLifetimeVIP || (u.VIPExpire != nil && u.VIPExpire.After(time.Now())) {
						return nil
					}
				}
			}
			return errors.New("该题库为 VIP 专属题库，仅限 VIP 会员刷题使用")
		}
	}
	return nil
}

// ApplyPublic 用户在首页对自己的私有题库提交公开申请
func (s *BankService) ApplyPublic(bankID uint, userID uint, userName string) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return nil, fmt.Errorf("题库不存在")
	}

	// 权限检查：只能对本人创建的题库提交公开申请
	if bank.CreatorID != userID && userID != 0 {
		return nil, errors.New("无权申请公开非本人创建的私有题库")
	}

	if bank.IsOfficial {
		return nil, errors.New("官方题库无需申请公开")
	}

	if bank.Visibility == "public" && bank.ReviewStatus == "approved" {
		return nil, errors.New("该题库已是公开状态，无需重复申请")
	}

	if bank.Visibility == "public" && bank.ReviewStatus == "pending" {
		return nil, errors.New("该题库已在审核中，请耐心等待管理员审批")
	}

	// 单次限制校验：检查用户是否已有其它公开题库正在审核中
	var pendingCount int64
	s.db.Model(&model.QuestionBank{}).
		Where("creator_id = ? AND visibility = 'public' AND review_status = 'pending'", userID).
		Count(&pendingCount)
	if pendingCount > 0 {
		return nil, errors.New("公开题库每次只能提交一个审核，由admin审批通过后才能再次提交")
	}

	if userName == "" {
		userName = "备考学员"
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		bank.Visibility = "public"
		bank.ReviewStatus = "pending"
		bank.CreatorName = userName
		bank.UpdatedAt = time.Now()
		if err := tx.Save(&bank).Error; err != nil {
			return err
		}

		// 向管理员发送微信服务通知提醒
		adminNotice := model.SystemNotification{
			UserID:    0,
			Type:      "admin_pending",
			Title:     "【微信服务通知】有新的公开题库待审批",
			Content:   fmt.Sprintf("用户「%s」申请将私有题库《%s》（共 %d 题）公开，请前往个人中心手动审批栏进行审核。", userName, bank.Title, bank.TotalCount),
			RelatedID: bank.ID,
			IsRead:    false,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&adminNotice).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &bank, nil
}

// UpdateBankCategory 修改题库分类（管理员可修改任意题库，普通用户可修改本人创建的题库）
func (s *BankService) UpdateBankCategory(bankID uint, categoryID uint, userID uint, userRole string) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err != nil {
		return nil, errors.New("题库不存在")
	}

	if userRole != "admin" {
		if bank.CreatorID != userID {
			return nil, errors.New("无权修改非本人创建题库的分类")
		}
	}

	var category model.Category
	if err := s.db.First(&category, categoryID).Error; err != nil {
		return nil, errors.New("目标分类不存在")
	}

	bank.CategoryID = category.ID
	bank.Category = category.Name
	bank.UpdatedAt = time.Now()

	if err := s.db.Save(&bank).Error; err != nil {
		return nil, fmt.Errorf("更新题库分类失败: %w", err)
	}

	return &bank, nil
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
