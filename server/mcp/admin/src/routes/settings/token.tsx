import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";
import { Modal } from "../../components/modal";
import { errText, useToast } from "../../components/toast";
import { fillEmpty, loadTemplates, type Tpl } from "../../lib/templates";

const SCOPES = [
  "read:context", "read:knowledge", "read:project", "read:registry",
  "read:inbox", "write:proposal", "write:inbox", "ops:audit",
];

type Token = {
  id: number;
  name: string;
  scopes: string[];
  status: string;
  description?: string;
  created_at: string;
  last_used_at?: string;
  use_count: number;
};

export function TokenPage() {
  const toast = useToast();
  const [list, setList] = useState<Token[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [scopes, setScopes] = useState<string[]>(["read:context"]);
  const [plaintext, setPlaintext] = useState("");
  const [copied, setCopied] = useState(false);
  const [tpls, setTpls] = useState<Tpl[]>([]);
  const [tplId, setTplId] = useState("");

  async function reload() {
    setList(await api<Token[]>("/api/auth_tokens"));
  }
  useEffect(() => {
    reload().catch((e) => toast.error(errText(e)));
    loadTemplates("token").then(setTpls).catch(() => setTpls([]));
  }, []);

  function toggle(s: string) {
    setScopes((cur) => cur.includes(s) ? cur.filter((x) => x !== s) : [...cur, s]);
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    try {
      const created = await api<{ token: string }>("/api/auth_tokens", "POST", { name, description, scopes });
      setPlaintext(created.token);
      setCopied(false);
      setName("");
      setDescription("");
      setTplId("");
      await reload();
      toast.success("Token 已签发。明文只出现这一次。");
    } catch (err) { toast.error(errText(err)); }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(plaintext);
      setCopied(true);
      toast.success("已复制明文");
    } catch {
      toast.error("复制失败，请手动选中");
    }
  }

  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold text-ink-1">Token</h2>
      <p className="mt-1 text-xs text-ink-3">明文只在签发 / 轮换时出现一次。撤销即作废，列表不再展示。admin:reindex 不是 Agent scope。</p>

      <form onSubmit={onCreate} className="mt-5 rounded-xl border border-border bg-card p-4">
        {tpls.length ? (
          <label className="mb-3 block text-xs text-ink-2">
            从模板填入
            <select
              className="mt-1 w-full rounded-lg border border-border px-2 py-1.5"
              value={tplId}
              onChange={(e) => {
                const id = e.target.value;
                setTplId(id);
                const t = tpls.find((x) => x.id === id);
                if (!t) return;
                const next = fillEmpty({ name, description, scopes }, {
                  name: t.name || "",
                  description: t.description || "",
                  scopes: t.scopes || [],
                });
                setName(next.name);
                setDescription(next.description);
                if (!scopes.length && next.scopes.length) setScopes(next.scopes);
              }}
            >
              <option value="">不使用模板</option>
              {tpls.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
          </label>
        ) : null}
        <div className="grid grid-cols-2 gap-3">
          <label className="block text-xs text-ink-2">
            name <span className="text-destructive">*</span>
            <input className="mt-1 w-full rounded-lg border border-border px-3 py-2 text-sm" placeholder="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </label>
          <label className="block text-xs text-ink-2">
            description <span className="text-ink-4">选填</span>
            <input className="mt-1 w-full rounded-lg border border-border px-3 py-2 text-sm" placeholder="description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </label>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          {SCOPES.map((s) => (
            <label key={s} className="flex items-center gap-1 font-mono text-[11px]">
              <input type="checkbox" checked={scopes.includes(s)} onChange={() => toggle(s)} />
              {s}
            </label>
          ))}
        </div>
        <button className="mt-3 rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground">签发</button>
      </form>

      <div className="mt-5 space-y-2">
        {list.map((t) => (
          <div key={t.id} className="rounded-xl border border-border bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <div className="font-medium text-ink-1">{t.name} <span className="font-mono text-[10px] text-ink-3">{t.status}</span></div>
                {t.description ? <div className="mt-1 text-sm text-ink-2">{t.description}</div> : <div className="mt-1 text-xs text-ink-4">无描述</div>}
                <div className="mt-1 font-mono text-[11px] text-ink-3">{(t.scopes || []).join(" ")}</div>
              </div>
              <div className="flex shrink-0 gap-2">
                <button
                  className="rounded-lg border border-border px-2 py-1 text-xs"
                  onClick={async () => {
                    if (!confirm(`确认轮换 token「${t.name}」？旧明文立刻作废。`)) return;
                    try {
                      const r = await api<{ token: string }>(`/api/auth_tokens/${t.id}/rotate`, "POST");
                      setPlaintext(r.token);
                      setCopied(false);
                      await reload();
                      toast.success(`已轮换 ${t.name}。明文只出现这一次。`);
                    } catch (e) { toast.error(errText(e)); }
                  }}
                >轮换</button>
                <button
                  className="rounded-lg border border-destructive/40 px-2 py-1 text-xs text-destructive"
                  onClick={async () => {
                    if (!confirm(`作废 token「${t.name}」？撤销后从列表消失。`)) return;
                    try {
                      await api(`/api/auth_tokens/${t.id}/revoke`, "POST");
                      await reload();
                      toast.success(`已撤销 ${t.name}`);
                    } catch (e) { toast.error(errText(e)); }
                  }}
                >撤销</button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {plaintext ? (
        <Modal title="明文只出现一次" onClose={() => setPlaintext("")}>
          <p className="text-sm text-ink-2">立刻复制。关掉窗口就再也看不到。</p>
          <pre className="mt-3 break-all rounded-lg bg-muted p-3 font-mono text-xs">{plaintext}</pre>
          <div className="mt-3 flex justify-end gap-2">
            <button className="rounded-lg border border-border px-3 py-1.5 text-xs" onClick={() => setPlaintext("")}>关闭</button>
            <button className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground" onClick={copy}>{copied ? "已复制" : "复制"}</button>
          </div>
        </Modal>
      ) : null}
    </section>
  );
}
