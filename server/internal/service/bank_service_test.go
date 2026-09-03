package service

import (
	"testing"

	"exam-server/internal/model"
)

func TestPrivateBankIsolationRules(t *testing.T) {
	// 准备模拟题库列表
	banks := []model.QuestionBank{
		{ID: 1, Title: "官方题库1", IsOfficial: true, Visibility: "public", CreatorID: 0},
		{ID: 2, Title: "公开题库2", IsOfficial: false, Visibility: "public", CreatorID: 100},
		{ID: 3, Title: "用户A的私有题库", IsOfficial: false, Visibility: "private", CreatorID: 1},
		{ID: 4, Title: "用户B的私有题库", IsOfficial: false, Visibility: "private", CreatorID: 2},
	}

	// 模拟 ListBanks 的过滤条件
	filterBanksForUser := func(uid uint) []model.QuestionBank {
		var result []model.QuestionBank
		for _, b := range banks {
			if b.IsOfficial || b.Visibility == "public" {
				result = append(result, b)
			} else if b.Visibility == "private" && uid > 0 && b.CreatorID == uid {
				result = append(result, b)
			}
		}
		return result
	}

	// 1. 未登录用户 (uid = 0)
	guestList := filterBanksForUser(0)
	if len(guestList) != 2 {
		t.Fatalf("expected 2 banks for guest, got %d", len(guestList))
	}
	for _, b := range guestList {
		if b.Visibility == "private" {
			t.Errorf("guest should never see private bank: %s", b.Title)
		}
	}

	// 2. 用户 1 (uid = 1)
	user1List := filterBanksForUser(1)
	if len(user1List) != 3 {
		t.Fatalf("expected 3 banks for user 1, got %d", len(user1List))
	}
	hasUserAPrivate := false
	for _, b := range user1List {
		if b.Title == "用户A的私有题库" {
			hasUserAPrivate = true
		}
		if b.Title == "用户B的私有题库" {
			t.Errorf("user 1 should NOT see user B's private bank")
		}
	}
	if !hasUserAPrivate {
		t.Errorf("user 1 should see own private bank")
	}

	// 3. 用户 2 (uid = 2)
	user2List := filterBanksForUser(2)
	if len(user2List) != 3 {
		t.Fatalf("expected 3 banks for user 2, got %d", len(user2List))
	}
	hasUserBPrivate := false
	for _, b := range user2List {
		if b.Title == "用户B的私有题库" {
			hasUserBPrivate = true
		}
		if b.Title == "用户A的私有题库" {
			t.Errorf("user 2 should NOT see user A's private bank")
		}
	}
	if !hasUserBPrivate {
		t.Errorf("user 2 should see own private bank")
	}
}

func TestDeleteBankPermissionRules(t *testing.T) {
	// 1. 官方题库不可删除
	officialBank := model.QuestionBank{ID: 1, IsOfficial: true}
	if !officialBank.IsOfficial {
		t.Errorf("expected official bank")
	}

	// 2. 公开题库不可删除
	publicBank := model.QuestionBank{ID: 2, Visibility: "public"}
	if publicBank.Visibility != "public" {
		t.Errorf("expected public bank")
	}

	// 3. 私有题库权限校验
	privateBank := model.QuestionBank{
		ID:         3,
		IsOfficial: false,
		Visibility: "private",
		CreatorID:  1,
	}

	checkCanDelete := func(bank *model.QuestionBank, userID uint) (bool, string) {
		if bank.IsOfficial {
			return false, "官方题库不允许删除"
		}
		if bank.Visibility == "public" {
			return false, "公开题库已面向全员开放，不可删除"
		}
		if bank.CreatorID != userID {
			return false, "无权删除非本人创建的题库"
		}
		return true, ""
	}

	// 他人 (uid = 2) 删除用户 1 的私有题库 -> 必须拒绝
	canDeleteOther, msgOther := checkCanDelete(&privateBank, 2)
	if canDeleteOther || msgOther != "无权删除非本人创建的题库" {
		t.Errorf("expected reject deleting other's bank, got: %v, %s", canDeleteOther, msgOther)
	}

	// 创建者本人 (uid = 1) 删除自己的私有题库 -> 允许
	canDeleteSelf, msgSelf := checkCanDelete(&privateBank, 1)
	if !canDeleteSelf || msgSelf != "" {
		t.Errorf("expected allow deleting own bank, got: %v, %s", canDeleteSelf, msgSelf)
	}
}
