import { FormEvent, useEffect, useState, type ReactNode } from "react";
import { api } from "../../lib/api";

const SCOPES = [
  "read:context", "read:knowledge", "read:project", "read:registry",
  "read:inbox", "write:proposal", "write:inbox", "ops:audit",
];

type Tpl = {
  id: string;
  name: string;
  reason: string;
  target_type: string;
  operation: string;
  section?: string;
  payload: string;
  scopes?: string[];
  updated_at?: string;
};

export function TemplatesPage() {
  const [list, setList] = useState<Tpl[]>([]);
  const [cur, setCur] = useState<Tpl>({
    id: "", name: "", reason: "", target_type: "note", operation: "append", section: "", payload: "", scopes: ["write:proposal"],
  });
  const [error, setError] = useState("");

  async function reload() {
    setList(await api<Tpl[]>("/api/templates"));
  }
  useEffect(() => { reload().catch((e) => setError(String(e))); }, []);

  function toggle(s: string) {
    setCur((c) => ({ ...c, scopes: (c.scopes || []).includes(s) ? (c.scopes || []).filter((x) => x !== s) : [...(c.scopes || []), s] }));
  }

  async function onSave(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      if (cur.id && list.some((t) => t.id === cur.id)) {
        await api("/api/templates/" + encodeURIComponent(cur.id), "POST", cur);
      } else {
        const created = await api<Tpl>("/api/templates", "POST", cur);
        setCur(created);
      }
      await reload();
    } catch (err) { setError(String(err)); }
  }

  return (
    <section className="flex h-full overflow-hidden">
      <div className="w-72 shrink-0 overflow-auto border-r border-border p-4">
        <h2 className="text-lg font-semibold">模板</h2>
        <p className="mt-1 text-xs text-ink-3">proposal 模板 + scope 组合。</p>
        <button
          className="mt-3 w-full rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground"
          onClick={() => setCur({ id: "", name: "", reason: "", target_type: "note", operation: "append", section: "", payload: "", scopes: ["write:proposal"] })}
        >+ 新建</button>
        <div className="mt-3 space-y-1">
          {list.map((t) => (
            <button key={t.id} onClick={() => setCur(t)} className={`block w-full rounded-lg border px-3 py-2 text-left ${cur.id === t.id ? "border-primary bg-primary/10" : "border-border bg-card"}`}>
              <div className="truncate text-sm">{t.name}</div>
              <div className="font-mono text-[10px] text-ink-4">{t.operation} · {t.target_type}</div>
            </button>
          ))}
        </div>
      </div>
      <form onSubmit={onSave} className="min-w-0 flex-1 overflow-auto p-6">
        {error ? <p className="mb-3 text-sm text-destructive">{error}</p> : null}
        <div className="grid grid-cols-2 gap-3">
          <Field label="名称"><input className="w-full rounded-lg border border-border px-3 py-2 text-sm" value={cur.name} onChange={(e) => setCur({ ...cur, name: e.target.value })} /></Field>
          <Field label="ID"><input className="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={cur.id} onChange={(e) => setCur({ ...cur, id: e.target.value })} placeholder="空则按名称生成" /></Field>
          <Field label="target.type"><input className="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={cur.target_type} onChange={(e) => setCur({ ...cur, target_type: e.target.value })} /></Field>
          <Field label="operation"><input className="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={cur.operation} onChange={(e) => setCur({ ...cur, operation: e.target.value })} /></Field>
        </div>
        <Field label="原因"><input className="mt-3 w-full rounded-lg border border-border px-3 py-2 text-sm" value={cur.reason} onChange={(e) => setCur({ ...cur, reason: e.target.value })} /></Field>
        <Field label="section"><input className="mt-3 w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={cur.section || ""} onChange={(e) => setCur({ ...cur, section: e.target.value })} /></Field>
        <Field label="payload">
          <textarea className="mt-1 min-h-40 w-full rounded-lg border border-border px-3 py-2 font-mono text-sm" value={cur.payload} onChange={(e) => setCur({ ...cur, payload: e.target.value })} />
        </Field>
        <div className="mt-3 flex flex-wrap gap-2">
          {SCOPES.map((s) => (
            <label key={s} className="flex items-center gap-1 font-mono text-[11px]">
              <input type="checkbox" checked={(cur.scopes || []).includes(s)} onChange={() => toggle(s)} />
              {s}
            </label>
          ))}
        </div>
        <button className="mt-4 rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground">保存</button>
      </form>
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
