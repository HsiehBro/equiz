package service

import (
	"errors"
	"fmt"
	"math"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type PlanService struct {
	db *gorm.DB
}

func NewPlanService(db *gorm.DB) *PlanService {
	return &PlanService{db: db}
}

// SavePlan 创建或更新学习规划
func (s *PlanService) SavePlan(userID uint, req model.SavePlanRequest) (*model.StudyPlanProgressResponse, error) {
	if req.DailyGoal <= 0 {
		return nil, errors.New("每日目标题数必须大于 0")
	}

	// 验证题库是否存在
	var bank model.QuestionBank
	if err := s.db.First(&bank, req.BankID).Error; err != nil {
		return nil, errors.New("指定题库不存在")
	}

	var savedPlan model.StudyPlan

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 若设为激活状态，先将该用户其它所有规划置为非激活
		if req.IsActive {
			if err := tx.Model(&model.StudyPlan{}).
				Where("user_id = ?", userID).
				Update("is_active", false).Error; err != nil {
				return err
			}
		}

		var existing model.StudyPlan
		err := tx.Where("user_id = ? AND bank_id = ?", userID, req.BankID).First(&existing).Error
		if err == nil {
			// 更新现有规划
			existing.DailyGoal = req.DailyGoal
			existing.AppReminder = req.AppReminder
			existing.WechatReminder = req.WechatReminder
			existing.IsActive = req.IsActive
			existing.UpdatedAt = time.Now()

			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			savedPlan = existing
			return nil
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 检查是否为第一个规划，如果是且未传 is_active，默认设为 true
			var count int64
			tx.Model(&model.StudyPlan{}).Where("user_id = ?", userID).Count(&count)
			isActive := req.IsActive
			if count == 0 {
				isActive = true
			}

			newPlan := model.StudyPlan{
				UserID:         userID,
				BankID:         req.BankID,
				DailyGoal:      req.DailyGoal,
				AppReminder:    req.AppReminder,
				WechatReminder: req.WechatReminder,
				IsActive:       isActive,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := tx.Create(&newPlan).Error; err != nil {
				return err
			}
			savedPlan = newPlan
			return nil
		}

		return err
	})

	if err != nil {
		return nil, err
	}

	savedPlan.Bank = &bank
	return s.buildProgressResponse(userID, savedPlan), nil
}

// ListPlans 获取用户的所有学习规划列表及其动态进度
func (s *PlanService) ListPlans(userID uint) ([]model.StudyPlanProgressResponse, error) {
	var plans []model.StudyPlan
	err := s.db.Where("user_id = ?", userID).
		Preload("Bank").
		Order("is_active DESC, updated_at DESC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}

	var results []model.StudyPlanProgressResponse
	for _, p := range plans {
		resp := s.buildProgressResponse(userID, p)
		if resp != nil {
			results = append(results, *resp)
		}
	}

	return results, nil
}

// GetActivePlan 获取当前置顶生效的学习规划（用于首页卡片和个人中心直接展示）
func (s *PlanService) GetActivePlan(userID uint) (*model.StudyPlanProgressResponse, error) {
	var plan model.StudyPlan
	// 优先查询激活的
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).
		Preload("Bank").
		First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 若未显式激活，取最新的一条规划
			err2 := s.db.Where("user_id = ?", userID).
				Preload("Bank").
				Order("updated_at DESC").
				First(&plan).Error
			if err2 != nil {
				if errors.Is(err2, gorm.ErrRecordNotFound) {
					return nil, nil // 无规划
				}
				return nil, err2
			}
		} else {
			return nil, err
		}
	}

	return s.buildProgressResponse(userID, plan), nil
}

// DeletePlan 删除指定学习规划
func (s *PlanService) DeletePlan(userID uint, planID uint) error {
	var plan model.StudyPlan
	if err := s.db.Where("id = ? AND user_id = ?", planID, userID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("规划不存在或无权删除")
		}
		return err
	}

	return s.db.Delete(&plan).Error
}

// buildProgressResponse 计算规划关联题库的总题数、做题数、今日刷题数及预计完成时间
func (s *PlanService) buildProgressResponse(userID uint, plan model.StudyPlan) *model.StudyPlanProgressResponse {
	bankTitle := ""
	totalQuestions := 0
	if plan.Bank != nil {
		bankTitle = plan.Bank.Title
		totalQuestions = plan.Bank.TotalCount
	} else {
		var bank model.QuestionBank
		if err := s.db.First(&bank, plan.BankID).Error; err == nil {
			bankTitle = bank.Title
			totalQuestions = bank.TotalCount
		}
	}

	// 1. 计算已做题数（去重题数）
	var finishedCount int64
	s.db.Model(&model.UserRecord{}).
		Where("user_id = ? AND bank_id = ?", userID, plan.BankID).
		Distinct("question_id").
		Count(&finishedCount)

	// 2. 计算今日已刷题数 (今日 00:00:00 至今)
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var todayCount int64
	s.db.Model(&model.UserRecord{}).
		Where("user_id = ? AND bank_id = ? AND created_at >= ?", userID, plan.BankID, todayStart).
		Count(&todayCount)

	// 3. 计算预估完成天数
	remaining := totalQuestions - int(finishedCount)
	if remaining < 0 {
		remaining = 0
	}

	daysNeeded := 0
	if plan.DailyGoal > 0 && remaining > 0 {
		daysNeeded = int(math.Ceil(float64(remaining) / float64(plan.DailyGoal)))
	}

	estimatedFinishDate := ""
	if daysNeeded > 0 {
		finishTime := now.AddDate(0, 0, daysNeeded)
		estimatedFinishDate = finishTime.Format("2006-01-02")
	} else if remaining == 0 && totalQuestions > 0 {
		estimatedFinishDate = "已全部学完"
	} else {
		estimatedFinishDate = fmt.Sprintf("预计需要 %d 天", daysNeeded)
	}

	isTodayGoalReached := int(todayCount) >= plan.DailyGoal

	return &model.StudyPlanProgressResponse{
		ID:                  plan.ID,
		UserID:              plan.UserID,
		BankID:              plan.BankID,
		BankTitle:           bankTitle,
		DailyGoal:           plan.DailyGoal,
		AppReminder:         plan.AppReminder,
		WechatReminder:      plan.WechatReminder,
		IsActive:            plan.IsActive,
		TotalQuestions:      totalQuestions,
		FinishedQuestions:   int(finishedCount),
		TodayCount:          int(todayCount),
		DaysNeeded:          daysNeeded,
		IsTodayGoalReached:  isTodayGoalReached,
		EstimatedFinishDate: estimatedFinishDate,
		CreatedAt:           plan.CreatedAt,
		UpdatedAt:           plan.UpdatedAt,
	}
}
