"""
公网备案域名资产探测引擎
========================
功能：
- 根据企业名称查询备案域名列表
- 自动生成 HTTP/HTTPS URL
- 探测 URL 可访问性、状态码、页面标题
- 去重后写入待确认资产池

数据来源：
1. 第三方备案查询 API 接口
2. 备案信息数据库(如需本地部署)
"""

import asyncio
import logging
import re
import time
from typing import List, Dict, Any, Optional
from dataclasses import dataclass

import aiohttp
from bs4 import BeautifulSoup


logger = logging.getLogger(__name__)


@dataclass
class ICPDomain:
    """备案域名信息"""
    domain: str
    icp_number: str = ""
    company_name: str = ""
    home_url: str = ""
    ip_address: str = ""


@dataclass
class URLProbeResult:
    """URL探测结果"""
    url: str
    protocol: str  # http / https
    domain: str
    status_code: int = 0
    title: str = ""
    response_time_ms: int = 0
    is_accessible: bool = False
    error: str = ""


class PublicICPCrawler:
    """
    公网备案域名探测爬虫
    
    使用方式：
        crawler = PublicICPCrawler(api_key="xxx", api_url="https://api.example.com")
        result = await crawler.crawl("腾讯科技", task_id=123)
    """

    def __init__(self, api_key: str = "", api_url: str = "", timeout: int = 10):
        self.api_key = api_key
        self.api_url = api_url.rstrip("/")
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession(timeout=self.timeout)
        return self._session

    async def close(self):
        if self._session and not self._session.closed:
            await self._session.close()

    async def crawl(self, company_name: str, task_id: int) -> Dict[str, Any]:
        """
        执行完整的公网备案探测流程
        
        Args:
            company_name: 企业/公司名称
            task_id: 关联的任务ID
        
        Returns:
            包含发现域名、URL探测结果的字典
        """
        start_time = time.time()
        logger.info(f"[Task-{task_id}] 开始查询企业备案: {company_name}")

        # Step 1: 查询备案域名
        domains = await self.query_icp_domains(company_name)
        
        if not domains:
            logger.warning(f"[Task-{task_id}] 未找到企业 {company_name} 的备案域名")
            return {
                "task_id": task_id,
                "company": company_name,
                "domains": [],
                "urls_found": 0,
                "new_assets": 0,
                "message": "未找到备案域名",
            }

        # Step 2: 生成标准URL并探测
        urls_to_probe = []
        for d in domains:
            urls_to_probe.append(f"https://{d.domain}")
            urls_to_probe.append(f"http://{d.domain}")

        # Step 3: 并发探测所有URL
        probe_results = await self.batch_probe_urls(urls_to_probe, max_concurrent=20)

        # Step 4: 筛选有效结果并去重
        valid_results = [r for r in probe_results if r.is_accessible]
        unique_urls = {}
        for r in valid_results:
            key = (r.url.lower(), r.protocol)
            if key not in unique_urls:
                unique_urls[key] = r

        elapsed = time.time() - start_time
        new_count = len(unique_urls)

        logger.info(
            f"[Task-{task_id}] 公网备案探测完成: "
            f"企业={company_name}, 备案域名={len(domains)}, "
            f"有效URL={new_count}, 耗时={elapsed:.1f}s"
        )

        # TODO: 将结果写入 MySQL 数据库的 web_asset 表和 detect_record_company 表
        # await self.save_results(task_id, company_name, domains, list(unique_urls.values()))

        return {
            "task_id": task_id,
            "company": company_name,
            "icp_domains": [
                {"domain": d.domain, "icp": d.icp_number, "ip": d.ip_address}
                for d in domains
            ],
            "urls_found": len(valid_results),
            "new_assets": new_count,
            "probe_details": [
                {
                    "url": r.url,
                    "status_code": r.status_code,
                    "title": r.title,
                    "response_time_ms": r.response_time_ms,
                }
                for r in list(unique_urls.values())
            ],
            "elapsed_seconds": round(elapsed, 2),
        }

    async def query_icp_domains(self, company_name: str) -> List[ICPDomain]:
        """
        查询企业名下的备案域名列表
        
        支持的数据源优先级：
        1. 配置的第三方 API 接口
        2. 本地模拟数据(开发测试用)
        """
        if self.api_key and self.api_url:
            return await self._query_from_api(company_name)
        else:
            return self._mock_icp_data(company_name)

    async def _query_from_api(self, company_name: str) -> List[ICPDomain]:
        """调用第三方备案查询API"""
        session = await self._get_session()
        
        try:
            params = {
                "key": self.api_key,
                "company": company_name,
            }
            async with session.get(f"{self.api_url}/query", params=params) as resp:
                data = await resp.json()
                
                domains = []
                for item in data.get("data", []):
                    domains.append(ICPDomain(
                        domain=item.get("domain", ""),
                        icp_number=item.get("icp", ""),
                        company_name=item.get("company", ""),
                        ip_address=item.get("ip", ""),
                    ))
                return domains
                
        except Exception as e:
            logger.error(f"备案API调用失败: {e}")
            return []

    def _mock_icp_data(self, company_name: str) -> List[ICPDomain]:
        """
        模拟备案数据(开发环境使用)
        
        实际生产环境请替换为真实的备案查询接口
        """
        mock_domains = {
            "腾讯": ["qq.com", "tencent.com", "weixin.com", "wechat.com"],
            "阿里巴巴": ["alibaba.com", "aliyun.com", "taobao.com", "tmall.com"],
            "百度": ["baidu.com", "baidupcs.com", "bce.baidu.com"],
            "字节跳动": ["bytedance.com", "douyin.com", "toutiao.com"],
            "华为": ["huawei.com", "huaweicloud.com", "harmonyos.com"],
            "Example Corp": ["example.com", "www.example.com", "test.example.com"],
        }
        
        found = []
        for company, domain_list in mock_domains.items():
            if company in company_name or company_name in company:
                for d in domain_list:
                    found.append(ICPDomain(
                        domain=d,
                        icp_number=f"京ICP备{hash(d)%100000000}号",
                        company_name=company,
                        ip_address=f"93.184.{hash(d)%256}.{hash(d*2)%256}",
                    ))
        
        return found

    async def batch_probe_urls(self, urls: List[str], max_concurrent: int = 20) -> List[URLProbeResult]:
        """
        批量并发探测URL可访问性
        
        Args:
            urls: 待探测的URL列表
            max_concurrent: 最大并发数
        
        Returns:
            探测结果列表
        """
        semaphore = asyncio.Semaphore(max_concurrent)
        tasks = [self._probe_single_url(u, semaphore) for u in urls]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        final_results = []
        for r in results:
            if isinstance(r, URLProbeResult):
                final_results.append(r)
            elif isinstance(r, Exception):
                logger.error(f"URL探测异常: {r}")
        
        return final_results

    async def _probe_single_url(self, url: str, semaphore: asyncio.Semaphore) -> URLProbeResult:
        """探测单个URL"""
        protocol = "https" if url.startswith("https") else "http"
        result = URLProbeResult(url=url, protocol=protocol, domain=self._extract_domain(url))

        async with semaphore:
            session = await self._get_session()
            
            try:
                start = time.time()
                headers = {
                    "User-Agent": "Mozilla/5.0 CertMonitor/1.0 (SSL Certificate Monitor)",
                    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
                    "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
                }
                
                timeout = aiohttp.ClientTimeout(total=self.timeout.total or 10)
                async with session.get(url, headers=headers, timeout=timeout, 
                                       allow_redirects=True, ssl=False) as resp:
                    
                    elapsed_ms = int((time.time() - start) * 1000)
                    result.status_code = resp.status
                    result.response_time_ms = elapsed_ms
                    result.is_accessible = 200 <= resp.status < 400
                    
                    try:
                        html = await resp.text(errors="ignore")
                        result.title = self._extract_title(html)
                    except:
                        result.title = ""

            except asyncio.TimeoutError:
                result.error = f"连接超时({self.timeout.total}s)"
            except aiohttp.ClientError as e:
                result.error = f"连接错误: {str(e)}"
            except Exception as e:
                result.error = f"未知错误: {str(e)}"
        
        return result

    @staticmethod
    def _extract_domain(url: str) -> str:
        """从URL中提取域名"""
        match = re.match(r"https?://([^/:]+)", url)
        return match.group(1) if match else url

    @staticmethod
    def _extract_title(html: str) -> str:
        """从HTML中提取网页标题"""
        try:
            soup = BeautifulSoup(html, 'lxml')
            title = soup.find('title')
            if title and title.string:
                return title.string.strip()[:200]
        except:
            pass
        return ""
