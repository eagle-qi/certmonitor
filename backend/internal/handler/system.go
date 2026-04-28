package handler

import (
	"strconv"

	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SystemConfigHandler 系统配置管理
type SystemConfigHandler struct {
	db *gorm.DB
}

func NewSystemConfigHandler(db *gorm.DB) *SystemConfigHandler {
	return &SystemConfigHandler{db: db}
}

// List 配置列表
func (h *SystemConfigHandler) List(c *gin.Context) {
	var configs []model.SysConfig
	h.db.Order("config_group ASC, id ASC").Find(&configs)
	response.Success(c, configs)
}

// Update 更新单个配置项
func (h *SystemConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")

	var req struct {
		ConfigValue string `json:"config_value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	result := h.db.Model(&model.SysConfig{}).Where("config_key = ?", key).Update("config_value", req.ConfigValue)
	if result.RowsAffected == 0 {
		response.NotFound(c, "配置项不存在", nil)
		return
	}

	recordOperationLog(h.db, middleware.GetUserID(c), "system_config", "update", c,
		map[string]interface{}{"key": key, "value": req.ConfigValue}, 1)

	response.SuccessWithMessage(c, "配置更新成功(部分配置可能需要重启服务生效)", nil)
}
