package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"certmonitor/internal/config"
	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	certRedis "certmonitor/pkg/redis"
	"certmonitor/pkg/logger"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

func NewAuthHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, redis: rdb, config: cfg}
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 可以是用户名或邮箱
	Password string `json:"password" binding:"required"`
	Captcha   string `json:"captcha,omitempty"`
	CaptchaID string `json:"captcha_id,omitempty"`
}

// Login 用户登录（支持账号密码/邮箱密码/验证码登录）
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	ctx := c.Request.Context()
	ip := c.ClientIP()

	// 查找用户（支持用用户名或邮箱登录）
	var user model.SysUser
	err := h.db.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 安全考虑：不明确提示用户不存在或密码错误
			response.Unauthorized(c, "用户名或密码错误", nil)
			return
		}
		logger.Error("查询用户失败: %v", err)
		response.InternalError(c, "服务器内部错误", nil)
		return
	}

	// 检查账号状态
	if user.AccountStatus == model.AccountStatusDisabled {
		response.Forbidden(c, "账号已被禁用，请联系管理员", nil)
		return
	}
	if user.AccountStatus == model.AccountStatusDeleted {
		response.NotFound(c, "账号已注销", nil)
		return
	}
	if user.AccountStatus == model.AccountStatusLocked {
		response.Forbidden(c, "账号已被锁定，请稍后重试或联系管理员", nil)
		return
	}

	// 检查 Redis 中是否被锁定
	locked, _ := certRedis.IsAccountLocked(ctx, req.Username)
	if locked {
		response.Forbidden(c, fmt.Sprintf("连续登录失败次数过多，账号已锁定 %d 分钟，请稍后重试", certRedis.LockDuration/time.Minute), nil)
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// 记录登录失败
		count, _ := certRedis.RecordLoginFailure(ctx, req.Username)

		if count >= certRedis.LoginMaxAttempts-1 {
			// 更新数据库中的锁状态
			lockUntil := time.Now().Add(certRedis.LockDuration)
			h.db.Model(&user).Updates(map[string]interface{}{
				"login_fail_count": count,
				"lock_until":       lockUntil,
			})
			response.Forbidden(c, "连续登录失败次数过多，账号已锁定", nil)
		} else {
			h.db.Model(&user).Update("login_fail_count", count)
			response.Unauthorized(c, fmt.Sprintf("用户名或密码错误(剩余%d次尝试机会)", certRedis.LoginMaxAttempts-count), nil)
		}
		return
	}

	// 登录成功：清除失败记录
	certRedis.ClearLoginFailures(ctx, req.Username)
	h.db.Model(&user).Updates(map[string]interface{}{
		"last_login_ip":    ip,
		"last_login_time": time.Now(),
		"login_fail_count": 0,
		"lock_until":      nil,
	})

	// 获取用户角色
	var roles []model.SysRole
	h.db.Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role.id").
		Where("sys_user_role.user_id = ?", user.ID).Find(&roles)

	var roleCodes []string
	for _, r := range roles {
		roleCodes = append(roleCodes, r.RoleCode)
	}

	// 生成 JWT Token
	token, err := middleware.GenerateToken(
		user.ID, user.Username, user.Email, roleCodes,
		h.config.JWT.Secret, h.config.JWT.ExpireHours,
	)
	if err != nil {
		logger.Error("生成Token失败: %v", err)
		response.InternalError(c, "登录失败，请重试", nil)
		return
	}

	// 缓存用户会话信息到 Redis
	certRedis.SaveUserSession(ctx, user.ID, map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"real_name": user.RealName,
		"ip":       ip,
		"login_at": time.Now(),
	})

	// 记录操作日志
	go h.recordLog(&user, "auth", "login", ip, map[string]interface{}{"method": "password"}, 1)

	c.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "登录成功",
		Data: gin.H{
			"token":          token,
			"expire_hours":   h.config.JWT.ExpireHours,
			"user":           user,
			"roles":          roleCodes,
		},
	})
}

// RegisterRequest 注册请求结构体
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8,max=32"`
	RealName    string `json:"real_name" binding:"required,min=2,max=50"`
	DeptName    string `json:"dept_name" binding:"max=50"`
	Captcha     string `json:"captcha" binding:"required,len=6"`
}

// Register 邮箱自助注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	ctx := c.Request.Context()

	// 1. 检查注册开关是否开启
	registerEnabled := getSystemConfig(h.db, "register_enabled")
	if registerEnabled != "1" {
		response.Forbidden(c, "当前未开放注册功能", nil)
		return
	}

	// 2. 检查邮箱白名单
	emailWhitelist := getSystemConfig(h.db, "register_email_whitelist")
	if emailWhitelist != "" {
		domains := strings.Split(emailWhitelist, ",")
		allowed := false
		for _, d := range domains {
			d = strings.TrimSpace(d)
			if d != "" && strings.HasSuffix(req.Email, d) {
				allowed = true
				break
			}
		}
		if !allowed {
			response.Forbidden(c, "当前仅允许指定域名的邮箱注册", nil)
			return
		}
	}

	// 3. 校验密码复杂度
	if !validatePassword(req.Password) {
		response.BadRequest(c, "密码需包含大小写字母和数字，长度8-32位", nil)
		return
	}

	// 4. 验证邮箱验证码
	valid, err := certRedis.VerifyCaptcha(ctx, req.Email, req.Captcha, "register")
	if err != nil || !valid {
		response.BadRequest(c, "验证码无效或已过期", nil)
		return
	}

	// 5. 检查邮箱是否已存在
	count := int64(0)
	h.db.Model(&model.SysUser{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		response.BadRequest(c, "该邮箱已被注册", nil)
		return
	}

	// 6. 加密密码并创建用户
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码加密失败: %v", err)
		response.InternalError(c, "注册失败，请重试", nil)
		return
	}

	user := model.SysUser{
		Username:      req.Email[:strings.Index(req.Email, "@")], // 用@前的部分作为默认用户名
		RealName:      req.RealName,
		Email:         req.Email,
		Password:      string(hashedPwd),
		DeptName:      req.DeptName,
		AccountStatus: model.AccountStatusNormal,
		RegisterType:  model.RegisterTypeEmail,
	}

	result := h.db.Create(&user)
	if result.Error != nil {
		logger.Error("创建用户失败: %v", result.Error)
		response.InternalError(c, "注册失败，请重试", nil)
		return
	}

	// 7. 分配默认角色（查看员）
	defaultRoleID := uint64(5) // viewer 角色 ID
	h.db.Exec("INSERT IGNORE INTO sys_user_role(user_id,role_id) VALUES (?,?)",
		user.ID, defaultRoleID)

	// 8. 记录日志
	go func() {
		h.db.Create(&model.SysOperationLog{
			UserID:          &user.ID,
			Username:        user.Username,
			OperationModule: "auth",
			OperationType:   "register",
			RequestIP:       c.ClientIP(),
			RequestParam:    fmt.Sprintf(`{"email":"%s","name":"%s"}`, req.Email, req.RealName),
			OperationResult: 1,
			UserAgent:       c.GetHeader("User-Agent"),
		})
	}()

	response.SuccessWithMessage(c, "注册成功", gin.H{
		"user_id": user.ID,
		"email":   user.Email,
	})
}

// SendCaptchaRequest 发送验证码请求
type SendCaptchaRequest struct {
	Email      string `json:"email" binding:"required,email"`
	CaptchaType string `json:"captcha_type" binding:"required,oneof=register login reset_password"` // register / login / reset_password
}

// SendCaptcha 发送邮箱验证码
func (h *AuthHandler) SendCaptcha(c *gin.Context) {
	var req SendCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	ctx := c.Request.Context()

	// 1. 检查发送频率限制
	canSend, err := certRedis.CheckCaptchaRateLimit(ctx, req.Email)
	if err != nil {
		logger.Error("检查频率限制失败: %v", err)
		response.InternalError(c, "服务异常，请稍后重试", nil)
		return
	}
	if !canSend {
		response.BadRequest(c, "今日验证码发送已达上限，请明天再试", nil)
		return
	}

	// 2. 如果是注册类型，检查是否已注册
	if req.CaptchaType == "register" {
		count := int64(0)
		h.db.Model(&model.SysUser{}).Where("email = ?", req.Email).Count(&count)
		if count > 0 {
			response.BadRequest(c, "该邮箱已被注册，可直接登录", nil)
			return
		}
	}

	// 3. 生成6位数字验证码
	captcha := generateRandomNumber(6)

	// 4. 保存验证码到 Redis
	if err := certRedis.SaveCaptcha(ctx, req.Email, captcha, req.CaptchaType); err != nil {
		logger.Error("保存验证码失败: %v", err)
		response.InternalError(c, "服务异常，请稍后重试", nil)
		return
	}

	// 5. 发送邮件（异步执行）
	go sendEmailCaptcha(h.config.Mail, req.Email, captcha, req.CaptchaType)

	response.SuccessWithMessage(c, "验证码已发送，请在10分钟内完成验证", nil)
}

// SSOLogin SSO单点登录入口
func (h *AuthHandler) SSOLogin(c *gin.Context) {
	if !h.config.SSO.Enabled {
		response.Forbidden(c, "SSO单点登录功能未开启", nil)
		return
	}

	// 构造 SSO 授权跳转 URL
	state := generateRandomString(16)
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		h.config.SSO.AuthURL,
		h.config.SSO.ClientID,
		url.QueryEscape(h.config.SSO.RedirectURI),
		state,
	)

	// 将 state 存入 Redis 用于回调时校验
	certRedis.Set(c.Request.Context(), "certmonitor:sso:state:"+state, "", 10*time.Minute)

	c.JSON(http.StatusOK, response.Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"sso_url": authURL,
		},
	})
}

// SSOCallback SSO 回调处理
func (h *AuthHandler) SSOCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.BadRequest(c, "缺少必要的回调参数", nil)
		return
	}

	// TODO: 实现完整的 SSO 回调流程
	// 1. 使用 code 换取 access_token
	// 2. 获取用户信息
	// 3. 匹配或创建影子账号
	// 4. 生成 JWT Token 并返回

	response.Unauthorized(c, "SSO回调功能开发中...", nil)
}

// 辅助方法

func validatePassword(pwd string) bool {
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(pwd)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(pwd)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(pwd)
	lenOk := len(pwd) >= 8 && len(pwd) <= 32
	return hasUpper && hasLower && hasDigit && lenOk
}

func generateRandomNumber(length int) string {
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = byte('0' + rand.Intn(10))
	}
	return string(result)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func sendEmailCaptcha(mailCfg config.MailConfig, to, captcha, captchaType string) {
	// TODO: 实现真实的邮件发送逻辑（使用 SMTP 或第三方邮件服务 API）
	// 目前仅记录日志
	logger.Info("发送验证码邮件: 收件人=%s, 类型=%s, 验证码=%s", to, captchaType, captcha)
}

func getSystemConfig(db *gorm.DB, key string) string {
	var cfg model.SysConfig
	db.Where("config_key = ?", key).First(&cfg)
	if cfg.ConfigValue == nil {
		return ""
	}
	return *cfg.ConfigValue
}

func (h *AuthHandler) recordLog(user *model.SysUser, module, opType, ip string, param interface{}, result uint8) {
	paramJSON, _ := json.Marshal(param)
	h.db.Create(&model.SysOperationLog{
		UserID:          &user.ID,
		Username:        user.Username,
		OperationModule: module,
		OperationType:   opType,
		RequestIP:       ip,
		RequestParam:    string(paramJSON),
		OperationResult: result,
	})
}
