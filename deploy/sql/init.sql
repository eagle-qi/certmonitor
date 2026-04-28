-- ===========================================
-- CertMonitor 数据库初始化脚本
-- MySQL 8.0+ / utf8mb4 / InnoDB
-- ===========================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 创建数据库
CREATE DATABASE IF NOT EXISTS `certmonitor` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `certmonitor`;

-- ===========================================
-- 一、用户权限模块表
-- ===========================================

-- 1.1 系统用户表
CREATE TABLE IF NOT EXISTS `sys_user` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `username` VARCHAR(50) NOT NULL COMMENT '登录账号',
    `real_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '真实姓名',
    `email` VARCHAR(100) NOT NULL COMMENT '登录邮箱',
    `phone` VARCHAR(20) DEFAULT '' COMMENT '手机号',
    `password` VARCHAR(100) NOT NULL COMMENT '加密密码(BCrypt)',
    `dept_name` VARCHAR(50) DEFAULT '' COMMENT '所属部门',
    `avatar` VARCHAR(255) DEFAULT '' COMMENT '头像地址',
    `account_status` TINYINT NOT NULL DEFAULT 1 COMMENT '账号状态: 1正常 2禁用 3锁定 4已注销',
    `register_type` TINYINT NOT NULL DEFAULT 1 COMMENT '注册来源: 1手动创建 2邮箱注册 3SSO自动创建',
    `sso_unique_id` VARCHAR(100) DEFAULT NULL COMMENT 'SSO唯一标识',
    `last_login_ip` VARCHAR(50) DEFAULT '' COMMENT '最后登录IP',
    `last_login_time` DATETIME DEFAULT NULL COMMENT '最后登录时间',
    `login_fail_count` INT NOT NULL DEFAULT 0 COMMENT '连续登录失败次数',
    `lock_until` DATETIME DEFAULT NULL COMMENT '账号锁定截止时间',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `remark` VARCHAR(255) DEFAULT '',
    UNIQUE KEY `uk_email` (`email`),
    UNIQUE KEY `uk_username` (`username`),
    UNIQUE KEY `uk_sso_id` (`sso_unique_id`),
    KEY `idx_account_status` (`account_status`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统用户表';

-- 1.2 角色表
CREATE TABLE IF NOT EXISTS `sys_role` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `role_name` VARCHAR(50) NOT NULL COMMENT '角色名称',
    `role_code` VARCHAR(50) NOT NULL COMMENT '角色标识',
    `description` VARCHAR(255) DEFAULT '' COMMENT '角色描述',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序序号',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1启用 0禁用',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_role_code` (`role_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- 1.3 用户角色关联表
CREATE TABLE IF NOT EXISTS `sys_user_role` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `role_id` BIGINT NOT NULL COMMENT '角色ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_user_role` (`user_id`, `role_id`),
    KEY `idx_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户角色关联表';

-- 1.4 邮箱验证码表
CREATE TABLE IF NOT EXISTS `sys_email_captcha` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `email` VARCHAR(100) NOT NULL COMMENT '邮箱地址',
    `captcha` VARCHAR(10) NOT NULL COMMENT '验证码',
    `captcha_type` TINYINT NOT NULL DEFAULT 1 COMMENT '类型: 1注册 2登录 3重置密码',
    `expire_time` DATETIME NOT NULL COMMENT '过期时间',
    `used` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已使用: 0未使用 1已使用',
    `ip_address` VARCHAR(50) DEFAULT '' COMMENT '发送IP',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_email` (`email`),
    KEY `idx_expire_time` (`expire_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邮箱验证码记录表';

-- 1.5 SSO 登录日志表
CREATE TABLE IF NOT EXISTS `sys_sso_login_log` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT DEFAULT NULL COMMENT '关联用户ID',
    `sso_unique_id` VARCHAR(100) NOT NULL COMMENT 'SSO唯一标识',
    `login_ip` VARCHAR(50) NOT NULL COMMENT '登录IP',
    `login_status` TINYINT NOT NULL COMMENT '状态: 1成功 2失败',
    `error_msg` TEXT COMMENT '错误信息',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_sso_id` (`sso_unique_id`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SSO单点登录日志表';

-- ===========================================
-- 二、系统配置表
-- ===========================================

-- 2.1 系统全局配置表
CREATE TABLE IF NOT EXISTS `sys_config` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `config_key` VARCHAR(50) NOT NULL COMMENT '配置键',
    `config_value` TEXT COMMENT '配置值',
    `config_name` VARCHAR(100) NOT NULL COMMENT '配置名称',
    `config_group` VARCHAR(50) DEFAULT 'default' COMMENT '配置分组',
    `remark` VARCHAR(255) DEFAULT '' COMMENT '备注',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统全局配置表';

-- 2.2 探测黑白名单配置表
CREATE TABLE IF NOT EXISTS `sys_detect_rule` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `rule_type` TINYINT NOT NULL COMMENT '类型: 1IP黑名单 2IP白名单 3端口黑名单 4端口白名单 5域名黑名单',
    `rule_value` VARCHAR(255) NOT NULL COMMENT '规则值(IP/端口/域名)',
    `enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否启用: 1启用 0禁用',
    `remark` VARCHAR(255) DEFAULT '' COMMENT '备注',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_rule_type` (`rule_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='探测黑白名单规则配置表';

-- ===========================================
-- 三、资产业务核心表
-- ===========================================

-- 3.1 URL 资产主表（核心大表）
CREATE TABLE IF NOT EXISTS `web_asset` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    -- 基础资产字段
    `url_address` VARCHAR(500) NOT NULL COMMENT 'URL地址',
    `protocol_type` TINYINT NOT NULL COMMENT '协议: 1HTTP 2HTTPS',
    `ip_address` VARCHAR(50) DEFAULT '' COMMENT '公网/内网IP',
    `port` INT DEFAULT 0 COMMENT '开放端口',
    `company_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '所属公司/备案主体名称',
    `asset_status` TINYINT NOT NULL DEFAULT 1 COMMENT '资产状态: 1待确认 2已确认 3已失效 4已废弃',
    `asset_source` TINYINT NOT NULL DEFAULT 4 COMMENT '资产来源: 1备案探测 2内网探测 3批量导入 4手动新增',
    -- 业务系统信息
    `business_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '业务系统名称',
    `business_desc` TEXT COMMENT '业务系统描述',
    `job_position` TINYINT NOT NULL DEFAULT 0 COMMENT '归属岗位: 0未知 1开发 2测试 3生产 4预发布 5办公系统',
    -- 资产负责人信息
    `duty_user_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '资产负责人姓名',
    `duty_user_phone` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '负责人手机号',
    `duty_user_email` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '负责人工作邮箱',
    -- 项目关联信息
    `project_name` VARCHAR(100) DEFAULT '' COMMENT '所属项目名称',
    `project_manager_name` VARCHAR(50) DEFAULT '' COMMENT '项目经理名称',
    `project_manager_email` VARCHAR(100) DEFAULT '' COMMENT '项目经理邮箱',
    -- 辅助信息
    `dept_name` VARCHAR(50) DEFAULT '' COMMENT '归属部门',
    `remark` TEXT DEFAULT '' COMMENT '资产备注',
    -- 探测自动填充字段
    `response_code` VARCHAR(10) DEFAULT '' COMMENT 'URL响应状态码',
    `web_title` VARCHAR(500) DEFAULT '' COMMENT '网站标题',
    `icp_number` VARCHAR(50) DEFAULT '' COMMENT '备案号',
    `last_detect_time` DATETIME DEFAULT NULL COMMENT '最后探测时间',
    `confirm_user_id` BIGINT DEFAULT NULL COMMENT '审核人ID',
    `confirm_time` DATETIME DEFAULT NULL COMMENT '审核时间',
    `confirm_remark` VARCHAR(255) DEFAULT '' COMMENT '审核备注',
    -- 审计字段
    `creator_id` BIGINT DEFAULT NULL COMMENT '创建人ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_url_protocol` (`url_address`, `protocol_type`),
    KEY `idx_company` (`company_name`),
    KEY `idx_business` (`business_name`),
    KEY `idx_job_position` (`job_position`),
    KEY `idx_asset_status` (`asset_status`),
    KEY `idx_asset_source` (`asset_source`),
    KEY `idx_duty_user` (`duty_user_email`),
    KEY `idx_creator` (`creator_id`),
    KEY `idx_last_detect` (`last_detect_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='URL资产主表';

-- 3.2 资产批量导入错误日志表
CREATE TABLE IF NOT EXISTS `web_asset_import_log` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `import_task_id` BIGINT NOT NULL COMMENT '导入批次ID',
    `file_name` VARCHAR(200) NOT NULL COMMENT '导入文件名',
    `row_num` INT NOT NULL COMMENT '错误行号(Excel行号)',
    `error_type` VARCHAR(50) NOT NULL COMMENT '错误类型',
    `error_msg` TEXT NOT NULL COMMENT '错误原因',
    `row_data` JSON COMMENT '原始行数据(JSON)',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_import_task` (`import_task_id`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资产批量导入错误日志表';

-- 3.3 导入任务记录表
CREATE TABLE IF NOT EXISTS `web_asset_import_task` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `file_name` VARCHAR(200) NOT NULL COMMENT '文件名',
    `file_path` VARCHAR(500) NOT NULL COMMENT '文件存储路径',
    `total_rows` INT NOT NULL DEFAULT 0 COMMENT '总行数',
    `success_count` INT NOT NULL DEFAULT 0 COMMENT '成功条数',
    `fail_count` INT NOT NULL DEFAULT 0 COMMENT '失败条数',
    `task_status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1处理中 2完成 3失败',
    `task_result` TEXT COMMENT '任务结果摘要',
    `operator_id` BIGINT NOT NULL COMMENT '操作人ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `finish_time` DATETIME DEFAULT NULL COMMENT '完成时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资产导入任务记录表';

-- ===========================================
-- 四、探测任务表
-- ===========================================

-- 4.1 公网备案探测任务表
CREATE TABLE IF NOT EXISTS `detect_record_company` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `company_name` VARCHAR(100) NOT NULL COMMENT '探测企业名称',
    `icp_domains` JSON COMMENT '发现的备案域名列表(JSON)',
    `task_status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1待执行 2执行中 3成功 4失败',
    `success_count` INT NOT NULL DEFAULT 0 COMMENT '新增资产数量',
    `fail_count` INT NOT NULL DEFAULT 0 COMMENT '失败数量',
    `task_msg` TEXT COMMENT '任务日志信息',
    `cron_expr` VARCHAR(100) DEFAULT NULL COMMENT '定时任务Cron表达式',
    `is_periodic` TINYINT NOT NULL DEFAULT 0 COMMENT '是否周期性任务: 0否 1是',
    `next_run_time` DATETIME DEFAULT NULL COMMENT '下次执行时间',
    `create_user` BIGINT NOT NULL COMMENT '创建人ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `finish_time` DATETIME DEFAULT NULL COMMENT '任务完成时间',
    KEY `idx_task_status` (`task_status`),
    KEY `idx_create_user` (`create_user`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='公网备案探测任务表';

-- 4.2 内网 URL 探测任务表
CREATE TABLE IF NOT EXISTS `detect_record_intranet` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `task_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '任务名称',
    `ip_segment` VARCHAR(100) NOT NULL COMMENT '探测IP/网段(CIDR格式,如192.168.1.0/24)',
    `port_range` VARCHAR(100) DEFAULT '80,443,8080,8443,3000,9000' COMMENT '探测端口范围(逗号分隔)',
    `protocol_type` VARCHAR(10) DEFAULT 'ALL' COMMENT '探测协议: HTTP/HTTPS/ALL',
    `scan_rate_limit` INT DEFAULT 100 COMMENT '每秒最大扫描数(限流)',
    `task_status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1待执行 2执行中 3成功 4失败 5部分成功',
    `success_count` INT NOT NULL DEFAULT 0 COMMENT '新增资产数量',
    `updated_count` INT NOT NULL DEFAULT 0 COMMENT '更新资产数量',
    `scanned_ips` INT NOT NULL DEFAULT 0 COMMENT '已扫描IP数',
    `total_ips` INT NOT NULL DEFAULT 0 COMMENT '总需扫描IP数',
    `task_msg` TEXT COMMENT '任务执行日志',
    `cron_expr` VARCHAR(100) DEFAULT NULL COMMENT '定时表达式',
    `is_periodic` TINYINT NOT NULL DEFAULT 0 COMMENT '是否周期性任务',
    `next_run_time` DATETIME DEFAULT NULL COMMENT '下次执行时间',
    `create_user` BIGINT NOT NULL COMMENT '创建人ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `finish_time` DATETIME DEFAULT NULL COMMENT '完成时间',
    KEY `idx_task_status` (`task_status`),
    KEY `idx_create_user` (`create_user`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='内网网段探测任务表';

-- 4.3 内网探测结果明细表
CREATE TABLE IF NOT EXISTS `detect_intranet_detail` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `task_id` BIGINT NOT NULL COMMENT '关联内网探测任务ID',
    `ip_address` VARCHAR(50) NOT NULL COMMENT '目标IP',
    `port` INT NOT NULL DEFAULT 0 COMMENT '发现端口',
    `protocol` VARCHAR(10) DEFAULT '' COMMENT '协议 HTTP/HTTPS',
    `url` VARCHAR(500) DEFAULT '' COMMENT '生成的URL',
    `status_code` INT DEFAULT 0 COMMENT 'HTTP状态码',
    `title` VARCHAR(500) DEFAULT '' COMMENT '网页标题',
    `response_time_ms` INT DEFAULT 0 COMMENT '响应时间(毫秒)',
    `is_new` TINYINT NOT NULL DEFAULT 1 COMMENT '是否新发现的: 0已有 1新增',
    `asset_id` BIGINT DEFAULT NULL COMMENT '关联的资产ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_task_id` (`task_id`),
    KEY `idx_ip` (`ip_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='内网探测结果明细表';

-- ===========================================
-- 五、证书管理模块表
-- ===========================================

-- 5.1 SSL 证书信息主表
CREATE TABLE IF NOT EXISTS `ssl_cert_info` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `asset_id` BIGINT DEFAULT NULL COMMENT '关联资产ID',
    `domain_ip` VARCHAR(100) NOT NULL COMMENT '证书绑定域名/IP',
    `san_domains` TEXT COMMENT 'SAN域名列表(逗号分隔)',
    `cert_type` TINYINT NOT NULL COMMENT '证书类型: 1公网可信证书 2内网自签名证书',
    `cert_source` TINYINT NOT NULL COMMENT '证书来源: 1探测采集 2手动录入 3系统自助申请',
    `issuer` VARCHAR(255) DEFAULT '' COMMENT '颁发机构CA',
    `serial_no` VARCHAR(100) DEFAULT '' COMMENT '证书序列号',
    `encrypt_algorithm` VARCHAR(50) DEFAULT '' COMMENT '加密算法(RSA/ECDSA等)',
    `hash_algorithm` VARCHAR(50) DEFAULT '' COMMENT '哈希算法(SHA256/SHA384等)',
    `key_size` INT DEFAULT 0 COMMENT '密钥长度(位)',
    `valid_start_time` DATETIME NOT NULL COMMENT '证书生效时间',
    `valid_end_time` DATETIME NOT NULL COMMENT '证书过期时间',
    `days_remaining` INT GENERATED ALWAYS AS (DATEDIFF(`valid_end_time`, NOW())) STORED COMMENT '剩余天数(自动计算)',
    `cert_status` TINYINT NOT NULL DEFAULT 1 COMMENT '证书状态: 1正常 2即将过期 3已过期 4无效 5吊销',
    `auto_renew` TINYINT NOT NULL DEFAULT 0 COMMENT '是否自动续签: 0否 1是',
    `alert_days` INT NOT NULL DEFAULT 30 COMMENT '提前N天告警',
    `last_alert_time` DATETIME DEFAULT NULL COMMENT '最后告警时间',
    `apply_task_id` BIGINT DEFAULT NULL COMMENT '关联的申请任务ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_asset_id` (`asset_id`),
    `idx_domain_ip` (`domain_ip`),
    `idx_cert_status` (`cert_status`),
    `idx_valid_end` (`valid_end_time`),
    `idx_cert_source` (`cert_source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SSL证书信息主表';

-- 5.2 免费 HTTPS 证书申请任务表
CREATE TABLE IF NOT EXISTS `ssl_cert_apply_task` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `apply_type` TINYINT NOT NULL COMMENT '申请类型: 1公网域名 2内网IP',
    `apply_addr` VARCHAR(100) NOT NULL COMMENT '申请域名/IP',
    `san_addrs` TEXT COMMENT '额外SAN域名/IP列表(逗号分隔)',
    `asset_id` BIGINT DEFAULT NULL COMMENT '关联资产ID',
    `verify_method` TINYINT DEFAULT 1 COMMENT '校验方式: 1DNS验证 2HTTP文件验证(仅公网)',
    `dns_record_name` VARCHAR(255) DEFAULT '' COMMENT 'DNS验证记录名',
    `dns_record_value` VARCHAR(255) DEFAULT '' COMMENT 'DNS验证记录值',
    `http_verify_path` VARCHAR(255) DEFAULT '' COMMENT 'HTTP验证文件路径',
    `encrypt_algorithm` VARCHAR(50) DEFAULT 'RSA' COMMENT '加密算法 RSA/ECDSA',
    `key_size` INT DEFAULT 2048 COMMENT '密钥长度',
    `valid_days` INT DEFAULT 365 COMMENT '证书有效期(天)',
    `task_status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1申请中 2校验中 3签发成功 4签发失败',
    `fail_reason` TEXT COMMENT '失败原因',
    `cert_file_path` VARCHAR(255) DEFAULT NULL COMMENT '证书文件存储路径(zip包)',
    `private_key_encrypted` TINYINT NOT NULL DEFAULT 0 COMMENT '私钥是否加密存储: 0否 1是',
    `apply_user` BIGINT NOT NULL COMMENT '申请人ID',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `finish_time` DATETIME DEFAULT NULL COMMENT '签发完成时间',
    KEY `idx_task_status` (`task_status`),
    `idx_apply_addr` (`apply_addr`),
    KEY `idx_apply_user` (`apply_user`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='免费HTTPS证书申请任务表';

-- ===========================================
-- 六、预警通知模块表
-- ===========================================

-- 6.1 站内消息通知表
CREATE TABLE IF NOT EXISTS `notify_message` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `receiver_id` BIGINT NOT NULL COMMENT '接收人ID',
    `sender_id` BIGINT DEFAULT 0 COMMENT '发送人ID(0为系统)',
    `msg_type` TINYINT NOT NULL COMMENT '消息类型: 1系统通知 2证书预警 3资产异常 4任务通知 5审批通知',
    `msg_title` VARCHAR(200) NOT NULL COMMENT '消息标题',
    `msg_content` TEXT NOT NULL COMMENT '消息内容',
    `related_module` VARCHAR(50) DEFAULT '' COMMENT '关联模块(asset/cert/task)',
    `related_id` BIGINT DEFAULT 0 COMMENT '关联业务记录ID',
    `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读: 0未读 1已读',
    `read_time` DATETIME DEFAULT NULL COMMENT '阅读时间',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_receiver_id` (`receiver_id`),
    KEY `idx_is_read` (`is_read`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='站内消息通知表';

-- 6.2 告警规则配置表
CREATE TABLE IF NOT EXISTS `alert_rule` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `rule_name` VARCHAR(100) NOT NULL COMMENT '规则名称',
    `rule_type` TINYINT NOT NULL COMMENT '告警类型: 1证书即将过期 2证书已过期 3资产无法访问 4探测任务异常 5证书申请失败',
    `alert_threshold` INT DEFAULT 30 COMMENT '阈值(天数/次数等)',
    `notify_channels` VARCHAR(50) DEFAULT 'message,email' COMMENT '通知渠道(逗号分隔: message/email/sms)',
    `notify_template` TEXT DEFAULT NULL COMMENT '自定义通知模板',
    `enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否启用: 1启用 0禁用',
    `cooldown_minutes` INT DEFAULT 60 COMMENT '冷却间隔(分钟),避免重复告警',
    `remark` VARCHAR(255) DEFAULT '' COMMENT '备注',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警规则配置表';

-- 6.3 告警发送记录表
CREATE TABLE IF NOT EXISTS `alert_send_log` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `rule_id` BIGINT NOT NULL COMMENT '告警规则ID',
    `alert_type` TINYINT NOT NULL COMMENT '告警类型',
    `target_type` VARCHAR(20) NOT NULL COMMENT '推送目标类型: user/phone/email',
    `target_value` VARCHAR(100) NOT NULL COMMENT '推送目标值',
    `channel` VARCHAR(20) NOT NULL COMMENT '发送渠道: message/email/sms',
    `send_status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1待发送 2发送成功 3发送失败',
    `send_error` TEXT COMMENT '错误信息',
    `send_time` DATETIME DEFAULT NULL COMMENT '实际发送时间',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_rule_id` (`rule_id`),
    KEY `idx_alert_type` (`alert_type`),
    KEY `idx_send_status` (`send_status`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSETutf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警发送记录表';

-- ===========================================
-- 七、日志审计表
-- ===========================================

-- 7.1 系统操作审计日志表
CREATE TABLE IF NOT EXISTS `sys_operation_log` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
    `user_id` BIGINT DEFAULT NULL COMMENT '操作人ID',
    `username` VARCHAR(50) DEFAULT '' COMMENT '操作人账号',
    `operation_module` VARCHAR(50) NOT NULL COMMENT '操作模块(auth/asset/cert/detect/system)',
    `operation_type` VARCHAR(50) NOT NULL COMMENT '操作类型(create/update/delete/login/logout/import/export)',
    `request_method` VARCHAR(10) DEFAULT '' COMMENT '请求方法(GET/POST/PUT/DELETE)',
    `request_url` VARCHAR(500) DEFAULT '' COMMENT '请求URL',
    `request_ip` VARCHAR(50) NOT NULL COMMENT '操作IP',
    `request_param` TEXT COMMENT '请求参数(JSON格式)',
    `response_data` TEXT COMMENT '响应数据(脱敏)',
    `operation_result` TINYINT NOT NULL COMMENT '结果: 1成功 2失败',
    `error_msg` TEXT COMMENT '错误信息',
    `execution_time_ms` INT DEFAULT 0 COMMENT '执行耗时(毫秒)',
    `user_agent` VARCHAR(500) DEFAULT '' COMMENT '客户端UA',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY `idx_user_id` (`user_id`),
    `idx_module` (`operation_module`),
    `idx_type` (`operation_type`),
    `idx_result` (`operation_result`),
    `idx_create_time` (`create_time`),
    KEY `idx_request_ip` (`request_ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统操作审计日志表';

-- ===========================================
-- 八、索引优化与约束
-- ===========================================

-- 添加外键约束(可选，影响性能时可关闭)
ALTER TABLE `ssl_cert_info` ADD CONSTRAINT `fk_cert_asset`
    FOREIGN KEY (`asset_id`) REFERENCES `web_asset`(`id`) ON DELETE SET NULL;
    
ALTER TABLE `ssl_cert_apply_task` ADD CONSTRAINT `fk_apply_asset`
    FOREIGN KEY (`asset_id`) REFERENCES `web_asset`(`id`) ON DELETE SET NULL;

SET FOREIGN_KEY_CHECKS = 1;
