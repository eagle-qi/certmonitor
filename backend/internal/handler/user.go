package handler

import (
	"strconv"

	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/logger"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Username    string `json:"username" binding:"required,min=2,max=50"`
	RealName    string `json:"real_name" binding:"required,min=2,max=50"`
	Email       string `json:"email" binding:"required,email"`
	Phone       string `json:"phone" binding:"omitempty,max=20"`
	Password    string `json:"password" binding:"required,min=8,max=32"`
	DeptName    string `json:"dept_name" binding:"omitempty,max=50"`
	Remark      string `json:"remark" binding:"omitempty,max=255"`
	RoleIDs     []uint64 `json:"role_ids"` // 分配的角色ID列表
}

// List 用户列表（分页+筛选）
func (h *UserHandler) List(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.SysUser{})

	// 多维度筛选
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("username LIKE ? OR real_name LIKE ? OR email LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("account_status = ?", s)
	}
	if registerType := c.Query("register_type"); registerType != "" {
		rt, _ := strconv.Atoi(registerType)
		query = query.Where("register_type = ?", rt)
	}
	if dept := c.Query("dept"); dept != "" {
		query = query.Where("dept_name LIKE ?", "%"+dept+"%")
	}

	// 统计总数
	var total int64
	query.Count(&total)

	// 分页查询
	var users []model.SysUser
	offsetVal, _ := c.Get("offset")
	offset := 0
	if ov, ok := offsetVal.(int); ok {
		offset = ov
	}
	order := "create_time DESC"
	query.Preload("Roles").Offset(offset).Limit(pageSize.(int)).Order(order).Find(&users)

	// 脱敏：不返回密码
	for i := range users {
		users[i].Password = ""
	}

	response.PageSuccess(c, users, total, page.(int), pageSize.(int))
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	// 检查用户名和邮箱唯一性
	count := int64(0)
	h.db.Model(&model.SysUser{}).
		Where("username = ? OR email = ?", req.Username, req.Email).
		Count(&count)
	if count > 0 {
		response.BadRequest(c, "用户名或邮箱已存在", nil)
		return
	}

	// 加密密码
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码加密失败: %v", err)
		response.InternalError(c, "创建用户失败", nil)
		return
	}

	currentUserID := middleware.GetUserID(c)

	user := model.SysUser{
		Username:      req.Username,
		RealName:      req.RealName,
		Email:         req.Email,
		Phone:         req.Phone,
		Password:      string(hashedPwd),
		DeptName:      req.DeptName,
		AccountStatus: model.AccountStatusNormal,
		RegisterType:  model.RegisterTypeManual,
		Remark:        req.Remark,
	}

	tx := h.db.Begin()

	if tx.Create(&user).Error != nil {
		tx.Rollback()
		logger.Error("创建用户DB失败: %v", err)
		response.InternalError(c, "创建用户失败", nil)
		return
	}

	// 分配角色
	if len(req.RoleIDs) > 0 {
		for _, roleID := range req.RoleIDs {
			tx.Exec("INSERT INTO sys_user_role(user_id,role_id) VALUES (?,?) ON DUPLICATE KEY UPDATE user_id=user_id",
				user.ID, roleID)
		}
	}

	tx.Commit()

	// 重新查询包含角色的完整数据
	h.db.Preload("Roles").First(&user, user.ID)
	user.Password = ""

	// 记录审计日志
	recordOperationLog(h.db, currentUserID, "user", "create", c, map[string]interface{}{
		"target_user_id": user.ID, "target_username": user.Username,
	}, 1)

	response.SuccessWithMessage(c, "创建成功", user)
}

// Update 更新用户
func (h *UserHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的用户ID", nil)
		return
	}

	var existing model.SysUser
	if h.db.First(&existing, id).Error != nil {
		response.NotFound(c, "用户不存在", nil)
		return
	}

	var req struct {
		RealName  string  `json:"real_name" binding:"omitempty,min=2,max=50"`
		Phone     string  `json:"phone" binding:"omitempty,max=20"`
		DeptName  string  `json:"dept_name" binding:"omitempty,max=50"`
		Remark    string  `json:"remark" binding:"omitempty,max=255"`
		RoleIDs   []uint64 `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	updateData := map[string]interface{}{}
	if req.RealName != "" { updateData["real_name"] = req.RealName }
	if req.Phone != "" { updateData["phone"] = req.Phone }
	if req.DeptName != "" { updateData["dept_name"] = req.DeptName }
	updateData["remark"] = req.Remark

	tx := h.db.Begin()

	if len(updateData) > 0 {
		tx.Model(&existing).Updates(updateData)
	}

	// 更新角色
	if len(req.RoleIDs) > 0 {
		tx.Where("user_id = ?", id).Delete(&model.SysUserRole{})
		for _, rid := range req.RoleIDs {
			tx.Exec("INSERT IGNORE INTO sys_user_role(user_id,role_id) VALUES (?,?)", id, rid)
		}
	}

	tx.Commit()

	h.db.Preload("Roles").First(&existing, id)
	existing.Password = ""

	recordOperationLog(h.db, middleware.GetUserID(c), "user", "update", c,
		map[string]interface{}{"target_user_id": id}, 1)

	response.SuccessWithMessage(c, "更新成功", existing)
}

// Delete 删除(注销)用户
func (h *UserHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	var user model.SysUser
	if h.db.First(&user, id).Error != nil {
		response.NotFound(c, "用户不存在", nil)
		return
	}

	// 不允许删除自己
	if id == middleware.GetUserID(c) {
		response.BadRequest(c, "不能删除自己的账户", nil)
		return
	}

	// 软删除：标记为已注销
	h.db.Model(&user).Update("account_status", model.AccountStatusDeleted)

	recordOperationLog(h.db, middleware.GetUserID(c), "user", "delete", c,
		map[string]interface{}{"target_user_id": id, "deleted_username": user.Username}, 1)

	response.SuccessWithMessage(c, "用户已注销", nil)
}

// UpdateStatus 启用/禁用用户
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	var req struct {
		Status uint8 `json:"status" binding:"required,oneof=1 2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	result := h.db.Model(&model.SysUser{ID: id}).Update("account_status", req.Status)
	if result.RowsAffected == 0 {
		response.NotFound(c, "用户不存在", nil)
		return
	}

	statusText := "启用"
	if req.Status == 2 {
		statusText = "禁用"
	}

	recordOperationLog(h.db, middleware.GetUserID(c), "user", "status_change", c,
		map[string]interface{}{"target_user_id": id, "new_status": statusText}, 1)

	response.SuccessWithMessage(c, "用户已"+statusText, nil)
}

// ResetPassword 重置用户密码
func (h *UserHandler) ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8,max=32"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	var user model.SysUser
	if h.db.First(&user, id).Error != nil {
		response.NotFound(c, "用户不存在", nil)
		return
	}

	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	h.db.Model(&user).Update("password", string(hashedPwd))

	recordOperationLog(h.db, middleware.GetUserID(c), "user", "reset_password", c,
		map[string]interface{}{"target_user_id": id}, 1)

	response.SuccessWithMessage(c, "密码重置成功", nil)
}

// AssignRoles 为用户分配角色
func (h *UserHandler) AssignRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	var req struct {
		RoleIDs []uint64 `json:"role_ids" binding:"required,min=1,dive,uint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	tx := h.db.Begin()
	tx.Where("user_id = ?", id).Delete(&model.SysUserRole{})
	for _, rid := range req.RoleIDs {
		tx.Exec("INSERT IGNORE INTO sys_user_role(user_id,role_id) VALUES (?,?)", id, rid)
	}
	tx.Commit()

	response.SuccessWithMessage(c, "角色分配成功", nil)
}
