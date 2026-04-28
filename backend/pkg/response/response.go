package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRequestID 从上下文获取请求ID
func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	requestID, exists := c.Get("request_id")
	if !exists {
		return ""
	}
	if id, ok := requestID.(string); ok {
		return id
	}
	return ""
}

// Response 统一响应结构体
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	PageInfo  *PageInfo   `json:"page_info,omitempty"`
}

// PageInfo 分页信息
type PageInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int  `json:"total_pages"`
}

// PageData 带分页的响应数据
type PageData struct {
	List interface{} `json:"list"`
	Page PageInfo     `json:"page_info"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   "success",
		Data:      data,
		RequestID: GetRequestID(c),
	})
}

// SuccessWithMessage 成功响应(自定义消息)
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   message,
		Data:      data,
		RequestID: GetRequestID(c),
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   "success",
		Data:      list,
		RequestID: GetRequestID(c),
		PageInfo: &PageInfo{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, httpCode int, message string, data interface{}) {
	c.JSON(httpCode, Response{
		Code:      httpCode,
		Message:   message,
		Data:      data,
		RequestID: GetRequestID(c),
	})
}

// BadRequest 参数错误响应 (400)
func BadRequest(c *gin.Context, message string, data interface{}) {
	Error(c, http.StatusBadRequest, message, data)
}

// Unauthorized 未授权响应 (401)
func Unauthorized(c *gin.Context, message string, data interface{}) {
	Error(c, http.StatusUnauthorized, message, data)
}

// Forbidden 禁止访问响应 (403)
func Forbidden(c *gin.Context, message string, data interface{}) {
	Error(c, http.StatusForbidden, message, data)
}

// NotFound 未找到资源响应 (404)
func NotFound(c *gin.Context, message string, data interface{}) {
	Error(c, http.StatusNotFound, message, data)
}

// InternalError 内部服务器错误响应 (500)
func InternalError(c *gin.Context, message string, data interface{}) {
	Error(c, http.StatusInternalServerError, message, data)
}
