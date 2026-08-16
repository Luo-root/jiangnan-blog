import { useState, type ComponentPropsWithoutRef, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import rehypeHighlight from "rehype-highlight";
import { Check, Copy } from "lucide-react";
import { slugifyHeading } from "@/lib/posts";

function PreWithCopy({ children, ...props }: ComponentPropsWithoutRef<"pre">) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    const pre = (props as { ref?: React.Ref<HTMLPreElement> }).ref as HTMLPreElement | undefined;
    const codeEl = pre?.querySelector("code") ?? document.activeElement?.querySelector("code");
    const text = codeEl?.textContent || "";
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  return (
    <div className="group relative">
      <button
        onClick={handleCopy}
        aria-label="复制代码"
        className="absolute right-3 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-md border border-border/50 bg-background/80 text-muted-foreground opacity-0 backdrop-blur transition-all hover:text-foreground group-hover:opacity-100"
      >
        {copied ? <Check size={14} /> : <Copy size={14} />}
      </button>
      <pre {...props}>{children}</pre>
    </div>
  );
}

function HeadingWithId({
  level,
  children,
}: {
  level: 2 | 3;
  children: ReactNode;
}) {
  const text = extractText(children);
  const id = slugifyHeading(text);
  const Tag = (`h${level}` as "h2" | "h3");
  return <Tag id={id}>{children}</Tag>;
}

function extractText(node: ReactNode): string {
  if (typeof node === "string") return node;
  if (Array.isArray(node)) return node.map(extractText).join("");
  if (node && typeof node === "object" && "props" in node) {
    return extractText((node as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}

interface MarkdownRendererProps {
  content: string;
}

export function MarkdownRenderer({ content }: MarkdownRendererProps) {
  return (
    <div className="prose-blog">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex, [rehypeHighlight, { detect: true, ignoreMissing: true }]]}
        components={{
          pre: PreWithCopy,
          h2: ({ children }) => <HeadingWithId level={2}>{children}</HeadingWithId>,
          h3: ({ children }) => <HeadingWithId level={3}>{children}</HeadingWithId>,
          a: ({ href, children }) => (
            <a href={href} target={href?.startsWith("http") ? "_blank" : undefined} rel={href?.startsWith("http") ? "noopener noreferrer" : undefined}>
              {children}
            </a>
          ),
          img: ({ src, alt }) => (
            <img src={src} alt={alt} loading="lazy" />
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
