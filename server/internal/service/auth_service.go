package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"exam-server/internal/config"
	"exam-server/internal/middleware"
	"exam-server/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

// GenerateUniqueRandomNickname 生成全局唯一且符合规范的随机默认昵称 (如: 用户_7e2a9b4f)
func (s *AuthService) GenerateUniqueRandomNickname() string {
	for attempts := 0; attempts < 20; attempts++ {
		b := make([]byte, 4) // 4 字节 = 8 位十六进制字符
		if _, err := rand.Read(b); err != nil {
			break
		}
		randomHex := hex.EncodeToString(b)
		candidate := fmt.Sprintf("用户_%s", randomHex)

		if s.db != nil {
			var count int64
			if err := s.db.Model(&model.User{}).Where("nickname = ?", candidate).Count(&count).Error; err == nil && count == 0 {
				return candidate
			}
		} else {
			return candidate
		}
	}
	// 极端高并发下的微秒戳安全兜底
	return fmt.Sprintf("用户_%x", time.Now().UnixNano()%0xFFFFFFFF)
}

// GenerateToken 为用户签发 JWT
func (s *AuthService) GenerateToken(user *model.User) (string, error) {
	expireTime := time.Now().Add(time.Duration(s.cfg.JWT.ExpireHours) * time.Hour)
	claims := middleware.CustomClaims{
		UserID: user.ID,
		OpenID: user.OpenID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// MockLogin 模拟开发快速登录 (自动建表或查找对应测试用户)
func (s *AuthService) MockLogin(devIdentifier string, role string, nickname string) (*model.User, string, error) {
	if devIdentifier == "" {
		devIdentifier = "dev_user_test_001"
	}
	mockOpenID := fmt.Sprintf("mock_%s", devIdentifier)

	if role == "" {
		role = "user"
	}

	var user model.User
	err := s.db.Where("open_id = ? OR open_id = ?", devIdentifier, mockOpenID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新用户初次登录：若未传昵称，默认生成全局唯一的随机字符串昵称
			if nickname == "" {
				nickname = s.GenerateUniqueRandomNickname()
			}
			user = model.User{
				OpenID:    mockOpenID,
				Nickname:  nickname,
				AvatarURL: "https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0",
				Role:      role,
			}
			if err := s.db.Create(&user).Error; err != nil {
				return nil, "", fmt.Errorf("failed to create mock user: %w", err)
			}
		} else {
			return nil, "", err
		}
	} else {
		// 老用户登录：只有当显式传了非空昵称且与当前不同时才更新，决不覆盖用户已有的真实昵称
		updates := map[string]interface{}{}
		if role != "" && user.Role != role {
			updates["role"] = role
			user.Role = role
		}
		if nickname != "" && user.Nickname != nickname {
			updates["nickname"] = nickname
			user.Nickname = nickname
		}
		if len(updates) > 0 {
			_ = s.db.Model(&user).Updates(updates).Error
		}
	}

	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

type WxSessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// WeChatLogin 微信真实 code2Session 流程
func (s *AuthService) WeChatLogin(code string) (*model.User, string, error) {
	if s.cfg.WeChat.AppID == "" || s.cfg.WeChat.AppSecret == "" || strings.HasPrefix(s.cfg.WeChat.AppID, "your_") {
		// 未配置真实微信 AppID 时自动降级为 Mock 模式 (未传 nickname，初次登录走随机字符串昵称)
		return s.MockLogin(fmt.Sprintf("wx_%s", code[:min(len(code), 8)]), "user", "")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.cfg.WeChat.AppID, s.cfg.WeChat.AppSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to call WeChat API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read WeChat response: %w", err)
	}

	var wxResp WxSessionResponse
	if err := json.Unmarshal(body, &wxResp); err != nil {
		return nil, "", fmt.Errorf("failed to decode WeChat response: %w", err)
	}

	if wxResp.ErrCode != 0 {
		return nil, "", fmt.Errorf("wechat login failed [%d]: %s", wxResp.ErrCode, wxResp.ErrMsg)
	}

	var user model.User
	err = s.db.Where("open_id = ?", wxResp.OpenID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户初次微信登录：默认昵称为随机字符串
			defaultNickname := s.GenerateUniqueRandomNickname()
			user = model.User{
				OpenID:    wxResp.OpenID,
				UnionID:   wxResp.UnionID,
				Nickname:  defaultNickname,
				AvatarURL: "",
				Role:      "user",
			}
			if err := s.db.Create(&user).Error; err != nil {
				return nil, "", fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return nil, "", err
		}
	}

	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}

func (s *AuthService) GetUserProfile(userID uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateProfileRequest 修改个人资料请求
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// UpdateProfile 修改个人资料，严格强制校验昵称全局唯一
func (s *AuthService) UpdateProfile(userID uint, req UpdateProfileRequest) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	oldNickname := user.Nickname
	nicknameChanged := false
	newNickname := strings.TrimSpace(req.Nickname)
	if newNickname != "" && newNickname != oldNickname {
		if len([]rune(newNickname)) > 20 {
			return nil, errors.New("昵称长度不能超过 20 个字符")
		}

		// 强制校验全局唯一：排除当前用户自身后，严禁与其他用户重复
		var count int64
		err := s.db.Model(&model.User{}).
			Where("nickname = ? AND id != ?", newNickname, userID).
			Count(&count).Error
		if err != nil {
			return nil, fmt.Errorf("校验昵称唯一性失败: %w", err)
		}
		if count > 0 {
			return nil, errors.New("该昵称已被其他考友使用，请更换一个唯一的昵称")
		}

		user.Nickname = newNickname
		nicknameChanged = true
	}

	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}

	user.UpdatedAt = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("更新个人资料失败: %w", err)
	}

	// 当昵称变更时，级联同步更新评论引用以及消息中心中历史冗余的旧昵称
	if nicknameChanged {
		// 1. 同步更新 user_notes 中被回复人相关字段
		_ = s.db.Model(&model.UserNote{}).
			Where("reply_to_user_id = ? OR reply_to_author = ?", userID, oldNickname).
			Update("reply_to_author", newNickname).Error

		// 2. 同步更新 system_notifications 中发送者名称与标题提示
		_ = s.db.Model(&model.SystemNotification{}).
			Where("replier_id = ? OR replier_name = ?", userID, oldNickname).
			Updates(map[string]interface{}{
				"replier_name": newNickname,
				"title":        gorm.Expr("REPLACE(title, ?, ?)", oldNickname, newNickname),
			}).Error
	}

	return &user, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
