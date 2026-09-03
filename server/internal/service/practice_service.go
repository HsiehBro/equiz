package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PracticeService struct {
	db *gorm.DB
}

func NewPracticeService(db *gorm.DB) *PracticeService {
	return &PracticeService{db: db}
}

type SubmitSingleRequest struct {
	BankID          uint     `json:"bank_id" binding:"required"`
	QuestionID      uint     `json:"question_id" binding:"required"`
	UserAnswer      []string `json:"user_answer" binding:"required"`
	DurationSeconds int      `json:"duration_seconds"`
}

type SubmitSingleResult struct {
	QuestionID    uint     `json:"question_id"`
	IsCorrect     bool     `json:"is_correct"`
	CorrectAnswer []string `json:"correct_answer"`
	Analysis      string   `json:"analysis"`
	UserAnswer    []string `json:"user_answer"`
}

// CompareAnswers 比对答案是否一致（规范化排序且大写）
func CompareAnswers(ans1, ans2 []string) bool {
	if len(ans1) != len(ans2) {
		return false
	}
	c1 := make([]string, len(ans1))
	c2 := make([]string, len(ans2))
	for i := range ans1 {
		c1[i] = strings.ToUpper(strings.TrimSpace(ans1[i]))
	}
	for i := range ans2 {
		c2[i] = strings.ToUpper(strings.TrimSpace(ans2[i]))
	}
	sort.Strings(c1)
	sort.Strings(c2)
	for i := range c1 {
		if c1[i] != c2[i] {
			return false
		}
	}
	return true
}

func (s *PracticeService) SubmitSingleAnswer(userID uint, req *SubmitSingleRequest) (*SubmitSingleResult, error) {
	var q model.Question
	if err := s.db.First(&q, req.QuestionID).Error; err != nil {
		return nil, errors.New("试题不存在")
	}

	isCorrect := CompareAnswers(req.UserAnswer, q.Answer)

	// 事务内记录作答与错题集状态
	err := s.db.Transaction(func(tx *gorm.DB) error {
		record := model.UserRecord{
			UserID:          userID,
			BankID:          req.BankID,
			QuestionID:      req.QuestionID,
			UserAnswer:      req.UserAnswer,
			IsCorrect:       isCorrect,
			DurationSeconds: req.DurationSeconds,
			CreatedAt:       time.Now(),
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		if !isCorrect {
			// 答错了，更新或新建错题记录
			var userErr model.UserError
			err := tx.Where("user_id = ? AND question_id = ?", userID, req.QuestionID).First(&userErr).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					userErr = model.UserError{
						UserID:      userID,
						BankID:      req.BankID,
						QuestionID:  req.QuestionID,
						WrongCount:  1,
						IsMastered:  false,
						LastWrongAt: time.Now(),
					}
					return tx.Create(&userErr).Error
				}
				return err
			}

			// 如果已存在该错题，更新计数并重置掌握状态
			userErr.WrongCount += 1
			userErr.IsMastered = false
			userErr.LastWrongAt = time.Now()
			return tx.Save(&userErr).Error
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &SubmitSingleResult{
		QuestionID:    q.ID,
		IsCorrect:     isCorrect,
		CorrectAnswer: q.Answer,
		Analysis:      q.Analysis,
		UserAnswer:    req.UserAnswer,
	}, nil
}

type SubmitExamRequest struct {
	BankID          uint                `json:"bank_id" binding:"required"`
	TotalSeconds    int                 `json:"total_seconds"`
	Answers         map[uint][]string   `json:"answers" binding:"required"` // question_id -> user_answer
}

type SubmitExamResult struct {
	TotalQuestions int     `json:"total_questions"`
	AnsweredCount  int     `json:"answered_count"`
	CorrectCount   int     `json:"correct_count"`
	WrongCount     int     `json:"wrong_count"`
	Score          float64 `json:"score"`
	AccuracyRate   float64 `json:"accuracy_rate"` // 正确率百分比
	TotalSeconds   int     `json:"total_seconds"`
}

func (s *PracticeService) SubmitExam(userID uint, req *SubmitExamRequest) (*SubmitExamResult, error) {
	var questions []model.Question
	if err := s.db.Where("bank_id = ?", req.BankID).Find(&questions).Error; err != nil {
		return nil, err
	}

	totalQ := len(questions)
	if totalQ == 0 {
		return nil, errors.New("题库题目为空")
	}

	correctCount := 0
	answeredCount := 0
	now := time.Now()

	var records []model.UserRecord
	var wrongQuestions []uint

	for _, q := range questions {
		userAns, exists := req.Answers[q.ID]
		if !exists || len(userAns) == 0 {
			continue
		}
		answeredCount++
		isCorr := CompareAnswers(userAns, q.Answer)
		if isCorr {
			correctCount++
		} else {
			wrongQuestions = append(wrongQuestions, q.ID)
		}

		records = append(records, model.UserRecord{
			UserID:          userID,
			BankID:          req.BankID,
			QuestionID:      q.ID,
			UserAnswer:      userAns,
			IsCorrect:       isCorr,
			DurationSeconds: req.TotalSeconds / max(1, totalQ),
			CreatedAt:       now,
		})
	}

	// 批量落库
	if len(records) > 0 {
		_ = s.db.Create(&records).Error
	}

	// 批量记录错题
	for _, qid := range wrongQuestions {
		userErr := model.UserError{
			UserID:      userID,
			BankID:      req.BankID,
			QuestionID:  qid,
			WrongCount:  1,
			IsMastered:  false,
			LastWrongAt: now,
		}
		s.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "question_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"wrong_count":   gorm.Expr("user_errors.wrong_count + 1"),
				"is_mastered":   false,
				"last_wrong_at": now,
			}),
		}).Create(&userErr)
	}

	wrongCount := answeredCount - correctCount
	accuracy := 0.0
	if answeredCount > 0 {
		accuracy = float64(correctCount) / float64(answeredCount) * 100
	}
	score := float64(correctCount) / float64(totalQ) * 100

	return &SubmitExamResult{
		TotalQuestions: totalQ,
		AnsweredCount:  answeredCount,
		CorrectCount:   correctCount,
		WrongCount:     wrongCount,
		Score:          score,
		AccuracyRate:   accuracy,
		TotalSeconds:   req.TotalSeconds,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
