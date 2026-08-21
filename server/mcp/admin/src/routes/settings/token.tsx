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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

const SCOPES = [
  "read:context", "read:knowledge", "read:project", "read:registry",
  "read:inbox", "write:proposal", "write:inbox", "ops:audit",
];
const DEFAULT_SCOPES = ["read:context"];

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
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [scopes, setScopes] = useState<string[]>([...DEFAULT_SCOPES]);
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

  function resetForm() {
    setName("");
    setDescription("");
    setScopes([...DEFAULT_SCOPES]);
    setTplId("");
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    try {
      const created = await api<{ token: string }>("/api/auth_tokens", "POST", { name, description, scopes });
      setPlaintext(created.token);
      setCopied(false);
      setCreating(false);
      resetForm();
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
      <div className="flex items-end justify-between">
        <div>
          <h2 className="text-xl font-bold text-ink-1">Token</h2>
          <p className="mt-1 text-sm text-ink-3">明文只在签发 / 轮换时出现一次。撤销即作废，列表不再展示。</p>
        </div>
        <Button onClick={() => { resetForm(); setCreating(true); }}>新建 Token</Button>
      </div>

      <div className="mt-5 overflow-hidden rounded-xl border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>说明</TableHead>
              <TableHead>scopes</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>创建</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {list.length === 0 ? (
              <TableRow><TableCell colSpan={6} className="py-10 text-center text-ink-4">还没有 Token。点右上角新建。</TableCell></TableRow>
            ) : list.map((t) => (
              <TableRow key={t.id}>
                <TableCell className="font-semibold">{t.name}</TableCell>
                <TableCell className="max-w-[220px] truncate text-ink-2">{t.description || "—"}</TableCell>
                <TableCell className="max-w-[280px] truncate font-mono text-[11px] text-ink-3">{(t.scopes || []).join(" ")}</TableCell>
                <TableCell><Badge className="border-transparent bg-emerald-600 text-white hover:bg-emerald-600">{t.status}</Badge></TableCell>
                <TableCell className="font-mono text-[12px] whitespace-nowrap">{(t.created_at || "").replace("T", " ").slice(0, 19)}</TableCell>
                <TableCell className="text-right">
                  <Button variant="link" className="h-auto px-2" onClick={() => setAsk({ kind: "rotate", t })}>轮换</Button>
                  <Button variant="link" className="h-auto px-2 text-destructive" onClick={() => setAsk({ kind: "revoke", t })}>撤销</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <AppDialog title="新建 Token" open={creating} onClose={() => setCreating(false)}>
        <form onSubmit={onCreate}>
          {tpls.length ? (
            <div className="mb-3">
              <p className="mb-1 text-sm font-medium text-ink-2">从模板填入</p>
              <SimpleSelect
                value={tplId}
                onValue={(id) => {
                  setTplId(id);
                  const t = tpls.find((x) => x.id === id);
                  if (!t) return;
                  const { next, filled } = fillEmpty(
                    { name, description, scopes },
                    { name: t.name || "", description: t.description || "", scopes: t.scopes || [] },
                    { name: "", description: "", scopes: DEFAULT_SCOPES },
                  );
                  setName(next.name);
                  setDescription(next.description);
                  setScopes(next.scopes);
                  toast.success(filled.length ? `已填入：${filled.join("、")}` : "表单已有内容，空字段才用模板填");
                }}
                items={[{ value: "", label: "不使用模板" }, ...tpls.map((t) => ({ value: t.id, label: t.name }))]}
              />
            </div>
          ) : null}
          <Label className="block text-sm font-medium">
            name <span className="text-destructive">*</span>
            <Input className="mt-1" placeholder="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </Label>
          <Label className="mt-3 block text-sm font-medium">
            description <span className="text-ink-4 font-normal">选填</span>
            <Input className="mt-1" placeholder="description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </Label>
          <p className="mt-3 text-sm font-medium">scopes</p>
          <div className="mt-2 flex flex-wrap gap-3">
            {SCOPES.map((s) => (
              <label key={s} className="flex items-center gap-2 font-mono text-[12px] font-medium">
                <Checkbox checked={scopes.includes(s)} onCheckedChange={() => toggle(s)} />
                {s}
              </label>
            ))}
          </div>
          <div className="mt-4 flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreating(false)}>取消</Button>
            <Button type="submit">签发</Button>
          </div>
        </form>
      </AppDialog>

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
