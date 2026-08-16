import { useState, useEffect, useRef } from "react";
import { Link } from "@tanstack/react-router";
import { Search, FileText, X } from "lucide-react";
import { searchPosts, type SearchResult } from "@/lib/search";

interface SearchDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SearchDialog({ open, onOpenChange }: SearchDialogProps) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setQuery("");
      setResults([]);
    }
  }, [open]);

  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    const timer = setTimeout(() => {
      setResults(searchPosts(query));
    }, 150);
    return () => clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onOpenChange(false);
    };
    if (open) window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onOpenChange]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center bg-black/40 backdrop-blur-md pt-[14vh] px-4"
      onClick={() => onOpenChange(false)}
    >
      <div
        className="search-panel w-full max-w-2xl overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 border-b border-border bg-card px-4">
          <Search size={18} className="text-muted-foreground" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索文章标题、标签、内容…"
            className="flex-1 bg-transparent py-4 text-sm outline-none focus-visible:outline-none placeholder:text-muted-foreground"
          />
          <button
            onClick={() => onOpenChange(false)}
            className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-secondary"
          >
            <X size={16} />
          </button>
        </div>

        <div className="max-h-[60vh] overflow-y-auto">
          {query.trim() === "" ? (
            <div className="px-4 py-12 text-center text-sm text-muted-foreground">
              输入关键词开始搜索
            </div>
          ) : results.length === 0 ? (
            <div className="px-4 py-12 text-center text-sm text-muted-foreground">
              未找到与「{query}」相关的文章
            </div>
          ) : (
            <ul className="py-2">
              {results.map((r) => (
                <li key={r.slug}>
                  <Link
                    to="/posts/$slug"
                    params={{ slug: r.slug }}
                    onClick={() => onOpenChange(false)}
                    className="search-result flex items-start gap-3 px-4 py-3 hover:bg-secondary"
                  >
                    <FileText size={16} className="mt-0.5 shrink-0 text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <div className="font-medium text-foreground">{r.title}</div>
                      <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">
                        {r.matchSnippet}
                      </p>
                      <div className="mt-1 flex gap-1.5">
                        {r.tags.slice(0, 3).map((t) => (
                          <span key={t} className="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
                            {t}
                          </span>
                        ))}
                      </div>
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
