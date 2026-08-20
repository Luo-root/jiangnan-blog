import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { UnifiedDiff } from "../../components/diff-viewer";

type Commit = { sha: string; author: string; date: string; subject: string };

export function GitPage() {
  const [list, setList] = useState<Commit[]>([]);
  const [sha, setSha] = useState("");
  const [diff, setDiff] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api<Commit[]>("/api/git/history?limit=40").then(setList).catch((e) => setError(String(e)));
  }, []);

  async function open(commit: string) {
    setSha(commit);
    setDiff(await api<{ diff: string }>(`/api/git/diff/${encodeURIComponent(commit)}`).then((r) => r.diff));
  }

  return (
    <section className="flex h-full overflow-hidden">
      <div className="w-80 shrink-0 overflow-auto border-r border-border p-4">
        <h2 className="text-lg font-semibold">Git 变更</h2>
        <p className="mt-1 text-xs text-ink-3">workbench HEAD 历史。点一条看 diff。</p>
        {error ? <p className="mt-3 text-sm text-destructive">{error}</p> : null}
        <div className="mt-3 space-y-1">
          {list.length === 0 ? <p className="text-sm text-ink-4">暂无 commit</p> : null}
          {list.map((c) => (
            <button
              key={c.sha}
              onClick={() => open(c.sha).catch((e) => setError(String(e)))}
              className={`block w-full rounded-lg border px-3 py-2 text-left ${sha === c.sha ? "border-primary bg-primary/10" : "border-border bg-card hover:border-primary/40"}`}
            >
              <div className="truncate text-[13px]">{c.subject}</div>
              <div className="mt-1 font-mono text-[10px] text-ink-4">{c.sha.slice(0, 8)} · {c.author} · {(c.date || "").slice(0, 10)}</div>
            </button>
          ))}
        </div>
      </div>
      <div className="min-w-0 flex-1 overflow-auto p-4">
        {sha ? <UnifiedDiff text={diff} /> : <p className="text-sm text-ink-4">选中左侧 commit 查看红绿 diff。</p>}
      </div>
    </section>
  );
}
