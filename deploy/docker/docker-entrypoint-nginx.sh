#!/bin/sh
# CertMonitor Nginx 启动脚本
# 如果不存在 SSL 证书，自动生成自签名证书（用于开发/测试环境）

SSL_DIR="/etc/nginx/ssl"
KEY_FILE="$SSL_DIR/privkey.pem"
CERT_FILE="$SSL_DIR/fullchain.pem"

if [ ! -f "$CERT_FILE" ] || [ ! -f "$KEY_FILE" ]; then
    echo "[Nginx] 未检测到 SSL 证书，正在生成自签名证书..."
    mkdir -p "$SSL_DIR"
    openssl req -x509 -nodes -days 3650 \
        -newkey rsa:2048 \
        -keyout "$KEY_FILE" \
        -out "$CERT_FILE" \
        -subj "/CN=localhost/O=CertMonitor/CN=Self-Signed" 2>/dev/null
    echo "[Nginx] 自签名证书已生成"
fi

exec /docker-entrypoint.sh nginx -g 'daemon off;'
