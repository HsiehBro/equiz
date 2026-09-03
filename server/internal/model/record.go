package model

import (
	"time"
)

type UserRecord struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"not null;index" json:"user_id"`
	BankID          uint       `gorm:"not null;index" json:"bank_id"`
	QuestionID      uint       `gorm:"not null;index" json:"question_id"`
	UserAnswer      AnswerList `gorm:"type:jsonb;not null" json:"user_answer"`
	IsCorrect       bool       `gorm:"not null;index" json:"is_correct"`
	DurationSeconds int        `gorm:"default:0" json:"duration_seconds"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (UserRecord) TableName() string {
	return "user_records"
}
