package model

import (
	"time"
)

// UserReferral 记录学员之间的邀请绑定与首单返奖状态
type UserReferral struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	InviterID  uint       `gorm:"index;not null" json:"inviter_id"`
	Inviter    *User      `gorm:"foreignKey:InviterID" json:"inviter,omitempty"`
	InviteeID  uint       `gorm:"uniqueIndex;not null" json:"invitee_id"` // 单个用户首购保护，仅能被唯一有效绑定一次
	Invitee    *User      `gorm:"foreignKey:InviteeID" json:"invitee,omitempty"`
	Status     string     `gorm:"size:32;default:'bound';not null" json:"status"` // 'bound' (已锁定关系待付费) | 'rewarded' (首单完成已返奖)
	PlanID     string     `gorm:"size:64;default:''" json:"plan_id"`              // 'monthly' | 'quarterly' | 'yearly' | 'lifetime'
	RewardDays int        `gorm:"default:0" json:"reward_days"`                   // 返赠时长天数（终身卡封顶 365 天）
	RewardedAt *time.Time `json:"rewarded_at,omitempty"`                          // 奖励生效时间
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (UserReferral) TableName() string {
	return "user_referrals"
}

// VIPOrder 记录会员充值/开通订单流水
type VIPOrder struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderNo   string    `gorm:"size:64;uniqueIndex;not null" json:"order_no"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	PlanID    string    `gorm:"size:64;not null" json:"plan_id"`
	Amount    float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status    string    `gorm:"size:32;default:'paid';not null" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (VIPOrder) TableName() string {
	return "vip_orders"
}
