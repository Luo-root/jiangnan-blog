type Line = { kind: "same" | "del" | "add"; text: string };

function splitLines(s: string) {
  return (s || "").replace(/\r\n/g, "\n").split("\n");
}

export function diffLines(before: string, after: string): Line[] {
  const a = splitLines(before);
  const b = splitLines(after);
  const n = a.length;
  const m = b.length;
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = a[i] === b[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  const out: Line[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ kind: "same", text: a[i] });
      i++;
      j++;
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      out.push({ kind: "del", text: a[i] });
      i++;
    } else {
      out.push({ kind: "add", text: b[j] });
      j++;
    }
  }
  while (i < n) out.push({ kind: "del", text: a[i++] });
  while (j < m) out.push({ kind: "add", text: b[j++] });
  return out;
}

export function DiffViewer({ before, after, leftLabel = "原文（红=删）", rightLabel = "变更后（绿=增）" }: {
  before: string;
  after: string;
  leftLabel?: string;
  rightLabel?: string;
}) {
  const lines = diffLines(before, after);
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <div className="grid grid-cols-2 border-b border-border bg-muted font-mono text-[10px] uppercase tracking-wider text-ink-4">
        <div className="px-3 py-2">{leftLabel}</div>
        <div className="border-l border-border px-3 py-2">{rightLabel}</div>
      </div>
      <div className="grid max-h-[480px] grid-cols-2 overflow-auto font-mono text-[12px] leading-6">
        <pre className="whitespace-pre-wrap p-3">
          {lines.filter((l) => l.kind !== "add").map((l, i) => (
            <div key={"l"+i} className={l.kind === "del" ? "bg-destructive/10 text-destructive" : ""}>
              {l.kind === "del" ? "- " : "  "}{l.text}
            </div>
          ))}
        </pre>
        <pre className="whitespace-pre-wrap border-l border-border p-3">
          {lines.filter((l) => l.kind !== "del").map((l, i) => (
            <div key={"r"+i} className={l.kind === "add" ? "bg-accent/10 text-accent" : ""}>
              {l.kind === "add" ? "+ " : "  "}{l.text}
            </div>
          ))}
        </pre>
      </div>
    </div>
  );
}

export function UnifiedDiff({ text }: { text: string }) {
  return (
    <pre className="h-full min-h-0 overflow-auto whitespace-pre-wrap rounded-xl border border-border bg-card p-4 font-mono text-[13px] leading-6">
      {(text || "（无 diff）").split("\n").map((line, i) => {
        const cls = line.startsWith("+") && !line.startsWith("+++")
          ? "text-accent"
          : line.startsWith("-") && !line.startsWith("---")
            ? "text-destructive"
            : line.startsWith("@@")
              ? "text-primary"
              : "text-ink-2";
        return <div key={i} className={cls}>{line}</div>;
      })}
    </pre>
  );
}
