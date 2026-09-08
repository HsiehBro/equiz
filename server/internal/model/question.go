package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type OptionItem struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type OptionsList []OptionItem

func (o OptionsList) Value() (driver.Value, error) {
	if o == nil {
		return "[]", nil
	}
	return json.Marshal(o)
}

func (o *OptionsList) Scan(value interface{}) error {
	if value == nil {
		*o = []OptionItem{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, o)
}

type AnswerList []string

func (a AnswerList) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *AnswerList) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, a)
}

type Question struct {
	ID              uint        `gorm:"primaryKey" json:"id"`
	BankID          uint        `gorm:"not null;index:idx_bank_sort,priority:1;index" json:"bank_id"`
	Type            string      `gorm:"size:32;not null;default:'单选'" json:"type"` // '单选' | '多选' | '判断'
	Section         string      `gorm:"size:128;default:''" json:"section"`
	Title           string      `gorm:"type:text;not null" json:"title"`
	Options         OptionsList `gorm:"type:jsonb;not null" json:"options"`
	Answer          AnswerList  `gorm:"type:jsonb;not null" json:"answer"`
	Difficulty      string      `gorm:"size:32;default:'中等'" json:"difficulty"`
	KnowledgePoint  string      `gorm:"size:128;default:''" json:"knowledge_point"`
	Analysis        string      `gorm:"type:text;default:''" json:"analysis"`
	KnowledgeDetail string      `gorm:"type:text;default:''" json:"knowledge_detail"`
	SortOrder       int         `gorm:"default:0;index:idx_bank_sort,priority:2" json:"sort_order"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`

	// 动态状态（当前用户作答情况与收藏状态，非数据库持久化）
	UserAnswer   []string `gorm:"-" json:"user_answer,omitempty"`
	IsBookmarked bool     `gorm:"-" json:"is_bookmarked,omitempty"`
	BankTitle    string   `gorm:"-" json:"bank_title,omitempty"`
}

func (Question) TableName() string {
	return "questions"
}
