import { FormEvent, useState } from "react";
import { api } from "../../lib/api";
import { errText, useToast } from "../../components/toast";

type Hit = {
  id: string;
  title: string;
  path_hint: string;
  kind: string;
  visibility: string;
  summary: string;
  matched_fields?: string[];
  score: number;
  matched_via?: string;
  signals?: Record<string, number>;
  excerpt?: string;
};

type SearchOut = {
  results?: Hit[];
  message?: string;
  suggestions?: string[];
  query_echo?: string;
  executed_signals?: string[];
  elapsed_ms?: number;
};

const KINDS = ["", "note", "article", "project", "skill", "mcp_server", "context_pack"];
const VIS = ["", "public", "private", "secret", "draft"];
const SORTS = ["score", "recency", "access", "hot"];

export function SearchPage() {
  const toast = useToast();
  const [q, setQ] = useState("");
  const [kind, setKind] = useState("");
  const [visibility, setVisibility] = useState("");
  const [tag, setTag] = useState("");
  const [sort, setSort] = useState("score");
  const [out, setOut] = useState<SearchOut | null>(null);
  const [open, setOpen] = useState<string | null>(null);
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);

  async function run(e?: FormEvent) {
    e?.preventDefault();
    setBusy(true);
    const params = new URLSearchParams({ sort, limit: "20" });
    if (q.trim()) params.set("q", q.trim());
    if (kind) params.set("kind", kind);
    if (visibility) params.set("visibility", visibility);
    if (tag) params.set("tag", tag);
    try {
      setOut(await api<SearchOut>(`/api/knowledge/search?${params}`));
      setOpen(null);
      setBody("");
    } catch (err) {
      toast.error(errText(err));
    } finally {
      setBusy(false);
    }
  }

  function clearAll() {
    setQ("");
    setKind("");
    setVisibility("");
    setTag("");
    setSort("score");
    setOut(null);
    setOpen(null);
    setBody("");
  }

  async function expand(id: string) {
    if (open === id) {
      setOpen(null);
      return;
    }
    const got = await api<{ body?: string }>(`/api/knowledge?id=${encodeURIComponent(id)}`);
    setOpen(id);
    setBody(got.body || "");
  }

  const hits = out?.results || [];
  const empty = out && hits.length === 0;

  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold text-ink-1">知识搜索</h2>
      <p className="mt-1 text-xs text-ink-3">关键词可空。kind / visibility / tag 任一有值即可列出。清除清全部条件 + 结果。</p>

      <form onSubmit={run} className="mt-4 rounded-xl border border-border bg-card p-4">
        <div className="flex gap-2">
          <input className="flex-1 rounded-lg border border-border px-3 py-2 text-sm" placeholder="输入关键词（可空）" value={q} onChange={(e) => setQ(e.target.value)} />
          <button disabled={busy} className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50">搜索</button>
          <button type="button" className="rounded-lg border border-border px-3 py-2 text-xs" onClick={clearAll}>清除</button>
        </div>
        <div className="mt-3 flex flex-wrap gap-2 text-xs">
          <label className="flex items-center gap-1">kind
            <select className="rounded border border-border px-2 py-1" value={kind} onChange={(e) => setKind(e.target.value)}>
              {KINDS.map((k) => <option key={k || "all"} value={k}>{k || "全部"}</option>)}
            </select>
          </label>
          <label className="flex items-center gap-1">visibility
            <select className="rounded border border-border px-2 py-1" value={visibility} onChange={(e) => setVisibility(e.target.value)}>
              {VIS.map((k) => <option key={k || "all"} value={k}>{k || "全部"}</option>)}
            </select>
          </label>
          <label className="flex items-center gap-1">tag
            <input className="rounded border border-border px-2 py-1" value={tag} onChange={(e) => setTag(e.target.value)} />
          </label>
          <label className="flex items-center gap-1">排序
            <select className="rounded border border-border px-2 py-1" value={sort} onChange={(e) => setSort(e.target.value)}>
              {SORTS.map((k) => <option key={k} value={k}>{k}</option>)}
            </select>
          </label>
        </div>
      </form>

      {out ? (
        <p className="mt-4 font-mono text-[11px] text-ink-3">
          结果: {hits.length} 条{out.elapsed_ms != null ? ` · ${out.elapsed_ms}ms` : ""}
        </p>
      ) : null}

      {empty ? (
        <div className="mt-4 rounded-xl border border-warning/40 bg-warning/10 p-5">
          <div className="font-semibold text-ink-1">{out?.message || "未查询到相关内容"}</div>
          {out?.query_echo ? <div className="mt-1 font-mono text-xs text-ink-3">query: {out.query_echo}</div> : null}
          <div className="mt-2 font-mono text-[11px] text-ink-3">执行信号: {(out?.executed_signals || []).join(" / ")}</div>
          <ul className="mt-3 list-disc pl-5 text-sm text-ink-2">
            {(out?.suggestions || []).map((s) => <li key={s}>{s}</li>)}
          </ul>
        </div>
      ) : null}

      <div className="mt-3 space-y-2">
        {hits.map((h, i) => (
          <article key={h.id} className="rounded-xl border border-border bg-card p-4">
            <div className="flex items-center gap-2">
              <span className="font-mono text-[11px] text-ink-4">{i + 1}.</span>
              <h3 className="text-sm font-semibold text-ink-1">{h.title || h.id}</h3>
              <span className="rounded-full bg-muted px-2 py-0.5 font-mono text-[10px]">{h.kind}</span>
              <span className="rounded-full bg-muted px-2 py-0.5 font-mono text-[10px]">{h.visibility}</span>
            </div>
            <div className="mt-1 font-mono text-[11px] text-ink-3">路径: {h.path_hint}</div>
            <div className="mt-1 font-mono text-[11px] text-ink-3">
              score {h.score.toFixed(2)} · {(h.matched_fields || []).map((f) => `${f} ${(h.signals?.[f] ?? 0).toFixed(1)}`).join(" / ")}
            </div>
            <p className="mt-2 text-sm text-ink-2">{h.summary || h.excerpt || "无摘要"}</p>
            <button className="mt-2 text-xs text-primary" onClick={() => expand(h.id).catch((e) => toast.error(errText(e)))}>
              {open === h.id ? "收起" : "展开"}
            </button>
            {open === h.id ? (
              <pre className="mt-3 max-h-72 overflow-auto whitespace-pre-wrap rounded-lg bg-muted p-3 font-mono text-[12px]">{body || "（空）"}</pre>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}
