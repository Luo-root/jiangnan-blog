import { FormEvent, useEffect, useState, type ReactNode } from "react";
import { api } from "../../lib/api";
import { errText, useToast } from "../../components/toast";
import { SimpleSelect } from "../../components/simple-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";

const SCOPES = [
  "read:context", "read:knowledge", "read:project", "read:registry",
  "read:inbox", "write:proposal", "write:inbox", "ops:audit",
];

type Tpl = {
  id: string;
  kind: string;
  name: string;
  reason: string;
  target_type: string;
  operation: string;
  section?: string;
  payload: string;
  title?: string;
  content?: string;
  tags?: string[];
  description?: string;
  scopes?: string[];
  updated_at?: string;
};

const empty = (): Tpl => ({
  id: "", kind: "proposal", name: "", reason: "", target_type: "note",
  operation: "append", section: "", payload: "", title: "", content: "",
  tags: [], description: "", scopes: ["write:proposal"],
});

export function TemplatesPage() {
  const toast = useToast();
  const [list, setList] = useState<Tpl[]>([]);
  const [cur, setCur] = useState<Tpl>(empty());

  async function reload() {
    setList(await api<Tpl[]>("/api/templates"));
  }
  useEffect(() => { reload().catch((e) => toast.error(errText(e))); }, []);

  function toggle(s: string) {
    setCur((c) => ({ ...c, scopes: (c.scopes || []).includes(s) ? (c.scopes || []).filter((x) => x !== s) : [...(c.scopes || []), s] }));
  }

  async function onSave(e: FormEvent) {
    e.preventDefault();
    try {
      if (cur.id && list.some((t) => t.id === cur.id)) {
        await api("/api/templates/" + encodeURIComponent(cur.id), "POST", cur);
        toast.success("模板已更新");
      } else {
        const created = await api<Tpl>("/api/templates", "POST", cur);
        setCur(created);
        toast.success("模板已创建");
      }
      await reload();
    } catch (err) { toast.error(errText(err)); }
  }

  return (
    <section className="flex h-full overflow-hidden">
      <div className="w-72 shrink-0 overflow-auto border-r border-border p-4">
        <h2 className="text-lg font-semibold text-ink-1">模板</h2>
        <p className="mt-1 text-xs text-ink-3">kind 三选一。只预填空字段，不跳过创建确认。</p>
        <Button className="mt-3 w-full" size="sm" onClick={() => setCur(empty())}>+ 新建</Button>
        <div className="mt-3 space-y-1">
          {list.map((t) => (
            <Button
              key={t.id}
              variant={cur.id === t.id ? "secondary" : "outline"}
              className="h-auto w-full flex-col items-start py-2"
              onClick={() => setCur({ ...empty(), ...t })}
            >
              <span className="truncate text-sm text-ink-1">{t.name}</span>
              <span className="font-mono text-[10px] text-ink-4">{t.kind} · {t.operation || t.title || "—"}</span>
            </Button>
          ))}
        </div>
      </div>
      <form onSubmit={onSave} className="min-w-0 flex-1 overflow-auto p-6">
        <div className="grid grid-cols-2 gap-3">
          <Field label="kind *">
            <SimpleSelect
              value={cur.kind}
              onValue={(v) => setCur({ ...cur, kind: v || "proposal" })}
              items={[
                { value: "inbox", label: "inbox" },
                { value: "proposal", label: "proposal" },
                { value: "token", label: "token" },
              ]}
            />
          </Field>
          <Field label="名称 *"><Input value={cur.name} onChange={(e) => setCur({ ...cur, name: e.target.value })} required /></Field>
          <Field label="ID"><Input className="font-mono" value={cur.id} onChange={(e) => setCur({ ...cur, id: e.target.value })} placeholder="空则按名称生成" /></Field>
        </div>

        {cur.kind === "proposal" ? (
          <>
            <div className="mt-3 grid grid-cols-2 gap-3">
              <Field label="target.type"><Input className="font-mono" value={cur.target_type} onChange={(e) => setCur({ ...cur, target_type: e.target.value })} /></Field>
              <Field label="operation"><Input className="font-mono" value={cur.operation} onChange={(e) => setCur({ ...cur, operation: e.target.value })} /></Field>
            </div>
            <Field label="原因"><Input className="mt-3" value={cur.reason} onChange={(e) => setCur({ ...cur, reason: e.target.value })} /></Field>
            <Field label="section"><Input className="mt-3 font-mono" value={cur.section || ""} onChange={(e) => setCur({ ...cur, section: e.target.value })} /></Field>
            <Field label="payload">
              <Textarea className="mt-1 min-h-40 font-mono" value={cur.payload} onChange={(e) => setCur({ ...cur, payload: e.target.value })} />
            </Field>
            <div className="mt-3 flex flex-wrap gap-3">
              {SCOPES.map((s) => (
                <label key={s} className="flex items-center gap-2 font-mono text-[11px]">
                  <Checkbox checked={(cur.scopes || []).includes(s)} onCheckedChange={() => toggle(s)} />
                  {s}
                </label>
              ))}
            </div>
          </>
        ) : null}

        {cur.kind === "inbox" ? (
          <>
            <Field label="title"><Input className="mt-3" value={cur.title || ""} onChange={(e) => setCur({ ...cur, title: e.target.value })} /></Field>
            <Field label="content">
              <Textarea className="mt-1 min-h-40" value={cur.content || ""} onChange={(e) => setCur({ ...cur, content: e.target.value })} />
            </Field>
            <Field label="tags"><Input className="mt-3" value={(cur.tags || []).join(", ")} onChange={(e) => setCur({ ...cur, tags: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) })} placeholder="逗号分隔" /></Field>
          </>
        ) : null}

        {cur.kind === "token" ? (
          <>
            <Field label="description"><Input className="mt-3" value={cur.description || ""} onChange={(e) => setCur({ ...cur, description: e.target.value })} /></Field>
            <div className="mt-3 flex flex-wrap gap-3">
              {SCOPES.map((s) => (
                <label key={s} className="flex items-center gap-2 font-mono text-[11px]">
                  <Checkbox checked={(cur.scopes || []).includes(s)} onCheckedChange={() => toggle(s)} />
                  {s}
                </label>
              ))}
            </div>
          </>
        ) : null}

        <Button type="submit" size="sm" className="mt-4">保存</Button>
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
