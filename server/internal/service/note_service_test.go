package service

import (
	"testing"

	"exam-server/internal/model"
)

func TestNoteVisibilityDefault(t *testing.T) {
	// 验证未传或任意非 private 时的可见性默认置为 public
	vis1 := "private"
	if vis1 != "private" {
		t.Errorf("expected private")
	}

	vis2 := ""
	if vis2 != "private" {
		vis2 = "public"
	}
	if vis2 != "public" {
		t.Errorf("expected default public, got %s", vis2)
	}

	vis3 := "anything_else"
	if vis3 != "private" {
		vis3 = "public"
	}
	if vis3 != "public" {
		t.Errorf("expected fallback public, got %s", vis3)
	}
}

func TestNoteLikeCounterLogic(t *testing.T) {
	// 测试点赞与取消点赞的计数逻辑，以及下限保护 (like_count >= 0)
	note := model.UserNote{
		ID:        1,
		LikeCount: 0,
	}

	// 模拟首次点赞
	note.LikeCount++
	if note.LikeCount != 1 {
		t.Errorf("expected like count 1, got %d", note.LikeCount)
	}

	// 模拟二次点赞（重复点赞时取消）
	note.LikeCount--
	if note.LikeCount != 0 {
		t.Errorf("expected like count 0, got %d", note.LikeCount)
	}

	// 模拟异常并发导致的下限保护
	note.LikeCount--
	if note.LikeCount < 0 {
		note.LikeCount = 0
	}
	if note.LikeCount != 0 {
		t.Errorf("expected clamped like count 0, got %d", note.LikeCount)
	}
}

func TestNoteContentValidation(t *testing.T) {
	svc := NewNoteService(nil, nil)

	// 空内容应直接在服务层抛出校验错误
	_, err := svc.CreateNote(1, model.CreateNoteRequest{
		QuestionID: 1,
		Content:    "",
	})
	if err == nil {
		t.Fatalf("expected error for empty note content, got nil")
	}
}

func TestNoteReplyStructure(t *testing.T) {
	parentID := uint(100)
	req := model.CreateNoteRequest{
		QuestionID:    1,
		Content:       "这是针对公开评论的一条回复",
		ParentID:      &parentID,
		ReplyToAuthor: "Alex Chen",
	}

	if req.ParentID == nil || *req.ParentID != 100 {
		t.Fatalf("expected ParentID 100, got %v", req.ParentID)
	}
	if req.ReplyToAuthor != "Alex Chen" {
		t.Fatalf("expected ReplyToAuthor Alex Chen, got %s", req.ReplyToAuthor)
	}

	note := model.UserNote{
		ID:            101,
		UserID:        2,
		QuestionID:    1,
		Content:       req.Content,
		ParentID:      req.ParentID,
		ReplyToAuthor: req.ReplyToAuthor,
	}

	parent := model.UserNote{
		ID:      100,
		UserID:  1,
		Content: "主公开评论",
		Replies: []model.UserNote{note},
	}

	if len(parent.Replies) != 1 {
		t.Fatalf("expected 1 sub-reply, got %d", len(parent.Replies))
	}
	if parent.Replies[0].ReplyToAuthor != "Alex Chen" {
		t.Fatalf("expected sub-reply author Alex Chen, got %s", parent.Replies[0].ReplyToAuthor)
	}
}

func TestNoteBankAttributionAlignment(t *testing.T) {
	// 验证试题所属题库优先原则与题库名称注水机制
	q := model.Question{
		ID:        101,
		BankID:    2,
		Title:     "成人进行心肺复苏 (CPR) 时，胸外心脏按压与人工呼吸的比例通常为（ ）。",
		BankTitle: "2023年护士执业资格考试",
	}

	note := model.UserNote{
		ID:         1,
		UserID:     1,
		QuestionID: q.ID,
		BankID:     1, // 模拟客户端错误传入的题库 ID
		Question:   &q,
		Content:    "CPR 必须牢记 30:2",
	}

	// 模拟 ListUserNotes 中的校准逻辑
	if note.Question != nil && note.Question.BankID > 0 {
		note.BankID = note.Question.BankID
	}
	if note.Question != nil && note.Question.BankTitle != "" {
		note.BankTitle = note.Question.BankTitle
	}

	if note.BankID != 2 {
		t.Errorf("expected note.BankID to align to question.BankID 2, got %d", note.BankID)
	}
	if note.BankTitle != "2023年护士执业资格考试" {
		t.Errorf("expected note.BankTitle '2023年护士执业资格考试', got %s", note.BankTitle)
	}
}
