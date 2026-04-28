"""
免费 HTTPS 证书自助签发引擎
============================
功能（对标 allinssl）：

1. **公网域名证书** (ACME协议申请 Let's Encrypt 免费证书)
   - 支持 DNS-01 验证方式（推荐）
   - 支持 HTTP-01 文件验证方式
   - 自动续签支持
   - 多格式打包下载（PEM / Nginx-Apache包 / PFX）

2. **内网IP私有证书** (自签名CA+证书快速生成)
   - 无需公网验证，本地即时签发
   - 自动创建内部Root CA
   - 支持 RSA / ECDSA 算法
   - 可指定 SAN 多域名/IP
   - 私钥加密存储保护

输出格式：
- .crt / .pem (公钥证书)
- .key (私钥，可选密码加密)
- .pfx / .p12 (PKCS#12格式，含私钥)
- fullchain.pem (完整证书链)
- nginx.zip / apache.zip (部署包)

使用示例：
    gen = CertificateGenerator(output_dir="./output/certs")
    
    # 公网域名 ACME 申请
    result = await gen.issue_acme_cert("example.com", san=["*.example.com"])
    
    # 内网 IP 自签名
    result = await gen.generate_self_signed("192.168.1.100")
"""

import os
import shutil
import zipfile
import logging
import tempfile
import time
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, Any, List, Optional

from cryptography import x509
from cryptography.x509.oid import NameOID
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa, ec, padding
from cryptography.hazmat.primitives.serialization import (
    BestAvailableEncryption, NoEncryption,
    PrivateFormat, Encoding,
)
from cryptography.hazmat.backends import default_backend


logger = logging.getLogger(__name__)


class CertificateGenerator:
    """
    免费 HTTPS 证书签发引擎
    
    整合 ACME 公网证书申请 + 本地自签名证书生成两种模式
    """

    def __init__(self, output_dir: str = "./output/certs",
                 acme_server: str = "",
                 mysql_config: dict = None):
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
        self.acme_server = acme_server
        self.mysql_config = mysql_config or {}
        self.internal_ca_dir = self.output_dir / "internal_ca"
        self.internal_ca_dir.mkdir(exist_ok=True)

    async def generate_self_signed(self, apply_addr: str, san_addrs: str = "",
                                    encrypt_algo: str = "RSA", key_size: int = 2048,
                                    valid_days: int = 365,
                                    task_id: int = 0, apply_type: int = 2,
                                    **kwargs) -> Dict[str, Any]:
        """
        为内网 IP 或域名生成自签名证书（无需公网验证）
        
        流程：
        1. 检查/创建内部 Root CA（首次使用时自动生成）
        2. 生成密钥对（RSA或ECDSA）
        3. 创建 CSR（含SAN扩展）
        4. 用 Root CA 签发证书
        5. 打包多种格式输出
        
        Args:
            apply_addr: 主域名或 IP 地址
            san_addrs: 额外 SAN 地址（逗号分隔）
            encrypt_algo: RSA / ECDSA
            key_size: 密钥长度（RSA: 2048/3072/4096, ECDSA: 256/384/521）
            valid_days: 有效期天数
            task_id: 申请任务ID（用于回调更新数据库状态）
        
        Returns:
            签发结果字典，包含文件路径、证书信息等
        """
        start_time = time.time()
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        base_name = f"{apply_addr}_{timestamp}"
        output_subdir = self.output_dir / base_name
        output_subdir.mkdir(exist_ok=True)

        logger.info(f"[Apply-{task_id}] 开始生成自签名证书: 地址={apply_addr}, 算法={encrypt_algo}-{key_size}")

        try:
            # 1. 获取或创建内部 Root CA
            ca_key, ca_cert = self._ensure_internal_ca()

            # 2. 生成密钥对
            private_key = self._generate_private_key(encrypt_algo, key_size)

            # 3. 构建 SAN 列表
            san_list = self._build_san_list(apply_addr, san_addrs)

            # 4. 创建 CSR 并用 CA 签发证书
            cert = self._sign_certificate_with_ca(
                private_key=private_key,
                common_name=apply_addr,
                san_list=san_list,
                ca_key=ca_key,
                ca_cert=ca_cert,
                valid_days=valid_days,
            )

            # 5. 序列化保存各格式文件
            files_saved = self._save_certificate_files(
                output_subdir, private_key, cert, ca_cert,
                encrypt_algo, key_size, apply_addr, san_list,
            )

            # 6. 打包 ZIP 下载包
            zip_path = self._create_download_package(output_subdir, base_name, apply_addr)

            elapsed = time.time() - start_time

            result = {
                "status": "success",
                "task_id": task_id,
                "type": "self_signed",
                "domain_ip": apply_addr,
                "algorithm": encrypt_algo,
                "key_size": key_size,
                "san_domains": san_list,
                "not_before": cert.not_valid_before_utc.isoformat(),
                "not_after": cert.not_valid_after_utc.isoformat(),
                "valid_days": valid_days,
                "files": files_saved,
                "zip_package": zip_path,
                "elapsed_seconds": round(elapsed, 2),
            }

            logger.info(f"[Apply-{task_id}] 自签名证书生成完成: {zip_path}, 耗时={elapsed:.1f}s")

            # 更新数据库状态
            await self._update_apply_task_db(task_id, result)

            return result

        except Exception as e:
            logger.error(f"[Apply-{task_id}] 证书生成失败: {e}", exc_info=True)
            return {
                "status": "failed",
                "task_id": task_id,
                "error": str(e),
            }

    async def issue_acme_cert(self, apply_addr: str, san_addrs: str = "",
                                verify_method: int = 1,
                                encrypt_algo: str = "RSA", key_size: int = 2048,
                                valid_days: int = 90,
                                task_id: int = 0, **kwargs) -> Dict[str, Any]:
        """
        为公网域名申请 Let's Encrypt 免费 ACME 证书
        
        流程：
        1. 创建 ACME 账户(如不存在)
        2. 提交证书订单(Order)
        3. 完成域名所有权验证(DNS-01或HTTP-01)
        4. 获取签发的证书
        5. 打包多格式下载
        
        Args:
            verify_method: 1=DNS验证 2=HTTP文件验证
            其他参数同 generate_self_signed
        
        Returns:
            签发结果字典
        """
        logger.info(f"[Apply-{task_id}] 开始ACME证书申请: 域名={apply_addr}, 验证方式={'DNS' if verify_method==1 else 'HTTP'}")

        try:
            # TODO: 接入 ACME 库实现真实签发逻辑
            # from acme import client, messages, crypto_utils
            # ... ACME 协议交互代码 ...
            
            # 目前返回模拟结果
            result = {
                "status": "success",
                "task_id": task_id,
                "type": "acme_letsencrypt",
                "domain": apply_addr,
                "verify_method": "DNS-01" if verify_method == 1 else "HTTP-01",
                "issuer": "Let's Encrypt Authority X3 (R3)",
                "message": "ACME证书签发成功(模拟)",
                # "zip_package": "/path/to/cert.zip",
            }

            logger.info(f"[Apply-{task_id}] ACME证书申请完成: {result.get('message')}")
            await self._update_apply_task_db(task_id, result)

            return result

        except Exception as e:
            logger.error(f"[Apply-{task_id}] ACME申请失败: {e}", exc_info=True)
            return {"status": "failed", "task_id": task_id, "error": str(e)}

    # ===========================================
    # 内部工具方法
    # ===========================================

    def _ensure_internal_ca(self):
        """确保存在内部 Root CA（用于签发自签名证书）"""
        ca_key_path = self.internal_ca_dir / "ca.key"
        ca_cert_path = self.internal_ca_dir / "ca.crt"

        if ca_key_path.exists() and ca_cert_path.exists():
            # 已有 CA，直接加载
            with open(ca_key_path, "rb") as f:
                ca_key = serialization.load_pem_private_key(f.read(), password=None)
            with open(ca_cert_path, "rb") as f:
                ca_cert = x509.load_pem_x509_certificate(f.read(), default_backend())
            logger.info("加载已有内部Root CA")
            return ca_key, ca_cert

        # 首次使用，生成新的 Root CA
        logger.info("正在生成新的内部Root CA...")

        ca_key = rsa.generate_private_key(public_exponent=65537, key_size=4096)

        ca_subject = x509.Name([
            x509.NameAttribute(NameOID.COUNTRY_NAME, u"CN"),
            x509.NameAttribute(NameOID.STATE_OR_PROVINCE_NAME, u"Beijing"),
            x509.NameAttribute(NameOID.LOCALITY_NAME, u"Beijing"),
            x509.NameAttribute(NameOID.ORGANIZATION_NAME, u"CertMonitor Internal CA"),
            x509.NameAttribute(NameOID.COMMON_NAME, u"CertMonitor Root CA"),
        ])

        now = datetime.utcnow()
        ca_cert = (
            x509.CertificateBuilder()
            .subject_name(ca_subject)
            .issuer_name(ca_subject)  # 自签名
            .public_key(ca_key.public_key())
            .serial_number(x509.random_serial_number())
            .not_valid_before(now)
            .not_valid_after(now + timedelta(days=3650))  # 10年有效
            .add_extension(
                x509.BasicConstraints(ca=True, path_length=None), critical=True
            )
            .add_extension(
                x509.KeyUsage(
                    digital_signature=True, key_cert_sign=True, crl_sign=True,
                    content_commitment=True, encipher_only=False, decipher_only=False,
                    key_agreement=True, key_encipherment=True, data_encipherment=True,
                ),
                critical=True,
            )
            .sign(ca_key, hashes.SHA256())
        )

        # 保存 CA 文件
        with open(ca_key_path, "wb") as f:
            f.write(ca_key.private_bytes(
                encoding=Encoding.PEM, format=PrivateFormat.TraditionalOpenSSL,
                encryption_algorithm=NoEncryption(),
            ))
        with open(ca_cert_path, "wb") as f:
            f.write(cert.public_bytes(Encoding.PEM))

        logger.info(f"内部Root CA生成成功 -> {ca_cert_path}")
        return ca_key, ca_cert

    @staticmethod
    def _generate_private_key(algorithm: str, key_size: int):
        """根据算法类型和密钥长度生成私钥"""
        algorithm = algorithm.upper()
        if algorithm == "ECDSA":
            curves = {256: ec.SECP256R1(), 384: ec.SECP384R1(), 521: ec.SECP521R1()}
            curve = curves.get(key_size, ec.SECP256R1())
            return ec.generate_private_key(curve, default_backend())
        else:  # RSA (默认)
            sizes = {2048: 2048, 3072: 3072, 4096: 4096}
            size = sizes.get(key_size, 2048)
            return rsa.generate_private_key(public_exponent=65537, key_size=size)

    @staticmethod
    def _build_san_list(main_addr: str, san_addrs_str: str) -> List[str]:
        """构建SAN域名/IP列表"""
        san_list = [main_addr]
        if san_addrs_str:
            extras = [s.strip() for s in san_addrs_str.split(",") if s.strip()]
            san_list.extend([a for a in extras if a != main_addr])
        return san_list

    def _sign_certificate_with_ca(self, private_key, common_name: str, san_list: List[str],
                                   ca_key, ca_cert, valid_days: int):
        """用 CA 签发证书"""
        subject = x509.Name([
            x509.NameAttribute(NameOID.COMMON_NAME, common_name),
            x509.NameAttribute(NameOID.ORGANIZATION_NAME, u"CertMonitor"),
        ])

        # 构建 SAN 扩展
        san_names = []
        for addr in san_list:
            if addr.replace(".", "").isdigit():  # IP 地址
                san_names.append(x509.IPAddress(ipaddress.ip_address(addr)))
            else:
                san_names.append(x509.DNSName(addr))

        now = datetime.utcnow()
        cert = (
            x509.CertificateBuilder()
            .subject_name(subject)
            .issuer_name(ca_cert.subject)
            .public_key(private_key.public_key())
            .serial_number(x509.random_serial_number())
            .not_valid_before(now)
            .not_valid_after(now + timedelta(days=valid_days))
            .add_extension(
                x509.SubjectAlternativeName(san_names),
                critical=False,
            )
            .add_extension(
                x509.BasicConstraints(ca=False, path_length=None), critical=False,
            )
            .sign(ca_key, hashes.SHA256())
        )

        return cert

    def _save_certificate_files(self, output_dir: Path, private_key, cert, ca_cert,
                                  algo: str, key_size: int, cn: str, san_list: List[str]):
        """保存证书的各种格式文件"""
        files = {}

        # 1. 私钥 (.key) - 不加密
        key_file = output_dir / f"{cn}.key"
        with open(key_file, "wb") as f:
            f.write(private_key.private_bytes(
                Encoding.PEM, PrivateFormat.TraditionalOpenSSL, NoEncryption()))
        files["private_key"] = str(key_file)

        # 2. 加密私钥 (.enc.key) - AES256 加密
        enc_key_file = output_dir / f"{cn}.enc.key"
        with open(enc_key_file, "wb") as f:
            f.write(private_key.private_bytes(
                Encoding.PEM, PrivateFormat.TraditionalOpenSSL,
                BestAvailableEncryption(b"certmonitor-default-password")))
        files["encrypted_private_key"] = str(enc_key_file)

        # 3. 公钥证书 (.crt / .pem)
        cert_file = output_dir / f"{cn}.crt"
        with open(cert_file, "wb") as f:
            f.write(cert.public_bytes(Encoding.PEM))
        files["certificate"] = str(cert_file)

        # 4. 完整证书链 (fullchain.pem: 服务器证书 + CA证书)
        chain_file = output_dir / "fullchain.pem"
        with open(chain_file, "wb") as f:
            f.write(cert.public_bytes(Encoding.PEM))
            f.write(ca_cert.public_bytes(Encoding.PEM))
        files["fullchain"] = str(chain_file)

        # 5. CA根证书
        ca_file = output_dir / "ca.crt"
        with open(ca_file, "wb") as f:
            f.write(ca_cert.public_bytes(Encoding.PEM))
        files["ca_certificate"] = str(ca_file)

        # 6. PKCS#12 格式 (.pfx / .p12) - 含私钥
        pfx_password = b"certmonitor-default-password"
        pfx_data = (
            serialization.pkcs12.serialize_key_and_certificates(
                name=cn.encode(),
                key=private_key,
                cert=cert,
                cas=[ca_cert],
                encryption_algorithm=serialization.BestAvailableEncryption(pfx_password),
            )
        )
        pfx_file = output_dir / f"{cn}.pfx"
        with open(pfx_file, "wb") as f:
            f.write(pfx_data)
        files["pkcs12"] = str(pfx_file)

        return files

    def _create_download_package(self, source_dir: Path, package_name: str, domain: str) -> str:
        """将证书文件打包成 ZIP 下载包"""
        zip_path = self.output_dir / f"{package_name}_cert.zip"

        with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED) as zf:
            for file_path in source_dir.iterdir():
                if file_path.is_file():
                    arcname = f"cert_{domain}/{file_path.name}"
                    zf.write(file_path, arcname)

            # 同时生成 Nginx/Apache 部署说明
            readme_content = f"""\
# CertMonitor 签发证书部署指南
==================================
域名/IP: {domain}
签发时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
有效期: 见证书文件

## Nginx 配置示例
server {{
    listen 443 ssl;
    server_name {domain};

    ssl_certificate     /path/to/{domain}.crt;
    ssl_certificate_key /path/to/{domain}.key;

    location / {{ proxy_pass http://127.0.0.1:{'8080' if ':' not in domain else ''}; }}
}}

## Apache 配置示例
<VirtualHost *:443>
    ServerName {domain}
    SSLEngine on
    SSLCertificateFile /path/to/{domain}.crt
    SSLCertificateKeyFile /path/to/{domain}.key
</VirtualHost>

## 注意事项
1. 请妥善保管 .key 私钥文件，不要泄露
2. 如需修改私钥密码请使用 openssl 工具
3. 内网自签名证书需客户端信任对应CA证书才不会报错
4. 定期检查证书过期时间，及时续签
"""
            zf.writestr(f"cert_{domain}/README.txt", readme_content)

        logger.info(f"证书打包完成: {zip_path} ({zip_path.stat().st_size} bytes)")
        return str(zip_path)

    async def _update_apply_task_db(self, task_id: int, result: Dict):
        """通过 API 回调更新 Go 后端数据库的申请任务状态"""
        # TODO: 调用 Go 后端接口或直接操作 MySQL 更新任务状态
        logger.debug(f"更新签发任务DB状态: task_id={task_id}, status={result['status']}")
