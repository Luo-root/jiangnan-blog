import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";
import { AppDialog } from "../../components/app-dialog";
import { ConfirmDialog } from "../../components/confirm";
import { errText, useToast } from "../../components/toast";
import { fillEmpty, loadTemplates, type Tpl } from "../../lib/templates";
import { SimpleSelect } from "../../components/simple-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";

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
  const [ask, setAsk] = useState<{ kind: "rotate" | "revoke"; t: Token } | null>(null);

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
          <div className="mb-3">
            <p className="mb-1 text-xs text-ink-2">从模板填入</p>
            <SimpleSelect
              value={tplId}
              onValue={(id) => {
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
              items={[{ value: "", label: "不使用模板" }, ...tpls.map((t) => ({ value: t.id, label: t.name }))]}
            />
          </div>
        ) : null}
        <div className="grid grid-cols-2 gap-3">
          <Label className="block text-xs text-ink-2">
            name <span className="text-destructive">*</span>
            <Input className="mt-1" placeholder="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </Label>
          <Label className="block text-xs text-ink-2">
            description <span className="text-ink-4">选填</span>
            <Input className="mt-1" placeholder="description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </Label>
        </div>
        <div className="mt-3 flex flex-wrap gap-3">
          {SCOPES.map((s) => (
            <label key={s} className="flex items-center gap-2 font-mono text-[11px]">
              <Checkbox checked={scopes.includes(s)} onCheckedChange={() => toggle(s)} />
              {s}
            </label>
          ))}
        </div>
        <Button type="submit" size="sm" className="mt-3">签发</Button>
      </form>

      <div className="mt-5 space-y-2">
        {list.map((t) => (
          <Card key={t.id}>
            <CardContent className="flex items-center justify-between gap-3 p-4">
              <div className="min-w-0">
                <div className="flex items-center gap-2 font-medium text-ink-1">
                  {t.name} <Badge variant="secondary" className="font-mono text-[10px]">{t.status}</Badge>
                </div>
                {t.description ? <div className="mt-1 text-sm text-ink-2">{t.description}</div> : <div className="mt-1 text-xs text-ink-4">无描述</div>}
                <div className="mt-1 font-mono text-[11px] text-ink-3">{(t.scopes || []).join(" ")}</div>
              </div>
              <div className="flex shrink-0 gap-2">
                <Button variant="outline" size="sm" onClick={() => setAsk({ kind: "rotate", t })}>轮换</Button>
                <Button variant="destructive" size="sm" onClick={() => setAsk({ kind: "revoke", t })}>撤销</Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <AppDialog title="明文只出现一次" open={!!plaintext} onClose={() => setPlaintext("")}>
        <p className="text-sm text-ink-2">立刻复制。关掉窗口就再也看不到。</p>
        <pre className="mt-3 break-all rounded-lg bg-muted p-3 font-mono text-xs">{plaintext}</pre>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={() => setPlaintext("")}>关闭</Button>
          <Button size="sm" onClick={copy}>{copied ? "已复制" : "复制"}</Button>
        </div>
      </AppDialog>

      <ConfirmDialog
        open={!!ask}
        title={ask?.kind === "rotate" ? `轮换 token「${ask.t.name}」？` : `作废 token「${ask?.t.name}」？`}
        description={ask?.kind === "rotate" ? "旧明文立刻作废。新明文只出现这一次。" : "撤销后从列表消失。"}
        confirmLabel={ask?.kind === "rotate" ? "轮换" : "作废"}
        destructive={ask?.kind === "revoke"}
        onClose={() => setAsk(null)}
        onConfirm={async () => {
          if (!ask) return;
          const { kind, t } = ask;
          setAsk(null);
          try {
            if (kind === "rotate") {
              const r = await api<{ token: string }>(`/api/auth_tokens/${t.id}/rotate`, "POST");
              setPlaintext(r.token);
              setCopied(false);
              await reload();
              toast.success(`已轮换 ${t.name}。明文只出现这一次。`);
            } else {
              await api(`/api/auth_tokens/${t.id}/revoke`, "POST");
              await reload();
              toast.success(`已撤销 ${t.name}`);
            }
          } catch (e) { toast.error(errText(e)); }
        }}
      />
    </section>
  );
}
