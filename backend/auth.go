package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const authCookieName = "auth_token"

// loginRequest carries login payload.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// registerAuthRoutes registers login endpoint. It must be called before auth middleware.
func registerAuthRoutes(r *gin.Engine) string {
	// Auth config
	username := getEnv("AUTH_USERNAME", "admin")
	password := getEnv("AUTH_PASSWORD", "admin123")
	secret := getEnv("AUTH_SECRET", "supersecretkey")
	ttl := 24 * time.Hour

	r.POST("/api/auth/login", func(c *gin.Context) {
		var body loginRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_CREDENTIALS", "用户名或密码错误", gin.H{"hint": "请检查输入"})
			return
		}
		if body.Username != username || body.Password != password {
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误", nil)
			return
		}

		token, expiresAt, err := issueToken(secret, body.Username, ttl)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "TOKEN_ISSUE_FAILED", "生成登录凭证失败", gin.H{"detail": err.Error()})
			return
		}

		// Set HttpOnly cookie to ease browser usage
		c.SetCookie(authCookieName, token, int(ttl.Seconds()), "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"token": token, "expiresAt": expiresAt})
	})

	return secret
}

// authMiddleware validates JWT on protected routes.
func authMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := readTokenFromRequest(c)
		if token == "" {
			writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录", nil)
			c.Abort()
			return
		}

		if _, err := parseToken(token, secret); err != nil {
			writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "登录已过期或无效，请重新登录", gin.H{"detail": err.Error()})
			c.Abort()
			return
		}

		c.Next()
	}
}

func readTokenFromRequest(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if cookie, err := c.Cookie(authCookieName); err == nil {
		return cookie
	}
	return ""
}

func issueToken(secret, subject string, ttl time.Duration) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(time.Unix(expiresAt, 0)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return signed, expiresAt, nil
}

func parseToken(tokenStr string, secret string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
