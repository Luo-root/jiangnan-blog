import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { navigate } from "../../lib/nav";

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

export function ProposalsPage() {
  const [list, setList] = useState<Proposal[]>([]);
  const [error, setError] = useState("");
  useEffect(() => {
    api<Proposal[]>("/api/proposals").then(setList).catch((e) => setError(String(e)));
  }, []);
  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold">Proposal</h2>
      <p className="mt-1 text-xs text-ink-3">正式知识写入请求。点进去看详情、编辑、批准或拒绝。</p>
      {error ? <p className="mt-3 text-sm text-destructive">{error}</p> : null}
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
              <span className="rounded-full bg-muted px-2 py-0.5 font-mono text-[10px]">{LABEL[p.status] || p.status}</span>
            </div>
            <div className="mt-2 text-sm">{p.reason || "未填写原因"}</div>
            <div className="mt-2 font-mono text-[11px] text-ink-3">{p.target?.path || "未指定"} · {p.created_by}</div>
          </button>
        ))}
      </div>
    </section>
  );
}
