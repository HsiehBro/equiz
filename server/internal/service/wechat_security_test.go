package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"exam-server/internal/config"
)

func TestWeChatSecurity_NotConfigured(t *testing.T) {
	cfg := &config.Config{
		WeChat: config.WeChatConfig{
			AppID:     "",
			AppSecret: "",
		},
	}
	secService := NewWeChatSecurityService(cfg)
	passed, reason, err := secService.CheckMsgSec(context.Background(), "mock_openid", "测试内容")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Fatalf("expected passed=true when not configured, got false: %s", reason)
	}
}

func TestWeChatSecurity_PassAndRisky(t *testing.T) {
	var tokenCalls int32
	var checkCalls int32

	// 创建模拟微信开放平台 HTTP 服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. 获取 token
		if strings.Contains(r.URL.Path, "/cgi-bin/token") {
			atomic.AddInt32(&tokenCalls, 1)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "mock_access_token_123",
				"expires_in":   7200,
				"errcode":      0,
				"errmsg":       "ok",
			})
			return
		}

		// 2. 检查内容安全
		if strings.Contains(r.URL.Path, "/wxa/msg_sec_check") {
			atomic.AddInt32(&checkCalls, 1)
			var reqBody struct {
				Content string `json:"content"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)

			if strings.Contains(reqBody.Content, "违规招嫖") {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errcode": 0,
					"errmsg":  "ok",
					"result": map[string]interface{}{
						"suggest": "risky",
						"label":   20002, // 色情低俗
					},
				})
				return
			}

			// 正常内容放行
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"errcode": 0,
				"errmsg":  "ok",
				"result": map[string]interface{}{
					"suggest": "pass",
					"label":   100,
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		WeChat: config.WeChatConfig{
			AppID:     "mock_appid",
			AppSecret: "mock_secret",
		},
	}
	secService := NewWeChatSecurityService(cfg)
	secService.SetBaseURL(mockServer.URL)

	// 测试 1：正常内容放行
	passed, _, err := secService.CheckMsgSec(context.Background(), "user_001", "请问这道计算机网络的题目为什么选B？")
	if err != nil || !passed {
		t.Fatalf("expected pass for normal content, got passed=%v, err=%v", passed, err)
	}

	// 测试 2：命中微信违规模型拦截 (色情低俗)
	passedRisky, reason, err := secService.CheckMsgSec(context.Background(), "user_001", "这是含有违规招嫖的内容")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passedRisky {
		t.Fatalf("expected risky content to be rejected, got passed=true")
	}
	if !strings.Contains(reason, "色情低俗") {
		t.Errorf("expected reason to contain '色情低俗', got: %s", reason)
	}

	// 测试 3：Token 缓存复用，确保 tokenCalls 仅为 1 次
	if atomic.LoadInt32(&tokenCalls) != 1 {
		t.Errorf("expected token to be cached (1 call), got %d calls", tokenCalls)
	}
}

func TestWeChatSecurity_FallbackOnTimeout(t *testing.T) {
	// 模拟微信服务器超时/500异常，测试高可用优雅降级
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/cgi-bin/token") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "mock_token",
				"expires_in":   7200,
				"errcode":      0,
			})
			return
		}
		// 模拟网络故障返回 500
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		WeChat: config.WeChatConfig{
			AppID:     "mock_appid",
			AppSecret: "mock_secret",
		},
	}
	secService := NewWeChatSecurityService(cfg)
	secService.SetBaseURL(mockServer.URL)

	passed, _, err := secService.CheckMsgSec(context.Background(), "user_001", "任何考题讨论")
	if err != nil {
		t.Fatalf("unexpected error on fallback: %v", err)
	}
	// 网络故障时不卡死用户，优雅降级放行
	if !passed {
		t.Fatalf("expected fallback to pass=true on wechat 500 error")
	}
}
