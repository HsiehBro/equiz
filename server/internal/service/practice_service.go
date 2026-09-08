package service

import (
	"errors"
	"math"
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

	// 题目的题库归属严格以试题实际所属题库为准
	bankID := q.BankID
	if bankID == 0 {
		bankID = req.BankID
	}

	// 事务内记录作答与错题集状态
	err := s.db.Transaction(func(tx *gorm.DB) error {
		record := model.UserRecord{
			UserID:          userID,
			BankID:          bankID,
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
						BankID:      bankID,
						QuestionID:  req.QuestionID,
						WrongCount:  1,
						IsMastered:  false,
						LastWrongAt: time.Now(),
					}
					if err := tx.Create(&userErr).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				// 如果已存在该错题，更新计数、校正所属题库并重置掌握状态
				userErr.BankID = bankID
				userErr.WrongCount += 1
				userErr.IsMastered = false
				userErr.LastWrongAt = time.Now()
				if err := tx.Save(&userErr).Error; err != nil {
					return err
				}
			}
		}

		// 检查并更新当前活跃学习规划的打卡状态
		var activePlan model.StudyPlan
		if err := tx.Where("user_id = ? AND bank_id = ? AND is_active = ?", userID, bankID, true).First(&activePlan).Error; err == nil {
			now := time.Now()
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			var todayCount int64
			tx.Model(&model.UserRecord{}).Where("user_id = ? AND bank_id = ? AND created_at >= ?", userID, bankID, todayStart).Count(&todayCount)
			todayStr := now.Format("2006-01-02")
			if int(todayCount) >= activePlan.DailyGoal && activePlan.LastCheckInDate != todayStr {
				activePlan.LastCheckInDate = todayStr
				activePlan.CheckInDays += 1
				_ = tx.Save(&activePlan).Error
			}
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

type UserStatsResult struct {
	TotalQuestions   int `json:"total_questions"`
	CorrectQuestions int `json:"correct_questions"`
	AccuracyRate     int `json:"accuracy_rate"`
	CheckInDays      int `json:"check_in_days"`
}

// GetUserStats 获取用户学习数据看板核心统计指标
// 1. 累计打卡天数：由当前活跃计划提供
// 2. 累计答题：统计 user_records 总量（模拟考试写入 mock_records，严格排他）
// 3. 平均正确率：累计答对题数 / 累计答题题数 * 100，初始无记录时为 100
func (s *PracticeService) GetUserStats(userID uint) (*UserStatsResult, error) {
	var totalQuestions int64
	if err := s.db.Model(&model.UserRecord{}).Where("user_id = ?", userID).Count(&totalQuestions).Error; err != nil {
		return nil, err
	}

	var correctQuestions int64
	if err := s.db.Model(&model.UserRecord{}).Where("user_id = ? AND is_correct = ?", userID, true).Count(&correctQuestions).Error; err != nil {
		return nil, err
	}

	accuracyRate := 100
	if totalQuestions > 0 {
		accuracyRate = int(math.Round((float64(correctQuestions) / float64(totalQuestions)) * 100))
	}

	var activePlan model.StudyPlan
	checkInDays := 0
	if err := s.db.Where("user_id = ? AND is_active = ?", userID, true).First(&activePlan).Error; err == nil {
		checkInDays = activePlan.CheckInDays
	} else {
		// 若无显式激活计划，查询最新一条规划
		if err2 := s.db.Where("user_id = ?", userID).Order("updated_at DESC").First(&activePlan).Error; err2 == nil {
			checkInDays = activePlan.CheckInDays
		}
	}

	return &UserStatsResult{
		TotalQuestions:   int(totalQuestions),
		CorrectQuestions: int(correctQuestions),
		AccuracyRate:     accuracyRate,
		CheckInDays:      checkInDays,
	}, nil
}

