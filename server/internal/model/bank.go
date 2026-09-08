package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SyllabusWeightItem struct {
	Name    string  `json:"name"`
	Percent float64 `json:"percent"` // 百分比数值，如 20 代表 20%
}

type SyllabusWeightsList []SyllabusWeightItem

func (s SyllabusWeightsList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *SyllabusWeightsList) Scan(value interface{}) error {
	if value == nil {
		*s = []SyllabusWeightItem{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal SyllabusWeightsList value:", value))
	}
	return json.Unmarshal(bytes, s)
}

type QuestionBank struct {
	ID              uint                `gorm:"primaryKey" json:"id"`
	Title           string              `gorm:"size:255;not null;uniqueIndex" json:"title"`
	Category        string              `gorm:"size:64;default:'综合'" json:"category"`
	CategoryID      uint                `gorm:"default:1;index" json:"category_id"`
	Description     string              `gorm:"type:text" json:"description"`
	CoverURL        string              `gorm:"size:255;default:''" json:"cover_url"`
	TotalCount      int                 `gorm:"default:0" json:"total_count"`
	IsOfficial      bool                `gorm:"default:false;index" json:"is_official"`
	Visibility      string              `gorm:"size:32;default:'public';index" json:"visibility"` // 'public' (公开) | 'private' (私有)
	ReviewStatus    string              `gorm:"size:32;default:'approved';index" json:"review_status"` // 'approved' (已通过) | 'pending' (待审核) | 'rejected' (已驳回)
	IsVIP           bool                `gorm:"column:is_vip;default:false;index" json:"is_vip"`       // 是否为 VIP 专属题库
	CreatorID       uint                `gorm:"default:0;index" json:"creator_id"`
	CreatorName     string              `gorm:"size:64;default:''" json:"creator_name,omitempty"`     // 创建者昵称
	SyllabusWeights SyllabusWeightsList `gorm:"type:jsonb" json:"syllabus_weights,omitempty"`         // 考纲配比
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`

	// 动态关联统计字段（用于前端展示，非数据库直接持久化）
	Progress     int    `gorm:"-" json:"progress"`
	LastPractice string `gorm:"-" json:"last_practice,omitempty"`
}

func (QuestionBank) TableName() string {
	return "question_banks"
}
