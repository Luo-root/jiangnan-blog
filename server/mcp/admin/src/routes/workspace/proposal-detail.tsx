import { useEffect, useState, type ReactNode } from "react";
import { api } from "../../lib/api";
import { navigate } from "../../lib/nav";
import { DiffViewer } from "../../components/diff-viewer";

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
  receipt?: { status?: string; commit?: string; conflict_regions?: { excerpt?: string }[] };
};

export function ProposalDetailPage({ id }: { id: string }) {
  const [p, setP] = useState<Proposal | null>(null);
  const [reason, setReason] = useState("");
  const [op, setOp] = useState("");
  const [section, setSection] = useState("");
  const [target, setTarget] = useState("");
  const [content, setContent] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [original, setOriginal] = useState("");

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
  useEffect(() => { reload().catch((e) => setError(String(e))); }, [id]);

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
    } catch (e) { setError(String(e)); }
    finally { setBusy(false); }
  }
  async function review(status: string) {
    if (!confirm(status === "approved" ? "确认批准？" : "确认拒绝？")) return;
    setBusy(true);
    try {
      await api("/api/proposals/" + encodeURIComponent(id), "PUT", { status });
      await reload();
    } catch (e) { setError(String(e)); }
    finally { setBusy(false); }
  }

  if (!p) return <section className="p-6 text-sm text-ink-3">{error || "加载中…"}</section>;
  const canReview = p.status === "pending" || p.status === "conflict";

  return (
    <section className="h-full overflow-auto p-6">
      <button className="text-xs text-ink-3 hover:text-primary" onClick={() => navigate("/workspace/proposal")}>← 返回列表</button>
      <div className="mt-3 flex items-center gap-2">
        <h2 className="font-mono text-lg font-semibold text-primary">{p.id}</h2>
        <span className="rounded-full bg-muted px-2 py-0.5 font-mono text-[10px]">{p.status}</span>
      </div>
      {error ? <p className="mt-2 text-sm text-destructive">{error}</p> : null}
      <p className="mt-2 font-mono text-[11px] text-ink-3">{p.created_by} · {p.created_at} · base {p.base_commit || "—"}</p>

      <div className="mt-5 grid gap-3">
        <Field label="原因"><input className="w-full rounded-lg border border-border px-3 py-2 text-sm" value={reason} onChange={(e) => setReason(e.target.value)} /></Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="操作"><input className="w-full rounded-lg border border-border px-3 py-2 text-sm" value={op} onChange={(e) => setOp(e.target.value)} /></Field>
          <Field label="Section"><input className="w-full rounded-lg border border-border px-3 py-2 text-sm" value={section} onChange={(e) => setSection(e.target.value)} /></Field>
        </div>
        <Field label="目标路径"><input className="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={target} onChange={(e) => setTarget(e.target.value)} /></Field>
        <Field label="Payload">
          <textarea className="min-h-48 w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={content} onChange={(e) => setContent(e.target.value)} />
        </Field>
      </div>

      <div className="mt-5">
        <div className="mb-2 font-mono text-[10px] uppercase tracking-wider text-ink-4">Diff</div>
        <DiffViewer before={original} after={content} />
      </div>

      {p.receipt?.conflict_regions?.length ? (
        <pre className="mt-4 whitespace-pre-wrap rounded-lg bg-destructive/10 p-3 text-xs text-destructive">
          {p.receipt.conflict_regions.map((c) => c.excerpt).join("\n\n")}
        </pre>
      ) : null}

      <div className="mt-5 flex gap-2">
        <button disabled={busy} className="rounded-lg border border-border px-3 py-2 text-xs" onClick={save}>保存修改</button>
        {canReview ? (
          <>
            <button disabled={busy} className="rounded-lg bg-accent px-3 py-2 text-xs text-white" onClick={() => review("approved")}>批准</button>
            <button disabled={busy} className="rounded-lg bg-destructive px-3 py-2 text-xs text-white" onClick={() => review("rejected")}>拒绝</button>
          </>
        ) : null}
      </div>
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
