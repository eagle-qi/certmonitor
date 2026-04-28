package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"certmonitor/pkg/response"
)

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept, Authorization, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logMsg := fmt.Sprintf("[HTTP] %s | %3d | %13v | %15s | %-7s %s",
			GetRequestID(c),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)

		if statusCode >= 500 {
			logger.Error(logMsg)
		} else if statusCode >= 400 {
			logger.Warn(logMsg)
		} else {
			logger.Info(logMsg)
		}
	}
}

// Recovery 异常恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("PANIC: %v\n%s", err, debug.Stack())
				response.Error(c, http.StatusInternalServerError, "服务器内部错误", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// AuthRequired JWT认证中间件
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Unauthorized(c, "未提供有效的认证令牌", nil)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "令牌无效或已过期", nil)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			response.Unauthorized(c, "令牌格式异常", nil)
			c.Abort()
			return
		}

		// 将用户信息存入上下文，供后续处理器使用
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// RoleBasedAccessControl 基于角色的访问控制
// allowedRoles 允许访问的角色编码列表（为空表示仅需登录即可）
func RoleBasedAccessControl(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(allowedRoles) == 0 {
			c.Next() // 无角色要求，仅需登录
			return
		}

		userRoles, exists := c.Get("roles")
		if !exists {
			response.Forbidden(c, "无法获取用户权限信息", nil)
			c.Abort()
			return
		}

		roles, ok := userRoles.([]string)
		if !ok {
			response.Forbidden(c, "用户权限信息解析失败", nil)
			c.Abort()
			return
		}

		// 检查是否拥有所需角色之一
		for _, required := range allowedRoles {
			for _, has := range roles {
				if has == required || has == "super_admin" { // 超级管理员拥有所有权限
					c.Next()
					return
				}
			}
		}

		response.Forbidden(c, "权限不足，需要以下角色之一: "+strings.Join(allowedRoles, ", "), nil)
		c.Abort()
	}
}

// Pagination 分页参数解析中间件
func Pagination() gin.HandlerFunc {
	return func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "20")

		page, _ := strconv.Atoi(pageStr)
		pageSize, _ := strconv.Atoi(pageSizeStr)

		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		offset := (page - 1) * pageSize

		c.Set("page", page)
		c.Set("page_size", pageSize)
		c.Set("offset", offset)

		c.Next()
	}
}

// GetRequestID 从上下文获取请求ID
func GetRequestID(c *gin.Context) string {
	id, _ := c.Get("request_id")
	if id != nil {
		return id.(string)
	}
	return ""
}

// GetUserID 从上下文获取当前登录用户ID
func GetUserID(c *gin.Context) uint64 {
	id, _ := c.Get("user_id")
	if id != nil {
		return id.(uint64)
	}
	return 0
}

// GetUsername 从上下文获取当前登录用户名
func GetUsername(c *gin.Context) string {
	name, _ := c.Get("username")
	if name != nil {
		return name.(string)
	}
	return ""
}

// GenerateToken 生成JWT Token
func GenerateToken(userID uint64, username, email string, roles []string, jwtSecret string, expireHours int) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "certmonitor",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
