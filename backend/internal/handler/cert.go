package handler

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"certmonitor/internal/config"
	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CertHandler struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

func NewCertHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *CertHandler {
	return &CertHandler{db: db, redis: rdb, config: cfg}
}

// List 证书列表（分页+筛选）
func (h *CertHandler) List(c *gin.Context) {
	page, _ := c.Get("page")
	pageSize, _ := c.Get("page_size")

	query := h.db.Model(&model.SslCertInfo{}).Preload("Asset")

	if domain := c.Query("domain"); domain != "" {
		query = query.Where("domain_ip LIKE ?", "%"+domain+"%")
	}
	if status := c.Query("status"); status != "" {
		s, _ := strconv.Atoi(status)
		query = query.Where("cert_status = ?", s)
	}
	if certType := c.Query("cert_type"); certType != "" {
		ct, _ := strconv.Atoi(certType)
		query = query.Where("cert_type = ?", ct)
	}
	if source := c.Query("source"); source != "" {
		cs, _ := strconv.Atoi(source)
		query = query.Where("cert_source = ?", cs)
	}

	// 按过期时间排序（即将过期的排前面）
	order := "valid_end_time ASC"

	var total int64
	query.Count(&total)

	var certs []model.SslCertInfo
	offset, _ := c.Get("offset")
	query.Offset(offset).Limit(pageSize.(int)).Order(order).Find(&certs)

	response.PageSuccess(c, certs, total, page.(int), pageSize.(int))
}

// GetDetail 证书详情
func (h *CertHandler) GetDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var cert model.SslCertInfo
	if h.db.Preload("Asset").First(&cert, id).Error != nil {
		response.NotFound(c, "证书记录不存在", nil)
		return
	}

	response.Success(c, cert)
}

// CreateManualReq 手动录入证书请求
type CreateManualReq struct {
	AssetID        *uint64 `json:"asset_id,omitempty"` // 关联资产(可选)
	DomainIP       string  `json:"domain_ip" binding:"required,max=100"`
	SANDomains     string  `json:"san_domains,omitempty"`
	CertType       uint8   `json:"cert_type" binding:"required,oneof=1 2"`
	CertSource     uint8   `json:"cert_source" binding:"required,oneof=1 2 3"`
	Issuer         string  `json:"issuer" binding:"omitempty,max=255"`
	SerialNo       string  `json:"serial_no" binding:"omitempty,max=100"`
	EncryptAlgo    string  `json:"encrypt_algorithm" binding:"omitempty,max=50"`
	HashAlgo       string  `json:"hash_algorithm" binding:"omitempty,max=50"`
	KeySize        int     `json:"key_size" binding:"omitempty,gte=0"`
	ValidStartTime time.Time `json:"valid_start_time" binding:"required"`
	ValidEndTime   time.Time `json:"valid_end_time" binding:"required"`
	AlertDays      int     `json:"alert_days" binding:"omitempty,gte=0"`
	CertContent    string  `json:"cert_content,omitempty"` // PEM格式证书内容(可选上传解析)
}

// CreateManual 手动录入证书信息
func (h *CertHandler) CreateManual(c *gin.Context) {
	var req CreateManualReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: "+err.Error(), nil)
		return
	}

	// 如果提供了 PEM 格式的证书内容，尝试自动解析
	if req.CertContent != "" {
		parsed, err := parsePEMCertificate(req.CertContent)
		if err == nil && parsed != nil {
			if req.Issuer == "" { req.Issuer = parsed.Issuer }
			if req.SerialNo == "" { req.SerialNo = parsed.SerialNo }
			if req.EncryptAlgo == "" { req.EncryptAlgo = parsed.EncryptAlgorithm }
			if req.HashAlgo == "" { req.HashAlgo = parsed.HashAlgorithm }
			if req.KeySize == 0 { req.KeySize = parsed.KeySize }
			if req.SANDomains == "" && len(parsed.SANs) > 0 { req.SANDomains = strings.Join(parsed.SANs, ",") }
			if req.ValidStartTime.IsZero() { req.ValidStartTime = parsed.NotBefore }
			if req.ValidEndTime.IsZero() { req.ValidEndTime = parsed.NotAfter }
		}
	}

	// 自动计算证书状态
	daysRemaining := int(time.Until(req.ValidEndTime).Hours() / 24)
	certStatus := model.CertStatusNormal
	if daysRemaining <= 0 {
		certStatus = model.CertStatusExpired
	} else if daysRemaining <= 30 {
		certStatus = model.CertStatusExpiring
	}

	cert := model.SslCertInfo{
		AssetID:          req.AssetID,
		DomainIP:         req.DomainIP,
		SANDomains:       req.SANDomains,
		CertType:         req.CertType,
		CertSource:       req.CertSource,
		Issuer:           req.Issuer,
		SerialNo:         req.SerialNo,
		EncryptAlgorithm: req.EncryptAlgo,
		HashAlgorithm:    req.HashAlgo,
		KeySize:          req.KeySize,
		ValidStartTime:   req.ValidStartTime,
		ValidEndTime:     req.ValidEndTime,
		DaysRemaining:    daysRemaining,
		CertStatus:       certStatus,
		AlertDays:        req.AlertDays,
	}

	if h.db.Create(&cert).Error != nil {
		response.InternalError(c, "保存证书失败", nil)
		return
	}

	recordOperationLog(h.db, middleware.GetUserID(c), "cert", "create", c,
		map[string]interface{}{"cert_id": cert.ID, "domain": cert.DomainIP}, 1)

	response.SuccessWithMessage(c, "证书信息录入成功", cert)
}

// Update 更新证书信息
func (h *CertHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var existing model.SslCertInfo
	if h.db.First(&existing, id).Error != nil {
		response.NotFound(c, "证书记录不存在", nil)
		return
	}

	var req struct {
		AlertDays  int    `json:"alert_days" binding:"omitempty,gte=1"`
		AutoRenew  uint8  `json:"auto_renew" binding:"omitempty,oneof=0 1"`
		Remark     string `json:"remark" binding:"omitempty,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	h.db.Model(&existing).Updates(map[string]interface{}{
		"alert_days": req.AlertDays,
		"auto_renew": req.AutoRenew,
	})

	recordOperationLog(h.db, middleware.GetUserID(c), "cert", "update", c,
		map[string]interface{}{"cert_id": id}, 1)

	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除证书记录
func (h *CertHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	result := h.db.Delete(&model.SslCertInfo{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "证书记录不存在", nil)
		return
	}

	response.SuccessWithMessage(c, "证书记录已删除", nil)
}

// CollectCertByAsset 按资产ID采集证书（主动连接HTTPS获取证书信息）
func (h *CertHandler) CollectCertByAsset(c *gin.Context) {
	assetID, _ := strconv.ParseUint(c.Param("assetId"), 10, 64)

	var asset model.WebAsset
	if h.db.First(&asset, assetID).Error != nil {
		response.NotFound(c, "资产不存在", nil)
		return
	}

	if asset.ProtocolType != 2 { // 非HTTPS资产无法采集证书
		response.BadRequest(c, "该资产不是HTTPS协议，无法采集SSL证书", nil)
		return
	}

	// 从URL中提取域名
	urlParsed, err := url.Parse(asset.URLAddress)
	if err != nil {
		response.BadRequest(c, "URL地址解析失败", nil)
		return
	}
	host := urlParsed.Hostname()
	port := urlParsed.Port()

	// 连接并获取证书
	certInfo, err := fetchSSLCertificate(host, port)
	if err != nil {
		response.InternalError(c, fmt.Sprintf("采集证书失败: %v", err), nil)
		return
	}

	// 检查是否已有该域名的证书
	var existing model.SslCertInfo
	result := h.db.Where("domain_ip = ? AND asset_id = ?", host, assetID).First(&existing)

	if result.Error == nil {
		// 已存在则更新
		h.db.Model(&existing).Updates(certInfo)
		recordOperationLog(h.db, middleware.GetUserID(c), "cert", "collect_update", c,
			map[string]interface{}{"cert_id": existing.ID, "asset_id": assetID}, 1)
		response.SuccessWithMessage(c, "证书信息已更新", existing)
	} else {
		// 新增
		certInfo.AssetID = &assetID
		h.db.Create(&certInfo)
		recordOperationLog(h.db, middleware.GetUserID(c), "cert", "collect_new", c,
			map[string]interface{}{"cert_id": certInfo.ID, "asset_id": assetID}, 1)
		response.SuccessWithMessage(c, "证书采集成功", certInfo)
	}
}

// OverviewStats 证书统计概览
func (h *CertHandler) OverviewStats(c *gin.Context) {
	stats := make(map[string]int64)

	// 各状态数量
	h.db.Model(&model.SslCertInfo{}).
		Select("cert_status, COUNT(*) as count").
		Group("cert_status").
		Find(&[]map[string]interface{}{}).
		Scan(&stats)

	var total, normalCount, expiringCount, expiredCount, invalidCount int64

	h.db.Model(&model.SslCertInfo{}).Count(&total)
	h.db.Model(&model.SslCertInfo{}).Where("cert_status = ?", model.CertStatusNormal).Count(&normalCount)
	h.db.Model(&model.SslCertInfo{}).Where("cert_status = ?", model.CertStatusExpiring).Count(&expiringCount)
	h.db.Model(&model.SslCertInfo{}).Where("cert_status >= ?", model.CertStatusExpired).Count(&expiredCount)
	h.db.Model(&model.SslCertInfo{}).Where("cert_status IN ?", []uint8{4, 5}).Count(&invalidCount)

	// 即将过期细分(7天/15天/30天)
	var expireIn7d, expireIn15d, expireIn30d int64
	now := time.Now()
	h.db.Model(&model.SslCertInfo{}).
		Where("cert_status IN ? AND valid_end_time BETWEEN ? AND ?",
			[]uint8{1, 2}, now, now.AddDate(0, 0, 7)).Count(&expireIn7d)
	h.db.Model(&model.SslCertInfo{}).
		Where("cert_status IN ? AND valid_end_time BETWEEN ? AND ?",
			[]uint8{1, 2}, now, now.AddDate(0, 0, 15)).Count(&expireIn15d)
	h.db.Model(&model.SslCertInfo{}).
		Where("cert_status IN ? AND valid_end_time BETWEEN ? AND ?",
			[]uint8{1, 2}, now, now.AddDate(0, 0, 30)).Count(&expireIn30d)

	response.Success(c, gin.H{
		"total":              total,
		"normal":             normalCount,
		"expiring":           expiringCount,
		"expired":            expiredCount,
		"invalid_revoked":    invalidCount,
		"risk_in_7days":      expireIn7d,
		"risk_in_15days":     expireIn15d,
		"risk_in_30days":     expireIn30d,
	})
}

// RiskStats 风险统计分析
func (h *CertHandler) RiskStats(c *gin.Context) {
	// TODO: 实现更详细的风险统计：
	// - 按CA机构分布
	// - 按加密算法分布
	// - 按密钥长度分布
	// - 按证书来源分布
	// - 过期风险时间线数据

	response.Success(c, gin.H{
		"message": "风险统计功能开发中...",
	})
}

// DownloadCert 下载证书文件
func (h *CertHandler) DownloadCert(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64

	var applyTask model.SslCertApplyTask
	if h.db.First(&applyTask, id).Error != nil || applyTask.CertFilePath == "" {
		response.NotFound(c, "证书文件不存在或未签发成功", nil)
		return
	}

	// TODO: 返回证书文件下载流
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(applyTask.CertFilePath)))
	c.File(applyTask.CertFilePath)
}

// =========================================== 内部工具函数 ===========================================

type ParsedCertificate struct {
	Issuer            string
	SerialNo          string
	EncryptAlgorithm  string
	HashAlgorithm     string
	KeySize           int
	SANs              []string
	NotBefore         time.Time
	NotAfter          time.Time
}

func parsePEMCertificate(pemContent string) (*ParsedCertificate, error) {
	block, _ := pem.Decode([]byte(pemContent))
	if block == nil {
		return nil, fmt.Errorf("无效的PEM格式")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	keySize := 0
	switch k := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		keySize = k.N.BitLen()
	case *ecdsa.PublicKey:
		keySize = k.Params().BitSize
	}

	sans := make([]string, 0)
	for _, name := range cert.DNSNames {
		sans = append(sans, name)
	}
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	return &ParsedCertificate{
		Issuer:           cert.Issuer.CommonName,
		SerialNo:         fmt.Sprintf("%X", cert.SerialNumber),
		EncryptAlgorithm: cert.PublicKeyAlgorithm.String(),
		HashAlgorithm:    cert.SignatureAlgorithm.String(),
		KeySize:          keySize,
		SANs:             sans,
		NotBefore:        cert.NotBefore,
		NotAfter:         cert.NotAfter,
	}, nil
}

func fetchSSLCertificate(hostname, port string) (*model.SslCertInfo, error) {
	addr := hostname
	if port != "" && port != "443" {
		addr = addr + ":" + port
	}

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp", addr,
		&tls.Config{ServerName: hostname, InsecureSkipVerify: false},
	)
	if err != nil {
		return nil, fmt.Errorf("TLS连接失败: %w", err)
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0]

	keySize := 0
	switch k := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		keySize = k.N.BitLen()
	case *ecdsa.PublicKey:
		keySize = k.Params().BitSize
	}

	sans := make([]string, 0, len(cert.DNSNames)+len(cert.IPAddresses))
	sans = append(sans, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	daysRemaining := int(time.Until(cert.NotAfter).Hours() / 24)
	status := model.CertStatusNormal
	if daysRemaining <= 0 {
		status = model.CertStatusExpired
	} else if daysRemaining <= 30 {
		status = model.CertStatusExpiring
	}

	return &model.SslCertInfo{
		DomainIP:         hostname,
		SANDomains:       strings.Join(sans, ","),
		CertType:         1, // 默认公网可信
		CertSource:       1, // 探测采集
		Issuer:           cert.Issuer.CommonName,
		SerialNo:         fmt.Sprintf("%X", cert.SerialNumber),
		EncryptAlgorithm: cert.PublicKeyAlgorithm.String(),
		HashAlgorithm:    cert.SignatureAlgorithm.String(),
		KeySize:          keySize,
		ValidStartTime:   cert.NotBefore,
		ValidEndTime:     cert.NotAfter,
		DaysRemaining:    daysRemaining,
		CertStatus:       status,
	}, nil
}
