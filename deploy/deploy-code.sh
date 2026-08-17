#!/usr/bin/env bash
# VPS 端代码部署：解压 repo.tar.gz → 复用 node_modules → build → rsync public
# 用法（VPS 上）：bash /home/studio/app/deploy-code.sh
set -euo pipefail

APP=/home/studio/app
NEW=$APP/repo.new
OLD=$APP/repo
KEEP_PATTERN='node_modules|.git|dist'

echo "[1/5] 备份当前 repo -> repo.old.*"
TS=$(date +%s)
if [ -d "$OLD" ]; then
    mv "$OLD" "$APP/repo.old.$TS"
fi

echo "[2/5] 解压 repo.tar.gz -> $NEW"
mkdir -p "$NEW"
tar -xzf $APP/repo.tar.gz -C "$NEW"

echo "[3/5] 复用 node_modules（从最近的 repo.old.* 软链）"
# 注意：之前用 symlink 复用 node_modules 会在清理旧 backup 时链断裂
# 这里 fallback 到 npm install（首次或链断时），后续保留 fast path
PREV=$(ls -dt $APP/repo.old.* 2>/dev/null | head -1 || true)
LINKED=0
if [ -n "$PREV" ] && [ -d "$PREV/node_modules" ]; then
    # 解析到最终 target（symlink 链），验证是真实目录
    TARGET=$(readlink -f "$PREV/node_modules" 2>/dev/null || true)
    if [ -n "$TARGET" ] && [ -d "$TARGET" ]; then
        ln -s "$TARGET" "$NEW/node_modules"
        echo "    linked node_modules from $PREV -> $TARGET"
        LINKED=1
    fi
fi
if [ "$LINKED" -eq 0 ]; then
    echo "    [warn] no valid previous node_modules; running npm install (约 30-40s)"
    cd "$NEW"
    npm install --registry=https://registry.npmmirror.com --no-audit --no-fund 2>&1 | tail -3
fi

echo "[4/5] build（VAULT_ROOT=/home/studio/workbench）"
cd "$NEW"
VAULT_ROOT=/home/studio/workbench npm run build 2>&1 | tail -20

echo "[5/5] 切到新版本 + 部署 dist -> public"
# 把 NEW 替换为 repo（保证 post-receive 钩子找得到）
rm -rf "$OLD"
mv "$NEW" "$OLD"
rsync -a --delete "$OLD/dist/" "$APP/public/"

# 清理老的备份（保留最近 2 个）
ls -dt $APP/repo.old.* 2>/dev/null | tail -n +3 | while read d; do
    echo "    removing old backup: $d"
    rm -rf "$d"
done

echo "================================================"
echo "  deployed OK at $(date -Iseconds)"
echo "================================================"
