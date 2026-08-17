#!/usr/bin/env bash
# ============================================================
#  博客一键部署 (systemd + Caddy)
#  用法（VPS 上，root 权限）：
#     sudo bash setup-vps.sh
#
#  前置：
#    1. 域名 A 记录已指 VPS 公网 IP
#    2. 本地已 `ssh-copy-id -i studio.pem ubuntu@<VPS_IP>`
#    3. 已从 GitHub 拉取本仓库（手动或 clone）
# ============================================================

set -euo pipefail

# ---- 路径常量（按 user 确认：ubuntu 用户 / /home/studio/app 目录） ----
DEPLOY_USER="ubuntu"
APP_DIR="/home/studio/app"
REPO_DIR="$APP_DIR/repo"
DIST_DIR="$REPO_DIR/dist"
PUBLIC_DIR="$APP_DIR/public"
CADDYFILE="/etc/caddy/Caddyfile"
REPO_URL="https://github.com/Luo-root/jiangnan-blog.git"
BRANCH="main"
LOG_DIR="/var/log/blog-deploy"
WORKBENCH_DIR="/home/studio/workbench"
WORKBENCH_REPO="https://github.com/Luo-root/jiangnan-blog-vault.git"

# ---- 1. 基础依赖 ----
echo "[1/6] 安装基础依赖..."
apt-get update -qq
apt-get install -y -qq git curl rsync ca-certificates

# ---- 2. Node.js 22 (Vite 7 要求 20.19+ / 22.12+) ----
echo "[2/6] 安装 Node.js 22 (二进制)..."
export DEBIAN_FRONTEND=noninteractive
NEED_INSTALL=1
if command -v node >/dev/null 2>&1; then
    if [[ "$(node -v)" == v22.* ]]; then
        echo "    已装 node $(node -v)，跳过"
        NEED_INSTALL=0
    else
        echo "    当前 node $(node -v) 不是 22.x，强制重装"
    fi
fi
if [ "$NEED_INSTALL" = "1" ]; then
    NODE_VER=22.12.0
    cd /tmp
    curl -fsSLO "https://nodejs.org/dist/v${NODE_VER}/node-v${NODE_VER}-linux-x64.tar.xz"
    # 旧版改名（mv 不是 rm），让 tar 强制覆盖
    for f in node npm npx corepack; do
        [ -e /usr/local/bin/$f ] && mv -f /usr/local/bin/$f /usr/local/bin/${f}.old || true
    done
    [ -d /usr/local/include/node ] && mv -f /usr/local/include/node /usr/local/include/node.old || true
    [ -d /usr/local/lib/node_modules ] && mv -f /usr/local/lib/node_modules /usr/local/lib/node_modules.old || true
    [ -d /usr/local/share/doc/node ] && mv -f /usr/local/share/doc/node /usr/local/share/doc/node.old || true
    [ -e /usr/local/share/man/man1/node.1 ] && mv -f /usr/local/share/man/man1/node.1 /usr/local/share/man/man1/node.1.old || true
    tar -xJf "node-v${NODE_VER}-linux-x64.tar.xz" -C /usr/local --strip-components=1
    hash -r 2>/dev/null || true
    rm -f "node-v${NODE_VER}-linux-x64.tar.xz"
fi
echo "    node $(node -v) / npm $(npm -v)"

# ---- 3. 准备目录结构 + 解压工作台（内容源） ----
echo "[3/6] 准备目录结构 + 工作台..."
mkdir -p "$APP_DIR" "$PUBLIC_DIR" "$LOG_DIR" "$WORKBENCH_DIR"
chown -R "$DEPLOY_USER":"$DEPLOY_USER" "$APP_DIR" "$WORKBENCH_DIR"
chown -R "$DEPLOY_USER":"$DEPLOY_USER" "$LOG_DIR" 2>/dev/null || true

# 工作台内容（VAULT_ROOT 数据源）— 优先 /tmp/workbench.tar.gz
if [ -f /tmp/workbench.tar.gz ]; then
    rm -rf "$WORKBENCH_DIR"/*
    tar -xzf /tmp/workbench.tar.gz -C "$WORKBENCH_DIR"
    chown -R "$DEPLOY_USER":"$DEPLOY_USER" "$WORKBENCH_DIR"
    echo "    工作台已从 /tmp/workbench.tar.gz 解压到 $WORKBENCH_DIR"
fi

# ---- 4. 准备代码（优先 GitHub，失败 fallback tarball） ----
echo "[4/6] 准备代码..."
if [ -d "$REPO_DIR/.git" ]; then
    cd "$REPO_DIR"
    sudo -u "$DEPLOY_USER" git pull --ff-only origin "$BRANCH" 2>&1 || true
    cd "$REPO_DIR"
    sudo -u "$DEPLOY_USER" NODE_ENV=development npm install --include=dev
elif [ -f /tmp/blog.tar.gz ]; then
    echo "    本地 tarball 解压到 $REPO_DIR"
    mkdir -p "$REPO_DIR"
    tar -xzf /tmp/blog.tar.gz -C "$REPO_DIR"
    cd "$REPO_DIR"
    sudo -u "$DEPLOY_USER" NODE_ENV=development npm install --include=dev
elif timeout 90 sudo -u "$DEPLOY_USER" git clone --depth 1 -b "$BRANCH" "$REPO_URL" "$REPO_DIR" 2>/dev/null; then
    echo "    GitHub clone 成功"
    cd "$REPO_DIR"
    sudo -u "$DEPLOY_USER" NODE_ENV=development npm install --include=dev
else
    echo "ERROR: 既无 /tmp/blog.tar.gz 也连不上 GitHub，请先 scp 上传 tarball"
    exit 1
fi

# ---- 5. 安装并配置 Caddy ----
echo "[5/6] 安装 Caddy..."
if ! command -v caddy >/dev/null 2>&1; then
    apt-get install -y -qq debian-keyring debian-archive-keyring
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
    apt-get update -qq
    apt-get install -y -qq caddy
fi

# 写 Caddyfile（与 deploy/caddyfile 同步）
cat > "$CADDYFILE" <<EOF
# 遇见江楠 · 静态博客
# 自动 HTTPS (Let's Encrypt)
www.jiangnanstudio.cloud, jiangnanstudio.cloud {
    root * $PUBLIC_DIR
    encode gzip zstd
    try_files {path} {path}/ /index.html
    file_server
    header {
        Cache-Control "public, max-age=2592000"
    }
    @assets path /assets/*
    header @assets Cache-Control "public, max-age=2592000, immutable"
}
EOF

caddy validate --config "$CADDYFILE"
systemctl reload caddy || systemctl restart caddy
systemctl enable caddy

# ---- 6. 首次 build + 部署 ----
echo "[6/6] 首次 build + 部署..."
cd "$REPO_DIR"
if [ -d "$WORKBENCH_DIR" ] && [ "$(ls -A "$WORKBENCH_DIR" 2>/dev/null)" ]; then
    sudo -u "$DEPLOY_USER" env VAULT_ROOT="$WORKBENCH_DIR" npm run build
else
    echo "    警告：$WORKBENCH_DIR 为空，build 会用空 vault（页面会没文章）"
    sudo -u "$DEPLOY_USER" npm run build
fi
if [ -d "$DIST_DIR" ]; then
    rsync -a --delete "$DIST_DIR/" "$PUBLIC_DIR/"
    chown -R "$DEPLOY_USER":"$DEPLOY_USER" "$PUBLIC_DIR"
    echo "    部署完成: $(du -sh "$PUBLIC_DIR" 2>&1 | cut -f1)"
else
    echo "    build 失败：$DIST_DIR 不存在"
    exit 1
fi

# cron 定时检查 GitHub
cat > /etc/cron.d/blog-pull <<EOF
*/5 * * * * $DEPLOY_USER /home/studio/app/repo/deploy/blog-update.sh >> $LOG_DIR/update.log 2>&1
EOF
chmod 644 /etc/cron.d/blog-pull

echo
echo "================================================"
echo "  部署完成"
echo "  代码目录: $REPO_DIR"
echo "  构建产物: $DIST_DIR"
echo "  服务目录: $PUBLIC_DIR (Caddy root)"
echo "  日志目录: $LOG_DIR"
echo "  域名: jiangnanstudio.cloud / www.jiangnanstudio.cloud"
echo
echo "  Caddy 已自动申请 HTTPS 证书（前提：域名 A 记录已指本机）"
echo "  验证: systemctl status caddy"
echo "  查看: tail -f $LOG_DIR/update.log"
echo "================================================"
