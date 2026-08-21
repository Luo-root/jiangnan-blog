import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";
import { errText, useToast } from "../../components/toast";

type Entry = {
  ts: string;
  tool: string;
  client_id: string;
  scopes?: string[];
  args_digest?: string;
  result_status: string;
  duration_ms: number;
  error?: string;
  target_path?: string;
  commit?: string;
};

function toRFC3339(local: string): string | null {
  if (!local) return "";
  const d = new Date(local);
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}

export function AuditPage() {
  const toast = useToast();
  const [rows, setRows] = useState<Entry[]>([]);
  const [tool, setTool] = useState("");
  const [client, setClient] = useState("");
  const [since, setSince] = useState("");
  const [until, setUntil] = useState("");

  async function load(e?: FormEvent) {
    e?.preventDefault();
    const q = new URLSearchParams({ limit: "100" });
    if (tool) q.set("tool", tool);
    if (client) q.set("client_id", client);
    if (since) {
      const v = toRFC3339(since);
      if (v == null) { toast.error("请输入有效的日期和时间"); return; }
      if (v) q.set("since", v);
    }
    if (until) {
      const v = toRFC3339(until);
      if (v == null) { toast.error("请输入有效的日期和时间"); return; }
      if (v) q.set("until", v);
    }
    try {
      setRows(await api<Entry[]>(`/api/audit/recent?${q}`));
    } catch (err) {
      toast.error(errText(err));
    }
  }
  useEffect(() => { load().catch((e) => toast.error(errText(e))); }, []);

  return (
    <section className="flex h-full flex-col p-6">
      <h2 className="text-lg font-semibold text-ink-1">审计日志</h2>
      <p className="mt-1 text-xs text-ink-3">最小字段集。不展示 token / args 原文。按 client / tool / since–until 过滤。</p>
      <form onSubmit={load} className="mt-4 flex flex-wrap gap-2">
        <input className="rounded-lg border border-border px-3 py-2 font-mono text-xs" placeholder="tool" value={tool} onChange={(e) => setTool(e.target.value)} />
        <input className="rounded-lg border border-border px-3 py-2 font-mono text-xs" placeholder="client_id" value={client} onChange={(e) => setClient(e.target.value)} />
        <label className="flex items-center gap-1 text-xs text-ink-3">since
          <input type="datetime-local" className="rounded-lg border border-border px-3 py-2 font-mono text-xs" value={since} onChange={(e) => setSince(e.target.value)} />
        </label>
        <label className="flex items-center gap-1 text-xs text-ink-3">until
          <input type="datetime-local" className="rounded-lg border border-border px-3 py-2 font-mono text-xs" value={until} onChange={(e) => setUntil(e.target.value)} />
        </label>
        <button className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground">刷新</button>
      </form>
      <div className="mt-4 min-h-0 flex-1 overflow-auto rounded-xl border border-border bg-card">
        <table className="w-full text-left text-xs">
          <thead className="sticky top-0 bg-muted font-mono text-[10px] uppercase tracking-wider text-ink-4">
            <tr>
              <th className="px-3 py-2">ts</th>
              <th className="px-3 py-2">tool</th>
              <th className="px-3 py-2">client</th>
              <th className="px-3 py-2">status</th>
              <th className="px-3 py-2">ms</th>
              <th className="px-3 py-2">digest</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr><td className="px-3 py-8 text-center text-ink-4" colSpan={6}>暂无记录</td></tr>
            ) : rows.map((e, i) => (
              <tr key={i} className="border-t border-border">
                <td className="px-3 py-2 font-mono text-[11px] whitespace-nowrap">{(e.ts || "").replace("T", " ").slice(0, 19)}</td>
                <td className="px-3 py-2 font-mono">{e.tool}</td>
                <td className="px-3 py-2 font-mono text-ink-3">{e.client_id || "—"}</td>
                <td className="px-3 py-2">
                  <span className={e.result_status === "success" ? "text-accent" : "text-destructive"}>{e.result_status}</span>
                </td>
                <td className="px-3 py-2 font-mono">{e.duration_ms}</td>
                <td className="max-w-[180px] truncate px-3 py-2 font-mono text-ink-4" title={e.args_digest}>{e.args_digest || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
