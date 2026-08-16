import { Link, createFileRoute } from "@tanstack/react-router";
import { ArrowRight } from "lucide-react";
import { getAllPosts, getStats, type Post } from "@/lib/posts";
import { ArticleCard } from "@/components/article-card";
import { TagCloud } from "@/components/tag-cloud";
import { ThemeScenery } from "@/components/theme-scenery";
import { NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/")({
  component: Index,
});

type Stats = ReturnType<typeof getStats>;

function StatItem({ label, value, unit }: { label: string; value: number; unit: string }) {
  return (
    <div className="flex items-baseline gap-3">
      <span className="font-mono text-4xl font-bold tracking-tight text-ink-1">{value}</span>
      <span className="font-mono text-xs text-ink-3">{unit}</span>
      <span className="font-serif text-sm text-ink-2">{label}</span>
    </div>
  );
}

/* =========================================================
   明亮 · 朝曦：网格化 / 上升式（登山步道般的导览层级）
   ========================================================= */
function DawnHome({ posts, stats }: { posts: Post[]; stats: Stats }) {
  const grid = posts.slice(0, 6);

  return (
    <div className="dawn-only">
      <section className="home-hero relative overflow-hidden">
        <ThemeScenery className="absolute inset-0" />
        <div className="dawn-hero-veil pointer-events-none absolute inset-0" />
        <img src="/assets/theme/sun.png" alt="" className="hero-celestial pointer-events-none absolute left-32 top-20 hidden h-40 w-auto opacity-90 lg:block" />
        <div className="ink-blot animate-ink-drift pointer-events-none absolute -right-32 -top-24 h-[420px] w-[520px]" />
        <div className="dawn-identity pointer-events-none absolute right-6 top-20 hidden select-none lg:flex">
          <div className="flex flex-col items-center gap-5">
            <img src="/assets/theme/sun.png" alt="" className="h-20 w-auto" />
            <span className="vertical-text font-calligraphy text-2xl text-ink-3">春山可望</span>
            <span className="font-mono text-[10px] tracking-[0.35em] text-ink-4 [writing-mode:vertical-rl]">SUNRISE / GROW</span>
          </div>
        </div>

        <div className="relative mx-auto max-w-6xl px-4 pb-28 pt-20 sm:px-6 sm:pb-36 sm:pt-28">
          <div className="reveal-up max-w-3xl">
            <div className="flex items-center gap-4">
              <p className="font-mono text-xs tracking-[0.35em] text-primary">JINGSHI / DAWN</p>
              <span className="hairline w-16" />
              <p className="font-mono text-xs tracking-[0.35em] text-ink-3">SPRING KNOWLEDGE GARDEN</p>
            </div>
            <h1 className="mt-6 font-calligraphy text-6xl leading-[1.2] tracking-wide sm:text-7xl">
              朝<span className="text-[var(--dawn-cinnabar)]">曦</span>入山
              <br />
              万物<span className="relative inline-block">生长
                <span className="absolute -bottom-1 left-0 h-[3px] w-full bg-primary/70" style={{ borderRadius: "50% 40% 60% 45% / 100%" }} />
              </span>
            </h1>
            <p className="mt-6 max-w-xl text-base leading-loose text-ink-2 sm:text-lg">
              把每一页笔记摊在春日的山光里。双链像山径，知识沿着太阳升起的方向，慢慢长成一座可行的山。
            </p>
          </div>

          {/* 登山步道：上升网格导览（三卡为唯一导航入口） */}
          <div className="reveal mt-10 grid gap-5 sm:grid-cols-3" data-reveal-delay="150">
            <Link
              to="/posts"
              className="dawn-trail-step group relative flex flex-col gap-3 rounded-xl bg-card/70 p-6 ring-1 ring-ink-3/15 backdrop-blur-sm max-sm:mt-0"
            >
              <span className="font-calligraphy text-3xl text-primary">壹</span>
              <h3 className="font-serif text-lg font-bold tracking-wide">入卷浏览</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">沿山径拾级，遍览春山文簿</p>
              <ArrowRight size={16} className="mt-2 text-primary transition-transform group-hover:translate-x-1" />
            </Link>
            <Link
              to="/graph"
              className="dawn-trail-step group relative flex flex-col gap-3 rounded-xl bg-card/70 p-6 ring-1 ring-ink-3/15 backdrop-blur-sm max-sm:mt-0"
              style={{ marginTop: "-18px" }}
            >
              <span className="font-calligraphy text-3xl text-primary">贰</span>
              <h3 className="font-serif text-lg font-bold tracking-wide">览星图谱</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">力导向星图，见双链脉络</p>
              <ArrowRight size={16} className="mt-2 text-primary transition-transform group-hover:translate-x-1" />
            </Link>
            <a
              href="#tags"
              onClick={(e) => {
                e.preventDefault();
                document.getElementById("tags")?.scrollIntoView({ behavior: "smooth", block: "start" });
              }}
              className="dawn-trail-step group relative flex flex-col gap-3 rounded-xl bg-card/70 p-6 ring-1 ring-ink-3/15 backdrop-blur-sm max-sm:mt-0"
              style={{ marginTop: "-36px" }}
            >
              <span className="font-calligraphy text-3xl text-primary">叁</span>
              <h3 className="font-serif text-lg font-bold tracking-wide">标签印谱</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">闲章集萃，触类旁通</p>
              <ArrowRight size={16} className="mt-2 text-primary transition-transform group-hover:translate-x-1" />
            </a>
          </div>

          {/* 题跋统计：无卡片，mono 数字 + 发丝线 */}
          <div className="reveal mt-14 max-w-2xl" data-reveal-delay="200">
            <div className="flex flex-wrap items-center gap-x-12 gap-y-4 py-6">
              <StatItem label="文章" value={stats.posts} unit="POSTS" />
              <span className="hairline-y hidden h-8 sm:block" />
              <StatItem label="标签" value={stats.tags} unit="TAGS" />
              <span className="hairline-y hidden h-8 sm:block" />
              <StatItem label="双链" value={stats.links} unit="LINKS" />
            </div>
          </div>
        </div>
      </section>

      {/* 最新文章：网格化 */}
      <section className="relative mx-auto max-w-6xl px-4 py-20 sm:px-6">
        <div className="reveal flex items-end justify-between">
          <div className="flex items-center gap-5">
            <span className="vertical-text text-lg text-ink-3">新墨</span>
            <div>
              <h2 className="font-serif text-3xl font-bold tracking-wide">最新文章</h2>
              <p className="mt-1 font-mono text-xs tracking-widest text-ink-3">RECENT · {posts.length} 篇</p>
            </div>
          </div>
          <Link to="/posts" className="story-link font-serif text-sm tracking-widest text-primary">
            阅全部
          </Link>
        </div>
        <div className="reveal mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3" data-reveal-delay="80">
          {grid.map((post) => (
            <ArticleCard key={post.slug} post={post} variant="default" />
          ))}
        </div>
      </section>

      {/* 标签印谱 */}
      <section id="tags" className="relative scroll-mt-20 overflow-hidden">
        <div className="ink-blot pointer-events-none absolute -left-24 bottom-0 h-64 w-72 opacity-60" />
        <div className="mx-auto max-w-5xl px-4 py-16 sm:px-6">
          <div className="reveal flex items-center gap-5">
            <span className="seal h-9 w-9 text-sm">签</span>
            <div>
              <h2 className="font-serif text-2xl font-bold tracking-wide">标签印谱</h2>
              <p className="mt-1 font-mono text-xs tracking-widest text-ink-3">TAGS · 点击钤印筛选</p>
            </div>
          </div>
          <div className="reveal mt-8" data-reveal-delay="100">
            <TagCloud />
          </div>
        </div>
      </section>
    </div>
  );
}

/* =========================================================
   黑暗 · 夜隐：流线型 / 下沉式（江河流淌般的视觉引导）
   ========================================================= */
function NightHome({ posts, stats }: { posts: Post[]; stats: Stats }) {
  const stream = posts.slice(0, 6);

  return (
    <div className="night-only">
      <section className="home-hero relative overflow-hidden">
        <ThemeScenery className="absolute inset-0" />
        <div className="night-hero-veil pointer-events-none absolute inset-0" />
        <img src="/assets/theme/moon.png" alt="" className="hero-celestial pointer-events-none absolute left-32 top-20 hidden h-40 w-auto opacity-90 lg:block" />
        <div className="night-constellation pointer-events-none absolute right-[8%] top-12 h-48 w-80" />
        <div className="night-identity pointer-events-none absolute bottom-24 right-4 hidden select-none lg:flex">
          <div className="flex items-end gap-5">
            <span className="font-mono text-[10px] tracking-[0.35em] text-ink-4 [writing-mode:vertical-rl]">MOON RIVER / REST</span>
            <div className="flex flex-col items-center gap-5">
              <img src="/assets/theme/moon.png" alt="" className="h-20 w-auto" />
              <span className="vertical-text font-calligraphy text-2xl tracking-[0.08em] text-ink-2">江月沉静</span>
            </div>
          </div>
        </div>

        <div className="relative mx-auto max-w-4xl px-4 pb-32 pt-20 sm:px-6 sm:pb-44 sm:pt-28">
          <div className="reveal-up max-w-2xl">
            <div className="night-copy">
              <div className="flex items-center gap-4">
                <p className="font-mono text-xs tracking-[0.35em] text-primary">JINGSHI / NIGHT</p>
                <span className="night-dash w-16" />
                <p className="font-mono text-xs tracking-[0.35em] text-ink-4">WINTER MOON ARCHIVE</p>
              </div>
              <h1 className="mt-6 font-calligraphy text-6xl leading-[1.2] tracking-[0.08em] text-ink-1 sm:text-7xl">
                夜<span className="text-primary">航</span>江上
                <br />
                万念<span className="relative inline-block">归流
                  <span className="absolute -bottom-1 left-0 h-px w-full bg-primary/80" />
                </span>
              </h1>
              <p className="mt-6 max-w-xl text-base leading-loose text-ink-2 sm:text-lg">
                让文章成为冬夜江面上的微光。沿着反向链接回望，思想不再喧哗，只在月色与水声里彼此照见。
              </p>
              <div className="mt-10 flex flex-wrap gap-4">
                <Link
                  to="/posts"
                  className="fx-ripple inline-flex items-center gap-2 rounded-sm bg-primary px-6 py-3 font-serif text-sm font-medium tracking-widest text-primary-foreground"
                >
                  入卷浏览 <ArrowRight size={16} />
                </Link>
                <Link
                  to="/graph"
                  className="fx-ripple inline-flex items-center gap-2 rounded-sm border border-ink-4/50 px-6 py-3 font-serif text-sm font-medium tracking-widest text-ink-1"
                >
                  览星图谱
                </Link>
              </div>
            </div>
          </div>

          {/* 下沉式统计：低信息密度 */}
          <div className="reveal mt-12 max-w-md" data-reveal-delay="200">
            <div className="grid grid-cols-3 gap-6 py-6">
              <div><span className="font-mono text-3xl text-ink-1">{stats.posts}</span><span className="mt-1 block font-mono text-[10px] tracking-[0.2em] text-ink-4">MOON NOTES</span></div>
              <div><span className="font-mono text-3xl text-primary">{stats.links}</span><span className="mt-1 block font-mono text-[10px] tracking-[0.2em] text-ink-4">TIDAL LINKS</span></div>
              <div><span className="font-mono text-3xl text-ink-1">{stats.tags}</span><span className="mt-1 block font-mono text-[10px] tracking-[0.2em] text-ink-4">COLD TAGS</span></div>
            </div>
          </div>
        </div>
      </section>

      {/* 最新文章：流线型下沉串联 */}
      <section className="relative mx-auto max-w-3xl px-4 py-20 sm:px-6">
        <div className="reveal flex items-center gap-5">
          <span className="vertical-text text-lg text-ink-3">沉墨</span>
          <div>
            <h2 className="font-serif text-3xl font-bold tracking-wide">江流文章</h2>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-4">RIVER OF READING · {stream.length}</p>
          </div>
        </div>
        <NightWaves className="reveal mt-5 h-9 w-72 text-ink-4/60" data-reveal-delay="60" />

        <div className="relative mt-12 pl-8">
          <div className="night-flow-line pointer-events-none absolute left-2 top-2 bottom-2" />
          {stream.map((post, i) => (
            <div key={post.slug} className="reveal relative pb-10" data-reveal-delay={`${i * 70}`}>
              <span className="night-flow-node absolute -left-[1.62rem] top-3 h-3 w-3 rounded-full bg-night-moon" />
              <ArticleCard post={post} variant="compact" />
            </div>
          ))}
        </div>
      </section>

      {/* 标签：淡如江雾 */}
      <section className="relative overflow-hidden opacity-90">
        <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
          <div className="reveal flex items-center gap-5">
            <span className="seal h-9 w-9 text-sm">签</span>
            <div>
              <h2 className="font-serif text-2xl font-bold tracking-wide">寒江印谱</h2>
              <p className="mt-1 font-mono text-xs tracking-widest text-ink-4">TAGS · 轻触浮起</p>
            </div>
          </div>
          <div className="reveal mt-8" data-reveal-delay="100">
            <TagCloud />
          </div>
        </div>
      </section>
    </div>
  );
}

function Index() {
  const posts = getAllPosts();
  const stats = getStats();

  return (
    <div>
      <DawnHome posts={posts} stats={stats} />
      <NightHome posts={posts} stats={stats} />
    </div>
  );
}
