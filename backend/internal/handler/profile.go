package handler

import (

	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	db *gorm.DB
}

func NewProfileHandler(db *gorm.DB) *ProfileHandler {
	return &ProfileHandler{db: db}
}

// GetProfile 获取当前用户个人信息
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var user model.SysUser
	if h.db.Preload("Roles").First(&user, userID).Error != nil {
		response.NotFound(c, "用户信息不存在", nil)
		return
	}

	user.Password = ""
	response.Success(c, user)
}

// UpdateProfile 更新个人资料
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		RealName string `json:"real_name" binding:"omitempty,min=2,max=50"`
		Phone    string `json:"phone" binding:"omitempty,max=20"`
		Avatar   string `json:"avatar" binding:"omitempty,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	updates := make(map[string]interface{})
	if req.RealName != "" { updates["real_name"] = req.RealName }
	if req.Phone != "" { updates["phone"] = req.Phone }
	if req.Avatar != "" { updates["avatar"] = req.Avatar }

	if len(updates) > 0 {
		h.db.Model(&model.SysUser{ID: userID}).Updates(updates)
	}

	response.SuccessWithMessage(c, "资料更新成功", nil)
}

// ChangePassword 修改密码
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8,max=32"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	var user model.SysUser
	if h.db.First(&user, userID).Error != nil {
		response.NotFound(c, "用户不存在", nil)
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		response.BadRequest(c, "原密码错误", nil)
		return
	}

	// 加密新密码
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	h.db.Model(&user).Update("password", string(hashedPwd))

	response.SuccessWithMessage(c, "密码修改成功", nil)
}

// MyOperationLogs 我的操作日志
func (h *ProfileHandler) MyOperationLogs(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	var total int64
	h.db.Model(&model.SysOperationLog{}).Where("user_id = ?", userID).Count(&total)

	var logs []model.SysOperationLog
	offsetVal, _ := c.Get("offset")
	offset := 0
	if ov, ok := offsetVal.(int); ok {
		offset = ov
	}
	h.db.Where("user_id = ?", userID).
		Order("create_time DESC").
		Offset(offset).Limit(pageSize.(int)).Find(&logs)

	response.PageSuccess(c, logs, total, page.(int), pageSize.(int))
}
