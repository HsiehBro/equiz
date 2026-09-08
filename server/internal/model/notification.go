package model

import "time"

// SystemNotification 系统与审批通知模型
type SystemNotification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"` // 接收人ID (0 表示系统全局或所有管理员)
	Type      string    `gorm:"size:32;default:'system';index" json:"type"` // 'approval_pass' | 'approval_reject' | 'admin_pending' | 'activity' | 'comment_reply'
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	RelatedID       uint      `gorm:"default:0" json:"related_id"` // 关联的题库ID或评论ID/笔记ID
	BankID          uint      `gorm:"default:0" json:"bank_id,omitempty"`
	BankTitle       string    `gorm:"size:255" json:"bank_title,omitempty"`
	QuestionID      uint      `gorm:"default:0" json:"question_id,omitempty"`
	QuestionTitle   string    `gorm:"size:255" json:"question_title,omitempty"`
	ReplierID       uint      `gorm:"default:0" json:"replier_id,omitempty"`
	ReplierName     string    `gorm:"size:64" json:"replier_name,omitempty"`
	ReplierAvatar   string    `gorm:"size:255" json:"replier_avatar,omitempty"`
	OriginalContent string    `gorm:"type:text" json:"original_content,omitempty"`
	IsRead          bool      `gorm:"default:false;index" json:"is_read"`
	CreatedAt       time.Time `json:"created_at"`
}

func (SystemNotification) TableName() string {
	return "system_notifications"
}
