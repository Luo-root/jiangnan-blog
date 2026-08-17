import { Link, createFileRoute } from "@tanstack/react-router";
import { Calendar, ArrowUpRight } from "lucide-react";
import { getAllPosts } from "@/lib/posts";
import { ThemeScenery } from "@/components/theme-scenery";
import { DawnClouds, NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/archive")({
  component: ArchiveTimeline,
});

/** 解析 date 字符串（YYYY-MM-DD） */
function parseDate(s: string): Date | null {
  if (!s) return null;
  const d = new Date(s);
  return isNaN(d.getTime()) ? null : d;
}

const MONTH_LABELS = ["壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖", "拾", "拾壹", "拾贰"] as const;

function groupByYearMonth(posts: ReturnType<typeof getAllPosts>) {
  // posts 已经按 date 倒序排好，按 "年-月" 分组
  const groups: { year: number; month: number; posts: typeof posts }[] = [];
  const idx = new Map<string, number>();
  for (const p of posts) {
    const d = parseDate(p.date);
    if (!d) continue;
    const y = d.getFullYear();
    const m = d.getMonth() + 1;
    const key = `${y}-${m}`;
    let i = idx.get(key);
    if (i === undefined) {
      i = groups.length;
      idx.set(key, i);
      groups.push({ year: y, month: m, posts: [] });
    }
    groups[i].posts.push(p);
  }
  return groups;
}

function ArchiveTimeline() {
  const posts = getAllPosts();
  const groups = groupByYearMonth(posts);
  const yearCount = new Set(groups.map((g) => g.year)).size;

  return (
    <div className="archive-page relative mx-auto max-w-5xl overflow-hidden px-4 py-12 sm:px-6">
      <ThemeScenery className="posts-scenery absolute -inset-x-16 top-0 h-72 opacity-30" />

      {/* 页头 */}
      <div className="reveal relative">
        <div className="dawn-copy flex items-center gap-5">
          <img src="/assets/theme/sun.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-wide">年岁归档</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-3">
              DAWN ARCHIVE · {yearCount} 年 {posts.length} 卷
            </p>
          </div>
        </div>
        <div className="night-copy flex items-center gap-5">
          <img src="/assets/theme/moon.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-[0.08em]">岁时藏卷</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-4">
              NIGHT ARCHIVE · {yearCount} YEARS · {posts.length} NOTES
            </p>
          </div>
        </div>
        <div className="dawn-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <div className="night-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-night-moon/35 to-transparent" />
        <DawnClouds className="dawn-only mt-3 h-10 w-full max-w-xl text-ink-4/60" />
        <NightWaves className="night-only mt-3 h-9 w-full max-w-xl text-ink-4/60" />
        <p className="mt-6 max-w-2xl text-sm leading-relaxed text-muted-foreground">
          按时间倒序排列所有文章。日期取自 Obsidian 笔记 frontmatter。
        </p>
      </div>

      {/* 时间轴 */}
      {groups.length === 0 ? (
        <div className="mt-16 text-center text-muted-foreground">暂无归档</div>
      ) : (
        <div className="relative mt-12">
          {/* 中线：明=石青，暗=黛蓝 */}
          <div className="dawn-only absolute bottom-0 left-[3.5rem] top-0 w-px bg-gradient-to-b from-primary/40 via-primary/15 to-transparent" />
          <div className="night-only absolute bottom-0 left-[3.5rem] top-0 w-px bg-gradient-to-b from-night-moon/40 via-night-moon/15 to-transparent" />

          {groups.map((g, gi) => (
            <section key={`${g.year}-${g.month}`} className="reveal relative pb-10" data-reveal-delay={`${Math.min(gi * 40, 400)}`}>
              {/* 年月标头 */}
              <div className="relative mb-4 flex items-baseline gap-3 pl-0">
                <span className="dawn-copy w-14 shrink-0 text-right font-mono text-2xl font-bold text-primary">
                  {g.year}
                </span>
                <span className="night-copy w-14 shrink-0 text-right font-mono text-2xl font-bold text-night-moon">
                  {g.year}
                </span>
                <span className="font-calligraphy text-lg text-ink-3">
                  · {MONTH_LABELS[g.month - 1]}月
                </span>
                <span className="font-mono text-[10px] text-ink-4">{g.posts.length} 篇</span>
                <span className="hairline flex-1" />
              </div>

              {/* 列表 */}
              <ul className="space-y-1.5">
                {g.posts.map((p) => {
                  const d = parseDate(p.date);
                  const day = d ? d.getDate() : "";
                  return (
                    <li key={p.slug} className="relative pl-0">
                      <Link
                        to="/posts/$slug"
                        params={{ slug: p.slug }}
                        className="group flex items-baseline gap-4 rounded-lg px-3 py-2.5 transition-all hover:bg-card/60 sm:pl-0"
                      >
                        {/* 日期圆点 + 日 */}
                        <span className="relative flex w-14 shrink-0 items-center justify-end gap-2">
                          <span className="font-mono text-sm tabular-nums text-ink-3 group-hover:text-primary">
                            {String(day).padStart(2, "0")}
                          </span>
                          <span className="dawn-only h-1.5 w-1.5 rounded-full bg-primary/70 ring-2 ring-primary/20 group-hover:ring-primary/40" />
                          <span className="night-only h-1.5 w-1.5 rounded-full bg-night-moon/70 ring-2 ring-night-moon/20 group-hover:ring-night-moon/40" />
                        </span>

                        {/* 标题 */}
                        <span className="min-w-0 flex-1 truncate font-serif text-sm text-foreground transition-colors group-hover:text-primary">
                          {p.title}
                        </span>

                        {/* 标签 + 时长（仅日=有数据时显示） */}
                        <span className="hidden items-center gap-3 sm:flex">
                          {p.tags.slice(0, 2).map((t) => (
                            <span key={t} className="rounded-full bg-card/60 px-2 py-0.5 font-mono text-[10px] text-ink-3">
                              {t}
                            </span>
                          ))}
                          {p.readingTime > 0 && (
                            <span className="flex items-center gap-1 font-mono text-[10px] text-ink-4">
                              <Calendar size={10} />
                              {p.readingTime}min
                            </span>
                          )}
                          <ArrowUpRight
                            size={12}
                            className="text-muted-foreground opacity-0 transition-all group-hover:translate-x-0.5 group-hover:-translate-y-0.5 group-hover:opacity-100"
                          />
                        </span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
