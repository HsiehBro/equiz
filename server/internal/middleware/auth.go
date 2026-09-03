package middleware

import (
	"errors"
	"net/http"
	"strings"

	"exam-server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID uint   `json:"user_id"`
	OpenID string `json:"openid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "未提供认证 Token，请先登录"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "Token 格式错误，需为 Bearer <token>"))
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "Token 无效或已过期"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*CustomClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "无效的 Token Payload"))
			c.Abort()
			return
		}

		// 将解析后的用户上下文写入 Gin 上下文
		c.Set("user_id", claims.UserID)
		c.Set("openid", claims.OpenID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// GetCurrentUserID 从 Gin 上下文中获取当前登录用户 ID
func GetCurrentUserID(c *gin.Context) (uint, error) {
	val, exists := c.Get("user_id")
	if !exists {
		return 0, errors.New("user not authenticated")
	}
	uid, ok := val.(uint)
	if !ok {
		return 0, errors.New("invalid user id in context")
	}
	return uid, nil
}

// OptionalJWTAuthMiddleware 可选 JWT 认证中间件，若携带合法 Token 则注入上下文，若无则静默放行
func OptionalJWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			token, err := jwt.ParseWithClaims(parts[1], &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(*CustomClaims); ok {
					c.Set("user_id", claims.UserID)
					c.Set("openid", claims.OpenID)
					c.Set("role", claims.Role)
				}
			}
		}
		c.Next()
	}
}

