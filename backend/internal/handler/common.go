package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"certmonitor/internal/model"
	"certmonitor/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// recordOperationLog 记录操作审计日志
func recordOperationLog(db *gorm.DB, userID uint64, module, opType string, c *gin.Context, param interface{}, result uint8) {
	username := GetUsernameFromContext(c)
	paramJSON, _ := json.Marshal(param)
	reqMethod := c.Request.Method
	reqURL := c.Request.URL.String()
	requestIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	logEntry := model.SysOperationLog{
		UserID:          &userID,
		Username:        username,
		OperationModule: module,
		OperationType:   opType,
		RequestMethod:   reqMethod,
		RequestURL:      reqURL,
		RequestIP:       requestIP,
		RequestParam:    string(paramJSON),
		OperationResult: result,
		UserAgent:       userAgent,
		CreateTime:      time.Now(),
	}

	if err := db.Create(&logEntry).Error; err != nil {
		logger.Warn("写入审计日志失败: %v", err)
	}
}

func GetUsernameFromContext(c *gin.Context) string {
	name, exists := c.Get("username")
	if exists && name != nil {
		return name.(string)
	}
	return ""
}
