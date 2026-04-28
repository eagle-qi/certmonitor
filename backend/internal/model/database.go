package model

import (
	"fmt"
	"log"

	"certmonitor/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接和自动迁移
func InitDB(dbCfg config.DatabaseConfig, debug bool) (*gorm.DB, error) {
	var gormLogger logger.Interface
	if debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(mysql.Open(dbCfg.DSN()), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(0) // 不设置最大生命周期，让 MySQL 驱动自行管理

	// 自动迁移数据库表结构（开发环境使用，生产环境建议用 SQL 脚本管理）
	if debug {
		log.Println("开始自动迁移数据库...")
		err = autoMigrate(db)
		if err != nil {
			log.Printf("自动迁移警告(可忽略): %v\n", err)
		} else {
			log.Println("自动迁移完成")
		}
	}

	DB = db
	return db, nil
}

// autoMigrate 自动迁移所有数据表
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 用户权限表
		&SysUser{},
		&SysRole{},
		&SysUserRole{},
		&SysEmailCaptcha{},
		&SysSSOLoginLog{},
		// 系统配置表
		&SysConfig{},
		&SysDetectRule{},
		// 资产业务表
		&WebAsset{},
		&WebAssetImportLog{},
		&WebAssetImportTask{},
		// 探测任务表
		&DetectRecordCompany{},
		&DetectRecordIntranet{},
		&DetectIntranetDetail{},
		// 证书管理表
		&SslCertInfo{},
		&SslCertApplyTask{},
		// 预警通知表
		&NotifyMessage{},
		&AlertRule{},
		&AlertSendLog{},
		// 日志审计表
		&SysOperationLog{},
	)
}
