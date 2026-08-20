#!/usr/bin/env bash
# Install / update Jiangnan Workbase MCP on VPS.
# Expects these files already uploaded to /tmp/workbase-deploy/:
#   workbase-mcp
#   config.yaml
#   rebuild-blog.sh
#   jiangnan-workbase-mcp.service
#   post-receive-reindex.sh
set -euo pipefail

DEPLOY_SRC=${1:-/tmp/workbase-deploy}
WB=/home/studio/workbase
HOOK=/home/studio/vault.git/hooks/post-receive
MARKER='# >>> workbase-mcp reindex >>>'

echo "[1/6] create directories"
sudo mkdir -p "$WB/bin" "$WB/index" "$WB/proposals" "$WB/inbox"
sudo mkdir -p /home/studio/workbench/Workbase
sudo chown -R ubuntu:ubuntu "$WB"
sudo chown ubuntu:ubuntu /home/studio/workbench/Workbase || true

echo "[2/6] install binary + scripts"
install -m 0755 "$DEPLOY_SRC/workbase-mcp" "$WB/bin/workbase-mcp"
install -m 0755 "$DEPLOY_SRC/rebuild-blog.sh" "$WB/bin/rebuild-blog.sh"
install -m 0600 "$DEPLOY_SRC/config.yaml" "$WB/config.yaml"

echo "[3/6] install systemd unit"
sudo install -m 0644 "$DEPLOY_SRC/jiangnan-workbase-mcp.service" /etc/systemd/system/jiangnan-workbase-mcp.service
sudo systemctl daemon-reload
sudo systemctl enable jiangnan-workbase-mcp
sudo systemctl restart jiangnan-workbase-mcp

echo "[4/6] wait for health"
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf --max-time 2 -X POST http://127.0.0.1:8787/internal/reindex >/tmp/workbase-reindex.json; then
    echo "reindex ok: $(cat /tmp/workbase-reindex.json)"
    break
  fi
  sleep 1
  if [ "$i" = "10" ]; then
    echo "WARN: reindex not ready yet"
    sudo systemctl --no-pager --full status jiangnan-workbase-mcp || true
    sudo journalctl -u jiangnan-workbase-mcp -n 40 --no-pager || true
  fi
done

echo "[5/6] patch post-receive for reindex"
if grep -q "$MARKER" "$HOOK"; then
  echo "post-receive already patched"
else
  {
    echo ""
    echo "$MARKER"
    cat "$DEPLOY_SRC/post-receive-reindex.sh"
    echo "# <<< workbase-mcp reindex <<<"
  } >> "$HOOK"
  chmod +x "$HOOK"
  echo "post-receive patched"
fi

echo "[6/6] ownership"
sudo chown -R ubuntu:ubuntu "$WB"
# allow MCP apply commits into bare repo
sudo chown -R ubuntu:ubuntu /home/studio/vault.git /home/studio/workbench

echo "DONE"
sudo systemctl --no-pager --full status jiangnan-workbase-mcp | sed -n '1,20p'
ss -lntp | grep -E ':(8787|8788)\b' || true
