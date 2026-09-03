package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	if nickname == "" {
		nickname = fmt.Sprintf("开发者用户_%s", devIdentifier)
	}

	var user model.User
	err := s.db.Where("open_id = ? OR open_id = ?", devIdentifier, mockOpenID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		// 用户已存在时，如指定了 role 或 nickname 则同步更新，确保调试身份即时生效
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
	if s.cfg.WeChat.AppID == "" || s.cfg.WeChat.AppSecret == "" {
		// 未配置微信 AppID 时自动降级为 Mock 模式
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
			user = model.User{
				OpenID:    wxResp.OpenID,
				UnionID:   wxResp.UnionID,
				Nickname:  "备考学员",
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
