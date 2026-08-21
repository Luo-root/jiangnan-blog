import { useEffect, useState, type ReactNode } from "react";
import { api } from "../../lib/api";
import { navigate } from "../../lib/nav";
import { DiffViewer } from "../../components/diff-viewer";
import { CommentThread, type Comment } from "../../components/comments";
import { errText, useToast } from "../../components/toast";

type Proposal = {
  id: string;
  status: string;
  reason: string;
  created_by: string;
  created_at: string;
  base_commit?: string;
  target?: { type?: string; path?: string };
  operation?: { type?: string; section?: string };
  payload?: { format?: string; content?: string };
  comments?: Comment[];
  receipt?: { status?: string; commit?: string; conflict_regions?: { excerpt?: string }[] };
};

const STATUS_CLS: Record<string, string> = {
  pending: "bg-ink-3/20 text-ink-1",
  conflict: "bg-warning/20 text-warning",
  approved: "bg-primary/15 text-primary",
  applied: "bg-accent/20 text-accent",
  rejected: "bg-destructive/15 text-destructive",
};

export function ProposalDetailPage({ id }: { id: string }) {
  const toast = useToast();
  const [p, setP] = useState<Proposal | null>(null);
  const [reason, setReason] = useState("");
  const [op, setOp] = useState("");
  const [section, setSection] = useState("");
  const [target, setTarget] = useState("");
  const [content, setContent] = useState("");
  const [busy, setBusy] = useState(false);
  const [original, setOriginal] = useState("");
  const [openForm, setOpenForm] = useState(false);

  async function loadOriginal(path: string) {
    if (!path) { setOriginal(""); return; }
    try {
      const n = await api<{ body?: string }>(`/api/knowledge?id=${encodeURIComponent(path)}`);
      setOriginal(n.body || "");
    } catch {
      setOriginal("");
    }
  }

  async function reload() {
    const got = await api<Proposal>("/api/proposals/" + encodeURIComponent(id));
    setP(got);
    setReason(got.reason || "");
    setOp(got.operation?.type || "");
    setSection(got.operation?.section || "");
    setTarget(got.target?.path || "");
    setContent(got.payload?.content || "");
    await loadOriginal(got.target?.path || "");
  }
  useEffect(() => { reload().catch((e) => toast.error(errText(e))); }, [id]);

  const editable = p?.status === "pending" || p?.status === "conflict";
  const terminal = p?.status === "applied" || p?.status === "rejected" || p?.status === "approved";

  async function save() {
    setBusy(true);
    try {
      await api("/api/proposals/" + encodeURIComponent(id), "PATCH", {
        reason,
        target: { type: p?.target?.type || "note", path: target },
        operation: { type: op, section },
        payload: { format: "markdown", content },
      });
      await reload();
      toast.success("Proposal 已保存");
    } catch (e) { toast.error(errText(e)); }
    finally { setBusy(false); }
  }
  async function review(status: string) {
    if (!confirm(status === "approved" ? "确认批准并 apply？" : "确认拒绝该 Proposal？")) return;
    setBusy(true);
    try {
      await api("/api/proposals/" + encodeURIComponent(id), "PUT", { status });
      await reload();
      toast.success(status === "approved" ? "已批准" : "已拒绝");
    } catch (e) { toast.error(errText(e)); }
    finally { setBusy(false); }
  }

  if (!p) return <section className="p-6 text-sm text-ink-3">加载中…</section>;

  return (
    <section className="h-full overflow-auto p-6">
      <button className="text-xs text-ink-3 hover:text-primary" onClick={() => navigate("/workspace/proposal")}>← 返回列表</button>
      <div className="mt-3 flex items-center gap-2">
        <h2 className="font-mono text-lg font-semibold text-ink-1">{p.id}</h2>
        <span className={`rounded-full px-2 py-0.5 font-mono text-[10px] ${STATUS_CLS[p.status] || "bg-muted"}`}>{p.status}</span>
      </div>
      <p className="mt-2 font-mono text-[11px] text-ink-3">{p.created_by} · {p.created_at} · base {p.base_commit || "—"}</p>
      {terminal ? <p className="mt-2 text-xs text-ink-3">终态只读。要再改请开一条新 proposal。</p> : null}

      <div className="mt-5">
        <div className="mb-2 font-semibold text-ink-1">变更预览</div>
        <DiffViewer before={p.operation?.type === "create_file" ? "" : original} after={content} />
      </div>

      {p.receipt?.conflict_regions?.length ? (
        <pre className="mt-4 whitespace-pre-wrap rounded-lg bg-destructive/10 p-3 text-xs text-destructive">
          {p.receipt.conflict_regions.map((c) => c.excerpt).join("\n\n")}
        </pre>
      ) : null}

      <details className="mt-5 rounded-xl border border-border bg-card p-4" open={openForm || editable} onToggle={(e) => setOpenForm((e.target as HTMLDetailsElement).open)}>
        <summary className="cursor-pointer text-sm font-semibold text-ink-1">元数据 / 表单</summary>
        <div className="mt-3 grid gap-3">
          <Field label="原因"><input disabled={!editable} className="w-full rounded-lg border border-border px-3 py-2 text-sm disabled:bg-muted" value={reason} onChange={(e) => setReason(e.target.value)} /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="操作"><input disabled={!editable} className="w-full rounded-lg border border-border px-3 py-2 text-sm disabled:bg-muted" value={op} onChange={(e) => setOp(e.target.value)} /></Field>
            <Field label="Section"><input disabled={!editable} className="w-full rounded-lg border border-border px-3 py-2 text-sm disabled:bg-muted" value={section} onChange={(e) => setSection(e.target.value)} /></Field>
          </div>
          <Field label="目标路径"><input disabled={!editable} className="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm disabled:bg-muted" value={target} onChange={(e) => setTarget(e.target.value)} /></Field>
          <Field label="Payload">
            <textarea disabled={!editable} className="min-h-48 w-full rounded-lg border border-border px-3 py-2 font-mono text-sm disabled:bg-muted" value={content} onChange={(e) => setContent(e.target.value)} />
          </Field>
        </div>
      </details>

      {editable ? (
        <div className="mt-5 flex gap-2">
          <button disabled={busy} className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50" onClick={save}>保存修改</button>
          <button disabled={busy} className="rounded-lg bg-accent px-3 py-2 text-xs text-white disabled:opacity-50" onClick={() => review("approved")}>批准</button>
          <button disabled={busy} className="rounded-lg bg-destructive px-3 py-2 text-xs text-white disabled:opacity-50" onClick={() => review("rejected")}>拒绝</button>
        </div>
      ) : null}

      <CommentThread
        comments={p.comments || []}
        readOnly={!editable}
        onAppend={editable ? async (body) => {
          await api("/api/proposals/" + encodeURIComponent(id), "PATCH", { comment: { body } });
          await reload();
          toast.success("评论已追加");
        } : undefined}
      />
    </section>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block font-mono text-[10px] uppercase tracking-wider text-ink-4">{label}</span>
      {children}
    </label>
  );
}
