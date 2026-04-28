package handler

import (
	"fmt"
	"strconv"
	"time"

	"certmonitor/internal/model"
	"certmonitor/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StatisticsHandler struct {
	db *gorm.DB
}

func NewStatisticsHandler(db *gorm.DB) *StatisticsHandler {
	return &StatisticsHandler{db: db}
}

// AssetOverview 资产统计概览
func (h *StatisticsHandler) AssetOverview(c *gin.Context) {
	var total, confirmed, pending, invalid, deprecated int64

	h.db.Model(&model.WebAsset{}).Count(&total)
	h.db.Model(&model.WebAsset{}).Where("asset_status = ?", model.AssetStatusConfirmed).Count(&confirmed)
	h.db.Model(&model.WebAsset{}).Where("asset_status = ?", model.AssetStatusPending).Count(&pending)
	h.db.Model(&model.WebAsset{}).Where("asset_status >= ?", model.AssetStatusInvalid).Count(&invalid + deprecated)

	// 协议分布
	var httpCount, httpsCount int64
	h.db.Model(&model.WebAsset{}).Where("protocol_type = ?", 1).Count(&httpCount)
	h.db.Model(&model.WebAsset{}).Where("protocol_type = ?", 2).Count(&httpsCount)

	// 来源分布
	type SourceCount struct { Source uint8; Count int64 }
	var sourceStats []SourceCount
	h.db.Model(&model.WebAsset{}).
		Select("asset_source AS source, COUNT(*) AS count").
		Group("asset_source").Find(&sourceStats)

	sourceMap := map[string]int64{
		"icp_probe":    0,
		"intranet":     0,
		"batch_import": 0,
		"manual":       0,
	}
	for _, s := range sourceStats {
		switch s.Source {
		case 1: sourceMap["icp_probe"] = s.Count
		case 2: sourceMap["intranet"] = s.Count
		case 3: sourceMap["batch_import"] = s.Count
		case 4: sourceMap["manual"] = s.Count
		}
	}

	response.Success(c, gin.H{
		"total":          total,
		"status_distribution": gin.H{
			"confirmed":   confirmed,
			"pending":     pending,
			"invalid":     invalid,
		},
		"protocol_distribution": gin.H{
			"http":  httpCount,
			"https": httpsCount,
		},
		"source_distribution": sourceMap,
	})
}

// AssetDistribution 多维度资产分布统计
func (h *StatisticsHandler) AssetDistribution(c *gin.Context) {
	dimension := c.DefaultQuery("dimension", "company")

	switch dimension {
	case "company":
		var stats []map[string]interface{}
		h.db.Model(&model.WebAsset{}).
			Select("company_name, COUNT(*) as count").
			Group("company_name").Order("count DESC").Limit(20).Find(&stats)
		response.Success(c, stats)

	case "business":
		var stats []map[string]interface{}
		h.db.Model(&model.WebAsset{}).
			Select("business_name, COUNT(*) as count").
			Group("business_name").Order("count DESC").Limit(20).Find(&stats)
		response.Success(c, stats)

	case "position":
		positionLabels := map[uint8]string{
			0: "未知", 1: "开发", 2: "测试", 3: "生产", 4: "预发布", 5: "办公系统",
		}
		var stats []struct{ JobPosition uint8; Count int64 }
		h.db.Model(&model.WebAsset{}).
			Select("job_position, COUNT(*) AS count").
			Group("job_position").Order("job_position ASC").Find(&stats)

		result := make([]map[string]interface{}, len(stats))
		for i, s := range stats {
			result[i] = map[string]interface{}{
				"position": positionLabels[s.JobPosition],
				"position_code": s.JobPosition,
				"count":    s.Count,
			}
		}
		response.Success(c, result)

	case "project":
		var stats []map[string]interface{}
		h.db.Model(&model.WebAsset{}).
			Select("project_name, COUNT(*) as count").
			Where("project_name != ''").Group("project_name").Order("count DESC").Limit(20).Find(&stats)
		response.Success(c, stats)

	default:
		response.BadRequest(c, "不支持的维度参数，可选: company/business/position/project", nil)
	}
}

// DetectOverview 探测任务统计
func (h *StatisticsHandler) DetectOverview(c *gin.Context) {
	var companyTotal, companySuccess, companyFail int64
	var intranetTotal, intranetSuccess intranetPartial int64

	h.db.Model(&model.DetectRecordCompany{}).Count(&companyTotal)
	h.db.Model(&model.DetectRecordCompany{}).Where("task_status = 3").Count(&companySuccess)
	h.db.Model(&model.DetectRecordCompany{}).Where("task_status = 4").Count(&companyFail)

	h.db.Model(&model.DetectRecordIntranet{}).Count(&intranetTotal)
	h.db.Model(&model.DetectRecordIntranet{}).Where("task_status = 3").Count(&intranetSuccess)
	h.db.Model(&model.DetectRecordIntranet{}).Where("task_status = 5").Count(&intranetPartial)

	// 最近7天探测趋势
	trendData := make([]map[string]interface{}, 7)
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dayStart := date + " 00:00:00"
		dayEnd := date + " 23:59:59"

		var dayCompanyCount, dayIntranetCount int64
		h.db.Model(&model.DetectRecordCompany{}).
			Where("create_time BETWEEN ? AND ?", dayStart, dayEnd).Count(&dayCompanyCount)
		h.db.Model(&model.DetectRecordIntranet{}).
			Where("create_time BETWEEN ? AND ?", dayStart, dayEnd).Count(&dayIntranetCount)

		trendData[6-i] = map[string]interface{}{
			"date":         date,
			"company_count": dayCompanyCount,
			"intranet_count": dayIntranetCount,
		}
	}

	response.Success(c, gin.H{
		"company_tasks": gin.H{
			"total":   companyTotal,
			"success": companySuccess,
			"fail":    companyFail,
		},
		"intranet_tasks": gin.H{
			"total":   intranetTotal,
			"success": intranetSuccess,
			"partial": intranetPartial,
		},
		"recent_7days_trend": trendData,
	})
}

// DashboardData 仪表盘综合数据（首页概览）
func (h *StatisticsHandler) DashboardData(c *gin.Context) {
	// 并行查询多个维度的统计数据
	data := make(chan map[string]interface{}, 5)

	go func() {
		var total int64
		h.db.Model(&model.WebAsset{}).Count(&total)
		data <- map[string]interface{}{"asset_total": total}
	}()

	go func() {
		var count int64
		h.db.Model(&model.SslCertInfo{}).Where("cert_status IN ?", []uint8{2, 3}).Count(&count)
		data <- map[string]interface{}{"risk_cert_count": count}
	}()

	go func() {
		var count int64
		h.db.Model(&model.DetectRecordIntranet{}).Where("task_status IN ?", []uint8{1, 2}).Count(&count)
		data <- map[string]interface{}{"running_detect_tasks": count}
	}()

	go func() {
		var count int64
		h.db.Model(&model.SslCertApplyTask{}).Where("task_status IN ?", []uint8{1, 2}).Count(&count)
		data <- map[string]interface{}{"pending_cert_apply": count}
	}()

	go func() {
		var count int64
		h.db.Model(&model.NotifyMessage{}).Where("receiver_id = ? AND is_read = 0",
			middleware.GetUserID(c)).Count(&count)
		data <- map[string]interface{}{"unread_messages": count}
	}()

	dashboard := make(map[string]interface{})
	for i := 0; i < 5; i++ {
		for k, v := range <-data {
			dashboard[k] = v
		}
	}

	response.Success(c, dashboard)
}
