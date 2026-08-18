#!/usr/bin/env bash
# Rebuild public blog from current workbench checkout.
# Used by workbase.rebuild_cmd after MCP proposal apply.
set -euo pipefail

REPO="/home/studio/app/repo"
WORKBENCH="/home/studio/workbench"
PUBLIC="/home/studio/app/public"
LOG="/home/studio/app/deploy.log"

echo "[$(date -Iseconds)] workbase rebuild start" >> "$LOG"
cd "$REPO"
VAULT_ROOT="$WORKBENCH" npm run build >> "$LOG" 2>&1
rsync -a --delete "$REPO/dist/" "$PUBLIC/" >> "$LOG" 2>&1
echo "[$(date -Iseconds)] workbase rebuild OK" >> "$LOG"
