package handler

import (
	"fmt"
	"strconv"

	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RoleHandler struct {
	db *gorm.DB
}

func NewRoleHandler(db *gorm.DB) *RoleHandler {
	return &RoleHandler{db: db}
}

// Create 创建角色
func (h *RoleHandler) Create(c *gin.Context) {
	var role model.SysRole
	if err := c.ShouldBindJSON(&role); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	// 检查角色编码唯一性
	var count int64
	h.db.Model(&model.SysRole{}).Where("role_code = ?", role.RoleCode).Count(&count)
	if count > 0 {
		response.BadRequest(c, "角色编码已存在", nil)
		return
	}

	role.Status = 1
	h.db.Create(&role)

	recordOperationLog(h.db, middleware.GetUserID(c), "role", "create", c,
		map[string]interface{}{"role_id": role.ID, "role_code": role.RoleCode}, 1)

	response.SuccessWithMessage(c, "角色创建成功", role)
}

// List 角色列表
func (h *RoleHandler) List(c *gin.Context) {
	var roles []model.SysRole
	h.db.Order("sort_order ASC, id ASC").Find(&roles)
	response.Success(c, roles)
}

// Update 更新角色
func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var existing model.SysRole
	if h.db.First(&existing, id).Error != nil {
		response.NotFound(c, "角色不存在", nil)
		return
	}

	var updates struct {
		RoleName   string `json:"role_name,omitempty"`
		Description string `json:"description,omitempty"`
		SortOrder  int   `json:"sort_order,omitempty"`
		Status     uint8  `json:"status,omitempty"`
	}
	c.ShouldBindJSON(&updates)

	h.db.Model(&existing).Updates(updates)

	response.SuccessWithMessage(c, "角色更新成功", nil)
}

// Delete 删除角色
func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 检查是否有用户使用该角色
	var userCount int64
	h.db.Table("sys_user_role").Where("role_id = ?", id).Count(&userCount)
	if userCount > 0 {
		response.BadRequest(c, fmt.Sprintf("该角色下还有 %d 个用户，请先解除关联后再删除", userCount), nil)
		return
	}

	result := h.db.Delete(&model.SysRole{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "角色不存在", nil)
		return
	}

	response.SuccessWithMessage(c, "角色已删除", nil)
}
