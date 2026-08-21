import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { navigate } from "../../lib/nav";
import { errText, useToast } from "../../components/toast";

type Proposal = {
  id: string;
  status: string;
  reason: string;
  created_by: string;
  created_at: string;
  target?: { path?: string };
  operation?: { type?: string };
};

const LABEL: Record<string, string> = {
  pending: "待审批", approved: "已批准", rejected: "已拒绝", applied: "已应用", conflict: "冲突",
};
const CLS: Record<string, string> = {
  pending: "bg-ink-3/15 text-ink-1",
  conflict: "bg-warning/20 text-warning",
  approved: "bg-primary/15 text-primary",
  applied: "bg-accent/20 text-accent",
  rejected: "bg-destructive/15 text-destructive",
};

export function ProposalsPage() {
  const toast = useToast();
  const [list, setList] = useState<Proposal[]>([]);
  useEffect(() => {
    api<Proposal[]>("/api/proposals").then(setList).catch((e) => toast.error(errText(e)));
  }, []);
  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold text-ink-1">Proposal</h2>
      <p className="mt-1 text-xs text-ink-3">正式知识写入请求。点进去先看 diff，再决定批准或拒绝。</p>
      <div className="mt-4 space-y-2">
        {list.length === 0 ? <p className="text-sm text-ink-4">暂无 Proposal</p> : null}
        {list.map((p) => (
          <button
            key={p.id}
            onClick={() => navigate("/workspace/proposal/" + encodeURIComponent(p.id))}
            className="block w-full rounded-xl border border-border bg-card p-4 text-left hover:border-primary/40"
          >
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs text-primary">{p.id}</span>
              <span className={`rounded-full px-2 py-0.5 font-mono text-[10px] ${CLS[p.status] || "bg-muted"}`}>{LABEL[p.status] || p.status}</span>
            </div>
            <div className="mt-2 text-sm text-ink-1">{p.reason || "未填写原因"}</div>
            <div className="mt-2 font-mono text-[11px] text-ink-3">{p.target?.path || "未指定"} · {p.created_by}</div>
          </button>
        ))}
      </div>
    </section>
  );
}
