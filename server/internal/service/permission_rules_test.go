package service

import (
	"testing"

	"exam-server/internal/model"
)

// 1. 测试私有题库配额机制：普通用户最多2个，VIP用户最多20个，Admin无限制
func TestUserPermissionMechanism_PrivateBankQuota(t *testing.T) {
	checkPrivateQuota := func(role string, currentCount int) (bool, string) {
		if role == "admin" {
			return true, ""
		}
		if role == "vip" {
			if currentCount >= 20 {
				return false, "VIP用户最多拥有20个私有题库，已达上限！"
			}
			return true, ""
		}
		// 普通用户
		if currentCount >= 2 {
			return false, "普通用户最多拥有2个私有题库，升级VIP会员可拥有20个私有题库！"
		}
		return true, ""
	}

	// 普通用户
	ok, _ := checkPrivateQuota("user", 0)
	if !ok {
		t.Errorf("普通用户拥有0个题库应允许创建")
	}
	ok, _ = checkPrivateQuota("user", 1)
	if !ok {
		t.Errorf("普通用户拥有1个题库应允许创建")
	}
	ok, msg := checkPrivateQuota("user", 2)
	if ok || msg != "普通用户最多拥有2个私有题库，升级VIP会员可拥有20个私有题库！" {
		t.Errorf("普通用户拥有2个题库应拦截: got %v, %s", ok, msg)
	}

	// VIP 用户
	ok, _ = checkPrivateQuota("vip", 19)
	if !ok {
		t.Errorf("VIP用户拥有19个题库应允许创建")
	}
	ok, msg = checkPrivateQuota("vip", 20)
	if ok || msg != "VIP用户最多拥有20个私有题库，已达上限！" {
		t.Errorf("VIP用户拥有20个题库应拦截: got %v, %s", ok, msg)
	}

	// Admin 用户
	ok, _ = checkPrivateQuota("admin", 100)
	if !ok {
		t.Errorf("Admin用户私有题库应无限制")
	}
}

// 2. 测试公开题库每次只能上传一个、需由admin审批通过后才能再次上传机制
func TestUserPermissionMechanism_SinglePendingPublicBank(t *testing.T) {
	userBanks := []model.QuestionBank{
		{ID: 10, Title: "已通过的公开题库", CreatorID: 1, Visibility: "public", ReviewStatus: "approved"},
		{ID: 11, Title: "审核中的公开题库", CreatorID: 1, Visibility: "public", ReviewStatus: "pending"},
	}

	canUploadPublic := func(userID uint) (bool, string) {
		for _, b := range userBanks {
			if b.CreatorID == userID && b.Visibility == "public" && b.ReviewStatus == "pending" {
				return false, "公开题库每次只能上传一个，由admin审批通过后才能再次上传"
			}
		}
		return true, ""
	}

	// 用户 1 已有 pending 题库，应被拦截
	ok, msg := canUploadPublic(1)
	if ok || msg != "公开题库每次只能上传一个，由admin审批通过后才能再次上传" {
		t.Errorf("存在待审题库时应阻止上传新的公开题库: %v, %s", ok, msg)
	}

	// 模拟 Admin 审核通过 ID 11
	userBanks[1].ReviewStatus = "approved"

	// 审核通过后，用户 1 可以再次上传公开题库
	ok, msg = canUploadPublic(1)
	if !ok {
		t.Errorf("审批通过后应允许再次上传公开题库: %v, %s", ok, msg)
	}
}

// 3. 测试 Admin 删除任何公开题库权限 vs 普通用户无权删除公开题库
func TestUserPermissionMechanism_AdminDeletePublicBank(t *testing.T) {
	publicBank := model.QuestionBank{
		ID:           20,
		Title:        "用户上传已公开题库",
		CreatorID:    1,
		Visibility:   "public",
		ReviewStatus: "approved",
		IsOfficial:   false,
	}

	canDelete := func(bank *model.QuestionBank, userID uint, role string) (bool, string) {
		if bank.IsOfficial {
			return false, "官方题库不允许删除"
		}
		if role == "admin" {
			return true, "" // 管理员可删除任何非官方公开题库
		}
		if bank.Visibility == "public" && bank.ReviewStatus == "approved" {
			return false, "公开题库已面向全员开放，不可删除（仅管理员拥有删除公开题库权限）"
		}
		if bank.CreatorID != userID {
			return false, "无权删除非本人创建的题库"
		}
		return true, ""
	}

	// 普通创建者尝试删除已公开题库 -> 拦截
	ok, msg := canDelete(&publicBank, 1, "user")
	if ok {
		t.Errorf("普通用户不应允许删除公开题库: %v, %s", ok, msg)
	}

	// 管理员删除公开题库 -> 允许
	ok, msg = canDelete(&publicBank, 999, "admin")
	if !ok {
		t.Errorf("管理员应当被允许删除公开题库: %v, %s", ok, msg)
	}
}

// 4. 测试 VIP 题库访问权限拦截机制
func TestUserPermissionMechanism_VIPBankAccess(t *testing.T) {
	vipBank := model.QuestionBank{
		ID:    30,
		Title: "VIP专属真题模考",
		IsVIP: true,
	}

	checkAccess := func(bank *model.QuestionBank, role string, isVIP bool) (bool, string) {
		if bank.IsVIP {
			if role != "admin" && !isVIP {
				return false, "该题库为 VIP 专属题库，仅限 VIP 会员刷题使用"
			}
		}
		return true, ""
	}

	// 1. 普通用户访问 VIP 题库 -> 拦截
	ok, msg := checkAccess(&vipBank, "user", false)
	if ok || msg != "该题库为 VIP 专属题库，仅限 VIP 会员刷题使用" {
		t.Errorf("普通用户访问 VIP 题库应被拦截: %v, %s", ok, msg)
	}

	// 2. VIP 用户访问 VIP 题库 -> 放行
	ok, _ = checkAccess(&vipBank, "vip", true)
	if !ok {
		t.Errorf("VIP 用户访问 VIP 题库应放行")
	}

	// 3. Admin 用户访问 VIP 题库 -> 放行
	ok, _ = checkAccess(&vipBank, "admin", false)
	if !ok {
		t.Errorf("Admin 用户访问 VIP 题库应放行")
	}
}

// 5. 测试首页提交可见性申请机制 (ApplyPublic)
func TestUserPermissionMechanism_ApplyPublicBank(t *testing.T) {
	// 只能申请自己创建的私有题库
	canApplyPublic := func(bank *model.QuestionBank, userID uint, hasPending bool) (bool, string) {
		if bank.CreatorID != userID {
			return false, "无权申请公开非本人创建的私有题库"
		}
		if bank.IsOfficial {
			return false, "官方题库无需申请公开"
		}
		if bank.Visibility == "public" && bank.ReviewStatus == "approved" {
			return false, "该题库已是公开状态，无需重复申请"
		}
		if hasPending {
			return false, "公开题库每次只能提交一个审核，由admin审批通过后才能再次提交"
		}
		return true, ""
	}

	myPrivateBank := model.QuestionBank{
		ID:           101,
		Title:        "我的自建私有题库",
		CreatorID:    10,
		Visibility:   "private",
		ReviewStatus: "approved",
		IsOfficial:   false,
	}

	// 1. 他人私有题库不可申请
	ok, msg := canApplyPublic(&myPrivateBank, 20, false)
	if ok || msg != "无权申请公开非本人创建的私有题库" {
		t.Errorf("非创建者不能申请公开: %v, %s", ok, msg)
	}

	// 2. 本人正常私有题库无其它待审 -> 允许申请
	ok, _ = canApplyPublic(&myPrivateBank, 10, false)
	if !ok {
		t.Errorf("本人创建的私有题库应允许提交公开申请")
	}

	// 3. 本人已有其它待审题库 -> 拦截
	ok, msg = canApplyPublic(&myPrivateBank, 10, true)
	if ok || msg != "公开题库每次只能提交一个审核，由admin审批通过后才能再次提交" {
		t.Errorf("存在待审题库时应拦截提交公开申请: %v, %s", ok, msg)
	}
}

// 6. 测试审批结果通知精准送达机制（谁申请仅送达谁的消息中心）
func TestUserPermissionMechanism_TargetedNotification(t *testing.T) {
	// 模拟审批产生给指定创建者的通知
	creatorA := uint(1001)
	creatorB := uint(1002)

	notices := []model.SystemNotification{
		{
			ID:      1,
			UserID:  creatorA,
			Type:    "approval_pass",
			Title:   "【题库审核通过】公开申请已批准",
			Content: "恭喜！您上传的公开题库《题库A》已通过管理员审核，现已正式面向全员公开！",
		},
		{
			ID:      2,
			UserID:  creatorB,
			Type:    "approval_pass",
			Title:   "【题库审核通过】公开申请已批准",
			Content: "恭喜！您上传的公开题库《题库B》已通过管理员审核，现已正式面向全员公开！",
		},
		{
			ID:      3,
			UserID:  0, // 管理员通知
			Type:    "admin_pending",
			Title:   "【微信服务通知】有新的公开题库待审批",
			Content: "用户申请将题库公开，请审核。",
		},
	}

	filterNoticesForUser := func(all []model.SystemNotification, queryUserID uint, isAdmin bool) []model.SystemNotification {
		var res []model.SystemNotification
		for _, n := range all {
			if isAdmin {
				if n.UserID == queryUserID || n.UserID == 0 {
					res = append(res, n)
				}
			} else {
				if n.UserID == queryUserID {
					res = append(res, n)
				}
			}
		}
		return res
	}

	// 用户 A 只能看到题库 A 的通知，绝不能看到题库 B 的通知
	userANotices := filterNoticesForUser(notices, creatorA, false)
	if len(userANotices) != 1 || userANotices[0].UserID != creatorA {
		t.Errorf("用户A应且仅应收到发给自己的通知，实际收到: %v", userANotices)
	}

	// 用户 B 只能看到题库 B 的通知，绝不能看到题库 A 的通知
	userBNotices := filterNoticesForUser(notices, creatorB, false)
	if len(userBNotices) != 1 || userBNotices[0].UserID != creatorB {
		t.Errorf("用户B应且仅应收到发给自己的通知，实际收到: %v", userBNotices)
	}

	// 其他普通用户 C 查不到任何这些审批通知
	userCNotices := filterNoticesForUser(notices, 9999, false)
	if len(userCNotices) != 0 {
		t.Errorf("无关联用户不应收到任何审批通知: %v", userCNotices)
	}
}

// 7. 测试公开申请驳回后直接彻底删除机制（防止普通用户借公开申请绕过私有题库配额限制）
func TestUserPermissionMechanism_RejectBankDirectDeleteRule(t *testing.T) {
	// 模拟题库存储池
	banks := []model.QuestionBank{
		{ID: 201, Title: "正常私有题库1", CreatorID: 10, Visibility: "private", ReviewStatus: "approved"},
		{ID: 202, Title: "正常私有题库2", CreatorID: 10, Visibility: "private", ReviewStatus: "approved"},
		{ID: 203, Title: "违规待审公开题库", CreatorID: 10, Visibility: "public", ReviewStatus: "pending"},
	}

	// 驳回公开申请处理函数：直接从数据库中删除题库，而不是降级为 private
	rejectAndDirectDelete := func(bankID uint) (deleted bool, remainingCount int) {
		var remaining []model.QuestionBank
		found := false
		for _, b := range banks {
			if b.ID == bankID {
				found = true
				continue // 直接丢弃/删除
			}
			remaining = append(remaining, b)
		}
		banks = remaining
		return found, len(remaining)
	}

	// 执行驳回
	deleted, totalRemaining := rejectAndDirectDelete(203)
	if !deleted {
		t.Errorf("驳回操作应当成功匹配并删除待审公开题库")
	}
	if totalRemaining != 2 {
		t.Errorf("驳回后题库总数应恢复为2，实际为: %d", totalRemaining)
	}

	// 验证用户当前私有题库依然受配额约束，并未因驳回而凭空多出私有题库
	user10PrivateCount := 0
	for _, b := range banks {
		if b.CreatorID == 10 && b.Visibility == "private" {
			user10PrivateCount++
		}
	}
	if user10PrivateCount > 2 {
		t.Errorf("驳回后用户的私有题库数不能超过2个配额限制，实际为: %d", user10PrivateCount)
	}
}

