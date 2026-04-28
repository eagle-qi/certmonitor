package router

import (
	"certmonitor/internal/handler"
	"certmonitor/internal/middleware"
	"certmonitor/pkg/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Setup 初始化并注册所有路由
func Setup(r *gin.Engine, db *gorm.DB, rdb *redis.Client, cfg *config.Config) *gin.Engine {
	api := r.Group("/api/v1")

	// ==================== 公开接口（无需登录）====================
	public := api.Group("")
	{
		// 用户认证
		auth := public.Group("/auth")
		authHandler := handler.NewAuthHandler(db, rdb, cfg)
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/send-captcha", authHandler.SendCaptcha)
			auth.GET("/sso/login", authHandler.SSOLogin)
			auth.GET("/sso/callback", authHandler.SSOCallback)
		}
	}

	// ==================== 需要认证的接口 ====================
	protected := api.Group("")
	protected.Use(middleware.AuthRequired(cfg.JWT.Secret))
	{
		// ==================== 用户管理 ====================
		userGroup := protected.Group("/users")
		userHandler := handler.NewUserHandler(db)
		userGroup.Use(middleware.RoleBasedAccessControl("super_admin"))
		{
			userGroup.GET("", userHandler.List)                    // 用户列表
			userGroup.POST("", userHandler.Create)                 // 创建用户
			userGroup.PUT("/:id", userHandler.Update)              // 更新用户
			userGroup.DELETE("/:id", userHandler.Delete)           // 删除(注销)用户
			userGroup.PATCH("/:id/status", userHandler.UpdateStatus) // 启用/禁用
			userGroup.PATCH("/:id/reset-password", userHandler.ResetPassword) // 重置密码
			userGroup.PATCH("/:id/roles", userHandler.AssignRoles)           // 分配角色
		}

		// 个人中心（当前登录用户）
		me := protected.Group("/me")
		meHandler := handler.NewProfileHandler(db)
		{
			me.GET("", meHandler.GetProfile)
			me.PUT("", meHandler.UpdateProfile)
			me.PUT("/password", meHandler.ChangePassword)
			me.GET("/logs", meHandler.MyOperationLogs)
		}

		// ==================== 角色管理 ====================
		roleGroup := protected.Group("/roles")
		roleHandler := handler.NewRoleHandler(db)
		roleGroup.Use(middleware.RoleBasedAccessControl("super_admin"))
		{
			roleGroup.GET("", roleHandler.List)
			roleGroup.POST("", roleHandler.Create)
			roleGroup.PUT("/:id", roleHandler.Update)
			roleGroup.DELETE("/:id", roleHandler.Delete)
		}

		// ==================== 资产管理 ====================
		assetGroup := protected.Group("/assets")
		assetHandler := handler.NewAssetHandler(db, rdb, cfg)
		{
			assetGroup.Use(middleware.Pagination())
			assetGroup.GET("", assetHandler.List)                  // 资产列表(分页+筛选)
			assetGroup.POST("", assetHandler.Create)               // 新增资产(需 asset_admin 角色)
			assetGroup.GET("/:id", assetHandler.GetDetail)         // 资产详情
			assetGroup.PUT("/:id", assetHandler.Update)            // 编辑资产(需 asset_admin)
			assetGroup.DELETE("/:id", assetHandler.Delete)         // 删除资产(需 asset_admin)

			// 资产审核流程
			assetGroup.PATCH("/:id/confirm", assetHandler.Confirm)   // 审核通过
			assetGroup.PATCH("/:id/reject", assetHandler.Reject)     // 审核驳回

			// 批量导入
			assetGroup.POST("/import", assetHandler.BatchImport)     // 批量导入
			assetGroup.GET("/template/download", assetHandler.DownloadTemplate) // 下载导入模板
			assetGroup.GET("/import/logs/:taskId", assetHandler.ImportLog)      // 导入错误日志

			// 导出
			assetGroup.GET("/export", assetHandler.Export)             // 导出Excel
		}

		// ==================== 探测任务管理 ====================
		detectGroup := protected.Group("/detect")
		detectHandler := handler.NewDetectHandler(db, rdb, cfg)
		{
			// 公网备案探测
			detectGroup.POST("/company", detectHandler.CreateCompanyDetectTask)  // 创建公网探测任务
			detectGroup.GET("/company/tasks", detectHandler.ListCompanyTasks)    // 探测任务列表
			detectGroup.GET("/company/tasks/:id", detectHandler.CompanyTaskDetail)

			// 内网网段探测
			detectGroup.POST("/intranet", detectHandler.CreateIntranetDetectTask)
			detectGroup.GET("/intranet/tasks", detectHandler.ListIntranetTasks)
			detectGroup.GET("/intranet/tasks/:id", detectHandler.IntranetTaskDetail)
			detectGroup.GET("/intranet/tasks/:id/details", detectHandler.IntranetTaskDetails)
		}

		// ==================== 证书管理 ====================
		certGroup := protected.Group("/certificates")
		certHandler := handler.NewCertHandler(db, rdb, cfg)
		certGroup.Use(middleware.Pagination())
		{
			certGroup.GET("", certHandler.List)                     // 证书列表(分页+筛选)
			certGroup.GET("/:id", certHandler.GetDetail)            // 证书详情
			certGroup.POST("", certHandler.CreateManual)            // 手动录入证书
			certGroup.PUT("/:id", certHandler.Update)               // 更新证书信息
			certGroup.DELETE("/:id", certHandler.Delete)            // 删除证书记录

			// 自动采集
			certGroup.POST("/collect/:assetId", certHandler.CollectCertByAsset) // 按资产采集证书

			// 统计概览
			certGroup.GET("/stats/overview", certHandler.OverviewStats)        // 证书统计概览
			certGroup.GET("/stats/risk", certHandler.RiskStats)                // 风险统计

			// 下载
			certGroup.GET("/:id/download", certHandler.DownloadCert)           // 下载证书文件
		}

		// ==================== 证书自助申请签发 ====================
		applyGroup := protected.Group("/cert-apply")
		applyHandler := handler.NewCertApplyHandler(db, cfg)
		{
			applyGroup.POST("", applyHandler.SubmitApplication)       // 提交申请
			applyGroup.GET("/tasks", applyHandler.ListTasks)          // 申请任务列表
			applyGroup.GET("/tasks/:id", applyHandler.TaskDetail)     // 任务详情
			applyGroup.POST("/tasks/:id/retry", applyHandler.Retry)   // 重试失败任务
			applyGroup.GET("/tasks/:id/download", applyHandler.DownloadCertPackage) // 下载证书包
		}

		// ==================== 消息通知 ====================
		msgGroup := protected.Group("/messages")
		msgHandler := handler.NewMessageHandler(db)
		{
			msgGroup.GET("", msgHandler.List)                         // 消息列表
			msgGroup.GET("/unread-count", msgHandler.UnreadCount)     // 未读数量
			msgGroup.PATCH("/:id/read", msgHandler.MarkRead)          // 标记已读
			msgGroup.PATCH("/read-all", msgHandler.MarkAllRead)       // 全部已读
		}

		// ==================== 告警规则 ====================
		alertGroup := protected.Group("/alerts/rules")
		alertHandler := handler.NewAlertHandler(db)
		alertGroup.Use(middleware.RoleBasedAccessControl("super_admin", "cert_admin"))
		{
			alertGroup.GET("", alertHandler.List)
			alertGroup.POST("", alertHandler.Create)
			alertGroup.PUT("/:id", alertHandler.Update)
			alertGroup.DELETE("/:id", alertHandler.Delete)
			alertGroup.PATCH("/:id/toggle", alertHandler.ToggleEnable)
		}

		// ==================== 系统配置 ====================
		configGroup := protected.Group("/system/config")
		sysConfigHandler := handler.NewSystemConfigHandler(db)
		configGroup.Use(middleware.RoleBasedAccessControl("super_admin"))
		{
			configGroup.GET("", sysConfigHandler.List)
			configGroup.PUT("/:key", sysConfigHandler.Update)
		}

		// ==================== 操作日志审计 ====================
		logGroup := protected.Group("/logs")
		logHandler := handler.NewLogHandler(db)
		logGroup.Use(middleware.RoleBasedAccessControl("super_admin"))
		{
			logGroup.Use(middleware.Pagination())
			logGroup.GET("", logHandler.List)
			logGroup.GET("/:id", logHandler.Detail)
			logGroup.GET("/export", logHandler.Export)
		}

		// ==================== 统计报表 ====================
		statsGroup := protected.Group("/statistics")
		statsHandler := handler.NewStatisticsHandler(db)
		{
			statsGroup.GET("/assets/overview", statsHandler.AssetOverview)    // 资产统计概览
			statsGroup.GET("/assets/distribution", statsHandler.AssetDistribution) // 资产分布统计
			statsGroup.GET("/detect/overview", statsHandler.DetectOverview)  // 探测统计
			statsGroup.GET("/dashboard", statsHandler.DashboardData)        // 仪表盘数据
		}
	}

	return r
}
