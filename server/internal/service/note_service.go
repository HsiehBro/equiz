package service

import (
	"errors"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type NoteService struct {
	db *gorm.DB
}

func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{db: db}
}

// CreateNote 创建题目笔记或公开评论
func (s *NoteService) CreateNote(userID uint, req model.CreateNoteRequest) (*model.UserNote, error) {
	if req.Content == "" {
		return nil, errors.New("笔记内容不能为空")
	}

	// 检查题目是否存在
	var question model.Question
	if err := s.db.First(&question, req.QuestionID).Error; err != nil {
		return nil, errors.New("关联试题不存在")
	}

	bankID := req.BankID
	if bankID == 0 {
		bankID = question.BankID
	}

	visibility := req.Visibility
	if visibility != "private" {
		visibility = "public"
	}

	note := model.UserNote{
		UserID:     userID,
		BankID:     bankID,
		QuestionID: req.QuestionID,
		Content:    req.Content,
		Visibility: visibility,
		LikeCount:  0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.Create(&note).Error; err != nil {
		return nil, err
	}

	// 填充附加展示字段
	var user model.User
	if err := s.db.First(&user, userID).Error; err == nil {
		note.User = &user
		note.AuthorName = user.Nickname
		note.AuthorAvatar = user.AvatarURL
	}
	note.QuestionTitle = question.Title

	return &note, nil
}

// UpdateNote 修改个人笔记内容或可见范围
func (s *NoteService) UpdateNote(userID uint, noteID uint, req model.UpdateNoteRequest) (*model.UserNote, error) {
	var note model.UserNote
	if err := s.db.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("笔记不存在或无权修改")
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Content != "" {
		updates["content"] = req.Content
		note.Content = req.Content
	}
	if req.Visibility == "public" || req.Visibility == "private" {
		updates["visibility"] = req.Visibility
		note.Visibility = req.Visibility
	}
	updates["updated_at"] = time.Now()

	if len(updates) > 1 {
		if err := s.db.Model(&note).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return &note, nil
}

// DeleteNote 删除个人笔记
func (s *NoteService) DeleteNote(userID uint, noteID uint) error {
	var note model.UserNote
	if err := s.db.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("笔记不存在或无权删除")
		}
		return err
	}

	return s.db.Delete(&note).Error
}

// ListUserNotes 获取当前用户的笔记列表（包含私密与公开），支持按题库与可见性筛选
func (s *NoteService) ListUserNotes(userID uint, bankID uint, visibility string, page, pageSize int) ([]model.UserNote, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&model.UserNote{}).Where("user_id = ?", userID)
	if bankID > 0 {
		query = query.Where("bank_id = ?", bankID)
	}
	if visibility == "public" || visibility == "private" {
		query = query.Where("visibility = ?", visibility)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var notes []model.UserNote
	offset := (page - 1) * pageSize
	err := query.Preload("User").Preload("Question").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&notes).Error
	if err != nil {
		return nil, 0, err
	}

	// 查出该用户点赞过的所有笔记 ID
	var noteIDs []uint
	for _, n := range notes {
		noteIDs = append(noteIDs, n.ID)
	}
	likedMap := make(map[uint]bool)
	if len(noteIDs) > 0 {
		var likes []model.UserNoteLike
		s.db.Where("user_id = ? AND note_id IN ?", userID, noteIDs).Find(&likes)
		for _, l := range likes {
			likedMap[l.NoteID] = true
		}
	}

	for i := range notes {
		if notes[i].User != nil {
			notes[i].AuthorName = notes[i].User.Nickname
			notes[i].AuthorAvatar = notes[i].User.AvatarURL
		}
		if notes[i].Question != nil {
			notes[i].QuestionTitle = notes[i].Question.Title
		}
		notes[i].IsLiked = likedMap[notes[i].ID]
	}

	return notes, total, nil
}

// ListQuestionComments 获取指定题目的公开评论（所有人可见），按点赞数倒序与时间倒序排列
func (s *NoteService) ListQuestionComments(questionID uint, currentUserID uint, page, pageSize int) ([]model.UserNote, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&model.UserNote{}).
		Where("question_id = ? AND visibility = 'public'", questionID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var comments []model.UserNote
	offset := (page - 1) * pageSize
	err := query.Preload("User").
		Order("like_count DESC, created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&comments).Error
	if err != nil {
		return nil, 0, err
	}

	// 若当前用户登录，查询点赞状态
	likedMap := make(map[uint]bool)
	if currentUserID > 0 && len(comments) > 0 {
		var noteIDs []uint
		for _, c := range comments {
			noteIDs = append(noteIDs, c.ID)
		}
		var likes []model.UserNoteLike
		s.db.Where("user_id = ? AND note_id IN ?", currentUserID, noteIDs).Find(&likes)
		for _, l := range likes {
			likedMap[l.NoteID] = true
		}
	}

	for i := range comments {
		if comments[i].User != nil {
			comments[i].AuthorName = comments[i].User.Nickname
			comments[i].AuthorAvatar = comments[i].User.AvatarURL
		}
		comments[i].IsLiked = likedMap[comments[i].ID]
	}

	return comments, total, nil
}

// ToggleNoteLike 切换点赞状态（防重 + 事务增减计数）
func (s *NoteService) ToggleNoteLike(userID uint, noteID uint) (bool, int, error) {
	var note model.UserNote
	if err := s.db.First(&note, noteID).Error; err != nil {
		return false, 0, errors.New("笔记不存在")
	}

	var liked bool
	var newCount int

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var like model.UserNoteLike
		err := tx.Where("user_id = ? AND note_id = ?", userID, noteID).First(&like).Error
		if err == nil {
			// 已点赞，取消点赞
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			newCount = note.LikeCount - 1
			if newCount < 0 {
				newCount = 0
			}
			if err := tx.Model(&model.UserNote{}).Where("id = ?", noteID).Update("like_count", newCount).Error; err != nil {
				return err
			}
			liked = false
			return nil
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 未点赞，添加点赞
			newLike := model.UserNoteLike{
				UserID:    userID,
				NoteID:    noteID,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			newCount = note.LikeCount + 1
			if err := tx.Model(&model.UserNote{}).Where("id = ?", noteID).Update("like_count", newCount).Error; err != nil {
				return err
			}
			liked = true
			return nil
		}

		return err
	})

	if err != nil {
		return false, 0, err
	}

	return liked, newCount, nil
}
