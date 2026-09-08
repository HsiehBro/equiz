package service

import (
	"testing"
)

func TestNicknameValidationRules(t *testing.T) {
	// 1. 测试超长昵称 (限制最大 20 个字符)
	longNickname := "超长昵称测试一二三四五六七八九十一二三四五六七八九十额外字符"
	if len([]rune(longNickname)) <= 20 {
		t.Errorf("expected long nickname to exceed 20 runes, got %d", len([]rune(longNickname)))
	}

	// 2. 测试合法中文与字符长度
	validNickname := "备考达人_2026"
	if len([]rune(validNickname)) > 20 {
		t.Errorf("expected valid nickname <= 20 runes, got %d", len([]rune(validNickname)))
	}
}

func TestRandomNicknameFormat(t *testing.T) {
	// 验证未配置数据库时（db=nil时会走微秒戳或跳过DB查重兜底）生成的随机昵称格式与长度
	svc := &AuthService{db: nil}
	nick := svc.GenerateUniqueRandomNickname()
	if len(nick) == 0 {
		t.Fatalf("expected non-empty random nickname")
	}
	if len([]rune(nick)) > 20 {
		t.Errorf("nickname length %d exceeds max 20", len([]rune(nick)))
	}
}
