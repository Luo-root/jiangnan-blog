import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { errText, useToast } from "../../components/toast";
import { Card, CardContent } from "@/components/ui/card";

type Heat = { resource_id: string; count: number; last_access: string; score: number };

export function AccessPage() {
  const toast = useToast();
  const [rows, setRows] = useState<Heat[] | null>(null);
  useEffect(() => {
    api<Heat[]>("/api/heat").then(setRows).catch((e) => toast.error(errText(e)));
  }, []);
  const list = rows || [];
  const max = list[0]?.score || 0;
  const sum = list.reduce((n, x) => n + (x.count || 0), 0);
  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold text-ink-1">访问热度</h2>
      <p className="mt-1 text-xs text-ink-3">score = count × exp(-days / half_life)。条按 score，不是 count。只统计 MCP get。</p>
      <div className="mt-4 grid grid-cols-3 gap-3">
        <Stat label="上榜资源" value={String(list.length)} />
        <Stat label="最高 score" value={max ? max.toFixed(3) : "0"} />
        <Stat label="累计读取" value={String(sum)} />
      </div>
      <div className="mt-4 rounded-xl border border-border bg-card p-4">
        {rows && list.length === 0 ? (
          <p className="text-sm text-ink-2">暂无访问记录。热度只统计 MCP get，不统计后台浏览。</p>
        ) : null}
        {list.slice(0, 50).map((h) => (
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
    <Card>
      <CardContent className="p-4">
        <div className="text-xs text-ink-3">{label}</div>
        <div className="mt-1 font-mono text-xl text-primary">{value}</div>
      </CardContent>
    </Card>
  );
}
