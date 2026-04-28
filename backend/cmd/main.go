package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"certmonitor/internal/config"
	"certmonitor/internal/middleware"
	"certmonitor/internal/model"
	"certmonitor/internal/router"
	"certmonitor/pkg/logger"
	"certmonitor/pkg/redis"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化日志
	logger.Init(cfg.Log)

	// 3. 初始化 Redis
	redisClient, redisErr := redis.NewClient(cfg.Redis)
	if redisErr != nil {
		logger.Fatal("Redis 连接失败: %v", redisErr)
	}
	defer redisClient.Close()
	logger.Info("Redis 连接成功")

	// 4. 初始化数据库
	db, dbErr := model.InitDB(cfg.Database, cfg.App.Mode == "debug")
	if dbErr != nil {
		logger.Fatal("MySQL 连接失败: %v", dbErr)
	}

	// 自动迁移数据库表结构
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 5. 初始化 Gin 引擎
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 6. 注册全局中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())

	// 7. 注册路由
	apiRouter := router.Setup(r, db, redisClient, cfg)

	// 8. 启动定时任务调度器
	scheduler := NewScheduler(db, redisClient, cfg)
	scheduler.Start()
	defer scheduler.Stop()

	// 9. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      apiRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("CertMonitor 服务启动成功, 监听端口: %d", cfg.App.Port)
	logger.Info("API文档地址: http://localhost:%d/api/v1/docs", cfg.App.Port)

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在优雅关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器强制关闭: %v", err)
	}
	logger.Info("服务已停止")
}

// Scheduler 定时任务调度器
type Scheduler struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
	cron   *cron.Cron
}

func NewScheduler(db *gorm.DB, r *redis.Client, cfg *config.Config) *Scheduler {
	return &Scheduler{
		db:     db,
		redis:  r,
		config: cfg,
		cron:   cron.New(),
	}
}

func (s *Scheduler) Start() {
	// 每1小时执行一次证书过期巡检
	s.cron.AddFunc("0 * * * *", func() {
		s.checkCertificateExpiry()
	})

	// 每5分钟清理过期的验证码
	s.cron.AddFunc("*/5 * * * *", func() {
		s.cleanExpiredCaptchas()
	})

	// 每天凌晨2点检查待执行的周期性探测任务
	s.cron.AddFunc("0 2 * * *", func() {
		s.checkPeriodicDetectTasks()
	})

	s.cron.Start()
	logger.Info("定时任务调度器已启动")
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	logger.Info("定时任务调度器已停止")
}

// checkCertificateExpiry 证书过期巡检
func (s *Scheduler) checkCertificateExpiry() {
	logger.Info("开始证书过期巡检...")

	var certs []model.SslCertInfo
	s.db.Where("cert_status = ?", 1).Where("valid_end_time <= DATE_ADD(NOW(), INTERVAL 30 DAY)").Find(&certs)

	for _, cert := range certs {
		daysRemaining := int(time.Until(cert.ValidEndTime).Hours() / 24)

		if daysRemaining <= 0 {
			// 已过期
			s.db.Model(&cert).Update("cert_status", 3)
			s.sendAlert(cert, "expired")
			logger.Warn("证书已过期: %s (ID:%d)", cert.DomainIP, cert.ID)
		} else if daysRemaining <= 7 {
			s.db.Model(&cert).Update("cert_status", 2)
			s.sendAlert(cert, "expiring_soon_7")
			logger.Warn("证书即将过期(<7天): %s 剩余%d天", cert.DomainIP, daysRemaining)
		} else if daysRemaining <= 15 || daysRemaining <= 30 {
			s.db.Model(&cert).Update("cert_status", 2)
			s.sendAlert(cert, "expiring_soon")
			logger.Info("证书即将到期: %s 剩余%d天", cert.DomainIP, daysRemaining)
		}
	}
	logger.Info("证书过期巡检完成, 共检查 %d 张证书", len(certs))
}

// sendAlert 发送告警通知
func (s *Scheduler) sendAlert(cert model.SslCertInfo, alertType string) {
	// TODO: 实现告警发送逻辑（邮件、短信、站内消息）
	logger.Debug("发送告警: 证书ID=%d, 类型=%s", cert.ID, alertType)
}

// cleanExpiredCaptchas 清理过期验证码
func (s *Scheduler) cleanExpiredCaptchas() {
	result := s.db.Where("expire_time < NOW() AND used = 0").Delete(&model.SysEmailCaptcha{})
	if result.RowsAffected > 0 {
		logger.Info("已清理 %d 条过期验证码", result.RowsAffected)
	}
}

// checkPeriodicDetectTasks 检查周期性探测任务
func (s *Scheduler) checkPeriodicDetectTasks() {
	logger.Debug("检查周期性探测任务...")
	// TODO: 实现周期性探测任务的触发逻辑
}
