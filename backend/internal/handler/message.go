package handler

import (
	"strconv"

	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageHandler struct {
	db *gorm.DB
}

func NewMessageHandler(db *gorm.DB) *MessageHandler {
	return &MessageHandler{db: db}
}

// List 消息列表
func (h *MessageHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.NotifyMessage{}).Where("receiver_id = ?", userID)

	if msgType := c.Query("type"); msgType != "" {
		t, _ := strconv.Atoi(msgType)
		query = query.Where("msg_type = ?", t)
	}
	if isRead := c.Query("is_read"); isRead != "" {
		r, _ := strconv.Atoi(isRead)
		query = query.Where("is_read = ?", r)
	}

	var total int64
	query.Count(&total)

	var messages []model.NotifyMessage
	offset, _ := c.Get("offset")
	query.Order("create_time DESC").Offset(offset).Limit(pageSize.(int)).Find(&messages)

	response.PageSuccess(c, messages, total, page.(int), pageSize.(int))
}

// UnreadCount 未读消息数量
func (h *MessageHandler) UnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var count int64
	h.db.Model(&model.NotifyMessage{}).
		Where("receiver_id = ? AND is_read = 0", userID).
		Count(&count)

	response.Success(c, gin.H{"unread_count": count})
}

// MarkRead 标记单条消息为已读
func (h *MessageHandler) MarkRead(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := middleware.GetUserID(c)

	result := h.db.Model(&model.NotifyMessage{}).
		Where("id = ? AND receiver_id = ?", id, userID).
		Update("is_read", 1)

	if result.RowsAffected == 0 {
		response.NotFound(c, "消息不存在或无权操作", nil)
		return
	}

	now := time.Now()
	h.db.Model(&model.NotifyMessage{ID: id}).Update("read_time", now)

	response.SuccessWithMessage(c, "已标记为已读", nil)
}

// MarkAllRead 全部标记已读
func (h *MessageHandler) MarkAllRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	now := time.Now()

	h.db.Model(&model.NotifyMessage{}).
		Where("receiver_id = ? AND is_read = 0", userID).
		Updates(map[string]interface{}{
			"is_read":   1,
			"read_time": now,
		})

	response.SuccessWithMessage(c, "全部已读", nil)
}
