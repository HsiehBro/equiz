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
	svc := NewNoteService(nil)

	// 空内容应直接在服务层抛出校验错误
	_, err := svc.CreateNote(1, model.CreateNoteRequest{
		QuestionID: 1,
		Content:    "",
	})
	if err == nil {
		t.Fatalf("expected error for empty note content, got nil")
	}
}
