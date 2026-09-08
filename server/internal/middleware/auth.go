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
		var tokenString string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			} else {
				c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "Token 格式错误，需为 Bearer <token>"))
				c.Abort()
				return
			}
		} else if queryToken := c.Query("token"); queryToken != "" {
			tokenString = queryToken
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "未提供认证 Token，请先登录"))
			c.Abort()
			return
		}

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
		var tokenString string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		} else if queryToken := c.Query("token"); queryToken != "" {
			tokenString = queryToken
		}

		if tokenString != "" {
			token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
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

// GetCurrentUserRole 从 Gin 上下文中获取当前登录用户角色
func GetCurrentUserRole(c *gin.Context) string {
	val, exists := c.Get("role")
	if !exists {
		return ""
	}
	if role, ok := val.(string); ok {
		return role
	}
	return ""
}

// AdminRequiredMiddleware 必须是管理员角色才能访问的权限中间件
func AdminRequiredMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetCurrentUserRole(c)
		if role != "admin" {
			c.JSON(http.StatusForbidden, model.ErrorResponse(403, "权限不足：该操作仅系统管理员可用"))
			c.Abort()
			return
		}
		c.Next()
	}
}
