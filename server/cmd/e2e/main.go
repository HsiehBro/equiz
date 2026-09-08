package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type TestRunner struct {
	BaseURL     string
	Client      *http.Client
	Token       string
	UserID      uint
	TotalSteps  int
	PassedSteps int
	FailedSteps int
	Errors      []string
}

func NewTestRunner(baseURL string) *TestRunner {
	return &TestRunner{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (r *TestRunner) Request(method, path string, body interface{}, auth bool) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal json failed: %w", err)
		}
		bodyReader = bytes.NewReader(bs)
	}

	reqURL := r.BaseURL + path
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if auth && r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http execute failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read response body failed: %w", err)
	}

	return resp, respBytes, nil
}

func (r *TestRunner) RunStep(name string, fn func() error) {
	r.TotalSteps++
	start := time.Now()
	err := fn()
	elapsed := time.Since(start).Milliseconds()

	if err == nil {
		r.PassedSteps++
		fmt.Printf("  [PASS] %02d. %-48s (%d ms)\n", r.TotalSteps, name, elapsed)
	} else {
		r.FailedSteps++
		errMsg := fmt.Sprintf("Step %02d [%s] failed: %v", r.TotalSteps, name, err)
		r.Errors = append(r.Errors, errMsg)
		fmt.Printf("  [FAIL] %02d. %-48s (%d ms)\n         -> Error: %v\n", r.TotalSteps, name, elapsed, err)
	}
}

func main() {
	urlFlag := flag.String("url", "http://127.0.0.1:8080", "API Base URL")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Println("         智能刷题助手 - Go 后端 API 端到端 (E2E) 全链路自动化集成测试")
	fmt.Printf("         测试目标服务地址: %s\n", *urlFlag)
	fmt.Printf("         测试启动时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("================================================================================")

	runner := NewTestRunner(*urlFlag)

	// 数据暂存
	var vipBankID uint = 1
	var freeBankID uint = 2
	var question1ID uint
	var question1Answer []string
	var question2ID uint
	var createdNoteID uint

	// Step 1: 探活接口
	runner.RunStep("服务探活检查 (GET /health)", func() error {
		resp, body, err := runner.Request("GET", "/health", nil, false)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if !strings.Contains(string(body), "exam-server") {
			return fmt.Errorf("response body does not contain exam-server: %s", string(body))
		}
		return nil
	})

	// Step 2: 未授权访问拦截测试 (401 负向边界)
	runner.RunStep("未授权访问拦截边界测试 (GET /api/v1/auth/profile)", func() error {
		resp, _, err := runner.Request("GET", "/api/v1/auth/profile", nil, false)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusUnauthorized {
			return fmt.Errorf("expected status 401, got %d", resp.StatusCode)
		}
		return nil
	})

	// Step 3: Mock 普通用户登录与换发 Token
	testUserID := fmt.Sprintf("e2e_%d", time.Now().UnixNano())
	runner.RunStep("普通学员登录与 JWT 换发 (POST /api/v1/auth/mock-login)", func() error {
		payload := map[string]string{
			"dev_user_id": testUserID,
			"role":        "user",
			"nickname":    fmt.Sprintf("学员_%s", testUserID),
		}
		resp, body, err := runner.Request("POST", "/api/v1/auth/mock-login", payload, false)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var data struct {
			Token string `json:"token"`
			User  struct {
				ID uint `json:"id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(apiResp.Data, &data); err != nil {
			return err
		}

		if data.Token == "" || data.User.ID == 0 {
			return fmt.Errorf("token or user_id is empty")
		}

		runner.Token = data.Token
		runner.UserID = data.User.ID
		return nil
	})

	// Step 4: 登录后获取当前用户 Profile
	runner.RunStep("个人信息查询验证 (GET /api/v1/auth/profile)", func() error {
		resp, body, err := runner.Request("GET", "/api/v1/auth/profile", nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		return nil
	})

	// Step 5: 题库分类查询
	runner.RunStep("题库分类列表检索 (GET /api/v1/categories)", func() error {
		resp, body, err := runner.Request("GET", "/api/v1/categories", nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		return nil
	})

	// Step 6: 题库列表检索与分类归属识别
	runner.RunStep("题库列表获取与 VIP/免费题库校验 (GET /api/v1/banks)", func() error {
		resp, body, err := runner.Request("GET", "/api/v1/banks", nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var banks []struct {
			ID         uint   `json:"id"`
			Title      string `json:"title"`
			IsVIP      bool   `json:"is_vip"`
			TotalCount int    `json:"total_count"`
		}
		if err := json.Unmarshal(apiResp.Data, &banks); err != nil {
			return err
		}

		if len(banks) == 0 {
			return fmt.Errorf("no banks found in database; please run seed first")
		}

		for _, b := range banks {
			if b.IsVIP && vipBankID == 1 {
				vipBankID = b.ID
			} else if !b.IsVIP {
				freeBankID = b.ID
			}
		}
		return nil
	})

	// Step 7: 普通用户越权访问 VIP 题库拦截验证 (403 负向边界)
	runner.RunStep("普通学员越权访问 VIP 题库拦截 (GET /banks/:id/questions)", func() error {
		path := fmt.Sprintf("/api/v1/banks/%d/questions", vipBankID)
		resp, _, err := runner.Request("GET", path, nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusForbidden {
			return fmt.Errorf("expected 403 Forbidden for normal user accessing VIP bank, got %d", resp.StatusCode)
		}
		return nil
	})

	// Step 8: 免费题库正常作答权限验证
	runner.RunStep("普通学员访问免费题库试题 (GET /banks/:id/questions)", func() error {
		path := fmt.Sprintf("/api/v1/banks/%d/questions", freeBankID)
		resp, body, err := runner.Request("GET", path, nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		return nil
	})

	// Step 9: 学员开通/购买 VIP 会员 (POST /api/v1/vip/purchase)
	runner.RunStep("学员购买/开通连续包月 VIP (POST /api/v1/vip/purchase)", func() error {
		payload := map[string]string{
			"plan_id": "monthly",
		}
		resp, body, err := runner.Request("POST", "/api/v1/vip/purchase", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		// 刷新个人资料，确保 VIP 状态已更新
		_, bodyProf, err := runner.Request("GET", "/api/v1/auth/profile", nil, true)
		if err != nil {
			return err
		}
		var profResp APIResponse
		_ = json.Unmarshal(bodyProf, &profResp)
		var userProf struct {
			Role      string `json:"role"`
			VIPExpire string `json:"vip_expire"`
		}
		_ = json.Unmarshal(profResp.Data, &userProf)
		if userProf.Role != "vip" && userProf.Role != "admin" {
			return fmt.Errorf("user role is not vip after purchase: %s", string(bodyProf))
		}

		// 通过 mock-login 重新签发含最新 VIP claims 的 Token
		loginPayload := map[string]string{
			"dev_user_id": testUserID,
			"role":        "vip",
		}
		respLogin, bodyLogin, err := runner.Request("POST", "/api/v1/auth/mock-login", loginPayload, false)
		if err == nil && respLogin.StatusCode == 200 {
			var apiResp APIResponse
			_ = json.Unmarshal(bodyLogin, &apiResp)
			var data struct {
				Token string `json:"token"`
			}
			_ = json.Unmarshal(apiResp.Data, &data)
			if data.Token != "" {
				runner.Token = data.Token
			}
		}
		return nil
	})

	// Step 10: VIP 会员成功解锁并获取 VIP 专属题库全量题目
	runner.RunStep("VIP 学员解锁题库并加载全量试题 (GET /banks/:id/questions)", func() error {
		path := fmt.Sprintf("/api/v1/banks/%d/questions", vipBankID)
		resp, body, err := runner.Request("GET", path, nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var questions []struct {
			ID     uint     `json:"id"`
			Answer []string `json:"answer"`
		}
		if err := json.Unmarshal(apiResp.Data, &questions); err != nil {
			return err
		}

		if len(questions) < 2 {
			return fmt.Errorf("expected at least 2 questions in bank %d, got %d", vipBankID, len(questions))
		}

		question1ID = questions[0].ID
		question1Answer = questions[0].Answer
		question2ID = questions[1].ID
		return nil
	})

	// Step 11: 单题作答 - 正向正确回答判定
	runner.RunStep("单题作答正确判定 (POST /api/v1/practice/submit-single)", func() error {
		payload := map[string]interface{}{
			"bank_id":          vipBankID,
			"question_id":      question1ID,
			"user_answer":      question1Answer,
			"duration_seconds": 12,
		}
		resp, body, err := runner.Request("POST", "/api/v1/practice/submit-single", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var result struct {
			IsCorrect bool `json:"is_correct"`
		}
		if err := json.Unmarshal(apiResp.Data, &result); err != nil {
			return err
		}

		if !result.IsCorrect {
			return fmt.Errorf("expected is_correct = true for correct answer")
		}
		return nil
	})

	// Step 12: 单题作答 - 反向错误回答判定（触发错题库记录）
	runner.RunStep("单题作答错误判定与记录 (POST /api/v1/practice/submit-single)", func() error {
		payload := map[string]interface{}{
			"bank_id":          vipBankID,
			"question_id":      question2ID,
			"user_answer":      []string{"WRONG_OPT_E2E"},
			"duration_seconds": 25,
		}
		resp, body, err := runner.Request("POST", "/api/v1/practice/submit-single", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var result struct {
			IsCorrect bool `json:"is_correct"`
		}
		if err := json.Unmarshal(apiResp.Data, &result); err != nil {
			return err
		}

		if result.IsCorrect {
			return fmt.Errorf("expected is_correct = false for wrong answer")
		}
		return nil
	})

	// Step 13: 错题集自动沉淀验证
	runner.RunStep("错题集自动沉淀校验 (GET /api/v1/errors)", func() error {
		path := fmt.Sprintf("/api/v1/errors?bank_id=%d", vipBankID)
		resp, body, err := runner.Request("GET", path, nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var errorsList []struct {
			QuestionID uint `json:"question_id"`
			WrongCount int  `json:"wrong_count"`
			IsMastered bool `json:"is_mastered"`
		}
		if err := json.Unmarshal(apiResp.Data, &errorsList); err != nil {
			return err
		}

		found := false
		for _, e := range errorsList {
			if e.QuestionID == question2ID && !e.IsMastered {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("wrong question %d was not found in active error bank", question2ID)
		}
		return nil
	})

	// Step 14: 错题标记掌握与移出
	runner.RunStep("错题标记掌握与移出 (POST /api/v1/errors/:questionId/master)", func() error {
		path := fmt.Sprintf("/api/v1/errors/%d/master", question2ID)
		resp, body, err := runner.Request("POST", path, nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		return nil
	})

	// Step 15: 试题收藏与取消收藏切换
	runner.RunStep("试题收藏与取消收藏闭环 (POST /api/v1/favorites/toggle)", func() error {
		// 1. 收藏
		payload := map[string]interface{}{"question_id": question1ID}
		resp, body, err := runner.Request("POST", "/api/v1/favorites/toggle", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}
		var res1 struct {
			IsBookmarked bool `json:"is_bookmarked"`
		}
		if err := json.Unmarshal(apiResp.Data, &res1); err != nil {
			return err
		}
		if !res1.IsBookmarked {
			return fmt.Errorf("expected is_bookmarked = true")
		}

		// 2. 取消收藏
		resp2, body2, err := runner.Request("POST", "/api/v1/favorites/toggle", payload, true)
		if err != nil {
			return err
		}
		if resp2.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp2.StatusCode, string(body2))
		}
		var apiResp2 APIResponse
		_ = json.Unmarshal(body2, &apiResp2)
		var res2 struct {
			IsBookmarked bool `json:"is_bookmarked"`
		}
		_ = json.Unmarshal(apiResp2.Data, &res2)
		if res2.IsBookmarked {
			return fmt.Errorf("expected is_bookmarked = false after second toggle")
		}
		return nil
	})

	// Step 16: 全真模考交卷与评分诊断
	runner.RunStep("模考整卷提交与得分核验 (POST /api/v1/practice/submit-exam)", func() error {
		answersMap := map[string][]string{
			fmt.Sprintf("%d", question1ID): question1Answer,
			fmt.Sprintf("%d", question2ID): {"WRONG_CHOICE"},
		}
		payload := map[string]interface{}{
			"bank_id":       vipBankID,
			"total_seconds": 95,
			"answers":       answersMap,
		}
		resp, body, err := runner.Request("POST", "/api/v1/practice/submit-exam", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}

		var result struct {
			TotalQuestions int     `json:"total_questions"`
			AnsweredCount  int     `json:"answered_count"`
			CorrectCount   int     `json:"correct_count"`
			WrongCount     int     `json:"wrong_count"`
			Score          float64 `json:"score"`
		}
		if err := json.Unmarshal(apiResp.Data, &result); err != nil {
			return err
		}

		if result.AnsweredCount != 2 || result.CorrectCount != 1 || result.WrongCount != 1 {
			return fmt.Errorf("unexpected exam stats: answered=%d, correct=%d, wrong=%d",
				result.AnsweredCount, result.CorrectCount, result.WrongCount)
		}
		return nil
	})

	// Step 17: 学习规划制定与主计划校验
	runner.RunStep("学习目标制定与置顶激活 (POST /api/v1/plans)", func() error {
		payload := map[string]interface{}{
			"bank_id":         vipBankID,
			"daily_goal":      25,
			"is_active":       true,
			"app_reminder":    true,
			"wechat_reminder": false,
		}
		resp, body, err := runner.Request("POST", "/api/v1/plans", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		// 验证当前激活主规划
		respActive, bodyActive, err := runner.Request("GET", "/api/v1/plans/active", nil, true)
		if err != nil {
			return err
		}
		if respActive.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200 for active plan, got %d: %s", respActive.StatusCode, string(bodyActive))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(bodyActive, &apiResp); err != nil {
			return err
		}
		var activePlan struct {
			BankID    uint `json:"bank_id"`
			DailyGoal int  `json:"daily_goal"`
		}
		if err := json.Unmarshal(apiResp.Data, &activePlan); err != nil {
			return err
		}
		if activePlan.BankID != vipBankID || activePlan.DailyGoal != 25 {
			return fmt.Errorf("active plan mismatch: bank_id=%d, daily_goal=%d", activePlan.BankID, activePlan.DailyGoal)
		}
		return nil
	})

	// Step 18: 题目社区笔记与评论互动
	runner.RunStep("题目笔记发布、点赞与评论流 (POST /api/v1/notes)", func() error {
		// 1. 发布笔记
		payload := map[string]interface{}{
			"bank_id":     vipBankID,
			"question_id": question1ID,
			"content":     "【E2E测试笔记】本题是核心高频考点，请务必注意题目中的限定条件！",
			"visibility":  "public",
		}
		resp, body, err := runner.Request("POST", "/api/v1/notes", payload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return err
		}
		var note struct {
			ID uint `json:"id"`
		}
		if err := json.Unmarshal(apiResp.Data, &note); err != nil {
			return err
		}
		createdNoteID = note.ID

		// 2. 点赞笔记
		likePath := fmt.Sprintf("/api/v1/notes/%d/like", createdNoteID)
		respLike, bodyLike, err := runner.Request("POST", likePath, nil, true)
		if err != nil {
			return err
		}
		if respLike.StatusCode != http.StatusOK {
			return fmt.Errorf("like failed with status %d: %s", respLike.StatusCode, string(bodyLike))
		}

		// 3. 验证题目公开评论
		commentPath := fmt.Sprintf("/api/v1/questions/%d/comments", question1ID)
		respComments, bodyComments, err := runner.Request("GET", commentPath, nil, true)
		if err != nil {
			return err
		}
		if respComments.StatusCode != http.StatusOK {
			return fmt.Errorf("get comments failed with status %d: %s", respComments.StatusCode, string(bodyComments))
		}
		return nil
	})

	// Step 19: 裂变活动与邀请奖励查询
	runner.RunStep("裂变活动邀请奖励查询 (GET /api/v1/activity/referral/rewards)", func() error {
		resp, body, err := runner.Request("GET", "/api/v1/activity/referral/rewards", nil, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		return nil
	})

	// Step 20: 参数错误边界拦截校验 (400 负向边界)
	runner.RunStep("缺失必要参数请求拦截校验 (POST /api/v1/practice/submit-single)", func() error {
		badPayload := map[string]interface{}{
			"invalid_field": "some_value",
		}
		resp, _, err := runner.Request("POST", "/api/v1/practice/submit-single", badPayload, true)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("expected status 400 for bad payload, got %d", resp.StatusCode)
		}
		return nil
	})

	// 测试结果汇总报告
	fmt.Println("================================================================================")
	fmt.Printf("集成测试执行完成: 共 %d 项用例 | 通过: %d | 失败: %d | 通过率: %.1f%%\n",
		runner.TotalSteps, runner.PassedSteps, runner.FailedSteps,
		float64(runner.PassedSteps)/float64(runner.TotalSteps)*100.0)

	if runner.FailedSteps > 0 {
		fmt.Println("\n失败用例明细:")
		for _, errStr := range runner.Errors {
			fmt.Printf("  - %s\n", errStr)
		}
		fmt.Println("================================================================================")
		os.Exit(1)
	}

	fmt.Println("状态: ALL E2E INTEGRATION TESTS PASSED (100% 成功)")
	fmt.Println("================================================================================")
}
