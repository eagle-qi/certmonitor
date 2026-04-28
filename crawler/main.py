"""
CertMonitor URL探测与证书采集引擎
=====================================
Python服务，负责：
- 公网备案域名资产自动探测与URL发现
- 内网IP/CIDR网段HTTP/HTTPS端口扫描
- SSL/TLS证书信息自动采集与过期监测
- 免费 HTTPS 证书自助签发（ACME协议 + 自签名）

通过 FastAPI 提供 REST API，供 Go 后端异步调用执行任务。
"""

import os
import sys
import logging
from contextlib import asynccontextmanager
from typing import Optional, List, Dict, Any

import uvicorn
from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings

from core.public_crawler import PublicICPCrawler
from core.intranet_scanner import IntranetScanner
from core.cert_collector import CertificateCollector
from core.cert_generator import CertificateGenerator

# ===========================================
# 配置管理
# ===========================================
class Settings(BaseSettings):
    """应用配置 - 从环境变量或 .env 加载"""

    # 服务配置
    APP_NAME: str = "CertMonitor-Crawler"
    HOST: str = "0.0.0.0"
    PORT: int = 8000
    DEBUG: bool = True
    LOG_LEVEL: str = "INFO"

    # Go 后端连接配置
    GO_BACKEND_URL: str = "http://127.0.0.1:8080"

    # MySQL数据库（直接操作用于写入探测结果）
    MYSQL_HOST: str = "127.0.0.1"
    MYSQL_PORT: int = 3306
    MYSQL_USER: str = "root"
    MYSQL_PASSWORD: str = "CertMonitor@2024"
    MYSQL_DATABASE: str = "certmonitor"

    # Redis缓存
    REDIS_HOST: str = "127.0.0.1"
    REDIS_PORT: int = 6379
    REDIS_DB: int = 0

    # 备案查询接口
    ICP_API_KEY: str = ""
    ICP_API_URL: str = ""

    # 探测默认参数
    DEFAULT_TIMEOUT: int = 10          # 单次请求超时秒数
    MAX_CONCURRENT: int = 50           # 最大并发数
    RATE_LIMIT_PER_SEC: int = 100      # 每秒最大请求数

    # 证书签发配置
    CERT_OUTPUT_DIR: str = "./output/certs"
    ACME_SERVER_URL: str = "https://acme-v02.api.letsencrypt.org/directory"  # Let's Encrypt生产环境
    ACME_STAGING_URL: str = "https://acme-staging-v02.api.letsencrypt.org/directory"  # 测试环境

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()

# 日志配置
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format="%(asctime)s [%(levelname)s] %(name)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)
logger = logging.getLogger(__name__)

# ===========================================
# 初始化核心组件
# ===========================================

public_crawler: Optional[PublicICPCrawler] = None
intranet_scanner: Optional[IntranetScanner] = None
cert_collector: Optional[CertificateCollector] = None
cert_generator: Optional[CertificateGenerator] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """FastAPI 应用生命周期管理"""
    global public_crawler, intranet_scanner, cert_collector, cert_generator

    logger.info("正在初始化探测引擎组件...")

    # 初始化各模块
    public_crawler = PublicICPCrawler(
        api_key=settings.ICP_API_KEY,
        api_url=settings.ICP_API_URL,
        timeout=settings.DEFAULT_TIMEOUT,
    )

    intranet_scanner = IntranetScanner(
        max_concurrent=settings.MAX_CONCURRENT,
        rate_limit=settings.RATE_LIMIT_PER_SEC,
        timeout=settings.DEFAULT_TIMEOUT,
    )

    cert_collector = CertificateCollector(timeout=settings.DEFAULT_TIMEOUT)

    cert_generator = CertificateGenerator(
        output_dir=settings.CERT_OUTPUT_DIR,
        acme_server=settings.ACME_SERVER_URL,
        mysql_config={
            "host": settings.MYSQL_HOST,
            "port": settings.MYSQL_PORT,
            "user": settings.MYSQL_USER,
            "password": settings.MYSQL_PASSWORD,
            "database": settings.MYSQL_DATABASE,
        }
    )

    # 确保证书输出目录存在
    os.makedirs(settings.CERT_OUTPUT_DIR, exist_ok=True)

    logger.info("CertMonitor Crawler 引擎启动成功!")
    yield

    logger.info("正在关闭探测引擎...")
    # 清理资源
    if intranet_scanner:
        await intranet_scanner.close()

    logger.info("引擎已停止")


# 创建 FastAPI 实例
app = FastAPI(
    title="CertMonitor Crawler Engine",
    description="URL探测、内网扫描、SSL证书采集与签发引擎",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ===========================================
# API 路由定义
# ===========================================

@app.get("/health")
async def health_check():
    """健康检查接口"""
    return {"status": "ok", "service": "certmonitor-crawler", "version": "1.0.0"}


# ==================== 公网备案探测接口 ====================

class PublicDetectRequest(BaseModel):
    task_id: int
    company_name: str
    detect_type: str = "public_icp"


@app.post("/api/v1/detect/public")
async def detect_public_assets(req: PublicDetectRequest, background_tasks: BackgroundTasks):
    """
    触发公网备案域名资产探测
    由 Go 后端调用此接口异步执行探测任务
    """
    try:
        background_tasks.add_task(run_public_detection, req.task_id, req.company_name)
        return {
            "code": 200,
            "message": "公网探测任务已接收，正在后台执行",
            "data": {"task_id": req.task_id}
        }
    except Exception as e:
        logger.error(f"启动公网探测任务失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


async def run_public_detection(task_id: int, company_name: str):
    """后台执行公网备案探测任务"""
    logger.info(f"[Task-{task_id}] 开始公网备案探测: 企业={company_name}")

    try:
        result = await public_crawler.crawl(company_name, task_id)
        logger.info(f"[Task-{task_id}] 公网探测完成: 发现{len(result.get('domains', []))}个域名")
    except Exception as e:
        logger.error(f"[Task-{task_id}] 公网探测异常: {e}", exc_info=True)


# ==================== 内网URL探测接口 ====================

class IntranetDetectRequest(BaseModel):
    task_id: int
    task_name: str
    ip_segment: str                    # CIDR格式: 192.168.1.0/24
    port_range: str = "80,443,8080,8443,3000,9000"
    protocol_type: str = "ALL"         # HTTP / HTTPS / ALL
    rate_limit: int = 100
    detect_type: str = "intranet_scan"


@app.post("/api/v1/detect/intranet")
async def detect_intranet_assets(req: IntranetDetectRequest, background_tasks: BackgroundTasks):
    """
    触发内网IP/网段 URL 探测
    扫描指定网段的开放端口并检测HTTP/HTTPS服务
    """
    try:
        background_tasks.add_task(
            run_intranet_detection,
            req.task_id, req.task_name, req.ip_segment,
            req.port_range, req.protocol_type, req.rate_limit
        )
        return {
            "code": 200,
            "message": "内网探测任务已接收，正在后台执行",
            "data": {"task_id": req.task_id}
        }
    except Exception as e:
        logger.error(f"启动内网探测任务失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


async def run_intranet_detection(task_id: int, task_name: str, ip_segment: str,
                                  port_range: str, protocol_type: str, rate_limit: int):
    """后台执行内网网段探测任务"""
    logger.info(f"[Task-{task_id}] 开始内网探测: 网段={ip_segment}, 协议={protocol_type}, 端口={port_range}")

    try:
        ports = [int(p.strip()) for p in port_range.split(",") if p.strip().isdigit()]
        result = await intranet_scanner.scan(
            task_id=task_id,
            ip_segment=ip_segment,
            ports=ports,
            protocol=protocol_type,
            rate_limit=rate_limit,
        )
        logger.info(f"[Task-{task_id}] 内网探测完成: 发现{result.get('new_count', 0)}个新资产")
    except Exception as e:
        logger.error(f"[Task-{task_id}] 内网探测异常: {e}", exc_info=True)


# ==================== SSL证书采集接口 ====================

class CertCollectRequest(BaseModel):
    asset_id: int
    domain: str
    port: int = 443
    collect_type: str = "auto"   # auto / manual


@app.post("/api/v1/cert/collect")
async def collect_certificate(req: CertCollectRequest):
    """
    对指定域名/IP进行SSL证书信息采集
    建立TLS连接并提取证书详情
    """
    try:
        cert_info = await cert_collector.collect(req.domain, req.port)
        return {
            "code": 200,
            "message": "证书采集成功",
            "data": cert_info
        }
    except Exception as e:
        logger.error(f"证书采集失败({req.domain}:{req.port}): {e}")
        raise HTTPException(status_code=500, detail=f"证书采集失败: {str(e)}")


# ==================== 证书签发接口 ====================

class CertApplyRequest(BaseModel):
    task_id: int
    apply_type: int                   # 1公网域名 2内网IP
    apply_addr: str                   # 申请的域名或IP
    san_addrs: str = ""               # 额外SAN地址(逗号分隔)
    verify_method: int = 1            # 1DNS验证 2HTTP文件验证(仅公网)
    encrypt_algorithm: str = "RSA"    # RSA / ECDSA
    key_size: int = 2048              # 密钥长度
    valid_days: int = 365             # 证书有效期(天)


@app.post("/api/v1/cert/apply")
async def apply_certificate(req: CertApplyRequest, background_tasks: BackgroundTasks):
    """
    提交免费HTTPS证书申请
    - 公网域名: 使用ACME协议申请 Let's Encrypt 免费证书
    - 内网IP: 快速生成自签名CA+证书
    """
    try:
        background_tasks.add_task(run_cert_apply, req.dict())
        return {
            "code": 200,
            "message": "证书签发任务已提交",
            "data": {"task_id": req.task_id}
        }
    except Exception as e:
        logger.error(f"提交证书申请失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


async def run_cert_apply(params: dict):
    """后台执行证书签发"""
    task_id = params.get("task_id")
    apply_addr = params.get("apply_addr")
    apply_type = params.get("apply_type")

    logger.info(f"[Apply-{task_id}] 开始证书签发: 类型={'公网' if apply_type == 1 else '内网'}, 地址={apply_addr}")

    try:
        if apply_type == 1:
            # 公网域名 -> ACME协议签发
            result = await cert_generator.issue_acme_cert(**params)
        else:
            # 内网IP -> 自签名证书生成
            result = await cert_generator.generate_self_signed(**params)

        logger.info(f"[Apply-{task_id}] 证书签发完成: status={result.get('status')}")
    except Exception as e:
        logger.error(f"[Apply-{task_id}] 证书签发异常: {e}", exc_info=True)


# ==================== 批量证书检查接口（定时任务用）====================

@app.post("/api/v1/cert/batch-check")
async def batch_check_certificates(domains: List[str], background_tasks: BackgroundTasks):
    """
    批量检查多个域名的SSL证书状态
    主要用于定时巡检任务
    """
    background_tasks.add_task(run_batch_cert_check, domains)
    return {"code": 200, "message": f"批量检查任务已启动, 共{len(domains)}个域名"}


async def run_batch_cert_check(domains: List[str]):
    """批量证书状态检查"""
    results = []
    for domain in domains:
        try:
            info = await cert_collector.collect(domain, 443)
            results.append({"domain": domain, **info})
        except Exception as e:
            results.append({"domain": domain, "error": str(e)})
    
    logger.info(f"批量证书检查完成: 总数={len(domains)}, 成功={sum(1 for r in results if 'error' not in r)}")
    return results


# ===========================================
# 启动入口
# ===========================================

if __name__ == "__main__":
    print("""
    ╔══════════════════════════════════════╗
    ║   CertMonitor Crawler Engine v1.0     ║
    ║   URL探测 · 内网扫描 · 证书采集签发   ║
    ╚══════════════════════════════════════╝
    """)
    
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level=settings.LOG_LEVEL.lower(),
    )
