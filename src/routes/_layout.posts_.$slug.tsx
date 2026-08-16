import { Link, createFileRoute } from "@tanstack/react-router";
import { Calendar, Clock, ArrowLeft } from "lucide-react";
import { getPostBySlug } from "@/lib/posts";
import { MarkdownRenderer } from "@/components/markdown-renderer";
import { Backlinks } from "@/components/backlinks";
import { TOC } from "@/components/toc";
import { ThemeScenery } from "@/components/theme-scenery";
import { NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/posts_/$slug")({
  component: PostDetail,
});

function formatDate(date: string): string {
  if (!date) return "";
  const d = new Date(date);
  if (isNaN(d.getTime())) return date;
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

function PostDetail() {
  const { slug } = Route.useParams();
  const post = getPostBySlug(slug);

  if (!post) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-20 text-center">
        <p className="text-muted-foreground">文章不存在</p>
        <Link to="/posts" className="mt-4 inline-block text-primary hover:underline">
          返回文章列表
        </Link>
      </div>
    );
  }

  return (
    <article className="post-detail relative mx-auto max-w-6xl overflow-x-clip px-4 py-12 sm:px-6">
      <ThemeScenery className="post-scenery absolute -inset-x-24 top-0 h-[360px] opacity-20" />
      <Link
        to="/posts"
        className="reveal mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft size={14} /> 返回文章列表
      </Link>

      <div className="reveal relative" data-reveal-delay="50">
        <div className="ink-blot dawn-only pointer-events-none absolute -left-28 -top-24 h-80 w-80" />
        <div className="post-night-orbit night-only pointer-events-none absolute right-0 top-0 h-64 w-64" />
        <div className="dawn-only mb-5 flex items-center gap-3 font-mono text-[10px] tracking-[0.3em] text-ink-4"><img src="/assets/theme/sun.png" alt="" className="h-4 w-auto" /><span>SPRING NOTE / SUNLIT EDITION</span></div>
        <div className="night-copy mb-5 flex items-center gap-3 font-mono text-[10px] tracking-[0.3em] text-ink-4"><img src="/assets/theme/moon.png" alt="" className="h-4 w-auto" /><span>WINTER NOTE / MOONLIT EDITION</span></div>
        {post.cover && (
          <div className="mb-10 aspect-[1200/630] w-full overflow-hidden rounded-sm">
            <img src={post.cover} alt={post.title} className="h-full w-full object-cover" />
          </div>
        )}

        <div className="mb-3 flex flex-wrap gap-2">
          {post.tags.map((tag) => (
            <Link
              key={tag}
              to="/tags/$tag"
              params={{ tag }}
              className="rounded-full bg-card/60 px-2.5 py-0.5 font-mono text-xs text-secondary-foreground transition-all hover:bg-primary hover:text-primary-foreground hover:shadow-[0_4px_18px_-6px_var(--color-primary)]"
            >
              {tag}
            </Link>
          ))}
        </div>
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="dawn-only mb-2 font-mono text-[10px] tracking-[0.32em] text-primary">春日 · 山中一页</p>
            <p className="night-copy mb-2 font-mono text-[10px] tracking-[0.32em] text-primary">冬夜 · 江上回声</p>
            <h1 className="font-calligraphy text-4xl leading-snug tracking-wide sm:text-5xl">{post.title}</h1>
          </div>
        </div>
        <div className="mt-4 flex items-center gap-4 font-mono text-xs tracking-wider text-muted-foreground">
          <span className="flex items-center gap-1">
            <Calendar size={14} /> {formatDate(post.date)}
          </span>
          <span className="flex items-center gap-1">
            <Clock size={14} /> {post.readingTime} 分钟阅读
          </span>
        </div>
        {/* 卷首与正文之间：朝曦软笔触，夜间月光水波 */}
        <div className="dawn-only mt-10 h-px bg-gradient-to-r from-transparent via-primary/30 to-transparent" />
        <div className="night-only mt-10 h-px bg-gradient-to-r from-transparent via-night-moon/30 to-transparent" />
      </div>

      <div className="reveal mt-10 flex gap-12" data-reveal-delay="100">
        <div className="min-w-0 flex-1">
          <MarkdownRenderer content={post.content} />
          {/* 卷尾收束：明=花枝压卷，暗=波纹送月 */}
          <div className="night-only mt-16">
            <NightWaves className="h-10 w-full text-ink-4/70" />
          </div>
          <Backlinks slug={post.slug} />
        </div>
        <aside className="hidden w-56 shrink-0 lg:block">
          <div className="sticky top-24">
            <TOC content={post.content} />
          </div>
        </aside>
      </div>
    </article>
  );
}
