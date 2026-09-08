package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"exam-server/internal/config"
)

// MsgSecCheckResult 微信 msg_sec_check 审核结果
type MsgSecCheckResult struct {
	Suggest string `json:"suggest"` // pass: 通过; review: 需人工审核; risky: 违规
	Label   int    `json:"label"`   // 违规标签代码
}

// MsgSecCheckResponse 微信 msg_sec_check 外层响应
type MsgSecCheckResponse struct {
	ErrCode int                `json:"errcode"`
	ErrMsg  string             `json:"errmsg"`
	Result  *MsgSecCheckResult `json:"result"`
}

// TokenResponse 微信凭证响应
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// LabelDescriptions 微信违规标签代码映射表
var LabelDescriptions = map[int]string{
	100:   "正常",
	10001: "广告导流与营销违规",
	20001: "时政与意识形态敏感",
	20002: "色情低俗与招嫖",
	20003: "侮辱辱骂与网络暴力",
	20006: "违法犯罪与违禁品",
	20008: "欺诈刷单与网络黑灰产",
	20012: "低俗不良导向",
	20013: "侵权盗版",
}

// WeChatSecurityService 微信官方内容安全审核服务
type WeChatSecurityService struct {
	cfg            *config.Config
	httpClient     *http.Client
	baseURL        string
	accessToken    string
	tokenExpiresAt time.Time
	tokenMu        sync.RWMutex
}

// NewWeChatSecurityService 构造函数
func NewWeChatSecurityService(cfg *config.Config) *WeChatSecurityService {
	return &WeChatSecurityService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 2 * time.Second},
		baseURL:    "https://api.weixin.qq.com",
	}
}

// SetBaseURL 设置 API 基地址（主要供单元测试 mock）
func (s *WeChatSecurityService) SetBaseURL(url string) {
	s.baseURL = url
}

// GetAccessToken 获取微信 AccessToken，内置读写锁与过期自动刷新
func (s *WeChatSecurityService) GetAccessToken(ctx context.Context) (string, error) {
	s.tokenMu.RLock()
	if s.accessToken != "" && time.Now().Before(s.tokenExpiresAt) {
		token := s.accessToken
		s.tokenMu.RUnlock()
		return token, nil
	}
	s.tokenMu.RUnlock()

	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	// 双重检查
	if s.accessToken != "" && time.Now().Before(s.tokenExpiresAt) {
		return s.accessToken, nil
	}

	if s.cfg == nil || s.cfg.WeChat.AppID == "" || s.cfg.WeChat.AppSecret == "" {
		return "", fmt.Errorf("wechat app_id or app_secret not configured")
	}

	tokenURL := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		s.baseURL, s.cfg.WeChat.AppID, s.cfg.WeChat.AppSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call wechat token api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res TokenResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if res.ErrCode != 0 {
		return "", fmt.Errorf("wechat token error [%d]: %s", res.ErrCode, res.ErrMsg)
	}

	s.accessToken = res.AccessToken
	// 提前 300 秒刷新，保证缓冲
	expireSeconds := res.ExpiresIn - 300
	if expireSeconds <= 60 {
		expireSeconds = 60
	}
	s.tokenExpiresAt = time.Now().Add(time.Duration(expireSeconds) * time.Second)

	return s.accessToken, nil
}

// CheckMsgSec 调用微信 security.msgSecCheck v2 进行云端语义合规审查
// 返回: (passed bool, rejectReason string, err error)
func (s *WeChatSecurityService) CheckMsgSec(ctx context.Context, openID string, content string) (bool, string, error) {
	if content == "" {
		return true, "", nil
	}

	// 1. 若未配置微信 AppID 或 Secret (或仍为占位符)，优雅降级放行（不影响本地开发与测试）
	if s.cfg == nil || s.cfg.WeChat.AppID == "" || s.cfg.WeChat.AppSecret == "" || strings.HasPrefix(s.cfg.WeChat.AppID, "your_") {
		return true, "", nil
	}

	// 2. 获取/复用微信 AccessToken
	token, err := s.GetAccessToken(ctx)
	if err != nil {
		log.Printf("[WeChatSecurity] 获取 AccessToken 失败，降级放行: %v\n", err)
		return true, "", nil
	}

	// 3. 构造 msgSecCheck v2 请求
	secURL := fmt.Sprintf("%s/wxa/msg_sec_check?access_token=%s", s.baseURL, token)
	if openID == "" {
		openID = "mock_user"
	}
	payload := map[string]interface{}{
		"openid":  openID,
		"scene":   2, // 2: 论坛/发帖/评论场景
		"version": 2,
		"content": content,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return true, "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, secURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return true, "", nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[WeChatSecurity] 微信内容安全接口网络抖动或超时，降级放行: %v\n", err)
		return true, "", nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return true, "", nil
	}

	var secResp MsgSecCheckResponse
	if err := json.Unmarshal(respBody, &secResp); err != nil {
		log.Printf("[WeChatSecurity] 反序列化微信内容安全响应失败: %v, body: %s\n", err, string(respBody))
		return true, "", nil
	}

	// Token 失效（40001 / 42001），清空缓存以备下次重新拉取
	if secResp.ErrCode == 40001 || secResp.ErrCode == 42001 {
		s.tokenMu.Lock()
		s.accessToken = ""
		s.tokenMu.Unlock()
		log.Printf("[WeChatSecurity] AccessToken 已失效，已清空缓存，本次降级放行\n")
		return true, "", nil
	}

	// 其他微信接口非 0 返回（例如开发机 openid 非真机微信用户），记录日志并安全降级
	if secResp.ErrCode != 0 {
		log.Printf("[WeChatSecurity] 微信安全检测接口返回错误 [%d: %s]，降级放行\n", secResp.ErrCode, secResp.ErrMsg)
		return true, "", nil
	}

	// 4. 分析违规检测结果
	if secResp.Result != nil {
		if secResp.Result.Suggest == "risky" {
			labelName := LabelDescriptions[secResp.Result.Label]
			if labelName == "" {
				labelName = "违规内容"
			}
			reason := fmt.Sprintf("内容未通过微信安全审查（涉嫌%s），请修改后重新发布", labelName)
			return false, reason, nil
		}
	}

	return true, "", nil
}
