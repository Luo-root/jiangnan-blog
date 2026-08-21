import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { UnifiedDiff } from "../../components/diff-viewer";
import { errText, useToast } from "../../components/toast";

type Commit = { sha: string; author: string; date: string; subject: string };

export function GitPage() {
  const toast = useToast();
  const [list, setList] = useState<Commit[]>([]);
  const [sha, setSha] = useState("");
  const [diff, setDiff] = useState("");
  const [meta, setMeta] = useState<Commit | null>(null);

  useEffect(() => {
    api<Commit[]>("/api/git/history?limit=40").then(setList).catch((e) => toast.error(errText(e)));
  }, []);

  async function open(c: Commit) {
    setSha(c.sha);
    setMeta(c);
    setDiff(await api<{ diff: string }>(`/api/git/diff/${encodeURIComponent(c.sha)}`).then((r) => r.diff));
  }

  return (
    <section className="flex h-full overflow-hidden">
      <div className="w-72 shrink-0 overflow-auto border-r border-border p-4">
        <h2 className="text-lg font-semibold text-ink-1">Git 变更</h2>
        <p className="mt-1 text-xs text-ink-3">线性 HEAD 历史。点一条看 diff。</p>
        <div className="relative mt-4 ml-2">
          <div className="absolute bottom-2 left-[5px] top-2 w-px bg-border" />
          <div className="space-y-1">
            {list.length === 0 ? <p className="text-sm text-ink-4">暂无 commit</p> : null}
            {list.map((c) => (
              <button
                key={c.sha}
                onClick={() => open(c).catch((e) => toast.error(errText(e)))}
                className={`relative flex w-full gap-3 rounded-lg px-2 py-2 text-left ${sha === c.sha ? "bg-primary/10" : "hover:bg-muted"}`}
              >
                <span className={`mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full border-2 ${sha === c.sha ? "border-primary bg-primary" : "border-ink-4 bg-card"}`} />
                <span className="min-w-0">
                  <span className="block truncate text-[13px] text-ink-1">{c.subject}</span>
                  <span className="mt-0.5 block font-mono text-[10px] text-ink-3">{c.sha.slice(0, 8)} · {c.author} · {(c.date || "").slice(0, 10)}</span>
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden p-4">
        {sha && meta ? (
          <>
            <div className="mb-3">
              <div className="font-semibold text-ink-1">{meta.subject}</div>
              <div className="mt-1 font-mono text-[11px] text-ink-3">{meta.sha} · {meta.author} · {meta.date}</div>
            </div>
            <div className="min-h-0 flex-1 overflow-auto">
              <UnifiedDiff text={diff} />
            </div>
          </>
        ) : <p className="text-sm text-ink-4">选一条提交看 diff</p>}
      </div>
    </section>
  );
}
