import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";

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
  const [list, setList] = useState<Token[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [scopes, setScopes] = useState<string[]>(["read:context"]);
  const [plaintext, setPlaintext] = useState("");
  const [error, setError] = useState("");

  async function reload() {
    setList(await api<Token[]>("/api/auth_tokens"));
  }
  useEffect(() => { reload().catch((e) => setError(String(e))); }, []);

  function toggle(s: string) {
    setScopes((cur) => cur.includes(s) ? cur.filter((x) => x !== s) : [...cur, s]);
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const created = await api<{ token: string }>("/api/auth_tokens", "POST", { name, description, scopes });
      setPlaintext(created.token);
      setName("");
      setDescription("");
      await reload();
    } catch (err) { setError(String(err)); }
  }

  return (
    <section className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold">Token</h2>
      <p className="mt-1 text-xs text-ink-3">明文只在签发 / 轮换时出现一次。admin:reindex 不是 Agent scope。</p>
      {error ? <p className="mt-3 text-sm text-destructive">{error}</p> : null}
      {plaintext ? (
        <div className="mt-4 rounded-xl border border-warning/40 bg-warning/10 p-4 font-mono text-xs break-all">
          立刻复制：{plaintext}
        </div>
      ) : null}

      <form onSubmit={onCreate} className="mt-5 rounded-xl border border-border bg-card p-4">
        <div className="grid grid-cols-2 gap-3">
          <input className="rounded-lg border border-border px-3 py-2 text-sm" placeholder="name" value={name} onChange={(e) => setName(e.target.value)} />
          <input className="rounded-lg border border-border px-3 py-2 text-sm" placeholder="description" value={description} onChange={(e) => setDescription(e.target.value)} />
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
            <div className="flex items-center justify-between">
              <div>
                <div className="font-medium">{t.name} <span className="font-mono text-[10px] text-ink-3">{t.status}</span></div>
                <div className="mt-1 font-mono text-[11px] text-ink-3">{(t.scopes || []).join(" ")}</div>
              </div>
              <div className="flex gap-2">
                <button
                  className="rounded-lg border border-border px-2 py-1 text-xs"
                  onClick={async () => {
                    const r = await api<{ token: string }>(`/api/auth_tokens/${t.id}/rotate`, "POST");
                    setPlaintext(r.token);
                    await reload();
                  }}
                >轮换</button>
                <button
                  className="rounded-lg border border-destructive/40 px-2 py-1 text-xs text-destructive"
                  onClick={async () => {
                    await api(`/api/auth_tokens/${t.id}/revoke`, "POST");
                    await reload();
                  }}
                >撤销</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
