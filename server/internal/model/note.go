package model

import (
	"time"

	"gorm.io/gorm"
)

// UserNote 题目笔记与评论
type UserNote struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"not null;index:idx_user_note" json:"user_id"`
	User       *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	BankID     uint           `gorm:"not null;index:idx_bank_note" json:"bank_id"`
	Bank       *QuestionBank  `gorm:"foreignKey:BankID" json:"bank,omitempty"`
	QuestionID uint           `gorm:"not null;index:idx_question_note" json:"question_id"`
	Question   *Question      `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	Visibility string         `gorm:"size:16;not null;default:'public';index" json:"visibility"` // "public" | "private"
	LikeCount  int            `gorm:"default:0" json:"like_count"`
	ParentID      *uint          `gorm:"index:idx_parent_note" json:"parent_id,omitempty"` // 父级评论ID (用于嵌套回复)
	ReplyToUserID *uint          `gorm:"index:idx_reply_to_user" json:"reply_to_user_id,omitempty"` // 被回复人用户ID
	ReplyToUser   *User          `gorm:"foreignKey:ReplyToUserID" json:"reply_to_user,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// 动态运行时字段（非数据库持久化）
	IsLiked       bool       `gorm:"-" json:"is_liked"`
	AuthorName    string     `gorm:"-" json:"author_name,omitempty"`
	AuthorAvatar  string     `gorm:"-" json:"author_avatar,omitempty"`
	QuestionTitle string     `gorm:"-" json:"question_title,omitempty"`
	BankTitle     string     `gorm:"-" json:"bank_title"`
	ReplyToAuthor string     `gorm:"size:64" json:"reply_to_author,omitempty"`
	Replies       []UserNote `gorm:"-" json:"replies,omitempty"`
}

func (UserNote) TableName() string {
	return "user_notes"
}

// UserNoteLike 笔记点赞防重记录
type UserNoteLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_note_like,priority:1" json:"user_id"`
	NoteID    uint      `gorm:"not null;uniqueIndex:idx_user_note_like,priority:2" json:"note_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserNoteLike) TableName() string {
	return "user_note_likes"
}

// CreateNoteRequest 新增笔记/评论或回复请求
type CreateNoteRequest struct {
	BankID        uint   `json:"bank_id"`
	QuestionID    uint   `json:"question_id" binding:"required"`
	Content       string `json:"content" binding:"required"`
	Visibility    string `json:"visibility"` // 可选: "public" | "private"，默认 public
	ParentID      *uint  `json:"parent_id"`  // 可选: 回复的目标父评论ID
	ReplyToUserID *uint  `json:"reply_to_user_id"` // 可选: 被回复人用户ID
	ReplyToAuthor string `json:"reply_to_author"` // 可选: 被回复人昵称
}

// UpdateNoteRequest 更新笔记/评论请求
type UpdateNoteRequest struct {
	Content    string `json:"content"`
	Visibility string `json:"visibility"`
}
