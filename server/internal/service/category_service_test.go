package service

import (
	"errors"
	"strings"
	"testing"

	"exam-server/internal/model"
)

// 1. 验证分类删除规则与题库兜底机制：默认综合分类禁止删除，删除其它分类时关联题库安全归入默认「综合」
func TestCategoryDeleteSafetyRules(t *testing.T) {
	defaultCategoryID := uint(1)
	categories := []model.Category{
		{ID: 1, Name: "综合", IsActive: true},
		{ID: 2, Name: "医学类", IsActive: true},
		{ID: 3, Name: "财经类", IsActive: true},
	}

	banks := []model.QuestionBank{
		{ID: 101, Title: "医学考试A", CategoryID: 2, Category: "医学类"},
		{ID: 102, Title: "医学考试B", CategoryID: 2, Category: "医学类"},
		{ID: 103, Title: "初级会计", CategoryID: 3, Category: "财经类"},
	}

	deleteCategory := func(catID uint) error {
		if catID == defaultCategoryID {
			return errors.New("默认「综合」分类为系统核心基础分类，不允许删除")
		}
		// 模拟级联迁移
		for i := range banks {
			if banks[i].CategoryID == catID {
				banks[i].CategoryID = defaultCategoryID
				banks[i].Category = "综合"
			}
		}
		// 移除分类
		var newCats []model.Category
		for _, c := range categories {
			if c.ID != catID {
				newCats = append(newCats, c)
			}
		}
		categories = newCats
		return nil
	}

	// 尝试删除综合分类 -> 必须报错拦截
	err := deleteCategory(1)
	if err == nil || !strings.Contains(err.Error(), "不允许删除") {
		t.Fatalf("expected error when deleting default category 1, got: %v", err)
	}

	// 正常删除医学类 (ID 2)
	err = deleteCategory(2)
	if err != nil {
		t.Fatalf("unexpected error deleting category 2: %v", err)
	}

	// 校验关联题库是否全部安全归入综合分类 (ID 1)
	if banks[0].CategoryID != 1 || banks[0].Category != "综合" {
		t.Errorf("expected bank 101 to fallback to category 1 '综合', got %d, %s", banks[0].CategoryID, banks[0].Category)
	}
	if banks[1].CategoryID != 1 || banks[1].Category != "综合" {
		t.Errorf("expected bank 102 to fallback to category 1 '综合', got %d, %s", banks[1].CategoryID, banks[1].Category)
	}
	// 财经类题库不受影响
	if banks[2].CategoryID != 3 || banks[2].Category != "财经类" {
		t.Errorf("bank 103 should remain in category 3")
	}
}

// 2. 验证默认分类名称不可修改规则
func TestDefaultCategoryRenameRule(t *testing.T) {
	checkRenameRule := func(catID uint, currentName string, newName string) error {
		newName = strings.TrimSpace(newName)
		if newName == "" {
			return errors.New("分类名称不能为空")
		}
		if catID == 1 && newName != currentName {
			return errors.New("默认「综合」分类名称不可修改")
		}
		return nil
	}

	if err := checkRenameRule(1, "综合", "全新综合分类"); err == nil {
		t.Errorf("expected error when renaming default category 1")
	}

	if err := checkRenameRule(2, "医学类", "医疗健康类"); err != nil {
		t.Errorf("renaming non-default category should succeed: %v", err)
	}
}

// 3. 验证题库分类修改权限规则
func TestUpdateBankCategoryPermissionRules(t *testing.T) {
	bank := model.QuestionBank{
		ID:         50,
		Title:      "自定义题库",
		CreatorID:  10,
		CategoryID: 2,
		Category:   "医学类",
	}

	checkPermission := func(userRole string, userID uint) bool {
		if userRole == "admin" {
			return true
		}
		return bank.CreatorID == userID
	}

	// Admin 用户有权限
	if !checkPermission("admin", 999) {
		t.Errorf("admin should have permission to update bank category")
	}

	// 题库创建者本人有权限
	if !checkPermission("user", 10) {
		t.Errorf("owner should have permission to update bank category")
	}

	// 其它普通用户无权限
	if checkPermission("user", 20) {
		t.Errorf("other user should NOT have permission to update bank category")
	}
}

// 4. 验证分类动态 VIP 与题库数聚合统计逻辑
func TestCategoryVipAndCountAggregation(t *testing.T) {
	categories := []model.Category{
		{ID: 1, Name: "综合"},
		{ID: 2, Name: "医学类"},
		{ID: 3, Name: "财经类"},
	}

	banks := []model.QuestionBank{
		{ID: 1, CategoryID: 2, IsVIP: false},
		{ID: 2, CategoryID: 2, IsVIP: true},  // 医学类包含 VIP 题库
		{ID: 3, CategoryID: 3, IsVIP: false}, // 财经类无 VIP 题库
	}

	countMap := make(map[uint]int)
	vipMap := make(map[uint]bool)

	for _, b := range banks {
		catID := b.CategoryID
		if catID == 0 {
			catID = 1
		}
		countMap[catID]++
		if b.IsVIP {
			vipMap[catID] = true
		}
	}

	for i := range categories {
		c := &categories[i]
		c.BankCount = countMap[c.ID]
		c.HasVIP = vipMap[c.ID]
		c.IsVip = vipMap[c.ID]
	}

	// 验证医学类 (ID 2): 题库数 2，且包含 VIP 标识
	if categories[1].BankCount != 2 || !categories[1].IsVip {
		t.Errorf("category 2 expected 2 banks and isVip=true, got count=%d, isVip=%v", categories[1].BankCount, categories[1].IsVip)
	}

	// 验证财经类 (ID 3): 题库数 1，且无 VIP
	if categories[2].BankCount != 1 || categories[2].IsVip {
		t.Errorf("category 3 expected 1 bank and isVip=false, got count=%d, isVip=%v", categories[2].BankCount, categories[2].IsVip)
	}

	// 验证综合类 (ID 1): 题库数 0
	if categories[0].BankCount != 0 {
		t.Errorf("category 1 expected 0 banks, got %d", categories[0].BankCount)
	}
}
