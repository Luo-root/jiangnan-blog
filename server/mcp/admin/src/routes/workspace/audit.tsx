import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";
import { errText, useToast } from "../../components/toast";
import { DateRangePicker, dateToRFC3339 } from "../../components/date-time-picker";
import { AUDIT_BADGE } from "../../lib/status";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

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

export function AuditPage() {
  const toast = useToast();
  const [rows, setRows] = useState<Entry[]>([]);
  const [tool, setTool] = useState("");
  const [client, setClient] = useState("");
  const [since, setSince] = useState<Date | null>(null);
  const [until, setUntil] = useState<Date | null>(null);

  async function load(e?: FormEvent) {
    e?.preventDefault();
    const q = new URLSearchParams({ limit: "100" });
    if (tool) q.set("tool", tool);
    if (client) q.set("client_id", client);
    if (since) {
      const v = dateToRFC3339(since);
      if (v == null) { toast.error("请输入有效的日期和时间"); return; }
      if (v) q.set("since", v);
    }
    if (until) {
      const v = dateToRFC3339(until);
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
      <h2 className="text-xl font-bold text-ink-1">审计日志</h2>
      <p className="mt-1 text-sm text-ink-3">最小字段集。不展示 token / args 原文。日期用范围选择器。</p>
      <form onSubmit={load} className="mt-4 flex flex-wrap items-center gap-2">
        <Input className="h-9 w-40 font-mono text-sm" placeholder="tool" value={tool} onChange={(e) => setTool(e.target.value)} />
        <Input className="h-9 w-40 font-mono text-sm" placeholder="client_id" value={client} onChange={(e) => setClient(e.target.value)} />
        <DateRangePicker from={since} to={until} onChange={(a, b) => { setSince(a); setUntil(b); }} />
        <Button type="submit" size="sm">刷新</Button>
      </form>
      <div className="mt-4 min-h-0 flex-1 overflow-auto rounded-xl border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ts</TableHead>
              <TableHead>tool</TableHead>
              <TableHead>client</TableHead>
              <TableHead>status</TableHead>
              <TableHead>ms</TableHead>
              <TableHead>digest</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.length === 0 ? (
              <TableRow><TableCell className="py-8 text-center text-ink-4" colSpan={6}>暂无记录</TableCell></TableRow>
            ) : rows.map((e, i) => (
              <TableRow key={i}>
                <TableCell className="font-mono text-[12px] whitespace-nowrap">{(e.ts || "").replace("T", " ").slice(0, 19)}</TableCell>
                <TableCell className="font-mono font-medium">{e.tool}</TableCell>
                <TableCell className="font-mono text-ink-3">{e.client_id || "—"}</TableCell>
                <TableCell>
                  <Badge className={AUDIT_BADGE[e.result_status] || ""}>{e.result_status}</Badge>
                </TableCell>
                <TableCell className="font-mono">{e.duration_ms}</TableCell>
                <TableCell className="max-w-[180px] truncate font-mono text-ink-4" title={e.args_digest}>{e.args_digest || "—"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}
