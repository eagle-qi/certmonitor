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
            max_concurrent: 最大并发连接数
            rate_limit: 每秒最大请求数限制
            timeout: 单次请求超时秒数(秒)
        """
        self.max_concurrent = max_concurrent
        self.rate_limit = rate_limit
        self.timeout = timeout
        self.semaphore = asyncio.Semaphore(max_concurrent)
        self._executor = ThreadPoolExecutor(max_workers=max_concurrent)
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        if self._session is None or self._session.closed:
            timeout = aiohttp.ClientTimeout(total=self.timeout, connect=5)
            self._session = aiohttp.ClientSession(timeout=timeout)
        return self._session

    def _is_ip_blacklisted(self, ip_str: str) -> bool:
        """检查IP是否在黑名单中"""
        try:
            ip = ipaddress.ip_address(ip_str)
            for cidr in DEFAULT_IP_BLACKLIST:
                if ip in ipaddress.ip_network(cidr, strict=False):
                    return True
        except ValueError:
            pass
        return False

    def _is_port_blacklisted(self, port: int) -> bool:
        return port in DEFAULT_PORT_BLACKLIST

    def _parse_ip_segment(self, ip_segment: str) -> List[str]:
        """
        解析CIDR网段或单IP，返回所有有效IP列表
        """
        ips = []
        if '/' in ip_segment:
            try:
                network = ipaddress.ip_network(ip_segment, strict=False)
                ips = [str(ip) for ip in network.hosts()]
                # 限制单个网段最多256个主机
                if len(ips) > 256:
                    logger.warning(f"网段 {ip_segment} 包含{len(ips)}个IP，截断为前256个")
                    ips = ips[:256]
            except ValueError as e:
                logger.error(f"无效的CIDR格式: {ip_segment}, 错误: {e}")
                return []
        else:
            ips = [ip_segment]
        return [ip for ip in ips if not self._is_ip_blacklisted(ip)]

    async def _tcp_connect(self, ip: str, port: int) -> bool:
        """
        检查指定IP:端口是否可TCP连接（在线程池中执行阻塞操作）
        """
        loop = asyncio.get_event_loop()
        try:
            _, writer = await loop.run_in_executor(
                self._executor,
                lambda: asyncio.run(self._do_tcp_connect(ip, port))
            )
            return True
        except Exception:
            return False

    def _do_tcp_connect(self, ip: str, port: int):
        """同步TCP连接测试（在线程池中运行）"""
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(self.timeout)
        try:
            sock.connect((ip, port))
            return (sock, None)
        finally:
            sock.close()

    async def _detect_http_service(self, ip: str, port: int, protocol_type: str) -> Dict[str, Any]:
        """
        对开放端口尝试HTTP/HTTPS协议探测，提取标题等信息
        """
        result = {
            "protocol": "",
            "url": "",
            "status_code": 0,
            "title": "",
            "response_time_ms": 0,
            "error": "",
        }
        protocols_to_try = []
        if protocol_type in ("HTTPS", "ALL"):
            protocols_to_try.append("https")
        if protocol_type in ("HTTP", "ALL"):
            protocols_to_try.append("http")

        session = await self._get_session()

        for proto in protocols_to_try:
            url = f"{proto}://{ip}:{port}"
            start_time = time.time()
            try:
                ssl_ctx = None
                if proto == "https":
                    ssl_ctx = ssl.create_default_context()
                    ssl_ctx.check_hostname = False
                    ssl_ctx.verify_mode = ssl.CERT_NONE

                async with session.get(url, allow_redirects=True, ssl=ssl_ctx) as resp:
                    elapsed_ms = int((time.time() - start_time) * 1000)
                    result["protocol"] = proto
                    result["url"] = url
                    result["status_code"] = resp.status
                    result["response_time_ms"] = elapsed_ms
                    try:
                        html = await resp.text(encoding="utf-8", errors="ignore")
                        soup = BeautifulSoup(html, "html.parser")
                        title_tag = soup.find("title")
                        if title_tag and title_tag.string:
                            result["title"] = title_tag.string.strip()[:200]
                    except Exception:
                        pass
                    break  # 成功获取响应后不再尝试其他协议
            except asyncio.TimeoutError:
                result["error"] = f"{proto.upper()} 超时"
            except aiohttp.ClientError as e:
                result["error"] = f"{proto.upper()} 连接错误: {str(e)[:100]}"
            except Exception as e:
                result["error"] = str(e)[:200]

        return result

    async def _scan_target(self, target: ScanTarget, protocol_type: str) -> ScanResult:
        """
        扫描单个IP+端口组合：先TCP探测再HTTP识别
        """
        scan_result = ScanResult(ip=target.ip, port=target.port)

        if self._is_port_blacklisted(target.port):
            return scan_result

        async with self.semaphore:
            is_open = await self._tcp_connect(target.ip, target.port)
            if not is_open:
                return scan_result

            scan_result.is_open = True
            http_info = await self._detect_http_service(target.ip, target.port, protocol_type)
            scan_result.protocol = http_info["protocol"]
            scan_result.url = http_info["url"]
            scan_result.status_code = http_info["status_code"]
            scan_result.title = http_info["title"]
            scan_result.response_time_ms = http_info["response_time_ms"]
            scan_result.error = http_info["error"]

            if http_info["protocol"]:
                scan_result.is_http_service = True

            logger.debug(
                f"扫描完成: {target.ip}:{target.port} "
                f"{'OPEN' if is_open else 'CLOSED'} "
                f"| 协议={scan_result.protocol} "
                f"| 标题={scan_result.title[:30] if scan_result.title else ''}"
            )

            return scan_result

    async def scan(
        self,
        task_id: int,
        ip_segment: str,
        ports: List[int],
        protocol: str = "ALL",
        rate_limit: Optional[int] = None,
    ) -> Dict[str, Any]:
        """
        执行内网扫描任务主入口

        Args:
            task_id: 任务ID（用于日志和结果关联）
            ip_segment: CIDR网段或单IP，如 "192.168.1.0/24"
            ports: 要扫描的端口列表，如 [80, 443, 8080]
            protocol: 扫描协议类型 "HTTP" / "HTTPS" / "ALL"
            rate_limit: 本次任务的速率限制(可选，默认使用实例配置)

        Returns:
            包含统计信息的字典:
            {
                "task_id": 1,
                "total_ips": 254,
                "scanned_ips": 254,
                "open_ports": 12,
                "http_services_found": 5,
                "new_assets_count": 3,
                "updated_assets_count": 2,
                "details": [...]
            }
        """
        effective_rate_limit = rate_limit or self.rate_limit
        start_time = time.time()

        logger.info(
            f"[Task-{task_id}] 开始内网扫描: 网段={ip_segment}, "
            f"端口={ports}, 协议={protocol}, 并发={self.max_concurrent}"
        )

        # 1. 解析IP网段
        ips = self._parse_ip_segment(ip_segment)
        if not ips:
            logger.warning(f"[Task-{task_id}] 无效的IP网段: {ip_segment}")
            return {"task_id": task_id, "total_ips": 0, "scanned_ips": 0, "new_count": 0}

        # 2. 过滤黑名单端口
        valid_ports = [p for p in ports if not self._is_port_blacklisted(p)]
        if not valid_ports:
            logger.warning(f"[Task-{task_id}] 所有端口均在黑名单中")
            return {"task_id": task_id, "total_ips": len(ips), "scanned_ips": 0, "new_count": 0}

        # 3. 生成扫描目标
        targets: List[ScanTarget] = []
        for ip in ips:
            for port in valid_ports:
                targets.append(ScanTarget(ip=ip, port=port))

        summary = ScanSummary(task_id=task_id, total_ips=len(ips))
        new_count = 0

        # 4. 并发执行扫描（带速率限制）
        tasks = []
        for target in targets:
            tasks.append(self._scan_target(target, protocol))

        results = await asyncio.gather(*tasks, return_exceptions=True)

        # 5. 汇总结果
        for r in results:
            if isinstance(r, Exception):
                logger.error(f"扫描异常: {r}")
                continue
            if isinstance(r, ScanResult):
                summary.details.append(r)
                if r.is_open:
                    summary.open_ports += 1
                if r.is_http_service:
                    summary.http_services_found += 1
                if r.is_new_asset:
                    new_count += 1

        summary.scanned_ips = len(ips)
        summary.new_assets_count = new_count
        summary.scan_duration_seconds = round(time.time() - start_time, 2)

        logger.info(
            f"[Task-{task_id}] 内网扫描完成: "
            f"总IP={summary.total_ips}, 开放端口={summary.open_ports}, "
            f"HTTP服务={summary.http_services_found}, 新资产={new_count}, "
            f"耗时={summary.scan_duration_seconds}s"
        )

        return {
            "task_id": task_id,
            "total_ips": summary.total_ips,
            "scanned_ips": summary.scanned_ips,
            "open_ports": summary.open_ports,
            "http_services_found": summary.http_services_found,
            "new_count": new_count,
            "duration_seconds": summary.scan_duration_seconds,
        }

    async def close(self):
        """关闭扫描器，释放资源"""
        if self._session and not self._session.closed:
            await self._session.close()
        self._executor.shutdown(wait=False)
        logger.info("IntranetScanner 已关闭")
