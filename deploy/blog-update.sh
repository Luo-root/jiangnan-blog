#!/usr/bin/env bash
# ============================================================
#  博客代码更新（被 cron 调用或手动执行）
#  流程：git pull → npm ci → npm run build → rsync 到 public
#  失败时回退到上一版本（rsync 的 --delete 配合 --backup）
# ============================================================

set -euo pipefail

REPO_DIR="/home/studio/app/repo"
BUILD_DIR="/home/studio/app/build"
PUBLIC_DIR="/home/studio/app/public"
BACKUP_DIR="/home/studio/app/public.bak"
WORKBENCH_DIR="/home/studio/workbench"
LOG="/var/log/blog-deploy/update.log"

log() { echo "[$(date -Iseconds)] $*" >> "$LOG"; }

# 1) 拉取
cd "$REPO_DIR" || { log "FATAL: $REPO_DIR 不存在"; exit 1; }
git fetch origin main 2>>"$LOG" || { log "WARN: git fetch 失败（可能 VPS 无 GitHub 通路），跳过代码更新"; VAULT_FETCH_OK=0; }
VAULT_FETCH_OK=${VAULT_FETCH_OK:-1}
[ "$VAULT_FETCH_OK" = "0" ] && { log "工作台模式：仅 rebuild"; }
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main 2>/dev/null || echo "$LOCAL")
[ "$LOCAL" = "$REMOTE" ] && { log "no changes ($LOCAL)"; exit 0; }
log "update: $LOCAL -> $REMOTE"

# 2) 备份当前 public（失败可回滚）
if [ -d "$PUBLIC_DIR" ] && [ "$(ls -A "$PUBLIC_DIR" 2>/dev/null)" ]; then
    rm -rf "$BACKUP_DIR"
    cp -a "$PUBLIC_DIR" "$BACKUP_DIR"
fi

# 3) 拉代码 + 装依赖 + build（VAULT_ROOT 指向工作台）
git pull --ff-only origin main >> "$LOG" 2>&1 || { log "WARN: git pull 失败，保留旧版 build"; }
# devDependencies 必须装（vite / plugin-react 在 devDependencies）
NODE_ENV=development npm install --include=dev >> "$LOG" 2>&1 || npm install >> "$LOG" 2>&1
if [ -d "$WORKBENCH_DIR" ] && [ "$(ls -A "$WORKBENCH_DIR" 2>/dev/null)" ]; then
    VAULT_ROOT="$WORKBENCH_DIR" npm run build >> "$LOG" 2>&1
else
    npm run build >> "$LOG" 2>&1
fi

# 4) 同步到 public
if [ -d "$BUILD_DIR/dist" ]; then
    rsync -a --delete "$BUILD_DIR/dist/" "$PUBLIC_DIR/"
    log "deployed $REMOTE to public"
    rm -rf "$BACKUP_DIR"
else
    log "ERROR: dist/ not found after build, rolling back"
    if [ -d "$BACKUP_DIR" ]; then
        rm -rf "$PUBLIC_DIR"
        mv "$BACKUP_DIR" "$PUBLIC_DIR"
    fi
    exit 1
fi
