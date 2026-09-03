package model

import (
	"time"
)

type UserFavorite struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_user_fav_q,priority:1;index" json:"user_id"`
	QuestionID uint      `gorm:"not null;uniqueIndex:idx_user_fav_q,priority:2;index" json:"question_id"`
	BankID     uint      `gorm:"not null;index" json:"bank_id"`
	CreatedAt  time.Time `json:"created_at"`

	Question *Question     `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	Bank     *QuestionBank `gorm:"foreignKey:BankID" json:"bank,omitempty"`
}

func (UserFavorite) TableName() string {
	return "user_favorites"
}
