-- ===========================================
-- CertMonitor 种子数据脚本
-- 包含：预设角色、默认管理员、初始配置项、默认告警规则
-- ===========================================

USE `certmonitor`;

-- ===========================================
-- 1. 插入预设角色
-- ===========================================

INSERT INTO `sys_role` (`role_name`, `role_code`, `description`, `sort_order`, `status`) VALUES
('超级管理员', 'super_admin', '拥有系统所有权限', 1, 1),
('资产管理员', 'asset_admin', '负责资产的增删改查、批量导入、审核确认', 2, 1),
('证书管理员', 'cert_admin', '负责SSL证书的采集监控、自助申请签发', 3, 1),
('项目负责人', 'project_manager', '可查看和管理本项目下的资产和证书', 4, 1),
('普通查看员', 'viewer', '只读查看所有资产和证书信息', 5, 1);

-- ===========================================
-- 2. 插入默认超级管理员 (密码: Admin@123, BCrypt加密)
-- ===========================================

-- 密码: Admin@123 的BCrypt哈希值 (使用Go bcrypt生成)
INSERT INTO `sys_user` (`username`, `real_name`, `email`, `phone`, `password`, `dept_name`, `account_status`, `register_type`, `remark`) VALUES
('admin', '超级管理员', 'admin@certmonitor.local', '', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '技术部', 1, 1, '系统默认管理员账户');

-- 为管理员分配超级管理员角色
INSERT INTO `sys_user_role` (`user_id`, `role_id`) VALUES (1, 1);

-- ===========================================
-- 3. 插入测试普通用户
-- ===========================================

INSERT INTO `sys_user` (`username`, `real_name`, `email`, `phone`, `password`, `dept_name`, `account_status`, `register_type`) VALUES
('zhangsan', '张三', 'zhangsan@example.com', '13800138001', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '研发部', 1, 1),
('lisi', '李四', 'lisi@example.com', '13800138002', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '运维部', 1, 1);

INSERT INTO `sys_user_role` (`user_id`, `role_id`) VALUES (2, 4), (3, 2);

-- ===========================================
-- 4. 插入系统全局配置
-- ===========================================

INSERT INTO `sys_config` (`config_key`, `config_value`, `config_name`, `config_group`, `remark`) VALUES
-- 用户认证相关
('register_enabled', '1', '允许邮箱注册开关', 'auth', '开启后用户可通过邮箱自行注册'),
('register_email_whitelist', '@example.com,@company.com', '注册邮箱白名单', 'auth', '只允许指定域名的邮箱注册，留空表示不限制'),
('login_max_fail_count', '5', '最大登录失败次数', 'auth', '超过此次数将锁定账号'),
('login_lock_duration', '30', '账号锁定时长(分钟)', 'auth', '超过最大失败次数后的锁定时长'),
('session_timeout', '7200', '会话超时时间(秒)', 'auth', '无操作多久后需要重新登录'),
-- SSO配置
('sso_enabled', '0', 'SSO单点登录开关', 'sso', '是否开启企业SSO登录'),
('sso_default_role_id', '5', 'SSO默认角色ID', 'sso', 'SSO首次登录自动创建账号时分配的角色'),
('sso_allow_password_login', '1', 'SSO账号允许密码登录', 'sso', 'SSO创建的账号是否也允许用密码登录'),
-- 证书相关
('cert_alert_default_days', '30', '证书默认提前告警天数', 'cert', '证书过期前多少天开始告警'),
('cert_auto_check_enabled', '1', '自动证书巡检开关', 'cert', '是否自动定期检查证书有效期'),
('cert_check_interval', '3600', '证书巡检间隔(秒)', 'cert', '每隔多长时间检查一次所有证书有效期'),
-- 探测相关
('detect_rate_limit', '100', '探测速率限制(次/秒)', 'detect', '内网探测每秒最多请求数'),
('detect_timeout_seconds', '10', '探测超时时间(秒)', 'detect', '单个URL探测的超时时间'),
-- 通知相关
('mail_enabled', '0', '邮件通知开关', 'notify', '是否启用邮件通知'),
('sms_enabled', '0', '短信通知开关', 'notify', '是否启用短信通知');

-- ===========================================
-- 5. 插入默认告警规则
-- ===========================================

INSERT INTO `alert_rule` (`rule_name`, `rule_type`, `alert_threshold`, `notify_channels`, `enabled`, `cooldown_minutes`, `remark`) VALUES
('证书即将到期(30天)', 1, 30, 'message,email,sms', 1, 1440, '证书剩余<=30天时触发告警'),
('证书即将到期(15天)', 1, 15, 'message,email,sms', 1, 1440, '证书剩余<=15天时触发告警'),
('证书即将到期(7天)', 1, 7, 'message,email,sms', 1, 720, '证书剩余<=7天时触发告警'),
('证书已过期', 2, 0, 'message,email,sms', 1, 120, '证书已过期的紧急告警'),
('资产无法访问', 3, 3, 'message,email', 1, 480, '资产连续N次无法访问时告警'),
('探测任务异常', 4, 0, 'message,email', 1, 240, '探测任务执行失败的告警'),
('证书申请签发失败', 5, 0, 'message,email', 1, 60, '证书自助申请签发失败的告警');

-- ===========================================
-- 6. 插入默认探测规则(示例)
-- ===========================================

INSERT INTO `sys_detect_rule` (`rule_type`, `rule_value`, `enabled`, `remark`) VALUES
-- IP黑名单
(1, '127.0.0.1', 1, '禁止探测本机回环地址'),
(1, '169.254.0.0/16', 1, '禁止探测链路本地地址'),
(1, '224.0.0.0/4', 1, '禁止探测组播地址'),
(1, '255.255.255.255', 1, '禁止探测广播地址'),
-- 端口黑名单(危险端口不探测)
(3, '22', 1, 'SSH端口不探测'),
(3, '23', 1, 'Telnet端口不探测'),
(3, '3306', 1, 'MySQL端口不探测'),
(3, '6379', 1, 'Redis端口不探测'),
(3, '27017', 1, 'MongoDB端口不探测');

-- ===========================================
-- 7. 插入测试资产数据(可选)
-- ===========================================

INSERT INTO `web_asset` (`url_address`, `protocol_type`, `ip_address`, `port`, `company_name`, `asset_status`, `asset_source`, 
                          `business_name`, `job_position`, `duty_user_name`, `duty_user_phone`, `duty_user_email`,
                          `project_name`, `project_manager_name`, `dept_name`, `response_code`, `web_title`, `creator_id`) VALUES
('https://www.example.com', 2, '93.184.216.34', 443, 'Example Corp', 2, 4, '官网门户', 3, '张三', '13800138001', 'zhangsan@example.com', '企业官网项目', '李四', '市场部', '200', 'Example Domain', 1),
('https://api.example.com', 2, '93.184.216.35', 443, 'Example Corp', 2, 4, 'API网关', 1, '张三', '13800138001', 'zhangsan@example.com', '微服务架构', '李四', '研发部', '200', 'API Gateway', 1),
('http://internal.example.com', 1, '192.168.1.100', 8080, 'Example Corp', 2, 4, '内部OA系统', 5, '李四', '13800138002', 'lisi@example.com', 'OA办公项目', '李四', '运维部', '200', 'Internal OA System', 1);

-- ===========================================
-- 8. 插入测试证书数据(可选)
-- ===========================================

INSERT INTO `ssl_cert_info` (`asset_id`, `domain_ip`, `cert_type`, `cert_source`, `issuer`, `serial_no`, `encrypt_algorithm`, `hash_algorithm`, `key_size`, `valid_start_time`, `valid_end_time`, `cert_status`) VALUES
(1, 'www.example.com', 1, 1, "Let's Encrypt Authority X3", '03AC75F9A0A3F7B9B2C1D0E8F7A6B5C4D3E2F1A0', 'RSA', 'SHA256', 2048, NOW() - INTERVAL 180 DAY, NOW() + INTERVAL 185 DAY, 1),
(2, 'api.example.com', 1, 1, "Let's Encrypt Authority X3", '04BD86A0B1B4C8C2D3E4F5A6B7C8D9E0F1A2B3C4D5', 'RSA', 'SHA256', 2048, NOW() - INTERVAL 90 DAY, NOW() + INTERVAL 275 DAY, 1),
(3, 'internal.example.com', 2, 3, 'CertMonitor Internal CA', '05CE97B1C2C5D9D3E4F5A6B7C8D9E0F1A2B3C4D5E6', 'ECDSA', 'SHA256', 256, NOW() - INTERVAL 30 DAY, NOW() + INTERVAL 335 DAY, 1);

SELECT '种子数据插入完成！默认管理员账号: admin / Admin@123' AS message;
