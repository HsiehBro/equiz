package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type NoteService struct {
	db        *gorm.DB
	wechatSec *WeChatSecurityService
}

func NewNoteService(db *gorm.DB, wechatSec *WeChatSecurityService) *NoteService {
	return &NoteService{db: db, wechatSec: wechatSec}
}

// CreateNote 创建题目笔记、公开评论或针对评论的回复
func (s *NoteService) CreateNote(userID uint, req model.CreateNoteRequest) (*model.UserNote, error) {
	if req.Content == "" {
		return nil, errors.New("笔记内容不能为空")
	}

	// 检查题目是否存在
	var question model.Question
	if err := s.db.First(&question, req.QuestionID).Error; err != nil {
		return nil, errors.New("关联试题不存在")
	}

	// 题目在题库的归属严格以试题自身所属题库为准，防止客户端传参错乱
	bankID := question.BankID
	if bankID == 0 {
		bankID = req.BankID
	}

	visibility := req.Visibility
	if visibility != "private" {
		visibility = "public"
	}

	// 检查是否为对某条评论的回复
	var parentNote *model.UserNote
	if req.ParentID != nil && *req.ParentID > 0 {
		var p model.UserNote
		if err := s.db.First(&p, *req.ParentID).Error; err != nil {
			return nil, errors.New("回复的目标评论不存在")
		}
		parentNote = &p
		// 回复公开评论统一继承为公开可见
		if parentNote.Visibility == "public" {
			visibility = "public"
		}
	}

	// 公开评论及子回复强制进行双重敏感词与内容安全审查
	if visibility == "public" {
		// 第一道防线：本地 DFA 快速初筛 (0ms 纳秒级过滤，拦截硬性违规，省流量和配额)
		if hasSensitive, hitWord := CheckSensitiveContent(req.Content); hasSensitive {
			return nil, fmt.Errorf("公开评论包含敏感违规词汇【%s】，禁止发布", hitWord)
		}

		// 第二道防线：微信官方 security.msgSecCheck 云端语义合规审查 (小程序官方合规保障)
		if s.wechatSec != nil {
			var authorUser model.User
			var openID string
			if err := s.db.First(&authorUser, userID).Error; err == nil && authorUser.OpenID != "" {
				openID = authorUser.OpenID
			} else {
				openID = "mock_user"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			passed, reason, _ := s.wechatSec.CheckMsgSec(ctx, openID, req.Content)
			cancel()
			if !passed {
				return nil, errors.New(reason)
			}
		}
	}

	// 若为回复，提前查询被回复人信息与昵称，确保写入数据库持久化
	var replyToUserID *uint
	var replyToName string
	if parentNote != nil {
		replyToName = req.ReplyToAuthor
		replyToUserID = req.ReplyToUserID

		// 如果前端未提供 reply_to_user_id
		if replyToUserID == nil || *replyToUserID == 0 {
			if replyToName != "" {
				var targetUser model.User
				if err := s.db.Where("nickname = ?", replyToName).First(&targetUser).Error; err == nil {
					replyToUserID = &targetUser.ID
				}
			}
			// 如果仍未匹配到，默认目标为父级评论的作者
			if replyToUserID == nil || *replyToUserID == 0 {
				replyToUserID = &parentNote.UserID
			}
		}

		// 根据确定好的 replyToUserID 获取最新昵称
		if replyToUserID != nil && *replyToUserID > 0 {
			var u model.User
			if err := s.db.First(&u, *replyToUserID).Error; err == nil {
				replyToName = u.Nickname
			}
		} else if replyToName == "" {
			var parentUser model.User
			if err := s.db.First(&parentUser, parentNote.UserID).Error; err == nil {
				replyToName = parentUser.Nickname
			}
		}
	}

	note := model.UserNote{
		UserID:        userID,
		BankID:        bankID,
		QuestionID:    req.QuestionID,
		Content:       req.Content,
		Visibility:    visibility,
		LikeCount:     0,
		ParentID:      req.ParentID,
		ReplyToUserID: replyToUserID,
		ReplyToAuthor: replyToName,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
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
	var bank model.QuestionBank
	if err := s.db.First(&bank, bankID).Error; err == nil {
		note.Bank = &bank
		note.BankTitle = bank.Title
	}

	// 若为回复，向被回复人投递真实消息中心通知
	if parentNote != nil {
		targetUserID := parentNote.UserID
		if replyToUserID != nil && *replyToUserID > 0 {
			targetUserID = *replyToUserID
		}
		// 仅在被回复人不是自己时发送回复通知到消息中心
		if targetUserID != userID {
			replierName := note.AuthorName
			if replierName == "" {
				replierName = "考友"
			}
			notify := model.SystemNotification{
				UserID:          targetUserID,
				Type:            "comment_reply",
				Title:           fmt.Sprintf("%s 回复了你的公开评论", replierName),
				Content:         req.Content,
				RelatedID:       parentNote.ID,
				BankID:          parentNote.BankID,
				QuestionID:      parentNote.QuestionID,
				QuestionTitle:   question.Title,
				ReplierID:       userID,
				ReplierName:     replierName,
				ReplierAvatar:   note.AuthorAvatar,
				OriginalContent: parentNote.Content,
				IsRead:          false,
				CreatedAt:       time.Now(),
			}
			_ = s.db.Create(&notify).Error
		}
	}

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

	targetVis := note.Visibility
	if req.Visibility == "public" || req.Visibility == "private" {
		targetVis = req.Visibility
	}
	targetContent := note.Content
	if req.Content != "" {
		targetContent = req.Content
	}
	// 转为公开或已公开的评论更新内容时执行双重敏感词与内容安全校验
	if targetVis == "public" {
		// 第一道防线：本地 DFA 快速初筛
		if hasSensitive, hitWord := CheckSensitiveContent(targetContent); hasSensitive {
			return nil, fmt.Errorf("公开评论包含敏感违规词汇【%s】，禁止发布", hitWord)
		}

		// 第二道防线：微信官方 security.msgSecCheck 云端审查
		if s.wechatSec != nil {
			var user model.User
			var openID string
			if err := s.db.First(&user, userID).Error; err == nil && user.OpenID != "" {
				openID = user.OpenID
			} else {
				openID = "mock_user"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			passed, reason, _ := s.wechatSec.CheckMsgSec(ctx, openID, targetContent)
			cancel()
			if !passed {
				return nil, errors.New(reason)
			}
		}
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
	err := query.Preload("User").Preload("Question").Preload("ReplyToUser").Preload("Bank").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&notes).Error
	if err != nil {
		return nil, 0, err
	}

	// 收集并校验题目题库归属，确保试题在题库中归属严格一致
	bankIDs := make([]uint, 0)
	for i := range notes {
		if notes[i].Question != nil && notes[i].Question.BankID > 0 {
			notes[i].BankID = notes[i].Question.BankID
		}
		if notes[i].BankID > 0 {
			bankIDs = append(bankIDs, notes[i].BankID)
		}
	}
	bankTitleMap := make(map[uint]string)
	if len(bankIDs) > 0 {
		var banks []model.QuestionBank
		s.db.Where("id IN ?", bankIDs).Find(&banks)
		for _, b := range banks {
			bankTitleMap[b.ID] = b.Title
		}
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
			if notes[i].Question.BankID > 0 {
				notes[i].BankID = notes[i].Question.BankID
			}
		}
		if title, ok := bankTitleMap[notes[i].BankID]; ok {
			notes[i].BankTitle = title
		} else if notes[i].Bank != nil && notes[i].Bank.Title != "" {
			notes[i].BankTitle = notes[i].Bank.Title
		}
		if notes[i].ReplyToUser != nil && notes[i].ReplyToUser.Nickname != "" {
			notes[i].ReplyToAuthor = notes[i].ReplyToUser.Nickname
		}
		notes[i].IsLiked = likedMap[notes[i].ID]
	}

	return notes, total, nil
}

// ListQuestionComments 获取指定题目的公开评论（所有人可见），按点赞数倒序与时间倒序排列，并挂载子回复
func (s *NoteService) ListQuestionComments(questionID uint, currentUserID uint, page, pageSize int) ([]model.UserNote, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// 仅筛选顶级公开评论（parent_id 为空或为 0）
	query := s.db.Model(&model.UserNote{}).
		Where("question_id = ? AND visibility = 'public' AND (parent_id IS NULL OR parent_id = 0)", questionID)

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

	// 批量拉取所有已查出主评论的子回复
	if len(comments) > 0 {
		var topIDs []uint
		for _, c := range comments {
			topIDs = append(topIDs, c.ID)
		}

		var subReplies []model.UserNote
		subErr := s.db.Model(&model.UserNote{}).
			Preload("User").
			Preload("ReplyToUser").
			Where("parent_id IN (?) AND visibility = 'public'", topIDs).
			Order("created_at ASC").
			Find(&subReplies).Error

		parentAuthorMap := make(map[uint]string)
		parentUserMap := make(map[uint]uint)
		for _, c := range comments {
			if c.User != nil {
				parentAuthorMap[c.ID] = c.User.Nickname
				parentUserMap[c.ID] = c.UserID
			}
		}

		replyMap := make(map[uint][]model.UserNote)
		if subErr == nil && len(subReplies) > 0 {
			for i := range subReplies {
				if subReplies[i].User != nil {
					subReplies[i].AuthorName = subReplies[i].User.Nickname
					subReplies[i].AuthorAvatar = subReplies[i].User.AvatarURL
				}
				if subReplies[i].ParentID != nil {
					pID := *subReplies[i].ParentID
					// 动态展示被回复人的最新昵称：
					// 1. 若关联了 ReplyToUser，优先使用被回复用户的当前最新昵称
					// 2. 若 ReplyToUserID 对应主评论作者，使用主评论作者最新昵称
					// 3. 若 ReplyToAuthor 为空，降级回退主评论作者昵称
					if subReplies[i].ReplyToUser != nil && subReplies[i].ReplyToUser.Nickname != "" {
						subReplies[i].ReplyToAuthor = subReplies[i].ReplyToUser.Nickname
					} else if subReplies[i].ReplyToUserID != nil && parentUserMap[pID] != 0 && *subReplies[i].ReplyToUserID == parentUserMap[pID] {
						subReplies[i].ReplyToAuthor = parentAuthorMap[pID]
					} else if subReplies[i].ReplyToAuthor == "" {
						subReplies[i].ReplyToAuthor = parentAuthorMap[pID]
					}
					replyMap[pID] = append(replyMap[pID], subReplies[i])
				}
			}
		}

		for i := range comments {
			if reps, ok := replyMap[comments[i].ID]; ok {
				comments[i].Replies = reps
			} else {
				comments[i].Replies = []model.UserNote{}
			}
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
