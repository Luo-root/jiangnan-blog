import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { navigate } from "../../lib/nav";
import { errText, useToast } from "../../components/toast";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { PROPOSAL_BADGE } from "../../lib/status";

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
  const toast = useToast();
  const [list, setList] = useState<Proposal[]>([]);
  useEffect(() => {
    api<Proposal[]>("/api/proposals").then(setList).catch((e) => toast.error(errText(e)));
  }, []);
  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-xl font-bold text-ink-1">Proposal</h2>
      <p className="mt-1 text-sm text-ink-3">正式知识写入请求。点进去先看 diff，再决定批准或拒绝。</p>
      <div className="mt-4 space-y-2">
        {list.length === 0 ? <p className="text-sm text-ink-4">暂无 Proposal</p> : null}
        {list.map((p) => (
          <Card key={p.id} className="cursor-pointer hover:border-primary/40" onClick={() => navigate("/workspace/proposal/" + encodeURIComponent(p.id))}>
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <span className="font-mono text-xs text-primary">{p.id}</span>
                <Badge className={PROPOSAL_BADGE[p.status] || ""}>{LABEL[p.status] || p.status}</Badge>
              </div>
              <div className="mt-2 text-sm text-ink-1">{p.reason || "未填写原因"}</div>
              <div className="mt-2 font-mono text-[11px] text-ink-3">{p.target?.path || "未指定"} · {p.created_by}</div>
            </CardContent>
          </Card>
        ))}
      </div>
    </section>
  );
}
