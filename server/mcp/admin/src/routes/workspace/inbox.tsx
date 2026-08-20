import { useEffect, useState, type ReactNode } from "react";
import { api } from "../../lib/api";

type Item = {
  id: string;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  title: string;
  description: string;
  summary: string;
};

type Detail = Item & { content: string };

const COLS = [
  { status: "pending", name: "待处理" },
  { status: "reviewing", name: "待审核" },
  { status: "done", name: "已完成" },
  { status: "abandoned", name: "已废弃" },
];

export function InboxPage() {
  const [items, setItems] = useState<Item[]>([]);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [creating, setCreating] = useState(false);
  const [draft, setDraft] = useState("");
  const [edit, setEdit] = useState("");
  const [error, setError] = useState("");

  async function reload() {
    setItems(await api<Item[]>("/api/inbox"));
  }
  useEffect(() => { reload().catch((e) => setError(String(e))); }, []);

  async function open(id: string) {
    const d = await api<Detail>("/api/inbox/" + encodeURIComponent(id));
    setDetail(d);
    setEdit(d.content || "");
  }
  async function move(id: string, status: string) {
    await api("/api/inbox/" + encodeURIComponent(id), "PUT", { status });
    await reload();
  }

  return (
    <section className="flex h-full flex-col p-6">
      <div className="mb-4 flex items-end justify-between">
        <div>
          <h2 className="text-lg font-semibold">待办看板</h2>
          <p className="mt-1 text-xs text-ink-3">拖卡片换状态。这是真看板，不是下拉。</p>
        </div>
        <button className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground" onClick={() => { setCreating(true); setDraft(""); }}>
          + 新建待办
        </button>
      </div>
      {error ? <p className="mb-3 text-sm text-destructive">{error}</p> : null}
      <div className="grid min-h-0 flex-1 grid-cols-4 gap-3">
        {COLS.map((col) => {
          const list = items.filter((i) => i.status === col.status);
          return (
            <div
              key={col.status}
              className="flex min-h-0 flex-col overflow-hidden rounded-xl border border-border bg-card"
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => { const id = e.dataTransfer.getData("text/plain"); if (id) move(id, col.status).catch((err) => setError(String(err))); }}
            >
              <div className="flex items-center justify-between border-b border-border px-3 py-2 text-sm font-medium">
                {col.name}
                <span className="font-mono text-[11px] text-ink-3">{list.length}</span>
              </div>
              <div className="flex-1 space-y-2 overflow-auto p-2">
                {list.length === 0 ? <div className="py-8 text-center text-xs text-ink-4">— 空 —</div> : null}
                {list.map((it) => (
                  <article
                    key={it.id}
                    draggable
                    onDragStart={(e) => e.dataTransfer.setData("text/plain", it.id)}
                    onClick={() => open(it.id).catch((err) => setError(String(err)))}
                    className="cursor-grab rounded-lg border border-border bg-background p-3 hover:border-primary/40"
                  >
                    <div className="truncate text-[13px] font-semibold">{it.title || "未命名待办"}</div>
                    <div className="mt-1 line-clamp-2 text-[11px] text-ink-3">{it.description || it.summary || "点击查看"}</div>
                    <div className="mt-2 font-mono text-[10px] text-ink-4">{it.created_by} · {(it.updated_at || "").slice(0, 16)}</div>
                  </article>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {detail ? (
        <Modal title={detail.title || "待办详情"} onClose={() => setDetail(null)}>
          <pre className="whitespace-pre-wrap text-sm leading-7">{detail.content}</pre>
          <textarea className="mt-3 min-h-40 w-full rounded-lg border border-border p-2 text-sm" value={edit} onChange={(e) => setEdit(e.target.value)} />
          <div className="mt-3 flex justify-end gap-2">
            <button className="rounded-lg border border-border px-3 py-1.5 text-xs" onClick={() => setDetail(null)}>关闭</button>
            <button
              className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground"
              onClick={async () => {
                await api("/api/inbox/" + encodeURIComponent(detail.id), "PUT", { content: edit });
                await reload();
                await open(detail.id);
              }}
            >保存</button>
          </div>
        </Modal>
      ) : null}

      {creating ? (
        <Modal title="新建待办" onClose={() => setCreating(false)}>
          <textarea className="min-h-40 w-full rounded-lg border border-border p-2 text-sm" value={draft} onChange={(e) => setDraft(e.target.value)} placeholder="完整任务描述" />
          <div className="mt-3 flex justify-end gap-2">
            <button className="rounded-lg border border-border px-3 py-1.5 text-xs" onClick={() => setCreating(false)}>取消</button>
            <button
              className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground"
              onClick={async () => {
                if (!draft.trim()) return;
                await api("/api/inbox", "POST", { content: draft, created_by: "webui" });
                setCreating(false);
                await reload();
              }}
            >保存</button>
          </div>
        </Modal>
      ) : null}
    </section>
  );
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="fixed inset-0 z-20 flex items-center justify-center bg-ink-1/40 p-5" onClick={onClose}>
      <div className="max-h-[90vh] w-[640px] overflow-auto rounded-2xl border border-border bg-card p-5" onClick={(e) => e.stopPropagation()}>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold">{title}</h3>
          <button onClick={onClose} className="text-ink-3">×</button>
        </div>
        {children}
      </div>
    </div>
  );
}
