package handler

import (
	"strconv"
	"time"

	"certmonitor/internal/config"
	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"certmonitor/pkg/redis"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AssetHandler struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

func NewAssetHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *AssetHandler {
	return &AssetHandler{db: db, redis: rdb, config: cfg}
}

// CreateAssetReq 新增资产请求
type CreateAssetReq struct {
	URLAddress        string `json:"url_address" binding:"required,url"`
	ProtocolType      uint8  `json:"protocol_type" binding:"required,oneof=1 2"`
	IPAddress         string `json:"ip_address" binding:"omitempty,ip"`
	Port              int    `json:"port" binding:"omitempty,gte=0,lte=65535"`
	CompanyName       string `json:"company_name" binding:"required,max=100"`
	BusinessName      string `json:"business_name" binding:"required,max=100"`
	BusinessDesc      string `json:"business_desc" binding:"omitempty,max=2000"`
	JobPosition       uint8  `json:"job_position" binding:"required,oneof=0 1 2 3 4 5"`
	DutyUserName      string `json:"duty_user_name" binding:"required,max=50"`
	DutyUserPhone     string `json:"duty_user_phone" binding:"required,max=20"`
	DutyUserEmail     string `json:"duty_user_email" binding:"required,email"`
	ProjectName       string `json:"project_name" binding:"omitempty,max=100"`
	ProjectManagerName string `json:"project_manager_name" binding:"omitempty,max=50"`
	ProjectManagerEmail string `json:"project_manager_email" binding:"omitempty,email"`
	DeptName          string `json:"dept_name" binding:"omitempty,max=50"`
	Remark            string `json:"remark" binding:"omitempty,max=2000"`
}

// List 资产列表（分页+多维度筛选）
func (h *AssetHandler) List(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.WebAsset{})

	// URL模糊搜索
	if url := c.Query("url"); url != "" {
		query = query.Where("url_address LIKE ?", "%"+url+"%")
	}
	// 公司名称筛选
	if company := c.Query("company"); company != "" {
		query = query.Where("company_name = ?", company)
	}
	// 业务系统筛选
	if business := c.Query("business"); business != "" {
		query = query.Where("business_name = ?", business)
	}
	// 归属岗位筛选
	if position := c.Query("position"); position != "" {
		p, _ := strconv.Atoi(position)
		query = query.Where("job_position = ?", p)
	}
	// 负责人筛选
	if dutyUser := c.Query("duty_user"); dutyUser != "" {
		query = query.Where("duty_user_name LIKE ? OR duty_user_email LIKE ?",
			"%"+dutyUser+"%", "%"+dutyUser+"%")
	}
	// 项目名称筛选
	if project := c.Query("project"); project != "" {
		query = query.Where("project_name LIKE ?", "%"+project+"%")
	}
	// 协议类型筛选
	if protocol := c.Query("protocol"); protocol != "" {
		p, _ := strconv.Atoi(protocol)
		query = query.Where("protocol_type = ?", p)
	}
	// 资产状态筛选
	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("asset_status = ?", s)
	}
	// 资产来源筛选
	if source := c.Query("source"); source != "" {
		s, _ := strconv.Atoi(source)
		query = query.Where("asset_source = ?", s)
	}

	var total int64
	query.Count(&total)

	var assets []model.WebAsset
	offset, _ := c.Get("offset")
	order := "create_time DESC"
	query.Preload("Certificates").Offset(offset).Limit(pageSize.(int)).Order(order).Find(&assets)

	response.PageSuccess(c, assets, total, page.(int), pageSize.(int))
}

// GetDetail 资产详情
func (h *AssetHandler) GetDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var asset model.WebAsset
	if h.db.Preload("Certificates").First(&asset, id).Error != nil {
		response.NotFound(c, "资产不存在", nil)
		return
	}

	response.Success(c, asset)
}

// Create 新增资产
func (h *AssetHandler) Create(c *gin.Context) {
	var req CreateAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	// 唯一性检查
	var count int64
	h.db.Model(&model.WebAsset{}).Where("url_address = ? AND protocol_type = ?",
		req.URLAddress, req.ProtocolType).Count(&count)
	if count > 0 {
		response.BadRequest(c, "该URL+协议组合的资产已存在", nil)
		return
	}

	currentUserID := middleware.GetUserID(c)

	asset := model.WebAsset{
		URLAddress:        req.URLAddress,
		ProtocolType:      req.ProtocolType,
		IPAddress:         req.IPAddress,
		Port:              req.Port,
		CompanyName:       req.CompanyName,
		AssetStatus:       model.AssetStatusPending, // 默认待确认
		AssetSource:       model.SourceManual,
		BusinessName:      req.BusinessName,
		BusinessDesc:      req.BusinessDesc,
		JobPosition:       req.JobPosition,
		DutyUserName:      req.DutyUserName,
		DutyUserPhone:     req.DutyUserPhone,
		DutyUserEmail:     req.DutyUserEmail,
		ProjectName:       req.ProjectName,
		ProjectManagerName: req.ProjectManagerName,
		ProjectManagerEmail: req.ProjectManagerEmail,
		DeptName:          req.DeptName,
		Remark:            req.Remark,
		CreatorID:         &currentUserID,
	}

	if h.db.Create(&asset).Error != nil {
		response.InternalError(c, "创建资产失败", nil)
		return
	}

	recordOperationLog(h.db, currentUserID, "asset", "create", c,
		map[string]interface{}{"asset_id": asset.ID, "url": asset.URLAddress}, 1)

	response.SuccessWithMessage(c, "资产创建成功(状态为待确认)", asset)
}

// Update 编辑资产
func (h *AssetHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var existing model.WebAsset
	if h.db.First(&existing, id).Error != nil {
		response.NotFound(c, "资产不存在", nil)
		return
	}

	var req CreateAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	updates := map[string]interface{}{
		"url_address":          req.URLAddress,
		"protocol_type":        req.ProtocolType,
		"ip_address":           req.IPAddress,
		"port":                 req.Port,
		"company_name":         req.CompanyName,
		"business_name":        req.BusinessName,
		"business_desc":        req.BusinessDesc,
		"job_position":         req.JobPosition,
		"duty_user_name":       req.DutyUserName,
		"duty_user_phone":      req.DutyUserPhone,
		"duty_user_email":      req.DutyUserEmail,
		"project_name":         req.ProjectName,
		"project_manager_name": req.ProjectManagerName,
		"project_manager_email": req.ProjectManagerEmail,
		"dept_name":            req.DeptName,
		"remark":               req.Remark,
	}

	h.db.Model(&existing).Updates(updates)

	recordOperationLog(h.db, middleware.GetUserID(c), "asset", "update", c,
		map[string]interface{}{"asset_id": id}, 1)

	response.SuccessWithMessage(c, "资产更新成功", nil)
}

// Delete 删除资产
func (h *AssetHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	result := h.db.Delete(&model.WebAsset{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "资产不存在", nil)
		return
	}

	recordOperationLog(h.db, middleware.GetUserID(c), "asset", "delete", c,
		map[string]interface{}{"asset_id": id}, 1)

	response.SuccessWithMessage(c, "资产已删除", nil)
}

// Confirm 审核通过
func (h *AssetHandler) Confirm(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var asset model.WebAsset
	if h.db.First(&asset, id).Error != nil {
		response.NotFound(c, "资产不存在", nil)
		return
	}

	now := time.Now()
	userID := middleware.GetUserID(c)

	h.db.Model(&asset).Updates(map[string]interface{}{
		"asset_status":   model.AssetStatusConfirmed,
		"confirm_user_id": userID,
		"confirm_time":   now,
		"confirm_remark": "",
	})

	recordOperationLog(h.db, userID, "asset", "confirm", c,
		map[string]interface{}{"asset_id": id, "url": asset.URLAddress}, 1)

	response.SuccessWithMessage(c, "审核通过", nil)
}

// Reject 审核驳回
func (h *AssetHandler) Reject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req struct {
		Reason string `json:"reason" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入驳回原因", nil)
		return
	}

	var asset model.WebAsset
	if h.db.First(&asset, id).Error != nil {
		response.NotFound(c, "资产不存在", nil)
		return
	}

	now := time.Now()
	userID := middleware.GetUserID(c)

	h.db.Model(&asset).Updates(map[string]interface{}{
		"asset_status":   model.AssetStatusInvalid,
		"confirm_user_id": userID,
		"confirm_time":   now,
		"confirm_remark": req.Reason,
	})

	recordOperationLog(h.db, userID, "asset", "reject", c,
		map[string]interface{}{"asset_id": id, "reason": req.Reason}, 1)

	response.SuccessWithMessage(c, "已驳回: "+req.Reason, nil)
}

// BatchImport 批量导入资产（Excel文件上传+解析+校验+入库）
func (h *AssetHandler) BatchImport(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请选择要导入的Excel文件", nil)
		return
	}

	// 校验文件格式
	ext := filepath.Ext(file.Filename)
	if ext != ".xlsx" && ext != ".xls" && ext != ".csv" {
		response.BadRequest(c, "只支持 .xlsx/.xls/.csv 格式文件", nil)
		return
	}

	// 校验文件大小（最大10MB）
	if file.Size > h.config.Storage.MaxUploadSize {
		response.BadRequest(c, "文件大小不能超过10MB", nil)
		return
	}

	currentUserID := middleware.GetUserID(c)

	// 保存文件
	filePath := path.Join(h.config.Storage.Path, "import", time.Now().Format("20060102150405")+"_"+file.Filename)
	os.MkdirAll(path.Dir(filePath), 0755)
	c.SaveUploadedFile(file, filePath)

	// 创建导入任务记录
	task := model.WebAssetImportTask{
		FileName:   file.Filename,
		FilePath:   filePath,
		TaskStatus: 1, // 处理中
		OperatorID: currentUserID,
	}
	h.db.Create(&task)

	// 异步执行导入解析任务（避免阻塞HTTP响应）
	go h.processImportFile(task.ID, filePath, file.Filename, currentUserID)

	response.SuccessWithMessage(c, "导入任务已提交，正在后台处理中", gin.H{
		"task_id": task.ID,
	})
}

// processImportFile 异步处理导入文件
func (h *AssetHandler) processImportFile(taskID uint64, filePath, fileName string, operatorID uint64) {
	// TODO: 实现完整的 Excel 解析和批量入库逻辑
	// 1. 使用 excelize 库读取 Excel
	// 2. 逐行校验字段（必填、格式、枚举值）
	// 3. 基于 URL+Protocol 去重
	// 4. 合法数据写入 web_asset 表（状态设为待确认）
	// 5. 错误数据写入 web_asset_import_log 表
	// 6. 更新 import_task 的统计结果

	logger.Info("开始处理导入任务: taskID=%d, file=%s", taskID, fileName)

	// 模拟处理结果（实际实现需要接入 excelize 库）
	successCount := 0
	failCount := 0

	h.db.Model(&task).Where("id = ?", taskID).Updates(map[string]interface{}{
		"task_status":   2,
		"success_count": successCount,
		"fail_count":    failCount,
		"task_result":   fmt.Sprintf("共处理X条，成功%d条，失败%d条", successCount, failCount),
		"finish_time":   time.Now(),
	})
}

// DownloadTemplate 下载标准导入模板
func (h *AssetHandler) DownloadTemplate(c *gin.Context) {
	// TODO: 动态生成标准 Excel 导入模板（使用 excelize 库）
	// 包含所有必填字段、枚举值说明、示例数据

	templateContent := `
资产导入模板说明:
==================
必填字段(*):
  - URL地址*: 完整的URL地址，如 https://www.example.com
  - 协议类型*: HTTP(填1) 或 HTTPS(填2)
  - 所属公司*: 备案主体企业名称
  - 业务系统名称*: 自定义业务标识
  - 归属岗位*: 开发(1)/测试(2)/生产(3)/预发布(4)/办公系统(5)
  - 负责人姓名*:
  - 负责人手机号*: 11位手机号
  - 负责人工作邮箱*:

选填字段:
  - IP地址/解析IP
  - 端口号
  - 业务系统描述
  - 所属项目名称
  - 项目经理名称
  - 项目经理邮箱
  - 归属部门
  - 备注

注意:
  1. URL+协议 组合必须唯一
  2. 手机号、邮箱、IP地址会做格式校验
  3. 导入后资产默认进入"待确认"列表，等待管理员审核
`
	
	c.Header("Content-Disposition", "attachment; filename=asset_import_template.txt")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(templateContent))
}

// ImportLog 导入错误日志查看
func (h *AssetHandler) ImportLog(c *gin.Context) {
	taskID, _ := strconv.ParseUint(c.Param("taskId"), 10, 64)

	var logs []model.WebAssetImportLog
	h.db.Where("import_task_id = ?", taskID).Order("row_num ASC").Find(&logs)

	response.Success(c, logs)
}

// Export 导出资产Excel
func (h *AssetHandler) Export(c *gin.Context) {
	// TODO: 使用 excelize 库动态生成 Excel 文件并返回下载

	// 先获取符合筛选条件的数据（复用 List 的筛选逻辑）
	query := h.db.Model(&model.WebAsset{})

	if url := c.Query("url"); url != "" {
		query = query.Where("url_address LIKE ?", "%"+url+"%")
	}
	if company := c.Query("company"); company != "" {
		query = query.Where("company_name = ?", company)
	}
	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("asset_status = ?", s)
	}

	var assets []model.WebAsset
	query.Find(&assets)

	// TODO: 生成 Excel 并写入 ResponseWriter
	_ = assets

	c.Header("Content-Disposition", "attachment; filename=assets_export.xlsx")
	response.SuccessWithMessage(c, "导出功能开发中，请稍候", nil)
}
