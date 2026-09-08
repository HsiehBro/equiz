package model

import (
	"time"
)

// StudyPlan 学习规划与每日目标
type StudyPlan struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	UserID         uint          `gorm:"not null;uniqueIndex:idx_user_bank_plan,priority:1;index" json:"user_id"`
	BankID         uint          `gorm:"not null;uniqueIndex:idx_user_bank_plan,priority:2;index" json:"bank_id"`
	Bank           *QuestionBank `gorm:"foreignKey:BankID" json:"bank,omitempty"`
	DailyGoal      int           `gorm:"not null;default:30" json:"daily_goal"` // 每日目标题数
	AppReminder    bool          `gorm:"default:true" json:"app_reminder"`
	WechatReminder bool          `gorm:"default:false" json:"wechat_reminder"`
	IsActive       bool          `gorm:"default:false;index" json:"is_active"` // 是否为首页/个人中心置顶主计划
	CheckInDays    int           `gorm:"default:0" json:"check_in_days"`
	LastCheckInDate string       `gorm:"size:20;default:''" json:"last_check_in_date"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (StudyPlan) TableName() string {
	return "study_plans"
}

// SavePlanRequest 创建或保存学习规划请求
type SavePlanRequest struct {
	BankID         uint `json:"bank_id" binding:"required"`
	DailyGoal      int  `json:"daily_goal" binding:"required,min=1"`
	AppReminder    bool `json:"app_reminder"`
	WechatReminder bool `json:"wechat_reminder"`
	IsActive       bool `json:"is_active"`
	ResetCheckIn   bool `json:"reset_check_in"`
}

// StudyPlanProgressResponse 学习规划与今日动态进度响应
type StudyPlanProgressResponse struct {
	ID                  uint      `json:"id"`
	UserID              uint      `json:"user_id"`
	BankID              uint      `json:"bank_id"`
	BankTitle           string    `json:"bank_title"`
	IsVIP               bool      `json:"is_vip"`
	DailyGoal           int       `json:"daily_goal"`
	AppReminder         bool      `json:"app_reminder"`
	WechatReminder      bool      `json:"wechat_reminder"`
	IsActive            bool      `json:"is_active"`
	CheckInDays         int       `json:"check_in_days"`
	LastCheckInDate     string    `json:"last_check_in_date"`
	TotalQuestions      int       `json:"total_questions"`
	FinishedQuestions   int       `json:"finished_questions"`
	TodayCount          int       `json:"today_count"`
	DaysNeeded          int       `json:"days_needed"`
	IsTodayGoalReached  bool      `json:"is_today_goal_reached"`
	EstimatedFinishDate string    `json:"estimated_finish_date"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
