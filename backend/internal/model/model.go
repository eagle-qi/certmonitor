package model

import (
	"time"
)

// ===========================================
// 用户权限模型
// ===========================================

// SysUser 系统用户表
type SysUser struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username        string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	RealName        string         `gorm:"size:50;not null;default:''" json:"real_name"`
	Email           string         `gorm:"size:100;not null;uniqueIndex" json:"email"`
	Phone           string         `gorm:"size:20" json:"phone"`
	Password        string         `gorm:"size:100;not null" json:"-"` // 密码不返回前端
	DeptName        string         `gorm:"size:50" json:"dept_name"`
	Avatar          string         `gorm:"size:255" json:"avatar"`
	AccountStatus   uint8          `gorm:"not null;default:1;comment:'1正常 2禁用 3锁定 4已注销'" json:"account_status"`
	RegisterType    uint8          `gorm:"not null;default:1;comment:'1手动创建 2邮箱注册 3SSO'" json:"register_type"`
	SSOUniqueID     string         `gorm:"size:100;uniqueIndex" json:"-"`
	LastLoginIP     string         `gorm:"size:50" json:"last_login_ip"`
	LastLoginTime   *time.Time     `json:"last_login_time"`
	LoginFailCount  int            `gorm:"not null;default:0" json:"-"`
	LockUntil       *time.Time     `json:"-"`
	CreateTime      time.Time      `json:"create_time"`
	UpdateTime      time.Time      `json:"update_time"`
	Remark          string         `gorm:"size:255" json:"remark"`
	Roles           []SysRole      `gorm:"many2many:sys_user_role;" json:"roles,omitempty"`
}

func (SysUser) TableName() string { return "sys_user" }

const (
	AccountStatusNormal  uint8 = 1
	AccountStatusDisabled uint8 = 2
	AccountStatusLocked  uint8 = 3
	AccountStatusDeleted uint8 = 4
)

const (
	RegisterTypeManual uint8 = 1
	RegisterTypeEmail  uint8 = 2
	RegisterTypeSSO    uint8 = 3
)

// SysRole 角色表
type SysRole struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleName    string    `gorm:"size:50;not null;uniqueIndex" json:"role_name"`
	RoleCode    string    `gorm:"size:50;not null;uniqueIndex" json:"role_code"`
	Description string    `gorm:"size:255" json:"description"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	Status      uint8     `gorm:"not null;default:1" json:"status"`
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
}

func (SysRole) TableName() string { return "sys_role" }

// SysUserRole 用户角色关联表
type SysUserRole struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;index" json:"user_id"`
	RoleID    uint64    `gorm:"not null;index" json:"role_id"`
	CreateTime time.Time `json:"create_time"`
}

func (SysUserRole) TableName() string { return "sys_user_role" }

// SysEmailCaptcha 邮箱验证码表
type SysEmailCaptcha struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Email      string    `gorm:"size:100;not null;index" json:"email"`
	Captcha    string    `gorm:"size:10;not null" json:"captcha"`
	CaptchaType uint8    `gorm:"not null;default:1" json:"captcha_type"`
	ExpireTime time.Time `gorm:"not null;index" json:"expire_time"`
	Used       uint8     `gorm:"not null;default:0" json:"used"`
	IPAddress  string    `gorm:"size:50" json:"-"`
	CreateTime time.Time `json:"create_time"`
}

func (SysEmailCaptcha) TableName() string { return "sys_email_captcha" }

// SysSSOLoginLog SSO登录日志表
type SysSSOLoginLog struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      *uint64   `json:"user_id"`
	SSOUniqueID string    `gorm:"size:100;not null;index" json:"sso_unique_id"`
	LoginIP     string    `gorm:"size:50;not null" json:"login_ip"`
	LoginStatus uint8     `gorm:"not null" json:"login_status"`
	ErrorMsg    string    `gorm:"type:text" json:"error_msg,omitempty"`
	CreateTime  time.Time `json:"create_time"`
}

func (SysSSOLoginLog) TableName() string { return "sys_sso_login_log" }

// ===========================================
// 系统配置模型
// ===========================================

// SysConfig 系统全局配置表
type SysConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigKey   string    `gorm:"size:50;not null;uniqueIndex" json:"config_key"`
	ConfigValue *string   `gorm:"type:text" json:"config_value"`
	ConfigName  string    `gorm:"size:100;not null" json:"config_name"`
	ConfigGroup string    `gorm:"size:50;default:'default'" json:"config_group"`
	Remark      string    `gorm:"size:255" json:"remark"`
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
}

func (SysConfig) TableName() string { return "sys_config" }

// SysDetectRule 探测黑白名单规则表
type SysDetectRule struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleType  uint8     `gorm:"not null;index" json:"rule_type"` // 1黑名单IP 2白名单IP 3黑名单端口...
	RuleValue string    `gorm:"size:255;not null" json:"rule_value"`
	Enabled   uint8     `gorm:"not null;default:1" json:"enabled"`
	Remark    string    `gorm:"size:255" json:"remark"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

func (SysDetectRule) TableName() string { return "sys_detect_rule" }

// ===========================================
// 资产业务模型
// ===========================================

// WebAsset URL资产主表（核心大表）
type WebAsset struct {
	ID                uint64      `gorm:"primaryKey;autoIncrement" json:"id"`
	URLAddress        string      `gorm:"size:500;not null;uniqueIndex:uk_url_protocol,priority:1" json:"url_address"`
	ProtocolType      uint8       `gorm:"not null;uniqueIndex:uk_url_protocol,priority:2" json:"protocol_type"` // 1HTTP 2HTTPS
	IPAddress         string      `gorm:"size:50" json:"ip_address"`
	Port              int         `json:"port"`
	CompanyName       string      `gorm:"size:100;not null;default:'';index" json:"company_name"`
	AssetStatus       uint8       `gorm:"not null;default:1;index" json:"asset_status"` // 1待确认 2已确认 3已失效 4已废弃
	AssetSource       uint8       `gorm:"not null;default:4;index" json:"asset_source"` // 1备案探测 2内网探测 3批量导入 4手动新增
	BusinessName      string      `gorm:"size:100;not null;default:'';index" json:"business_name"`
	BusinessDesc      string      `gorm:"type:text" json:"business_desc,omitempty"`
	JobPosition       uint8       `gorm:"not null;default:0;index" json:"job_position"` // 0未知 1开发 2测试 3生产 4预发布 5办公系统
	DutyUserName      string      `gorm:"size:50;not null;default:''" json:"duty_user_name"`
	DutyUserPhone     string      `gorm:"size:20;not null;default:''" json:"duty_user_phone"`
	DutyUserEmail     string      `gorm:"size:100;not null;default:'';index" json:"duty_user_email"`
	ProjectName       string      `gorm:"size:100" json:"project_name,omitempty"`
	ProjectManagerName string     `gorm:"size:50" json:"project_manager_name,omitempty"`
	ProjectManagerEmail string    `gorm:"size:100" json:"project_manager_email,omitempty"`
	DeptName          string      `gorm:"size:50" json:"dept_name,omitempty"`
	Remark            string      `gorm:"type:text" json:"remark,omitempty"`
	// 探测自动填充字段
	ResponseCode      string      `gorm:"size:10" json:"response_code,omitempty"`
	WebTitle          string      `gorm:"size:500" json:"web_title,omitempty"`
	ICPNumber         string      `gorm:"size:50" json:"icp_number,omitempty"`
	LastDetectTime    *time.Time  `json:"last_detect_time,omitempty"`
	// 审核字段
	ConfirmUserID     *uint64     `json:"confirm_user_id,omitempty"`
	ConfirmTime       *time.Time  `json:"confirm_time,omitempty"`
	ConfirmRemark     string      `gorm:"size:255" json:"confirm_remark,omitempty"`
	// 审计字段
	CreatorID         *uint64     `json:"creator_id,omitempty"`
	CreateTime        time.Time   `json:"create_time"`
	UpdateTime        time.Time   `json:"update_time"`
	// 关联字段
	Certificates      []SslCertInfo `gorm:"foreignKey:AssetID" json:"certificates,omitempty"`
}

func (WebAsset) TableName() string { return "web_asset" }

const (
	AssetStatusPending   uint8 = 1 // 待确认
	AssetStatusConfirmed uint8 = 2 // 已确认
	AssetStatusInvalid   uint8 = 3 // 已失效
	AssetStatusDeprecated uint8 = 4 // 已废弃
)

const (
	SourceICPProbe   uint8 = 1 // 备案探测
	SourceIntranet   uint8 = 2 // 内网探测
	SourceBatchImport uint8 = 3 // 批量导入
	SourceManual     uint8 = 4 // 手动新增
)

const (
	JobPositionUnknown   uint8 = 0
	JobPositionDev       uint8 = 1
	JobPositionTest      uint8 = 2
	JobPositionProd      uint8 = 3
	JobPositionPreRelease uint8 = 4
	JobPositionOffice    uint8 = 5
)

// WebAssetImportLog 资产导入错误日志
type WebAssetImportLog struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ImportTaskID uint64   `gorm:"not null;index" json:"import_task_id"`
	FileName    string    `gorm:"size:200;not null" json:"file_name"`
	RowNum      int       `gorm:"not null" json:"row_num"`
	ErrorType   string    `gorm:"size:50;not null" json:"error_type"`
	ErrorMsg    string    `gorm:"type:text;not null" json:"error_msg"`
	RowData     string    `gorm:"type:json" json:"row_data,omitempty"`
	CreateTime  time.Time `gorm:"index" json:"create_time"`
}

func (WebAssetImportLog) TableName() string { return "web_asset_import_log" }

// WebAssetImportTask 导入任务记录表
type WebAssetImportTask struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FileName     string     `gorm:"size:200;not null" json:"file_name"`
	FilePath     string     `gorm:"size:500;not null" json:"-"`
	TotalRows    int        `gorm:"not null;default:0" json:"total_rows"`
	SuccessCount int        `gorm:"not null;default:0" json:"success_count"`
	FailCount    int        `gorm:"not null;default:0" json:"fail_count"`
	TaskStatus   uint8      `gorm:"not null;default:1" json:"task_status"` // 1处理中 2完成 3失败
	TaskResult   string     `gorm:"type:text" json:"task_result,omitempty"`
	OperatorID   uint64     `gorm:"not null" json:"operator_id"`
	CreateTime   time.Time  `json:"create_time"`
	FinishTime   *time.Time `json:"finish_time,omitempty"`
}

func (WebAssetImportTask) TableName() string { return "web_asset_import_task" }

// ===========================================
// 探测任务模型
// ===========================================

// DetectRecordCompany 公网备案探测任务表
type DetectRecordCompany struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CompanyName   string     `gorm:"size:100;not null" json:"company_name"`
	ICPDomains    string     `gorm:"type:json" json:"icp_domains,omitempty"`
	TaskStatus    uint8      `gorm:"not null;default:1;index" json:"task_status"` // 1待执行 2执行中 3成功 4失败
	SuccessCount  int        `gorm:"not null;default:0" json:"success_count"`
	FailCount     int        `gorm:"not null;default:0" json:"fail_count"`
	TaskMsg       string     `gorm:"type:text" json:"task_msg,omitempty"`
	CronExpr      string     `gorm:"size:100" json:"cron_expr,omitempty"`
	IsPeriodic    uint8      `gorm:"not null;default:0" json:"is_periodic"`
	NextRunTime   *time.Time `json:"next_run_time,omitempty"`
	CreateUser    uint64     `gorm:"not null;index" json:"create_user"`
	CreateTime    time.Time  `gorm:"index" json:"create_time"`
	FinishTime    *time.Time `json:"finish_time,omitempty"`
}

func (DetectRecordCompany) TableName() string { return "detect_record_company" }

// DetectRecordIntranet 内网URL探测任务表
type DetectRecordIntranet struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskName      string     `gorm:"size:100;not null;default:''" json:"task_name"`
	IPSegment     string     `gorm:"size:100;not null" json:"ip_segment"`
	PortRange     string     `gorm:"size:100;default:'80,443,8080,8443,3000,9000'" json:"port_range"`
	ProtocolType  string     `gorm:"size:10;default:'ALL'" json:"protocol_type"`
	ScanRateLimit int        `gorm:"default:100" json:"scan_rate_limit"`
	TaskStatus    uint8      `gorm:"not null;default:1;index" json:"task_status"`
	SuccessCount  int        `gorm:"not null;default:0" json:"success_count"`
	UpdatedCount  int        `gorm:"not null;default:0" json:"updated_count"`
	ScannedIPs    int        `gorm:"not null;default:0" json:"scanned_ips"`
	TotalIPs      int        `gorm:"not null;default:0" json:"total_ips"`
	TaskMsg       string     `gorm:"type:text" json:"task_msg,omitempty"`
	CronExpr      string     `gorm:"size:100" json:"cron_expr,omitempty"`
	IsPeriodic    uint8      `gorm:"not null;default:0" json:"is_periodic"`
	NextRunTime   *time.Time `json:"next_run_time,omitempty"`
	CreateUser    uint64     `gorm:"not null;index" json:"create_user"`
	CreateTime    time.Time  `gorm:"index" json:"create_time"`
	FinishTime    *time.Time `json:"finish_time,omitempty"`
}

func (DetectRecordIntranet) TableName() string { return "detect_record_intranet" }

// DetectIntranetDetail 内网探测结果明细表
type DetectIntranetDetail struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID         uint64    `gorm:"not null;index" json:"task_id"`
	IPAddress      string    `gorm:"size:50;not null;index" json:"ip_address"`
	Port           int       `gorm:"not null;default:0" json:"port"`
	Protocol       string    `gorm:"size:10;default:''" json:"protocol"`
	URL            string    `gorm:"size:500;default:''" json:"url,omitempty"`
	StatusCode     int       `gorm:"default:0" json:"status_code"`
	Title          string    `gorm:"size:500;default:''" json:"title,omitempty"`
	ResponseTimeMs int       `gorm:"default:0" json:"response_time_ms"`
	IsNew          uint8     `gorm:"not null;default:1" json:"is_new"`
	AssetID        *uint64   `json:"asset_id,omitempty"`
	CreateTime     time.Time `json:"create_time"`
}

func (DetectIntranetDetail) TableName() string { return "detect_intranet_detail" }

// ===========================================
// 证书管理模型
// ===========================================

// SslCertInfo SSL证书信息主表
type SslCertInfo struct {
	ID               uint64      `gorm:"primaryKey;autoIncrement" json:"id"`
	AssetID          *uint64     `gorm:"index" json:"asset_id,omitempty"`
	DomainIP         string      `gorm:"size:100;not null;index" json:"domain_ip"`
	SANDomains       string      `gorm:"type:text" json:"san_domains,omitempty"`
	CertType         uint8       `gorm:"not null" json:"cert_type"` // 1公网可信 2内网自签名
	CertSource       uint8       `gorm:"not null" json:"cert_source"` // 1探测采集 2手动录入 3自助申请
	Issuer           string      `gorm:"size:255;default:''" json:"issuer"`
	SerialNo         string      `gorm:"size:100" json:"serial_no,omitempty"`
	EncryptAlgorithm string      `gorm:"size:50;default:''" json:"encrypt_algorithm"`
	HashAlgorithm    string      `gorm:"size:50;default:''" json:"hash_algorithm"`
	KeySize          int         `gorm:"default:0" json:"key_size"`
	ValidStartTime   time.Time   `gorm:"not null" json:"valid_start_time"`
	ValidEndTime     time.Time   `gorm:"not null;index" json:"valid_end_time"`
	DaysRemaining    int         `gorm:"->:SELECT DATEDIFF(valid_end_time,NOW());<-:ignore" json:"days_remaining"`
	CertStatus       uint8       `gorm:"not null;default:1;index" json:"cert_status"` // 1正常 2即将过期 3已过期 4无效 5吊销
	AutoRenew        uint8       `gorm:"not null;default:0" json:"auto_renew"`
	AlertDays        int         `gorm:"not null;default:30" json:"alert_days"`
	LastAlertTime    *time.Time  `json:"last_alert_time,omitempty"`
	ApplyTaskID      *uint64     `json:"apply_task_id,omitempty"`
	CreateTime       time.Time   `json:"create_time"`
	UpdateTime       time.Time   `json:"update_time"`
	Asset            *WebAsset   `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (SslCertInfo) TableName() string { return "ssl_cert_info" }

const (
	CertStatusNormal      uint8 = 1
	CertStatusExpiring    uint8 = 2
	CertStatusExpired     uint8 = 3
	CertStatusInvalid     uint8 = 4
	CertStatusRevoked     uint8 = 5
)

// SslCertApplyTask 免费HTTPS证书申请任务表
type SslCertApplyTask struct {
	ID                 uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ApplyType          uint8      `gorm:"not null" json:"apply_type"` // 1公网域名 2内网IP
	ApplyAddr          string     `gorm:"size:100;not null;index" json:"apply_addr"`
	SANAddrs           string     `gorm:"type:text" json:"san_addrs,omitempty"`
	AssetID            *uint64    `json:"asset_id,omitempty"`
	VerifyMethod       uint8      `gorm:"default:1" json:"verify_method"` // 1DNS验证 2HTTP文件验证
	DNSRecordName      string     `gorm:"size:255;default:''" json:"dns_record_name,omitempty"`
	DNSRecordValue     string     `gorm:"size:255;default:''" json:"dns_record_value,omitempty"`
	HTTPVerifyPath     string     `gorm:"size:255;default:''" json:"http_verify_path,omitempty"`
	EncryptAlgorithm   string     `gorm:"size:50;default:'RSA'" json:"encrypt_algorithm"`
	KeySize            int        `gorm:"default:2048" json:"key_size"`
	ValidDays          int        `gorm:"default:365" json:"valid_days"`
	TaskStatus         uint8      `gorm:"not null;default:1;index" json:"task_status"` // 1申请中 2校验中 3签发成功 4签发失败
	FailReason         string     `gorm:"type:text" json:"fail_reason,omitempty"`
	CertFilePath       string     `gorm:"size:255" json:"cert_file_path,omitempty"`
	PrivateKeyEncrypted uint8     `gorm:"not null;default:0" json:"private_key_encrypted"`
	ApplyUser          uint64     `gorm:"not null;index" json:"apply_user"`
	CreateTime         time.Time  `gorm:"index" json:"create_time"`
	FinishTime         *time.Time `json:"finish_time,omitempty"`
}

func (SslCertApplyTask) TableName() string { return "ssl_cert_apply_task" }

// ===========================================
// 预警通知模型
// ===========================================

// NotifyMessage 站内消息通知表
type NotifyMessage struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ReceiverID    uint64     `gorm:"not null;index" json:"receiver_id"`
	SenderID      uint64     `gorm:"default:0" json:"sender_id,omitempty"`
	MsgType       uint8      `gorm:"not null" json:"msg_type"` // 1系统通知 2证书预警 3资产异常 4任务通知 5审批通知
	MsgTitle      string     `gorm:"size:200;not null" json:"msg_title"`
	MsgContent    string     `gorm:"type:text;not null" json:"msg_content"`
	RelatedModule string     `gorm:"size:50;default:''" json:"related_module,omitempty"`
	RelatedID     uint64     `gorm:"default:0" json:"related_id,omitempty"`
	IsRead        uint8      `gorm:"not null;default:0;index" json:"is_read"`
	ReadTime      *time.Time `json:"read_time,omitempty"`
	CreateTime    time.Time  `gorm:"index" json:"create_time"`
}

func (NotifyMessage)TableName() string { return "notify_message" }

// AlertRule 告警规则配置表
type AlertRule struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleName        string    `gorm:"size:100;not null" json:"rule_name"`
	RuleType        uint8     `gorm:"not null" json:"rule_type"` // 1即将过期 2已过期 3资产不可访问 4探测任务异常 5证书申请失败
	AlertThreshold  int       `gorm:"default:30" json:"alert_threshold"`
	NotifyChannels  string    `gorm:"size:50;default:'message,email'" json:"notify_channels"`
	NotifyTemplate  string    `gorm:"type:text" json:"notify_template,omitempty"`
	Enabled         uint8     `gorm:"not null;default:1" json:"enabled"`
	CooldownMinutes int       `gorm:"default:60" json:"cooldown_minutes"`
	Remark          string    `gorm:"size:255" json:"remark,omitempty"`
	CreateTime      time.Time `json:"create_time"`
	UpdateTime      time.Time `json:"update_time"`
}

func (AlertRule) TableName() string { return "alert_rule" }

// AlertSendLog 告警发送记录表
type AlertSendLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID       uint64    `gorm:"not null;index" json:"rule_id"`
	AlertType    uint8     `gorm:"not null;index" json:"alert_type"`
	TargetType   string    `gorm:"size:20;not null" json:"target_type"`
	TargetValue  string    `gorm:"size:100;not null" json:"target_value"`
	Channel      string    `gorm:"size:20;not null" json:"channel"`
	SendStatus   uint8     `gorm:"not null;default:1;index" json:"send_status"`
	SendError    string    `gorm:"type:text" json:"send_error,omitempty"`
	SendTime     *time.Time `json:"send_time,omitempty"`
	CreateTime   time.Time `gorm:"index" json:"create_time"`
}

func (AlertSendLog) TableName() string { return "alert_send_log" }

// ===========================================
// 日志审计模型
// ===========================================

// SysOperationLog 系统操作审计日志表
type SysOperationLog struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          *uint64   `json:"user_id,omitempty"`
	Username        string    `gorm:"size:50;default:''" json:"username,omitempty"`
	OperationModule string    `gorm:"size:50;not null;index" json:"operation_module"`
	OperationType   string    `gorm:"size:50;not null;index" json:"operation_type"`
	RequestMethod   string    `gorm:"size:10;default:''" json:"request_method,omitempty"`
	RequestURL      string    `gorm:"size:500;default:''" json:"request_url,omitempty"`
	RequestIP       string    `gorm:"size:50;not null;index" json:"request_ip"`
	RequestParam    string    `gorm:"type:text" json:"request_param,omitempty"`
	ResponseData    string    `gorm:"type:text" json:"response_data,omitempty"`
	OperationResult uint8     `gorm:"not null;index" json:"operation_result"` // 1成功 2失败
	ErrorMsg        string    `gorm:"type:text" json:"error_msg,omitempty"`
	ExecutionTimeMs int       `gorm:"default:0" json:"execution_time_ms,omitempty"`
	UserAgent       string    `gorm:"size:500;default:''" json:"user_agent,omitempty"`
	CreateTime      time.Time `gorm:"index" json:"create_time"`
}

func (SysOperationLog) TableName() string { return "sys_operation_log" }
