package service

import (
	"errors"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type FavoriteService struct {
	db *gorm.DB
}

func NewFavoriteService(db *gorm.DB) *FavoriteService {
	return &FavoriteService{db: db}
}

// ToggleFavorite 切换收藏状态，返回最新是否已收藏
func (s *FavoriteService) ToggleFavorite(userID uint, questionID uint) (bool, error) {
	var q model.Question
	if err := s.db.First(&q, questionID).Error; err != nil {
		return false, errors.New("试题不存在")
	}

	var fav model.UserFavorite
	err := s.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&fav).Error
	if err == nil {
		// 已收藏，则取消收藏
		if err := s.db.Delete(&fav).Error; err != nil {
			return true, err
		}
		return false, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 未收藏，则添加收藏
		newFav := model.UserFavorite{
			UserID:     userID,
			QuestionID: questionID,
			BankID:     q.BankID,
			CreatedAt:  time.Now(),
		}
		if err := s.db.Create(&newFav).Error; err != nil {
			return false, err
		}
		return true, nil
	}

	return false, err
}

// ListFavorites 获取用户的收藏题目列表
func (s *FavoriteService) ListFavorites(userID uint, bankID uint) ([]model.UserFavorite, error) {
	var favorites []model.UserFavorite

	query := s.db.Where("user_id = ?", userID)
	if bankID > 0 {
		query = query.Where("bank_id = ?", bankID)
	}

	err := query.Preload("Question").Preload("Bank").Order("created_at DESC").Find(&favorites).Error
	if err != nil {
		return nil, err
	}

	return favorites, nil
}
