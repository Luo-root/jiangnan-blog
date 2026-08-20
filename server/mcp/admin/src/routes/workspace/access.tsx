import { useEffect, useState } from "react";
import { api } from "../../lib/api";

type Heat = { resource_id: string; count: number; last_access: string; score: number };

export function AccessPage() {
  const [rows, setRows] = useState<Heat[]>([]);
  const [error, setError] = useState("");
  useEffect(() => {
    api<Heat[]>("/api/heat").then(setRows).catch((e) => setError(String(e)));
  }, []);
  const max = rows[0]?.score || 0;
  const sum = rows.reduce((n, x) => n + (x.count || 0), 0);
  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold">访问热度</h2>
      <p className="mt-1 text-xs text-ink-3">score = count × exp(-days / half_life)。条按 score，不是 count。</p>
      {error ? <p className="mt-3 text-sm text-destructive">{error}</p> : null}
      <div className="mt-4 grid grid-cols-3 gap-3">
        <Stat label="上榜资源" value={String(rows.length)} />
        <Stat label="最高 score" value={max ? max.toFixed(3) : "0"} />
        <Stat label="累计读取" value={String(sum)} />
      </div>
      <div className="mt-4 rounded-xl border border-border bg-card p-4">
        {rows.length === 0 ? <p className="text-sm text-ink-4">暂无访问记录</p> : null}
        {rows.slice(0, 50).map((h) => (
          <div key={h.resource_id} className="flex items-center gap-3 border-b border-border py-2 last:border-0">
            <div className="w-72 truncate font-mono text-xs text-ink-2" title={h.resource_id}>{h.resource_id}</div>
            <div className="h-2.5 flex-1 overflow-hidden rounded bg-muted">
              <div className="h-full bg-primary" style={{ width: `${max ? Math.round((h.score / max) * 100) : 0}%` }} />
            </div>
            <div className="w-14 text-right font-mono text-xs">{(h.score || 0).toFixed(3)}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <div className="text-xs text-ink-3">{label}</div>
      <div className="mt-1 font-mono text-xl text-primary">{value}</div>
    </div>
  );
}
