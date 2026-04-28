package handler

import (
	"strconv"

	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AlertHandler struct {
	db *gorm.DB
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{db: db}
}

// CreateRule 创建告警规则
func (h *AlertHandler) Create(c *gin.Context) {
	var rule model.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	rule.Enabled = 1
	h.db.Create(&rule)

	recordOperationLog(h.db, middleware.GetUserID(c), "alert_rule", "create", c,
		map[string]interface{}{"rule_id": rule.ID, "rule_name": rule.RuleName}, 1)

	response.SuccessWithMessage(c, "告警规则创建成功", rule)
}

// List 规则列表
func (h *AlertHandler) List(c *gin.Context) {
	var rules []model.AlertRule
	h.db.Find(&rules)
	response.Success(c, rules)
}

// Update 更新规则
func (h *AlertHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var existing model.AlertRule
	if h.db.First(&existing, id).Error != nil {
		response.NotFound(c, "规则不存在", nil)
		return
	}

	var updates model.AlertRule
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	h.db.Model(&existing).Updates(updates)

	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除规则
func (h *AlertHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64
	result := h.db.Delete(&model.AlertRule{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "规则不存在", nil)
		return
	}

	response.SuccessWithMessage(c, "规则已删除", nil)
}

// ToggleEnable 启用/禁用规则
func (h *AlertHandler) ToggleEnable(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64

	var req struct {
		Enabled uint8 `json:"enabled" binding:"required,oneof=0 1"`
	}
	c.ShouldBindJSON(&req)

	h.db.Model(&model.AlertRule{ID: id}).Update("enabled", req.Enabled)

	action := "启用"
	if req.Enabled == 0 { action = "禁用" }
	response.SuccessWithMessage(c, "规则已"+action, nil)
}
