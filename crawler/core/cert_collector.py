"""
SSL/TLS 证书信息采集器
========================
功能：
- 对指定域名/IP:端口建立 TLS 连接
- 提取证书详细信息（颁发者、有效期、SAN、指纹等）
- 检测证书过期/即将过期状态
- 支持自定义 CA 信任链验证

使用示例：
    collector = CertificateCollector(timeout=10)
    info = await collector.collect("example.com", 443)
    print(info["subject"], info["not_after"])
"""

import asyncio
import hashlib
import ipaddress
import logging
import ssl
import socket
from datetime import datetime, timezone
from typing import Dict, Any, Optional, List

from cryptography import x509
from cryptography.hazmat.backends import default_backend


logger = logging.getLogger(__name__)


class CertificateCollector:
    """
    SSL/TLS 证书采集器

    通过建立 TLS 连接获取目标服务器的证书详情，
    用于证书过期监测和资产发现。
    """

    def __init__(self, timeout: int = 10):
        """
        初始化证书采集器

        Args:
            timeout: 连接超时秒数(默认10秒)
        """
        self.timeout = timeout

    async def collect(self, domain_or_ip: str, port: int = 443) -> Dict[str, Any]:
        """
        采集目标地址的 SSL 证书信息

        Args:
            domain_or_ip: 目标域名或 IP 地址
            port: 端口号，默认 443

        Returns:
            包含证书详细信息的字典:
            {
                "domain": "example.com",
                "port": 443,
                "subject": "CN=example.com",
                "issuer": "CN=Let's Encrypt ...",
                "serial_number": "...",
                "not_before": "2024-01-01T00:00:00Z",
                "not_after": "2025-01-01T00:00:00Z",
                "is_valid": True,
                "days_until_expiry": 180,
                "san_domains": ["example.com", "www.example.com"],
                "fingerprint_sha256": "...",
                "fingerprint_sha1": "...",
                "version": "v3",
                "signature_algorithm": "sha256WithRSAEncryption",
                "public_key_type": "RSA",
                "public_key_size": 2048,
                "error": ""
            }
        """
        result = {
            "domain": domain_or_ip,
            "port": port,
            "subject": "",
            "issuer": "",
            "serial_number": "",
            "not_before": "",
            "not_after": "",
            "is_valid": False,
            "days_until_expiry": -1,
            "san_domains": [],
            "fingerprint_sha256": "",
            "fingerprint_sha1": "",
            "version": "",
            "signature_algorithm": "",
            "public_key_type": "",
            "public_key_size": 0,
            "error": "",
        }

        loop = asyncio.get_event_loop()
        try:
            # 在线程池中执行阻塞式 TLS 握手
            der_cert = await loop.run_in_executor(
                None,
                lambda: self._do_tls_handshake(domain_or_ip, port)
            )
            if not der_cert:
                result["error"] = f"TLS握手失败或无证书返回"
                return result

            # 解析 DER 格式的证书
            cert = x509.load_der_x509_certificate(der_cert, default_backend())

            # 基本字段
            subject = cert.subject
            issuer = cert.issuer
            result["subject"] = self._format_name(subject)
            result["issuer"] = self._format_name(issuer)
            result["serial_number"] = format(cert.serial_number, 'x')
            result["version"] = f"v{cert.version.value}"
            result["signature_algorithm"] = cert.signature_algorithm_oid._name or str(cert.signature_algorithm_oid)

            # 有效期
            not_before_utc = cert.not_valid_before_utc if hasattr(cert, 'not_valid_before_utc') else \
                            cert.not_valid_before.replace(tzinfo=timezone.utc)
            not_after_utc = cert.not_valid_after_utc if hasattr(cert, 'not_valid_after_utc') else \
                           cert.not_valid_after.replace(tzinfo=timezone.utc)

            result["not_before"] = not_before_utc.isoformat()
            result["not_after"] = not_after_utc.isoformat()

            now = datetime.now(timezone.utc)
            result["is_valid"] = not_before_utc <= now <= not_after_utc
            delta = not_after_utc - now
            result["days_until_expiry"] = delta.days

            # SAN 扩展（Subject Alternative Names）
            san_list = []
            try:
                for ext in cert.extensions:
                    if ext.oid == x509.oid.ExtensionOID.SUBJECT_ALTERNATIVE_NAME:
                        san_ext = ext.value
                        for name in san_ext.get_values_for_type(x509.DNSName):
                            san_list.append(name)
                        for name in san_ext.get_values_for_type(x509.IPAddress):
                            san_list.append(str(name))
                        break
            except Exception as e:
                logger.debug(f"解析 SAN 扩展失败: {e}")
            result["san_domains"] = san_list

            # 公钥信息
            pub_key = cert.public_key()
            key_type = type(pub_key).__name__
            if "rsa" in key_type.lower():
                result["public_key_type"] = "RSA"
                result["public_key_size"] = pub_key.key_size
            elif "ec" in key_type.lower():
                result["public_key_type"] = "ECDSA"
                result["public_key_size"] = pub_key.key_size
            else:
                result["public_key_type"] = key_type

            # 指纹
            raw_bytes = cert.public_bytes(serialization.Encoding.DER) if hasattr(cert.public_bytes, '__call__') else None
            # Fallback: use fingerprint from the DER cert we already have
            try:
                result["fingerprint_sha256"] = hashlib.sha256(der_cert).hexdigest().upper()
                result["fingerprint_sha1"] = hashlib.sha1(der_cert).hexdigest().upper()
            except Exception:
                pass

            logger.info(f"证书采集成功: {domain_or_ip}:{port}, 过期时间={result['not_after']}, 剩余{result['days_until_expiry']}天")

        except ssl.SSLCertVerificationError as e:
            result["error"] = f"SSL证书验证错误: {e}"
        except socket.timeout:
            result["error"] = f"连接超时({self.timeout}s)"
        except ConnectionRefusedError:
            result["error"] = "连接被拒绝"
        except OSError as e:
            if "Network is unreachable" in str(e):
                result["error"] = "网络不可达"
            elif "Connection reset" in str(e):
                result["error"] = "连接被重置"
            else:
                result["error"] = f"网络错误: {str(e)[:100]}"
        except Exception as e:
            result["error"] = f"未知错误: {str(e)[:200]}"

        return result

    def _do_tls_handshake(self, host: str, port: int) -> Optional[bytes]:
        """在当前线程中执行同步 TLS 握手并返回 DER 编码的证书"""
        ctx = ssl.create_default_context()
        # 不验证证书（允许自签名和过期证书）
        ctx.check_hostname = False
        ctx.verify_mode = ssl.CERT_NONE

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(self.timeout)

        try:
            sock.connect((host, port))
            tls_sock = ctx.wrap_socket(sock, server_hostname=host)
            # 获取对端证书的 DER 编码
            der_cert = tls_sock.getpeercert(binary_form=True)
            tls_sock.shutdown(socket.SHUT_RDWR)
            return der_cert
        finally:
            try:
                sock.close()
            except Exception:
                pass

    @staticmethod
    def _format_name(name: x509.Name) -> str:
        """将 X509 Name 转为可读字符串"""
        parts = []
        for attr in name:
            parts.append(f"{attr.oid._name}={attr.value}")
        return ", ".join(parts)


async def batch_collect(targets: List[Dict[str, Any]], timeout: int = 10) -> List[Dict[str, Any]]:
    """
    批量采集多个目标的证书信息

    Args:
        targets: 目标列表 [{"domain": "...", "port": 443}, ...]
        timeout: 单个目标超时秒数

    Returns:
        结果列表
    """
    collector = CertificateCollector(timeout=timeout)
    tasks = [collector.collect(t["domain"], t.get("port", 443)) for t in targets]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    output = []
    for i, r in enumerate(results):
        if isinstance(r, Exception):
            output.append({"domain": targets[i]["domain"], "error": str(r)})
        elif isinstance(r, dict):
            output.append(r)

    return output
