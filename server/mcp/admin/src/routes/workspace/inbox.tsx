import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { CommentThread, type Comment } from "../../components/comments";
import { MarkdownPreview } from "../../components/markdown";
import { AppDialog } from "../../components/app-dialog";
import { errText, useToast } from "../../components/toast";
import { fillEmpty, loadTemplates, type Tpl } from "../../lib/templates";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { SimpleSelect } from "../../components/simple-select";

type Item = {
  id: string;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  title: string;
  description: string;
  summary: string;
  comment_count?: number;
};

type Detail = Item & {
  content: string;
  tags?: string[];
  comments?: Comment[];
};

const COLS = [
  { status: "pending", name: "待处理", bar: "bg-slate-600", wrap: "border-slate-300 bg-slate-50" },
  { status: "reviewing", name: "待审核", bar: "bg-amber-500", wrap: "border-amber-400 bg-amber-50" },
  { status: "done", name: "已完成", bar: "bg-emerald-600", wrap: "border-emerald-400 bg-emerald-50" },
  { status: "abandoned", name: "已废弃", bar: "bg-zinc-400", wrap: "border-zinc-300 bg-zinc-100" },
];

const LEGAL: Record<string, string[]> = {
  pending: ["pending", "reviewing", "done", "abandoned"],
  reviewing: ["reviewing", "done", "abandoned"],
  done: ["done"],
  abandoned: ["abandoned"],
};

export function InboxPage() {
  const toast = useToast();
  const [items, setItems] = useState<Item[]>([]);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [creating, setCreating] = useState(false);
  const [draft, setDraft] = useState({ title: "", content: "", tags: "" });
  const [edit, setEdit] = useState("");
  const [editing, setEditing] = useState(false);
  const [tpls, setTpls] = useState<Tpl[]>([]);
  const [tplId, setTplId] = useState("");

  async function reload() {
    setItems(await api<Item[]>("/api/inbox"));
  }
  useEffect(() => {
    reload().catch((e) => toast.error(errText(e)));
    loadTemplates("inbox").then(setTpls).catch(() => setTpls([]));
  }, []);

  async function open(id: string) {
    const d = await api<Detail>("/api/inbox/" + encodeURIComponent(id));
    setDetail(d);
    setEdit(d.content || "");
    setEditing(false);
  }
  async function move(id: string, from: string, status: string) {
    if (from === status) return;
    if (!(LEGAL[from] || []).includes(status)) {
      toast.error(`不能把 ${from} 拖到 ${status}。状态不可逆，卡片留在原列。`);
      return;
    }
    try {
      await api("/api/inbox/" + encodeURIComponent(id), "PUT", { status });
      await reload();
      toast.success(`已移到${COLS.find((c) => c.status === status)?.name || status}`);
    } catch (e) {
      toast.error(errText(e));
    }
  }

  return (
    <section className="flex h-full flex-col p-6">
      <div className="mb-4 flex items-end justify-between">
        <div>
          <h2 className="text-xl font-bold text-ink-1">待办看板</h2>
          <p className="mt-1 text-sm text-ink-3">拖卡片换状态。非法拖拽会拒绝。终态仍可改正文和评论。</p>
        </div>
        <Button size="sm" onClick={() => { setCreating(true); setDraft({ title: "", content: "", tags: "" }); setTplId(""); }}>
          + 新建待办
        </Button>
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-4 gap-3">
        {COLS.map((col) => {
          const list = items.filter((i) => i.status === col.status);
          return (
            <div
              key={col.status}
              className={`flex min-h-0 flex-col overflow-hidden rounded-xl border ${col.wrap}`}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                const raw = e.dataTransfer.getData("text/plain");
                if (!raw) return;
                const [id, from] = raw.split("\t");
                if (id) move(id, from, col.status);
              }}
            >
              <div className="flex items-center justify-between border-b border-border px-3 py-2 text-sm font-medium text-ink-1">
                <span className="flex items-center gap-2">
                  <span className={`h-2 w-2 rounded-full ${col.bar}`} />
                  {col.name}
                </span>
                <span className="font-mono text-[11px] text-ink-3">{list.length}</span>
              </div>
              <div className="flex-1 space-y-2 overflow-auto p-2">
                {list.length === 0 ? <div className="py-8 text-center text-xs text-ink-4">— 空 —</div> : null}
                {list.map((it) => (
                  <article
                    key={it.id}
                    draggable
                    onDragStart={(e) => e.dataTransfer.setData("text/plain", `${it.id}\t${it.status}`)}
                    onClick={() => open(it.id).catch((err) => toast.error(errText(err)))}
                    className="cursor-grab rounded-lg border border-border bg-card p-3 hover:border-primary/40"
                  >
                    <div className="truncate text-[13px] font-semibold text-ink-1">{it.title || "未命名待办"}</div>
                    <div className="mt-1 line-clamp-2 text-[11px] text-ink-3">{it.description || it.summary || "点击查看"}</div>
                    <div className="mt-2 font-mono text-[10px] text-ink-4">
                      {it.created_by} · {(it.updated_at || "").slice(0, 16)}
                      {it.comment_count ? ` · ${it.comment_count} 评` : ""}
                    </div>
                  </article>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      <AppDialog title={detail?.title || "待办详情"} open={!!detail} onClose={() => setDetail(null)} wide>
        {detail ? (
          <>
            <p className="mb-3 font-mono text-[11px] text-ink-3">{detail.status} · {detail.created_by} · {(detail.updated_at || "").slice(0, 19)}</p>
            {editing ? (
              <Textarea className="min-h-40" value={edit} onChange={(e) => setEdit(e.target.value)} />
            ) : (
              <MarkdownPreview text={detail.content} />
            )}
            <div className="mt-3 flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setDetail(null)}>关闭</Button>
              {editing ? (
                <Button
                  size="sm"
                  onClick={async () => {
                    try {
                      await api("/api/inbox/" + encodeURIComponent(detail.id), "PUT", { content: edit });
                      await reload();
                      await open(detail.id);
                      toast.success("待办已保存");
                    } catch (e) {
                      toast.error(errText(e));
                    }
                  }}
                >保存</Button>
              ) : (
                <Button variant="outline" size="sm" onClick={() => setEditing(true)}>编辑</Button>
              )}
            </div>
            <CommentThread
              comments={detail.comments || []}
              onAppend={async (body) => {
                await api("/api/inbox/" + encodeURIComponent(detail.id), "PUT", { comment: { body } });
                await reload();
                await open(detail.id);
                toast.success("评论已追加");
              }}
            />
          </>
        ) : null}
      </AppDialog>

      <AppDialog title="新建待办" open={creating} onClose={() => setCreating(false)}>
        {tpls.length ? (
          <div className="mb-3">
            <p className="mb-1 text-xs text-ink-2">从模板填入</p>
            <SimpleSelect
              value={tplId}
              onValue={(id) => {
                setTplId(id);
                const t = tpls.find((x) => x.id === id);
                if (!t) {
                  toast.info("未选用模板");
                  return;
                }
                const { next, filled } = fillEmpty(draft, {
                  title: t.title || t.name || "",
                  content: t.content || "",
                  tags: (t.tags || []).join(", "),
                });
                setDraft(next);
                toast.success(filled.length ? `已填入：${filled.join("、")}` : "表单已有内容，空字段才用模板填");
              }}
              items={[{ value: "", label: "不使用模板" }, ...tpls.map((t) => ({ value: t.id, label: t.name }))]}
            />
          </div>
        ) : null}
        <Input className="mb-2" placeholder="标题（选填）" value={draft.title} onChange={(e) => setDraft({ ...draft, title: e.target.value })} />
        <Textarea className="min-h-40" value={draft.content} onChange={(e) => setDraft({ ...draft, content: e.target.value })} placeholder="完整任务描述（必填）" />
        <Input className="mt-2" placeholder="标签，逗号分隔（选填）" value={draft.tags} onChange={(e) => setDraft({ ...draft, tags: e.target.value })} />
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={() => setCreating(false)}>取消</Button>
          <Button
            size="sm"
            onClick={async () => {
              if (!draft.content.trim()) { toast.error("请填写待办正文"); return; }
              try {
                await api("/api/inbox", "POST", {
                  content: draft.content,
                  title: draft.title,
                  tags: draft.tags.split(",").map((s) => s.trim()).filter(Boolean),
                  created_by: "webui",
                });
                setCreating(false);
                await reload();
                toast.success("待办已创建");
              } catch (e) {
                toast.error(errText(e));
              }
            }}
          >保存</Button>
        </div>
      </AppDialog>
    </section>
  );
}
