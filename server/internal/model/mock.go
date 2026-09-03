package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SyllabusDiagnosticItem 单个大纲考点的得分与诊断
type SyllabusDiagnosticItem struct {
	Name         string  `json:"name"`          // 大纲考点名称
	TargetWeight float64 `json:"target_weight"` // 配置的出题百分比，如 20 代表 20%
	TotalCount   int     `json:"total_count"`   // 本考点实际出题数量
	CorrectCount int     `json:"correct_count"` // 本考点答对数量
	WrongCount   int     `json:"wrong_count"`   // 本考点答错数量
	ScoreRate    float64 `json:"score_rate"`    // 本考点正确率百分比 (0-100)
	StatusTag    string  `json:"status_tag"`    // '掌握良好' | '达标巩固' | '薄弱待强化'
	LevelClass   string  `json:"level_class"`   // 'level-success' | 'level-info' | 'level-warning'
	Advice       string  `json:"advice"`        // 考点强化建议
}

type SyllabusDiagnosticsList []SyllabusDiagnosticItem

func (s SyllabusDiagnosticsList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *SyllabusDiagnosticsList) Scan(value interface{}) error {
	if value == nil {
		*s = []SyllabusDiagnosticItem{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal SyllabusDiagnosticsList value:", value))
	}
	return json.Unmarshal(bytes, s)
}

// QuestionSnapshotItem 试卷每道题的作答与解析快照
type QuestionSnapshotItem struct {
	ID              uint        `json:"id"`
	Number          int         `json:"number"`
	Type            string      `json:"type"`
	Section         string      `json:"section"`
	Title           string      `json:"title"`
	Options         OptionsList `json:"options"`
	UserAnswer      []string    `json:"user_answer"`
	CorrectAnswer   []string    `json:"correct_answer"`
	IsCorrect       bool        `json:"is_correct"`
	Difficulty      string      `json:"difficulty"`
	KnowledgePoint  string      `json:"knowledge_point"`
	Analysis        string      `json:"analysis"`
	KnowledgeDetail string      `json:"knowledge_detail"`
}

type QuestionSnapshotsList []QuestionSnapshotItem

func (q QuestionSnapshotsList) Value() (driver.Value, error) {
	if q == nil {
		return "[]", nil
	}
	return json.Marshal(q)
}

func (q *QuestionSnapshotsList) Scan(value interface{}) error {
	if value == nil {
		*q = []QuestionSnapshotItem{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal QuestionSnapshotsList value:", value))
	}
	return json.Unmarshal(bytes, q)
}

// MockRecord 模考记录实体
type MockRecord struct {
	ID                  uint                    `gorm:"primaryKey" json:"id"`
	UserID              uint                    `gorm:"not null;index" json:"user_id"`
	BankID              uint                    `gorm:"not null;index" json:"bank_id"`
	ExamTitle           string                  `gorm:"size:255;not null" json:"exam_title"`
	Score               float64                 `gorm:"type:numeric(5,1);not null" json:"score"`
	TotalScore          float64                 `gorm:"type:numeric(5,1);default:100" json:"total_score"`
	TotalQuestions      int                     `gorm:"default:0" json:"total_questions"`
	CorrectCount        int                     `gorm:"default:0" json:"correct_count"`
	WrongCount          int                     `gorm:"default:0" json:"wrong_count"`
	UnansweredCount     int                     `gorm:"default:0" json:"unanswered_count"`
	TimeUsedSeconds     int                     `gorm:"default:0" json:"time_used_seconds"`
	RankingPercent      float64                 `gorm:"type:numeric(5,1);default:80" json:"ranking_percent"`
	IsPassed            bool                    `gorm:"default:false;index" json:"is_passed"`
	SyllabusDiagnostics SyllabusDiagnosticsList `gorm:"type:jsonb" json:"syllabus_diagnostics"`
	QuestionSnapshots   QuestionSnapshotsList   `gorm:"type:jsonb" json:"question_snapshots,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`

	// 辅助展示字段（格式化）
	TimeUsedText string `gorm:"-" json:"time_used_text,omitempty"`
	DateText     string `gorm:"-" json:"date_text,omitempty"`
	ScoreText    string `gorm:"-" json:"score_text,omitempty"`
}

func (MockRecord) TableName() string {
	return "mock_records"
}
