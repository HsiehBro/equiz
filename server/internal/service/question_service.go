package service

import (
	"exam-server/internal/model"

	"gorm.io/gorm"
)

type QuestionService struct {
	db *gorm.DB
}

func NewQuestionService(db *gorm.DB) *QuestionService {
	return &QuestionService{db: db}
}

// GetQuestionsByBankID 获取指定题库下的全部试题，并注水当前用户的最新作答和收藏状态
func (s *QuestionService) GetQuestionsByBankID(bankID uint, userID uint) ([]model.Question, error) {
	var questions []model.Question
	err := s.db.Where("bank_id = ?", bankID).Order("sort_order ASC, id ASC").Find(&questions).Error
	if err != nil {
		return nil, err
	}

	if userID > 0 && len(questions) > 0 {
		// 批量查询当前用户的收藏状态
		var favorites []model.UserFavorite
		s.db.Where("user_id = ? AND bank_id = ?", userID, bankID).Find(&favorites)
		favMap := make(map[uint]bool)
		for _, f := range favorites {
			favMap[f.QuestionID] = true
		}

		// 批量查询最新作答记录
		var records []model.UserRecord
		s.db.Where("user_id = ? AND bank_id = ?", userID, bankID).
			Order("id ASC").
			Find(&records)
		recordMap := make(map[uint][]string)
		for _, r := range records {
			recordMap[r.QuestionID] = r.UserAnswer
		}

		for i := range questions {
			qid := questions[i].ID
			questions[i].IsBookmarked = favMap[qid]
			if ans, exists := recordMap[qid]; exists {
				questions[i].UserAnswer = ans
			}
		}
	}

	return questions, nil
}

func (s *QuestionService) GetQuestionByID(questionID uint, userID uint) (*model.Question, error) {
	var q model.Question
	if err := s.db.First(&q, questionID).Error; err != nil {
		return nil, err
	}

	if userID > 0 {
		var fav model.UserFavorite
		if err := s.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&fav).Error; err == nil {
			q.IsBookmarked = true
		}

		var rec model.UserRecord
		if err := s.db.Where("user_id = ? AND question_id = ?", userID, questionID).Order("created_at DESC").First(&rec).Error; err == nil {
			q.UserAnswer = rec.UserAnswer
		}
	}

	return &q, nil
}
