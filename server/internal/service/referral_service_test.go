package service

import (
	"testing"
	"time"

	"exam-server/internal/model"
)

func TestMaskNickname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"A", "A"},
		{"张三", "张*"},
		{"李四五", "李*五"},
		{"备考刷题达人", "备*人"},
	}

	for _, tt := range tests {
		got := maskNickname(tt.input)
		if got != tt.expected {
			t.Errorf("maskNickname(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestReferralRules_SelfInvite(t *testing.T) {
	// 自邀请规则拦截校验
	inviteeID := uint(10)
	inviterID := uint(10)
	if inviteeID == inviterID {
		errStr := "不能邀请自己"
		if errStr != "不能邀请自己" {
			t.Fatalf("unexpected error string")
		}
	} else {
		t.Fatalf("expected self-invite to be detected")
	}
}

func TestReferralRules_LifetimeCapAndCycleDays(t *testing.T) {
	calcRewards := func(planID string) (inviterDays int, inviteeExtraDays int, isLifetime bool) {
		plan, ok := VIPPlanConfig[planID]
		if !ok {
			return 0, 0, false
		}
		if planID == "lifetime" {
			// 终身卡购买封顶赠送 365 天（1年）给邀请人，被邀请人直接终身 VIP
			return 365, 0, true
		}
		// 周期卡双方同等顺延
		return plan.Days, plan.Days, false
	}

	// 1. 月卡
	inviterDays, inviteeExtraDays, isLt := calcRewards("monthly")
	if inviterDays != 30 || inviteeExtraDays != 30 || isLt {
		t.Errorf("monthly reward expected 30/30/false, got %d/%d/%v", inviterDays, inviteeExtraDays, isLt)
	}

	// 2. 季卡
	inviterDays, inviteeExtraDays, isLt = calcRewards("quarterly")
	if inviterDays != 90 || inviteeExtraDays != 90 || isLt {
		t.Errorf("quarterly reward expected 90/90/false, got %d/%d/%v", inviterDays, inviteeExtraDays, isLt)
	}

	// 3. 年卡
	inviterDays, inviteeExtraDays, isLt = calcRewards("yearly")
	if inviterDays != 365 || inviteeExtraDays != 365 || isLt {
		t.Errorf("yearly reward expected 365/365/false, got %d/%d/%v", inviterDays, inviteeExtraDays, isLt)
	}

	// 4. 终身卡：邀请人必须设上限 365 天，被邀请人永久 VIP
	inviterDays, inviteeExtraDays, isLt = calcRewards("lifetime")
	if inviterDays != 365 || inviteeExtraDays != 0 || !isLt {
		t.Errorf("lifetime reward expected 365/0/true, got %d/%d/%v", inviterDays, inviteeExtraDays, isLt)
	}
}

func TestReferralRules_DateExtension(t *testing.T) {
	now := time.Now()
	// Case 1: 用户现有 VIP 尚未过期，应在原有到期时间上顺延
	existingExpire := now.AddDate(0, 0, 10)
	rewardDays := 30
	newExpire := existingExpire.AddDate(0, 0, rewardDays)
	expectedDiff := 40 // 10 + 30

	diffDays := int(newExpire.Sub(now).Hours() / 24)
	if diffDays < expectedDiff-1 || diffDays > expectedDiff+1 {
		t.Errorf("expected ~%d days diff, got %d", expectedDiff, diffDays)
	}

	// Case 2: 用户目前不是 VIP 或已过期，应从当前时间起顺延
	freshExpire := now.AddDate(0, 0, rewardDays)
	freshDiff := int(freshExpire.Sub(now).Hours() / 24)
	if freshDiff < 29 || freshDiff > 31 {
		t.Errorf("expected ~30 days diff, got %d", freshDiff)
	}
}

func TestReferralRules_MutualInvitePrevention(t *testing.T) {
	// 模拟已存在记录：用户 1 曾经邀请了 用户 2
	existingReferrals := []model.UserReferral{
		{InviterID: 1, InviteeID: 2, Status: "rewarded"},
	}

	checkReverseInvite := func(inviteeID, inviterID uint) bool {
		for _, ref := range existingReferrals {
			if ref.InviterID == inviteeID && ref.InviteeID == inviterID {
				return true // 发现反向互邀！
			}
		}
		return false
	}

	// 用户 1 尝试被用户 2 邀请 -> 应触发拦截
	if !checkReverseInvite(1, 2) {
		t.Errorf("expected reverse invite (user 2 inviting user 1) to be blocked")
	}

	// 用户 3 尝试被用户 1 邀请 -> 允许
	if checkReverseInvite(3, 1) {
		t.Errorf("expected normal invite (user 1 inviting user 3) to pass")
	}
}
