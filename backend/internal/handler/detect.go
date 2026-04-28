package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"certmonitor/internal/config"
	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DetectHandler struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

func NewDetectHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *DetectHandler {
	return &DetectHandler{db: db, redis: rdb, config: cfg}
}

// =========================================== 公网备案探测 ===========================================

// CompanyDetectRequest 公网探测请求
type CompanyDetectRequest struct {
	CompanyName string `json:"company_name" binding:"required,min=2,max=100"`
	IsPeriodic  bool   `json:"is_periodic"` // 是否周期性任务
	CronExpr    string `json:"cron_expr,omitempty"` // Cron表达式
}

// CreateCompanyDetectTask 创建公网备案探测任务
func (h *DetectHandler) CreateCompanyDetectTask(c *gin.Context) {
	var req CompanyDetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	currentUserID := middleware.GetUserID(c)

	task := model.DetectRecordCompany{
		CompanyName: req.CompanyName,
		TaskStatus:  1, // 待执行
		IsPeriodic:  boolToUint8(req.IsPeriodic),
		CronExpr:    req.CronExpr,
		CreateUser:  currentUserID,
	}

	h.db.Create(&task)

	// 调用 Python 探测引擎执行探测
	go h.executeCompanyDetection(task.ID, req.CompanyName)

	recordOperationLog(h.db, currentUserID, "detect", "create_company_task", c,
		map[string]interface{}{
			"task_id": task.ID, "company_name": req.CompanyName,
		}, 1)

	response.SuccessWithMessage(c, "探测任务已创建，正在后台执行", task)
}

// ListCompanyTasks 公网探测任务列表
func (h *DetectHandler) ListCompanyTasks(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.DetectRecordCompany{})
	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("task_status = ?", s)
	}

	var total int64
	query.Count(&total)

	var tasks []model.DetectRecordCompany
	offset, _ := c.Get("offset")
	query.Offset(offset).Limit(pageSize.(int)).Order("create_time DESC").Find(&tasks)

	response.PageSuccess(c, tasks, total, page.(int), pageSize.(int))
}

// CompanyTaskDetail 探测任务详情
func (h *DetectHandler) CompanyTaskDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var task model.DetectRecordCompany
	if h.db.First(&task, id).Error != nil {
		response.NotFound(c, "任务不存在", nil)
		return
	}

	response.Success(c, task)
}

// executeCompanyDetection 执行公网备案探测（调用Python引擎）
func (h *DetectHandler) executeCompanyDetection(taskID uint64, companyName string) {
	logger.Info("开始执行公网备案探测任务: taskID=%d, company=%s", taskID, companyName)

	// 更新状态为执行中
	h.db.Model(&model.DetectRecordCompany{ID: taskID}).Update("task_status", 2)

	// 构造请求体发送给 Python 探测引擎
	payload := map[string]interface{}{
		"task_id":       taskID,
		"company_name":  companyName,
		"detect_type":   "public_icp",
	}

	payloadBytes, _ := json.Marshal(payload)
	apiBaseURL := h.config.Crawler.APIBaseURL()
	resp, err := http.Post(apiBaseURL+"/api/v1/detect/public", "application/json", bytes.NewBuffer(payloadBytes))

	if err != nil {
		logger.Error("调用探测引擎失败: %v", err)
		h.db.Model(&model.DetectRecordCompany{ID: taskID}).Updates(map[string]interface{}{
			"task_status": 4,
			"fail_count":  1,
			"task_msg":    fmt.Sprintf("探测引擎连接失败: %v", err),
			"finish_time": time.Now(),
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// 根据引擎返回结果更新数据库
	// TODO: 处理探测结果，将新发现的资产写入 web_asset 表
	logger.Info("公网探测完成: taskID=%d, result=%s", taskID, string(body))
}

// =========================================== 内网网段探测 ===========================================

// IntranetDetectRequest 内网探测请求
type IntranetDetectRequest struct {
	TaskName     string `json:"task_name" binding:"required,max=100"`
	IPSegment    string `json:"ip_segment" binding:"required"` // CIDR格式: 192.168.1.0/24
	PortRange    string `json:"port_range" binding:"omitempty,default='80,443,8080'"`
	ProtocolType string `json:"protocol_type" binding:"omitempty,oneof=HTTP HTTPS ALL"` // ALL默认
	ScanRateLimit int   `json:"scan_rate_limit" binding:"omitempty,gte=1,lte=1000"`
	IsPeriodic   bool   `json:"is_periodic"`
	CronExpr     string `json:"cron_expr,omitempty"`
}

// CreateIntranetDetectTask 创建内网探测任务
func (h *DetectHandler) CreateIntranetDetectTask(c *gin.Context) {
	var req IntranetDetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	// 校验 IP/CIDR 格式
	if !isValidCIDR(req.IPSegment) {
		response.BadRequest(c, "无效的IP网段格式，请使用CIDR表示法(如192.168.1.0/24)", nil)
		return
	}

	currentUserID := middleware.GetUserID(c)

	// 计算总需扫描的 IP 数量
	totalIPs := countIPsInSegment(req.IPSegment)

	task := model.DetectRecordIntranet{
		TaskName:     req.TaskName,
		IPSegment:    req.IPSegment,
		PortRange:    req.PortRange,
		ProtocolType:  req.ProtocolType,
		ScanRateLimit: req.ScanRateLimit,
		TaskStatus:   1, // 待执行
		IsPeriodic:   boolToUint8(req.IsPeriodic),
		CronExpr:     req.CronExpr,
		CreateUser:   currentUserID,
		TotalIPs:     totalIPs,
	}

	h.db.Create(&task)

	// 调用 Python 探测引擎
	go h.executeIntranetDetection(task.ID, req)

	recordOperationLog(h.db, currentUserID, "detect", "create_intranet_task", c,
		map[string]interface{}{
			"task_id": task.ID, "ip_segment": req.IPSegment,
		}, 1)

	response.SuccessWithMessage(c, "内网探测任务已创建，正在后台执行", task)
}

// ListIntranetTasks 内网探测任务列表
func (h *DetectHandler) ListIntranetTasks(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.DetectRecordIntranet{})
	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("task_status = ?", s)
	}

	var total int64
	query.Count(&total)

	var tasks []model.DetectRecordIntranet
	offset, _ := c.Get("offset")
	query.Offset(offset).Limit(pageSize.(int)).Order("create_time DESC").Find(&tasks)

	response.PageSuccess(c, tasks, total, page.(int), pageSize.(int))
}

// IntranetTaskDetail 任务详情
func (h *DetectHandler) IntranetTaskDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var task model.DetectRecordIntranet
	if h.db.First(&task, id).Error != nil {
		response.NotFound(c, "任务不存在", nil)
		return
	}

	response.Success(c, task)
}

// IntranetTaskDetails 探测结果明细
func (h *DetectHandler) IntranetTaskDetails(c *gin.Context) {
	taskID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	var details []model.DetectIntranetDetail
	var total int64

	h.db.Model(&model.DetectIntranetDetail{}).Where("task_id = ?", taskID).Count(&total)
	offset, _ := c.Get("offset")
	h.db.Where("task_id = ?", taskID).
		Offset(offset).Limit(pageSize.(int)).
		Order("ip_address ASC, port ASC").
		Find(&details)

	response.PageSuccess(c, details, total, page.(int), pageSize.(int))
}

// executeIntranetDetection 执行内网探测（调用Python引擎）
func (h *DetectHandler) executeIntranetDetection(taskID uint64, req IntranetDetectRequest) {
	logger.Info("开始执行内网探测任务: taskID=%d, segment=%s", taskID, req.IPSegment)

	h.db.Model(&model.DetectRecordIntranet{ID: taskID}).Update("task_status", 2)

	payload := map[string]interface{}{
		"task_id":       taskID,
		"task_name":     req.TaskName,
		"ip_segment":    req.IPSegment,
		"port_range":    req.PortRange,
		"protocol_type": req.ProtocolType,
		"rate_limit":    req.ScanRateLimit,
		"detect_type":   "intranet_scan",
	}

	payloadBytes, _ := json.Marshal(payload)
	apiBaseURL := h.config.Crawler.APIBaseURL()
	resp, err := http.Post(apiBaseURL+"/api/v1/detect/intranet", "application/json", bytes.NewBuffer(payloadBytes))

	if err != nil {
		logger.Error("调用内网探测引擎失败: %v", err)
		h.db.Model(&model.DetectRecordIntranet{ID: taskID}).Updates(map[string]interface{}{
			"task_status": 4,
			"task_msg":    fmt.Sprintf("探测引擎连接失败: %v", err),
			"finish_time": time.Now(),
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	logger.Info("内网探测完成: taskID=%d, result=%s", taskID, string(body))
}

// 辅助函数

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func countIPsInSegment(cidr string) int {
	_, ipNet, _ := net.ParseCIDR(cidr)
	maskSize, _ := ipNet.Mask.Size()
	return 1 << (32 - maskSize)
}

func boolToUint8(b bool) uint8 {
	if b { return 1 }
	return 0
}
