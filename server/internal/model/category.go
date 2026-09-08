package model

import "time"

type Category struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null;uniqueIndex" json:"name"`
	Description string    `gorm:"size:255;default:''" json:"description"`
	Icon        string    `gorm:"size:255;default:''" json:"icon"`
	BgClass     string    `gorm:"size:64;default:'bg-gray-light'" json:"bg_class"`
	SortOrder   int       `gorm:"default:0;index" json:"sort_order"`
	IsActive    bool      `gorm:"default:true;index" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 动态统计与标识字段（非数据库持久化）
	BankCount int  `gorm:"-" json:"bank_count"`
	HasVIP    bool `gorm:"-" json:"has_vip"`
	IsVip     bool `gorm:"-" json:"is_vip"` // 适配小程序前端命名约定
}

func (Category) TableName() string {
	return "categories"
}
