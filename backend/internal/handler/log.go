package handler

import (
	"strconv"
	"time"

	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LogHandler 操作日志审计
type LogHandler struct {
	db *gorm.DB
}

func NewLogHandler(db *gorm.DB) *LogHandler {
	return &LogHandler{db: db}
}

// List 日志列表（分页+筛选）
func (h *LogHandler) List(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.SysOperationLog{})

	if module := c.Query("module"); module != "" {
		query = query.Where("operation_module = ?", module)
	}
	if opType := c.Query("type"); opType != "" {
		query = query.Where("operation_type = ?", opType)
	}
	if result := c.Query("result"); result != "" {
		r, _ := strconv.Atoi(result)
		query = query.Where("operation_result = ?", r)
	}
	if userID := c.Query("user_id"); userID != "" {
		uid, _ := strconv.ParseUint(userID, 10, 64)
		query = query.Where("user_id = ?", uid)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("username LIKE ? OR request_url LIKE ? OR operation_module LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if startTime := c.Query("start_time"); startTime != "" {
		t, _ := time.Parse(time.RFC3339, startTime)
		query = query.Where("create_time >= ?", t)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		t, _ := time.Parse(time.RFC3339, endTime)
		query = query.Where("create_time <= ?", t)
	}

	var total int64
	query.Count(&total)

	var logs []model.SysOperationLog
	offset, _ := c.Get("offset")
	query.Order("create_time DESC").Offset(offset).Limit(pageSize.(int)).Find(&logs)

	response.PageSuccess(c, logs, total, page.(int), pageSize.(int))
}

// Detail 日志详情
func (h *LogHandler) Detail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64

	var log model.SysOperationLog
	if h.db.First(&log, id).Error != nil {
		response.NotFound(c, "日志记录不存在", nil)
		return
	}

	response.Success(c, log)
}

// Export 导出日志（Excel）
func (h *LogHandler) Export(c *gin.Context) {
	// TODO: 使用 excelize 库导出日志到 Excel 文件
	response.SuccessWithMessage(c, "导出功能开发中...", nil)
}
