import { useState, useMemo } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { getAllPosts, getAllTags } from "@/lib/posts";
import { ArticleCard } from "@/components/article-card";
import { ThemeScenery } from "@/components/theme-scenery";
import { DawnClouds, NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/posts")({
  component: PostsList,
});

function PostsList() {
  const allPosts = getAllPosts();
  const tags = getAllTags();
  const [query, setQuery] = useState("");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const [sortBy, setSortBy] = useState<"date" | "title">("date");

  const filtered = useMemo(() => {
    let result = allPosts;
    if (activeTag) result = result.filter((p) => p.tags.includes(activeTag));
    if (query.trim()) {
      const q = query.toLowerCase();
      result = result.filter(
        (p) =>
          p.title.toLowerCase().includes(q) ||
          p.excerpt.toLowerCase().includes(q) ||
          p.tags.some((t) => t.toLowerCase().includes(q))
      );
    }
    if (sortBy === "title") {
      result = [...result].sort((a, b) => a.title.localeCompare(b.title));
    }
    return result;
  }, [allPosts, activeTag, query, sortBy]);

  return (
    <div className="posts-page relative mx-auto max-w-6xl overflow-hidden px-4 py-12 sm:px-6">
      <ThemeScenery className="posts-scenery absolute -inset-x-16 top-0 h-72 opacity-30" />
      <div className="reveal relative">
        <div className="dawn-copy flex items-center gap-5">
          <img src="/assets/theme/sun.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-wide">春山文簿</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-3">DAWN ARCHIVE · 共 {allPosts.length} 卷</p>
          </div>
        </div>
        <div className="night-copy flex items-center gap-5">
          <img src="/assets/theme/moon.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-[0.08em]">月下藏卷</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-4">NIGHT ARCHIVE · {allPosts.length} MOON NOTES</p>
          </div>
        </div>
        <div className="dawn-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <div className="night-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-night-moon/35 to-transparent" />
        {/* 页首意象带：明=云纹过山，暗=波纹横江 */}
        <DawnClouds className="dawn-only mt-3 h-10 w-full max-w-xl text-ink-4/60" />
        <NightWaves className="night-only mt-3 h-9 w-full max-w-xl text-ink-4/60" />
      </div>

      <div className="reveal relative mt-8 flex flex-col gap-4 sm:flex-row sm:items-center" data-reveal-delay="100">
        <div className="relative flex-1">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <span className="dawn-only absolute -top-6 left-0 font-calligraphy text-sm text-ink-4">在春风里检索旧笺</span>
          <span className="night-copy absolute -top-6 left-0 font-calligraphy text-sm tracking-[0.08em] text-ink-4">沿江寻找一页微光</span>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="文章 · 标题 · 标签"
            className="input-themed w-full rounded-lg border-0 bg-card/60 py-2 pl-9 pr-3 text-sm outline-none"
          />
        </div>
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as "date" | "title")}
          className="input-themed rounded-lg border-0 bg-card/60 px-3 py-2 text-sm outline-none"
        >
          <option value="date">按日期</option>
          <option value="title">按标题</option>
        </select>
      </div>

      <div className="reveal mt-4 flex flex-wrap gap-2" data-reveal-delay="150">
        <button
          onClick={() => setActiveTag(null)}
          className={`filter-pill rounded-full px-3 py-1 text-xs ${
            !activeTag
              ? "filter-pill--active bg-primary text-primary-foreground shadow-[0_4px_18px_-6px_var(--color-primary)]"
              : "bg-card/60 text-secondary-foreground hover:bg-card"
          }`}
        >
          全部
        </button>
        {tags.map(({ tag, count }) => (
          <button
            key={tag}
            onClick={() => setActiveTag(activeTag === tag ? null : tag)}
            className={`filter-pill rounded-full px-3 py-1 text-xs ${
              activeTag === tag
                ? "filter-pill--active bg-primary text-primary-foreground shadow-[0_4px_18px_-6px_var(--color-primary)]"
                : "bg-card/60 text-secondary-foreground hover:bg-card"
            }`}
          >
            {tag} ({count})
          </button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center gap-4 py-20 text-center">
          <img src="/assets/theme/sun.png" alt="" className="dawn-only h-16 w-auto opacity-60" />
          <img src="/assets/theme/moon.png" alt="" className="night-copy h-16 w-auto opacity-60" />
          <p className="text-muted-foreground">未找到匹配的文章</p>
          <button
            onClick={() => { setQuery(""); setActiveTag(null); }}
            className="text-sm text-primary hover:underline"
          >
            清除筛选
          </button>
        </div>
      ) : (
        <div className="mt-8">
          {filtered.map((post, i) => (
            <div key={post.slug} className="reveal" data-reveal-delay={`${Math.min(i * 60, 400)}`}>
              <ArticleCard post={post} variant="compact" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
