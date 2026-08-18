# Append-only fragment for /home/studio/vault.git/hooks/post-receive
# Do NOT include shebang or `set -euo` here — this is inlined into an existing hook.
echo "[workbase] triggering MCP reindex..."
if curl -sf --max-time 30 "http://127.0.0.1:8787/internal/reindex" >/tmp/workbase-reindex.json 2>/tmp/workbase-reindex.err; then
  echo "[workbase] reindex ok: $(cat /tmp/workbase-reindex.json)"
else
  echo "[workbase] WARN: reindex failed (MCP may be down): $(cat /tmp/workbase-reindex.err 2>/dev/null || true)"
fi
