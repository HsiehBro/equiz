package model

import (
	"time"
)

type UserError struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;uniqueIndex:idx_user_question,priority:1;index" json:"user_id"`
	BankID      uint      `gorm:"not null;index" json:"bank_id"`
	QuestionID  uint      `gorm:"not null;uniqueIndex:idx_user_question,priority:2;index" json:"question_id"`
	WrongCount  int       `gorm:"default:1" json:"wrong_count"`
	IsMastered  bool      `gorm:"default:false;index" json:"is_mastered"`
	LastWrongAt time.Time `json:"last_wrong_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联题目详情，方便查询错题列表时连带返回试题
	Question *Question `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
}

func (UserError) TableName() string {
	return "user_errors"
}
