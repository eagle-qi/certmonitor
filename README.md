# CertMonitor - URL 探测与证书生命周期管理系统

## 项目简介

CertMonitor 是面向企业**内外网 Web 资产治理 + SSL 证书全生命周期管控**的一体化平台，实现公网备案资产自动挖掘、内网网段定时 URL 探测、资产精细化多维台账管理、Excel 批量资产导入校验、证书自动采集过期预警、公网域名/内网 IP 免费 HTTPS 证书自助签发、完整用户账号体系（邮箱注册+SSO 单点登录）、细粒度权限控制、全流程操作审计、多维度可视化统计分析。

## 核心功能

| 模块 | 功能描述 |
|------|----------|
| 用户管理 | 账号密码登录、邮箱自助注册、企业 SSO 单点登录、RBAC 权限控制 |
| 资产管理 | URL 资产 CRUD、多维字段管理、批量导入导出、审核流程 |
| 公网探测 | 输入企业名称自动探测备案域名资产 |
| 内网探测 | 内网 IP/网段定时任务探测 HTTP/HTTPS 服务 |
| 证书监控 | SSL 证书自动采集、有效期监测、多阈值过期预警 |
| 证书签发 | 对标 allinssl，支持公网域名/内网 IP 免费 HTTPS 证书申请 |
| 预警通知 | 站内消息、邮件、短信多渠道风险预警推送 |
| 统计报表 | 资产、证书、探测任务多维度可视化统计 |

## 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                        接入层                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ Vue3 前端界面 │  │  Nginx 反向代理  │  │   API 网关(鉴权)    │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                       应用服务层                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │Go后端服务 │ │Python探测 │ │ 定时任务  │  │  预警通知服务   │   │
│  │(Gin框架) │ │ 引擎服务  │ │ 调度器   │  │ (邮件/短信)   │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                         数据层                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   MySQL 8.0   │  │    Redis     │  │    文件存储       │  │
│  │ (业务数据持久化)│  │(缓存/会话)  │  │ (证书包/日志)    │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 技术栈

- **后端**: Go 1.21+ / Gin Web Framework / GORM
- **前端**: Vue3 + Element Plus + ECharts
- **数据库**: MySQL 8.0
- **缓存**: Redis
- **探测引擎**: Python3 (requests, ssl, concurrent.futures)
- **任务调度**: Go cron + Python APScheduler
- **部署**: Docker Compose
- **认证**: JWT + SSO(OAuth2.0/OIDC/CAS)

## 快速开始

### 环境要求

- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- Python 3.9+

### 使用 Docker Compose 一键部署（推荐）

```bash
# 克隆项目
git clone <repository-url>
cd certmonitor

# 复制环境配置文件
cp .env.example .env
# 编辑 .env 配置数据库连接、Redis 等信息

# 一键启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 手动部署（开发环境）

#### 1. 初始化数据库

```bash
mysql -u root -p < deploy/sql/init.sql
mysql -u root -p < deploy/sql/seed.sql
```

#### 2. 启动 Redis

```bash
redis-server --port 6379
```

#### 3. 启动 Go 后端服务

```bash
cd backend
cp config.yaml.example config.yaml
# 编辑 config.yaml 配置数据库连接等信息
go mod download
go run main.go
```

#### 4. 启动 Python 探测引擎

```bash
cd crawler
pip install -r requirements.txt
python main.py
```

#### 5. 启动前端开发服务器

```bash
cd frontend
npm install
npm run dev
```

## 默认账号

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 超级管理员 | admin | Admin@123 |

## 项目结构

```
certmonitor/
├── backend/                    # Go 后端服务
│   ├── cmd/                    # 入口文件
│   ├── internal/               # 内部模块
│   │   ├── config/             # 配置管理
│   │   ├── middleware/         # 中间件(JWT/CORS/日志)
│   │   ├── model/              # 数据模型(GORM)
│   │   ├── router/             # 路由定义
│   │   ├── service/            # 业务逻辑层
│   │   ├── repository/         # 数据访问层
│   │   └── handler/            # HTTP 处理器
│   ├── pkg/                    # 公共工具包
│   ├── config.yaml             # 配置文件
│   └── go.mod                  # Go 模块依赖
├── crawler/                    # Python 探测引擎
│   ├── core/                   # 探测核心逻辑
│   │   ├── public_crawler.py   # 公网备案探测
│   │   ├── intranet_scanner.py # 内网扫描
│   │   ├── cert_collector.py   # 证书采集
│   │   └── cert_generator.py   # 证书生成(自签名/ACME)
│   ├── utils/                  # 工具函数
│   ├── config.py               # 配置
│   └── main.py                 # 入口
├── frontend/                   # Vue3 前端
│   ├── src/
│   │   ├── views/              # 页面视图
│   │   ├── components/         # 公共组件
│   │   ├── api/                # API 接口封装
│   │   ├── store/              # Pinia 状态管理
│   │   └── router/             # 路由配置
│   └── package.json
├── deploy/
│   ├── docker-compose.yml      # Docker 编排
│   ├── docker/                 # 各服务 Dockerfile
│   └── sql/                    # 数据库脚本
│       ├── init.sql            # 建表语句
│       └── seed.sql            # 种子数据
├── docs/                       # 项目文档
│   ├── api.md                  # API 接口文档
│   └── deployment.md           # 部署指南
├── .env.example                # 环境变量示例
└── README.md
```

## API 接口文档

详见 [docs/api.md](docs/api.md)

## 系统架构图

详见需求文档：
- [URL 探测与证书生命周期管理系统完整需求文档](docs/requirements.md)
- [系统架构设计与 MySQL 数据库设计](docs/architecture.md)

## 开发指南

### 添加新的业务模块

1. 在 `backend/internal/model/` 定义数据模型
2. 在 `backend/internal/repository/` 实现数据访问
3. 在 `backend/internal/service/` 实现业务逻辑
4. 在 `backend/internal/handler/` 实现 HTTP 处理器
5. 在 `backend/internal/router/` 注册路由

### 代码规范

- 遵循 Go 官方代码风格
- API 返回统一 JSON 格式
- 所有接口需要鉴权（白名单除外）
- 关键操作必须记录审计日志

## License

MIT License
