package handler

import (
	"fmt"
	"strconv"
	"time"

	"certmonitor/internal/config"
	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CertApplyHandler struct {
	db     *gorm.DB
	config *config.Config
}

func NewCertApplyHandler(db *gorm.DB, cfg *config.Config) *CertApplyHandler {
	return &CertApplyHandler{db: db, config: cfg}
}

// ApplyRequest 申请证书请求
type ApplyRequest struct {
	ApplyType      uint8   `json:"apply_type" binding:"required,oneof=1 2"` // 1公网域名 2内网IP
	ApplyAddr       string `json:"apply_addr" binding:"required,max=100"` // 域名或IP
	SANAddrs        string `json:"san_addrs,omitempty"` // 额外SAN域名/IP
	AssetID         *uint64 `json:"asset_id,omitempty"`
	VerifyMethod    uint8   `json:"verify_method" binding:"omitempty,oneof=1 2"` // 1DNS验证 2HTTP文件验证
	EncryptAlgorithm string `json:"encrypt_algorithm" binding:"omitempty,oneof=RSA ECDSA"`
	KeySize         int     `json:"key_size" binding:"omitempty,oneof=2048 3072 4096 256 384"`
	ValidDays       int     `json:"valid_days" binding:"omitempty,gte=1,lte=3650"`
}

// SubmitApplication 提交证书申请
func (h *CertApplyHandler) SubmitApplication(c *gin.Context) {
	var req ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	currentUserID := middleware.GetUserID(c)

	// 设置默认值
	if req.EncryptAlgorithm == "" { req.EncryptAlgorithm = "RSA" }
	if req.KeySize == 0 { req.KeySize = 2048 }
	if req.ValidDays == 0 { req.ValidDays = 365 }

	task := model.SslCertApplyTask{
		ApplyType:          req.ApplyType,
		ApplyAddr:           req.ApplyAddr,
		SANAddrs:            req.SANAddrs,
		AssetID:             req.AssetID,
		VerifyMethod:        req.VerifyMethod,
		EncryptAlgorithm:    req.EncryptAlgorithm,
		KeySize:             req.KeySize,
		ValidDays:           req.ValidDays,
		TaskStatus:          1, // 申请中
		ApplyUser:           currentUserID,
	}

	h.db.Create(&task)

	// 异步调用 Python 引擎执行签发（公网用ACME协议申请，内网自签名）
	go h.executeCertSign(&task)

	recordOperationLog(h.db, currentUserID, "cert_apply", "submit", c,
		map[string]interface{}{
			"task_id": task.ID, "type": task.ApplyType, "addr": task.ApplyAddr,
		}, 1)

	response.SuccessWithMessage(c, "证书申请已提交，正在后台处理中", task)
}

// ListTasks 申请任务列表
func (h *CertApplyHandler) ListTasks(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.SslCertApplyTask{}).Where("apply_user = ?", middleware.GetUserID(c))

	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("task_status = ?", s)
	}
	if applyType := c.Query("type"); applyType != "" {
		t, _ := strconv.Atoi(applyType)
		query = query.Where("apply_type = ?", t)
	}

	var total int64
	query.Count(&total)

	var tasks []model.SslCertApplyTask
	offset, _ := c.Get("offset")
	query.Offset(offset).Limit(pageSize.(int)).Order("create_time DESC").Find(&tasks)

	response.PageSuccess(c, tasks, total, page.(int), pageSize.(int))
}

// TaskDetail 任务详情
func (h *CertApplyHandler) TaskDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var task model.SslCertApplyTask
	if h.db.First(&task, id).Error != nil {
		response.NotFound(c, "任务不存在", nil)
		return
	}

	response.Success(c, task)
}

// Retry 重试失败的签发任务
func (h *CertApplyHandler) Retry(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var task model.SslCertApplyTask
	if h.db.First(&task, id).Error != nil || task.TaskStatus != 4 {
		response.BadRequest(c, "只有失败状态的任务可以重试", nil)
		return
	}

	// 重置状态并重新执行
	h.db.Model(&task).Updates(map[string]interface{}{
		"task_status": 1,
		"fail_reason": "",
	})
	go h.executeCertSign(&task)

	response.SuccessWithMessage(c, "已重新提交签发任务", nil)
}

// DownloadCertPackage 下载签发成功的证书包
func (h *CertApplyHandler) DownloadCertPackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64

	var task model.SslCertApplyTask
	if h.db.First(&task, id).Error != nil || task.CertFilePath == "" {
		response.NotFound(c, "证书文件不存在或未签发成功", nil)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=cert_%s_%d.zip",
		task.ApplyAddr, time.Now().Unix()))
	c.File(task.CertFilePath)
}

// executeCertSign 调用Python引擎执行证书签发
func (h *CertApplyHandler) executeCertSign(task *model.SslCertApplyTask) {
	logger.Info("开始处理证书签发任务: taskID=%d, addr=%s, type=%d",
		task.ID, task.ApplyAddr, task.ApplyType)

	h.db.Model(task).Update("task_status", 2) // 校验中

	payload := map[string]interface{}{
		"task_id":            task.ID,
		"apply_type":         task.ApplyType,
		"apply_addr":         task.ApplyAddr,
		"san_addrs":          task.SANAddrs,
		"verify_method":      task.VerifyMethod,
		"encrypt_algorithm":  task.EncryptAlgorithm,
		"key_size":           task.KeySize,
		"valid_days":         task.ValidDays,
	}

	// TODO: 调用 Python 签发引擎接口
	apiBaseURL := h.config.Crawler.APIBaseURL()

	if task.ApplyType == 1 {
		// 公网域名 -> 调用 ACME 协议签发（Let's Encrypt等）
		logger.Info("公网域名ACME证书签发: domain=%s", task.ApplyAddr)
	} else {
		// 内网 IP -> 自签名证书生成
		logger.Info("内网IP自签名证书生成: ip=%s", task.ApplyAddr)
	}

	// 模拟签发结果（实际需调用引擎API）
	// TODO: 根据真实返回结果更新数据库
	h.db.Model(task).Updates(map[string]interface{}{
		"task_status": 3,
		"finish_time": time.Now(),
	})

	// 如果签发成功且关联了资产，自动写入 ssl_cert_info 表
	if task.AssetID != nil && task.TaskStatus == 3 {
		validEnd := time.Now().AddDate(0, 0, task.ValidDays)
		newCert := model.SslCertInfo{
			AssetID:          task.AssetID,
			DomainIP:         task.ApplyAddr,
			CertType:         task.ApplyType,
			CertSource:       3, // 系统自助申请
			ValidStartTime:   time.Now(),
			ValidEndTime:     validEnd,
			DaysRemaining:    task.ValidDays,
			CertStatus:       1,
			EncryptAlgorithm: task.EncryptAlgorithm,
			KeySize:          task.KeySize,
			ApplyTaskID:      &task.ID,
		}
		h.db.Create(&newCert)

		// 更新申请任务的关联信息
		h.db.Model(task).Update("asset_id", task.AssetID)
	}
}
