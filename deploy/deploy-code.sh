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
PREV=$(ls -dt $APP/repo.old.* 2>/dev/null | head -1 || true)
if [ -n "$PREV" ] && [ -d "$PREV/node_modules" ]; then
    ln -s "$PREV/node_modules" "$NEW/node_modules"
    echo "    linked node_modules from $PREV"
else
    echo "    [warn] no previous node_modules found, will run npm ci"
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
