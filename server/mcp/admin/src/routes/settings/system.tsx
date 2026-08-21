import { useEffect, useState, type ReactNode } from "react";
import { api } from "../../lib/api";
import { errText, useToast } from "../../components/toast";

type Health = {
  ok: boolean;
  now: string;
  uptime_sec: number;
  listen?: { mcp?: string; admin?: string };
  paths?: Record<string, string>;
  sqlite?: Record<string, number>;
  index?: Record<string, number>;
  disk?: { path?: string; free_bytes?: number; total_bytes?: number; error?: string };
  git_head?: string;
};

function fmtBytes(n?: number) {
  if (!n) return "0";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

export function SystemPage() {
  const toast = useToast();
  const [h, setH] = useState<Health | null>(null);
  const [checking, setChecking] = useState(false);
  const [fail, setFail] = useState("");
  const [sampled, setSampled] = useState("");
  const [flash, setFlash] = useState(false);

  async function load(manual = false) {
    setChecking(true);
    try {
      const got = await api<Health>("/api/system/health");
      setH(got);
      setFail("");
      setSampled(new Date().toLocaleTimeString());
      if (manual) {
        setFlash(true);
        window.setTimeout(() => setFlash(false), 1600);
        toast.success("健康检查已更新");
      }
    } catch (e) {
      setFail(errText(e));
      if (manual) toast.error(errText(e));
    } finally {
      setChecking(false);
    }
  }
  useEffect(() => {
    load();
    const t = window.setInterval(() => load(), 15000);
    return () => window.clearInterval(t);
  }, []);

  const ok = !fail && !!h?.ok;
  return (
    <section className="h-full overflow-auto p-6">
      <div className="flex items-end justify-between">
        <div>
          <h2 className="text-lg font-semibold text-ink-1">System 健康</h2>
          <p className="mt-1 text-xs text-ink-3">进页开始 15s 轮询，离开停止。灯、采样时间、按钮态三件套。</p>
        </div>
        <button disabled={checking} className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50" onClick={() => load(true)}>
          {checking ? "检查中…" : flash ? "已更新" : "刷新"}
        </button>
      </div>
      <div className="mt-4 flex items-center gap-2 text-sm">
        <span className={`inline-block h-3 w-3 rounded-full ${ok ? "breath-ok" : "breath-bad"}`} />
        <span className="font-semibold text-ink-1">{ok ? "健康" : "异常"}</span>
        {fail ? <span className="text-destructive">{fail}</span> : null}
        {sampled ? <span className="font-mono text-[11px] text-ink-3">采样 {sampled}</span> : null}
      </div>
      {!h ? <p className="mt-6 text-sm text-ink-4">{checking ? "检查中…" : "加载中…"}</p> : (
        <>
          <div className="mt-4 grid grid-cols-4 gap-3">
            <Stat label="状态" value={h.ok ? "ok" : "down"} />
            <Stat label="uptime" value={`${h.uptime_sec}s`} />
            <Stat label="MCP" value={h.listen?.mcp || "—"} />
            <Stat label="Admin" value={h.listen?.admin || "—"} />
          </div>
          <div className="mt-4 grid grid-cols-2 gap-3">
            <Card title="索引条数">
              {Object.entries(h.index || {}).map(([k, v]) => (
                <Row key={k} k={k} v={String(v)} />
              ))}
            </Card>
            <Card title="SQLite">
              {Object.entries(h.sqlite || {}).map(([k, v]) => (
                <Row key={k} k={k} v={fmtBytes(v)} />
              ))}
            </Card>
            <Card title="磁盘">
              <Row k="path" v={h.disk?.path || "—"} />
              <Row k="free" v={fmtBytes(h.disk?.free_bytes)} />
              <Row k="total" v={fmtBytes(h.disk?.total_bytes)} />
              {h.disk?.error ? <Row k="error" v={h.disk.error} /> : null}
            </Card>
            <Card title="路径">
              {Object.entries(h.paths || {}).map(([k, v]) => (
                <Row key={k} k={k} v={v || "—"} />
              ))}
              <Row k="HEAD" v={h.git_head || "—"} />
            </Card>
          </div>
        </>
      )}
    </section>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <div className="text-xs text-ink-3">{label}</div>
      <div className="mt-1 truncate font-mono text-lg text-primary">{value}</div>
    </div>
  );
}
function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <h3 className="mb-2 text-sm font-semibold text-ink-1">{title}</h3>
      {children}
    </div>
  );
}
function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex gap-3 border-b border-border py-1.5 last:border-0">
      <div className="w-28 shrink-0 font-mono text-[11px] text-ink-4">{k}</div>
      <div className="min-w-0 break-all font-mono text-[11px]">{v}</div>
    </div>
  );
}
