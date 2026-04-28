# 系统架构设计与 MySQL 数据库设计

# 一、系统架构设计文档

## 1 文档说明

本文档基于**URL 探测及证书过期管理系统**全业务需求，输出标准企业级系统架构设计，包含架构整体思想、分层架构、部署架构、技术栈选型、模块交互设计，支撑后端开发、架构落地、分布式扩展。

## 2 系统总体设计思想

1. 采用**前后端分离 \+ 微服务模块化架构**，高内聚低耦合，便于独立开发、迭代、扩容；

2. 任务类业务（资产探测、证书签发、定时扫描）采用**异步任务调度**，不阻塞主业务流程；

3. 内外网探测引擎物理隔离，满足内网安全合规要求；

4. 统一身份认证体系，支持账号密码 / 邮箱注册 / SSO 单点登录多模式；

5. 数据分层存储：MySQL 主业务持久化 \+ 缓存高性能加速 \+ 文件存储非结构化资源；

6. 全链路日志审计、细粒度权限控制、多渠道预警通知，满足企业安全运维规范。

## 3 系统分层架构（五层标准架构）

### 3\.1 接入层

- 前端 Web 管理界面：系统所有功能可视化操作入口

- 统一 API 网关：请求路由、全局鉴权、流量限流、跨域处理、请求日志拦截

- 登录门户：账号密码登录、邮箱注册、企业 SSO 单点登录统一入口

### 3\.2 应用服务层

1. **身份认证与用户服务**

    - 用户 CRUD、账号生命周期管理

    - 邮箱注册、邮箱验证码收发校验

    - SSO 协议解析、回调鉴权、影子账号自动创建

    - 登录安全管控：密码锁定、异地登录、会话管理

2. **权限与审计服务**

    - 角色管理、资源权限分配、数据权限控制

    - 全链路操作日志采集、存储、查询、导出审计

3. **定时任务调度服务**

    - 公网备案探测定时调度

    - 内网 URL 网段扫描异步调度

    - 免费 HTTPS 证书签发任务调度

    - 证书过期巡检定时任务

4. **预警通知服务**

    - 站内消息生成与推送

    - 邮件通知分发、短信通知对接

    - 按负责人 / 项目经理定向消息投递

5. **报表统计服务**

    - 资产多维度统计、证书风险统计

    - 可视化图表组装、Excel 报表导出

### 3\.3 核心业务层

1. **资产管理核心模块**

    - 内外网资产探测结果接入

    - 资产单条新增 / 编辑 / 审核状态流转

    - 资产全维度业务、人员、项目、岗位关联管理

2. **资产批量导入模块**

    - Excel 模板动态生成

    - 文件解析、多规则字段校验、批量入库

3. **公网备案探测模块**

    - 企业备案信息查询、域名解析、URL 自动生成

4. **内网 URL 探测模块**

    - 网段存活扫描、HTTP/HTTPS 端口探测、资产状态采集

5. **SSL 证书生命周期模块**

    - HTTPS 证书自动采集、有效期监测、过期预警计算

6. **免费证书自助签发模块（对标 allinssl）**

    - 公网域名 DNS/HTTP 校验引擎

    - 内网 IP 自签名证书生成引擎

    - 多格式证书打包、自动入库、资产绑定

### 3\.4 探测执行层（独立引擎层）

- 外网备案探测引擎：公网域名连通性、状态码、页面标题探测

- 内网 Web 探测引擎：内网 IP / 网段端口扫描、Web 服务发现

- 探测规则引擎：速率限流、黑白名单、探测协议 / 端口管控

- 证书签发执行引擎：异步证书生成、校验、文件打包

### 3\.5 数据层

1. **关系型数据库 MySQL**：存储所有结构化业务主数据

2. **缓存数据库 Redis**：会话缓存、验证码缓存、任务状态缓存、热点数据加速

3. **文件存储**：导入模板、证书私钥包、报表文件、错误日志、系统归档日志

### 3\.6 外部接口层

- 第三方备案查询接口

- 邮件 / 短信通知推送接口

- 企业 SSO 统一身份认证接口（OAuth2\.0/OIDC/CAS）

- 免费证书签发上游底层接口

## 4 系统部署架构

1. **前端部署**：Nginx 静态资源部署，支持 HTTPS 访问、负载均衡

2. **后端服务部署**：模块化独立部署，支持横向扩容，内网探测服务隔离部署在内网区

3. **数据库部署**：MySQL 主从架构，保证数据高可用、读写分离

4. **缓存部署**：Redis 集群部署，保障高并发任务缓存稳定

5. **探测引擎部署**

    - 外网探测引擎：部署外网服务器

    - 内网探测引擎：仅部署内网隔离环境，禁止外网访问

6. **文件存储部署**：本地文件 / 分布式对象存储均可，证书私钥加密存储、权限隔离

## 5 核心技术栈选型建议

- 后端：Java SpringBoot / Go 微服务

- 前端：Vue3 \+ Element Plus

- 数据库：MySQL 8\.0

- 缓存：Redis

- 任务调度：XXL\-Job / 内置定时任务

- 探测引擎：自研 HTTP/HTTPS 扫描器

- 证书签发：兼容 allinssl 底层签发逻辑

- 身份认证：JWT \+ SSO 协议适配

## 6 系统架构文字总图（可直接绘图）

```Plain Text
【接入层】
├─ 前端Web管理界面
├─ 系统登录门户(账号密码/邮箱注册/SSO单点登录)
└─ 统一API网关(鉴权/路由/限流/日志)

【应用服务层】
├─ 用户身份认证服务(邮箱注册+验证码+SSO登录+会话安全)
├─ 权限&操作日志审计服务
├─ 分布式任务调度服务(探测任务+证书签发+证书巡检)
├─ 多渠道预警通知服务(站内/邮件/短信)
└─ 可视化报表统计服务

【核心业务层】
├─ 资产管理模块(资产CRUD+审核流程+多维字段管理)
├─ 资产批量导入模块(模板下载+文件校验+批量入库)
├─ 公网备案资产探测模块
├─ 内网IP/网段URL探测模块
├─ SSL证书全生命周期管理模块
└─ 免费HTTPS证书自助签发模块(allinssl能力)

【探测执行层】
├─ 外网备案探测引擎
├─ 内网Web资产探测引擎
├─ 探测规则引擎(黑白名单/速率/端口管控)
└─ 证书签发底层执行引擎

【数据层】
├─ MySQL(业务主数据持久化)
├─ Redis(缓存/会话/验证码/任务状态)
└─ 文件存储(证书包/导入模板/报表/日志)

【外部接口层】
├─ 域名备案查询第三方接口
├─ 邮件/短信推送接口
├─ 企业SSO统一身份认证接口
└─ 免费证书签发上游依赖接口
```

---

# 二、MySQL 数据库架构设计文档

## 1 设计说明

1. 数据库版本：**MySQL 8\.0**

2. 字符集：`utf8mb4`，排序规则：`utf8mb4\_unicode\_ci`

3. 存储引擎：InnoDB，支持事务、行级锁、主从复制

4. 设计原则：

    - 分表清晰、字段冗余合理、关联外键逻辑清晰

    - 满足资产探测、证书管理、用户权限、批量导入、证书申请全业务

    - 预留扩展字段，支持后续业务迭代

## 2 数据库整体表结构分类

1. 用户权限类：用户表、角色表、用户角色关联表、邮箱验证码表、SSO 登录日志表

2. 系统配置类：系统全局配置表、探测黑白名单配置表

3. 资产业务类：URL 资产主表、资产批量导入错误日志表

4. 探测任务类：公网备案探测任务表、内网 URL 探测任务表

5. 证书管理类：SSL 证书信息表、免费证书申请任务表

6. 日志审计类：系统操作日志表

## 3 详细数据表设计

### 3\.1 用户权限模块表

#### 3\.1\.1 sys\_user 用户主表

```sql
CREATE TABLE sys_user (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    username VARCHAR(50) NOT NULL COMMENT '登录账号',
    real_name VARCHAR(50) NOT NULL COMMENT '真实姓名',
    email VARCHAR(100) NOT NULL UNIQUE COMMENT '登录邮箱',
    phone VARCHAR(20) COMMENT '手机号',
    password VARCHAR(100) NOT NULL COMMENT '加密密码',
    dept_name VARCHAR(50) COMMENT '所属部门',
    account_status TINYINT NOT NULL DEFAULT 1 COMMENT '账号状态 1正常 2禁用 3锁定 4已注销',
    register_type TINYINT NOT NULL DEFAULT 1 COMMENT '注册来源 1手动创建 2邮箱注册 3SSO自动创建',
    sso_unique_id VARCHAR(100) COMMENT 'SSO唯一标识',
    last_login_ip VARCHAR(50) COMMENT '最后登录IP',
    last_login_time DATETIME COMMENT '最后登录时间',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    remark VARCHAR(255) COMMENT '备注'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统用户表';
```

#### 3\.1\.2 sys\_role 角色表

```sql
CREATE TABLE sys_role (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    role_name VARCHAR(50) NOT NULL COMMENT '角色名称',
    role_code VARCHAR(50) NOT NULL UNIQUE COMMENT '角色标识',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    remark VARCHAR(255)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色表';
```

预置角色编码：

- super\_admin：超级管理员

- asset\_admin：资产管理员

- cert\_admin：证书管理员

- project\_manager：项目负责人

- viewer：普通查看员

#### 3\.1\.3 sys\_user\_role 用户角色关联表

```sql
CREATE TABLE sys_user_role (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    role_id BIGINT NOT NULL COMMENT '角色ID',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_role(user_id,role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表';
```

#### 3\.1\.4 sys\_email\_captcha 邮箱验证码表

```sql
CREATE TABLE sys_email_captcha (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(100) NOT NULL COMMENT '邮箱地址',
    captcha VARCHAR(10) NOT NULL COMMENT '验证码',
    expire_time DATETIME NOT NULL COMMENT '过期时间',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_email(email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮箱验证码记录表';
```

#### 3\.1\.5 sys\_sso\_login\_log SSO 登录审计日志

```sql
CREATE TABLE sys_sso_login_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT COMMENT '关联用户ID',
    sso_unique_id VARCHAR(100) NOT NULL COMMENT 'SSO唯一标识',
    login_ip VARCHAR(50) NOT NULL COMMENT '登录IP',
    login_status TINYINT NOT NULL COMMENT '登录状态 1成功 2失败',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='SSO单点登录日志表';
```

### 3\.2 系统配置表

#### 3\.2\.1 sys\_config 系统全局配置表

```sql
CREATE TABLE sys_config (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    config_key VARCHAR(50) NOT NULL UNIQUE COMMENT '配置键',
    config_value VARCHAR(500) COMMENT '配置值',
    config_name VARCHAR(100) NOT NULL COMMENT '配置名称',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统全局配置表';
```

配置项包含：邮箱注册开关、邮箱白名单域名、SSO 开关、证书预警默认天数、探测限流参数等。

### 3\.3 资产核心业务表

#### 3\.3\.1 web\_asset URL 资产主表（核心大表）

```sql
CREATE TABLE web_asset (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    url_address VARCHAR(500) NOT NULL COMMENT 'URL地址',
    protocol_type TINYINT NOT NULL COMMENT '协议 1HTTP 2HTTPS',
    ip_address VARCHAR(50) COMMENT '公网/内网IP',
    port INT COMMENT '开放端口',
    company_name VARCHAR(100) NOT NULL COMMENT '所属公司名称',
    asset_status TINYINT NOT NULL DEFAULT 1 COMMENT '资产状态 1待确认 2已确认 3已失效 4已废弃',
    asset_source TINYINT NOT NULL COMMENT '资产来源 1备案探测 2内网探测 3批量导入 4手动新增',
    -- 业务系统信息
    business_name VARCHAR(100) NOT NULL COMMENT '业务系统名称',
    business_desc TEXT COMMENT '业务系统描述',
    job_position TINYINT NOT NULL COMMENT '归属岗位 1开发 2测试 3生产 4预发布 5办公系统',
    -- 资产负责人信息
    duty_user_name VARCHAR(50) NOT NULL COMMENT '资产负责人姓名',
    duty_user_phone VARCHAR(20) NOT NULL COMMENT '负责人手机号',
    duty_user_email VARCHAR(100) NOT NULL COMMENT '负责人工作邮箱',
    -- 项目关联信息
    project_name VARCHAR(100) COMMENT '所属项目名称',
    project_manager_name VARCHAR(50) COMMENT '项目经理名称',
    project_manager_email VARCHAR(100) COMMENT '项目经理邮箱',
    -- 辅助信息
    dept_name VARCHAR(50) COMMENT '归属部门',
    remark VARCHAR(255) COMMENT '资产备注',
    -- 探测自动填充字段
    response_code VARCHAR(10) COMMENT 'URL响应状态码',
    web_title VARCHAR(255) COMMENT '网站标题',
    last_detect_time DATETIME COMMENT '最后探测时间',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_url_protocol(url_address,protocol_type),
    KEY idx_company(company_name),
    KEY idx_business(business_name),
    KEY idx_job_position(job_position),
    KEY idx_asset_status(asset_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='URL资产主表';
```

#### 3\.3\.2 web\_asset\_import\_log 资产批量导入错误日志

```sql
CREATE TABLE web_asset_import_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    file_name VARCHAR(200) NOT NULL COMMENT '导入文件名',
    row_num INT NOT NULL COMMENT '错误行号',
    error_msg TEXT NOT NULL COMMENT '错误原因',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_create_time(create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资产批量导入错误日志表';
```

### 3\.4 探测任务表

#### 3\.4\.1 detect\_record\_company 公网备案探测任务表

```sql
CREATE TABLE detect_record_company (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    company_name VARCHAR(100) NOT NULL COMMENT '探测企业名称',
    task_status TINYINT NOT NULL COMMENT '任务状态 1待执行 2执行中 3成功 4失败',
    success_count INT DEFAULT 0 COMMENT '新增资产数量',
    fail_count INT DEFAULT 0 COMMENT '失败数量',
    task_msg TEXT COMMENT '任务日志信息',
    cron_expr VARCHAR(100) COMMENT '定时任务表达式',
    create_user BIGINT NOT NULL COMMENT '创建人ID',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finish_time DATETIME COMMENT '任务完成时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='公网备案探测任务表';
```

#### 3\.4\.2 detect\_record\_intranet 内网 URL 探测任务表

```sql
CREATE TABLE detect_record_intranet (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    ip_segment VARCHAR(100) NOT NULL COMMENT '探测IP/网段',
    port_range VARCHAR(100) COMMENT '探测端口范围',
    protocol_type VARCHAR(50) COMMENT '探测协议 HTTP/HTTPS/ALL',
    task_status TINYINT NOT NULL COMMENT '任务状态 1待执行 2执行中 3成功 4失败',
    success_count INT DEFAULT 0 COMMENT '新增资产数量',
    task_msg TEXT COMMENT '任务执行日志',
    cron_expr VARCHAR(100) COMMENT '定时表达式',
    create_user BIGINT NOT NULL COMMENT '创建人ID',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finish_time DATETIME COMMENT '完成时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='内网网段探测任务表';
```

### 3\.5 证书管理模块表

#### 3\.5\.1 ssl\_cert\_info SSL 证书信息主表

```sql
CREATE TABLE ssl_cert_info (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    asset_id BIGINT COMMENT '关联资产ID',
    domain_ip VARCHAR(100) NOT NULL COMMENT '证书绑定域名/IP',
    cert_type TINYINT NOT NULL COMMENT '证书类型 1公网可信证书 2内网自签名证书',
    cert_source TINYINT NOT NULL COMMENT '证书来源 1探测采集 2手动录入 3系统自助申请',
    issuer VARCHAR(255) COMMENT '颁发机构',
    serial_no VARCHAR(100) COMMENT '证书序列号',
    encrypt_algorithm VARCHAR(50) COMMENT '加密算法',
    valid_start_time DATETIME NOT NULL COMMENT '证书生效时间',
    valid_end_time DATETIME NOT NULL COMMENT '证书过期时间',
    cert_status TINYINT NOT NULL COMMENT '证书状态 1正常 2即将过期 3已过期 4无效 5吊销',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_asset_id(asset_id),
    KEY idx_cert_status(cert_status),
    KEY idx_valid_end(valid_end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='SSL证书信息主表';
```

#### 3\.5\.2 ssl\_cert\_apply\_task 免费证书申请任务表

```sql
CREATE TABLE ssl_cert_apply_task (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    apply_type TINYINT NOT NULL COMMENT '申请类型 1公网域名 2内网IP',
    apply_addr VARCHAR(100) NOT NULL COMMENT '申请域名/IP',
    asset_id BIGINT COMMENT '关联资产ID',
    task_status TINYINT NOT NULL COMMENT '任务状态 1申请中 2校验中 3签发成功 4签发失败',
    fail_reason TEXT COMMENT '失败原因',
    cert_file_path VARCHAR(255) COMMENT '证书文件存储路径',
    apply_user BIGINT NOT NULL COMMENT '申请人ID',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finish_time DATETIME COMMENT '签发完成时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='免费HTTPS证书申请任务表';
```

### 3\.6 日志审计表

#### 3\.6\.1 sys\_operation\_log 系统全量操作日志

```sql
CREATE TABLE sys_operation_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT COMMENT '操作人ID',
    username VARCHAR(50) COMMENT '操作人账号',
    operation_module VARCHAR(50) NOT NULL COMMENT '操作模块',
    operation_type VARCHAR(50) NOT NULL COMMENT '操作类型',
    request_ip VARCHAR(50) NOT NULL COMMENT '操作IP',
    request_param TEXT COMMENT '请求参数',
    operation_result TINYINT NOT NULL COMMENT '操作结果 1成功 2失败',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统操作审计日志表';
```

## 4 数据库索引设计原则

1. 高频查询字段建立普通索引：资产状态、归属岗位、证书过期时间、任务状态

2. 唯一业务键建立唯一索引：URL \+ 协议、用户邮箱、用户角色关联

3. 大表避免过度索引，写入型表精简索引，查询型表优化组合索引

4. 时间范围查询字段优先建立索引，提升报表统计、过期巡检效率

## 5 数据表关联关系说明

1. **用户 \- 角色**：sys\_user 多对多 sys\_role，通过 sys\_user\_role 中间表关联

2. **资产 \- 证书**：web\_asset 一对多 ssl\_cert\_info，一个资产可绑定多张证书

3. **资产 \- 导入日志**：web\_asset\_import\_log 记录批量导入失败资产明细

4. **证书申请任务 \- 证书主表**：签发成功后自动写入 ssl\_cert\_info，并关联 asset\_id 绑定资产

5. **探测任务 \- 资产**：探测任务执行完成后，批量生成数据写入 web\_asset 资产表

## 6 数据库优化建议

1. 大表（web\_asset、sys\_operation\_log）建议按时间分表或历史数据归档

2. 证书过期巡检、资产状态扫描定时 SQL 增加索引优化，避免全表扫描

3. 开启 MySQL 慢查询日志，定期优化慢 SQL

4. 核心业务表定期碎片整理，提升 InnoDB 读写性能

5. 日志类表支持自动清理策略，避免数据无限膨胀

> （注：文档部分内容可能由 AI 生成）
