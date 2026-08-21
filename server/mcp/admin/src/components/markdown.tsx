import type { ReactNode } from "react";

function inline(text: string): ReactNode[] {
  const out: ReactNode[] = [];
  const re = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\*[^*]+\*)|(\[[^\]]+\]\([^)]+\))/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const tok = m[0];
    if (tok.startsWith("`")) {
      out.push(<code key={i++} className="rounded bg-muted px-1 font-mono text-[12px]">{tok.slice(1, -1)}</code>);
    } else if (tok.startsWith("**")) {
      out.push(<strong key={i++}>{tok.slice(2, -2)}</strong>);
    } else if (tok.startsWith("*")) {
      out.push(<em key={i++}>{tok.slice(1, -1)}</em>);
    } else {
      const lm = tok.match(/^\[([^\]]+)\]\(([^)]+)\)$/);
      out.push(<a key={i++} href={lm?.[2]} className="text-primary underline" target="_blank" rel="noreferrer">{lm?.[1]}</a>);
    }
    last = m.index + tok.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

export function MarkdownPreview({ text }: { text: string }) {
  const src = text || "";
  const blocks: ReactNode[] = [];
  const lines = src.replace(/\r\n/g, "\n").split("\n");
  let i = 0;
  let k = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (line.startsWith("```")) {
      const buf: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        buf.push(lines[i]);
        i++;
      }
      i++;
      blocks.push(<pre key={k++} className="overflow-auto rounded-lg bg-muted p-3 font-mono text-[12px] leading-6">{buf.join("\n")}</pre>);
      continue;
    }
    if (/^#{1,6}\s/.test(line)) {
      const level = line.match(/^#+/)![0].length;
      const cls = level <= 2 ? "text-base font-semibold text-ink-1" : "text-sm font-semibold text-ink-1";
      blocks.push(<div key={k++} className={cls}>{inline(line.replace(/^#+\s/, ""))}</div>);
      i++;
      continue;
    }
    if (/^\s*[-*]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*[-*]\s+/, ""));
        i++;
      }
      blocks.push(
        <ul key={k++} className="list-disc space-y-1 pl-5">
          {items.map((it, n) => <li key={n}>{inline(it)}</li>)}
        </ul>,
      );
      continue;
    }
    if (/^\s*\d+\.\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*\d+\.\s+/, ""));
        i++;
      }
      blocks.push(
        <ol key={k++} className="list-decimal space-y-1 pl-5">
          {items.map((it, n) => <li key={n}>{inline(it)}</li>)}
        </ol>,
      );
      continue;
    }
    if (line.trim() === "") {
      i++;
      continue;
    }
    const para: string[] = [];
    while (i < lines.length && lines[i].trim() !== "" && !lines[i].startsWith("```") && !/^#{1,6}\s/.test(lines[i]) && !/^\s*[-*]\s+/.test(lines[i]) && !/^\s*\d+\.\s+/.test(lines[i])) {
      para.push(lines[i]);
      i++;
    }
    blocks.push(<p key={k++}>{inline(para.join(" "))}</p>);
  }
  if (blocks.length === 0) return <p className="text-sm text-ink-4">（空）</p>;
  return <div className="space-y-2 text-sm leading-7 text-ink-1">{blocks}</div>;
}
