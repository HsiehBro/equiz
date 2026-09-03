package model

import (
	"time"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OpenID    string    `gorm:"column:open_id;size:128;uniqueIndex;not null" json:"openid"`
	UnionID   string    `gorm:"column:union_id;size:128;index" json:"unionid,omitempty"`
	Nickname  string    `gorm:"size:64;default:'备考学员'" json:"nickname"`
	AvatarURL string    `gorm:"type:text;default:''" json:"avatar_url"`
	Role      string    `gorm:"size:32;default:'user'" json:"role"` // 'user' | 'admin' | 'vip'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
