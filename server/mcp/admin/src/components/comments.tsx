import { useState } from "react";
import { errText, useToast } from "./toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";

export type Comment = {
  id: string;
  author_type: string;
  author: string;
  at: string;
  body: string;
  reply_to?: string;
};

export function CommentThread({ comments, onAppend, readOnly }: {
  comments: Comment[];
  onAppend?: (body: string) => Promise<void>;
  readOnly?: boolean;
}) {
  const toast = useToast();
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const list = comments || [];
  return (
    <div className="mt-4">
      <div className="mb-2 font-mono text-[10px] uppercase tracking-wider text-ink-4">评论 · {list.length}</div>
      {list.length === 0 ? <p className="text-xs text-ink-4">还没有评论。</p> : null}
      <div className="space-y-2">
        {list.map((c) => (
          <div key={c.id} className="rounded-lg border border-border bg-muted/40 p-3">
            <div className="flex items-center gap-2 font-mono text-[10px] text-ink-3">
              <Badge variant="secondary">{c.author_type}</Badge>
              <span>{c.author}</span>
              <span>{(c.at || "").replace("T", " ").slice(0, 19)}</span>
            </div>
            <p className="mt-1 whitespace-pre-wrap text-sm text-ink-1">{c.body}</p>
          </div>
        ))}
      </div>
      {readOnly || !onAppend ? null : (
        <div className="mt-3">
          <Textarea
            className="min-h-20"
            placeholder="追加评论。评论不改状态。"
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          <div className="mt-2 flex justify-end">
            <Button
              size="sm"
              disabled={busy || !body.trim()}
              onClick={async () => {
                setBusy(true);
                try {
                  await onAppend(body.trim());
                  setBody("");
                } catch (e) {
                  toast.error(errText(e));
                } finally {
                  setBusy(false);
                }
              }}
            >发送评论</Button>
          </div>
        </div>
      )}
    </div>
  );
}
