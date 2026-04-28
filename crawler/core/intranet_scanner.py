"""
内网IP/CIDR网段 URL 扫描引擎
================================
功能：
- 解析 CIDR 网段，枚举所有 IP 地址
- 对指定端口范围进行 TCP 连接扫描
- 检测开放的 HTTP/HTTPS 服务
- 提取网页标题、状态码等元数据
- 结果写入数据库，自动关联已有资产或新增待确认资产

安全特性：
- 支持速率限流控制
- IP/端口黑白名单过滤
- 内网数据隔离，不对外传输
- 异步并发扫描提升效率

使用示例：
    scanner = IntranetScanner(max_concurrent=100, rate_limit=50, timeout=5)
    result = await scanner.scan(
        task_id=1,
        ip_segment="192.168.1.0/24",
        ports=[80, 443, 8080],
        protocol="ALL",
        rate_limit=50,
    )
"""

import asyncio
import ipaddress
import logging
import socket
import ssl
import time
from typing import List, Dict, Any, Optional, Set, Tuple
from dataclasses import dataclass, field
from concurrent.futures import ThreadPoolExecutor

import aiohttp
from bs4 import BeautifulSoup


logger = logging.getLogger(__name__)


@dataclass
class ScanTarget:
    """单个扫描目标"""
    ip: str
    port: int
    protocol_guess: str = ""  # HTTP / HTTPS / UNKNOWN


@dataclass
class ScanResult:
    """单个IP+端口的扫描结果"""
    ip: str
    port: int
    is_open: bool = False
    is_http_service: bool = False
    protocol: str = ""          # http 或 https
    url: str = ""
    status_code: int = 0
    title: str = ""
    response_time_ms: int = 0
    error: str = ""
    is_new_asset: bool = True   # 是否为新发现的资产(相对于已有数据库记录)


@dataclass
class ScanSummary:
    """一次扫描任务的汇总统计"""
    task_id: int
    total_ips: int = 0
    scanned_ips: int = 0
    open_ports: int = 0
    http_services_found: int = 0
    new_assets_count: int = 0
    updated_assets_count: int = 0
    scan_duration_seconds: float = 0
    details: List[ScanResult] = field(default_factory=list)


# 默认端口黑名单(危险端口不扫描)
DEFAULT_PORT_BLACKLIST: Set[int] = {
    22,     # SSH
    23,     # Telnet
    3306,   # MySQL
    5432,   # PostgreSQL
    6379,   # Redis
    27017,  # MongoDB
    11211,  # Memcached
    9200,   # Elasticsearch
    5672,   # RabbitMQ
}

# 默认IP地址黑名单
DEFAULT_IP_BLACKLIST = [
    "127.0.0.0/8",       # 回环地址
    "169.254.0.0/16",    # 链路本地
    "224.0.0.0/4",      # 组播
    "255.255.255.255/32", # 广播
]


class IntranetScanner:
    """
    内网 URL 探测扫描器
    
    特点：
    - 基于 asyncio + aiohttp 实现高并发异步扫描
    - TCP端口检测 + HTTP协议识别两阶段扫描
    - 支持自定义黑白名单过滤
    - 信号量控制并发度防止网络拥塞
    """

    def __init__(self, max_concurrent: int = 50, rate_limit: int = 100, timeout: int = 10):
        """
        初始化内网扫描器
        
        Args:
            max_concur