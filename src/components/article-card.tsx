import { Link } from "@tanstack/react-router";
import { cva, type VariantProps } from "class-variance-authority";
import { Calendar, Clock, ArrowRight } from "lucide-react";
import type { Post } from "@/lib/posts";

const cardVariants = cva("group relative flex", {
  variants: {
    variant: {
      default: "flex-col gap-4 overflow-hidden p-5",
      compact: "flex-col gap-1.5 px-4 py-5 transition-colors sm:flex-row sm:items-baseline sm:gap-6",
      row: "flex-col gap-1.5 px-4 py-6 transition-colors sm:grid sm:grid-cols-[3rem_1fr_auto] sm:items-baseline sm:gap-6",
      featured: "flex-col gap-8 overflow-hidden p-5 md:grid md:grid-cols-[5fr_7fr] md:items-center md:gap-12 md:p-10",
    },
  },
  defaultVariants: { variant: "default" },
});

interface ArticleCardProps extends VariantProps<typeof cardVariants> {
  post: Post;
  index?: number;
}

/* 国风墨色封面：黛青 / 松烟 / 赭石 / 藤黄 / 紫檀 / 石绿 */
const INK_COVERS = [
  "from-[#3d5a73] to-[#1f3245]",
  "from-[#3c423c] to-[#1c211c]",
  "from-[#6e4a36] to-[#38241a]",
  "from-[#7d6428] to-[#42351a]",
  "from-[#503c52] to-[#2a1f2c]",
  "from-[#3f5d4e] to-[#1f3027]",
];

const CN_NUM = ["壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖", "拾", "拾壹", "拾贰"];

function formatDate(date: string): string {
  if (!date) return "";
  const d = new Date(date);
  if (isNaN(d.getTime())) return date;
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

function CoverImage({ src, alt, className }: { src: string; alt: string; className?: string }) {
  if (!src) {
    const hash = alt.charCodeAt(0) % INK_COVERS.length;
    return (
      <div className={`relative flex items-center justify-center overflow-hidden bg-gradient-to-br ${INK_COVERS[hash]} ${className}`}>
        {/* 纸纹 + 墨晕 */}
        <div className="paper-grain absolute inset-0 opacity-25" />
        <div
          className="absolute inset-0"
          style={{
            background:
              "radial-gradient(ellipse 70% 60% at 30% 25%, rgba(255,255,255,0.10), transparent 65%), radial-gradient(ellipse 60% 55% at 75% 80%, rgba(0,0,0,0.28), transparent 70%)",
          }}
        />
        <span className="relative font-serif text-6xl font-bold text-white/85 drop-shadow-lg">
          {alt.charAt(0) || "·"}
        </span>
        <span className="absolute bottom-3 right-3 h-6 w-6 rounded-[3px] border border-white/40 font-serif text-[11px] leading-6 text-center text-white/70">
          {alt.charAt(1) || "识"}
        </span>
      </div>
    );
  }
  return (
    <img
      src={src}
      alt={alt}
      loading="lazy"
      className={`${className} object-cover transition-transform duration-500 group-hover:scale-105`}
    />
  );
}

function MetaRow({ post, className = "" }: { post: Post; className?: string }) {
  return (
    <div className={`flex items-center gap-4 font-mono text-xs text-muted-foreground ${className}`}>
      <span className="flex items-center gap-1">
        <Calendar size={12} /> {formatDate(post.date)}
      </span>
      <span className="flex items-center gap-1">
        <Clock size={12} /> {post.readingTime} 分钟
      </span>
    </div>
  );
}

function TagList({ tags, max = 3 }: { tags: string[]; max?: number }) {
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1 font-mono text-xs text-ink-3">
      {tags.slice(0, max).map((tag) => (
        <span key={tag}>#{tag}</span>
      ))}
    </div>
  );
}

export function ArticleCard({ post, variant = "default", index }: ArticleCardProps) {
  return (
    <Link to="/posts/$slug" params={{ slug: post.slug }} className={`${cardVariants({ variant })} theme-article-card`}>
      {variant === "default" && (
        <>
          <div className="relative aspect-[16/10] w-full overflow-hidden rounded-2xl">
            <CoverImage src={post.cover} alt={post.title} className="h-full w-full" />
          </div>
          <div className="flex flex-1 flex-col gap-2.5 px-1">
            <TagList tags={post.tags} />
            <h3 className="font-serif text-xl font-bold leading-snug tracking-wide text-foreground transition-colors group-hover:text-primary">
              {post.title}
            </h3>
            <p className="line-clamp-2 text-sm leading-relaxed text-muted-foreground">
              {post.excerpt}
            </p>
            <div className="mt-auto pt-3">
              <MetaRow post={post} />
            </div>
          </div>
        </>
      )}

      {variant === "compact" && (
        <>
          <div className="min-w-0 flex-1">
            <h3 className="font-serif text-lg font-bold leading-snug tracking-wide text-foreground transition-colors group-hover:text-primary">
              {post.title}
            </h3>
            <p className="mt-1 line-clamp-1 text-sm text-muted-foreground">{post.excerpt}</p>
          </div>
          <MetaRow post={post} className="shrink-0" />
        </>
      )}

      {variant === "row" && (
        <>
          <span className="hidden font-mono text-sm tracking-widest text-ink-4 sm:block">
            {CN_NUM[(index ?? 0) % CN_NUM.length]}
          </span>
          <div className="min-w-0">
            <h3 className="font-serif text-xl font-bold leading-snug tracking-wide text-foreground transition-colors group-hover:text-primary">
              {post.title}
            </h3>
            <p className="mt-1.5 line-clamp-1 text-sm text-muted-foreground">{post.excerpt}</p>
            <div className="mt-2">
              <TagList tags={post.tags} />
            </div>
          </div>
          <div className="flex items-center gap-3 sm:flex-col sm:items-end sm:gap-1.5">
            <MetaRow post={post} />
            <ArrowRight
              size={14}
              className="text-primary opacity-0 transition-all group-hover:translate-x-0.5 group-hover:opacity-100"
            />
          </div>
        </>
      )}

      {variant === "featured" && (
        <>
          <div className="relative aspect-[16/10] w-full overflow-hidden rounded-2xl">
            <CoverImage src={post.cover} alt={post.title} className="h-full w-full" />
          </div>
          <div className="flex flex-1 flex-col gap-4">
            <p className="font-mono text-xs tracking-[0.3em] text-primary">卷首 · FEATURED</p>
            <h3 className="font-serif text-3xl font-bold leading-tight tracking-wide transition-colors group-hover:text-primary">
              {post.title}
            </h3>
            <p className="line-clamp-3 text-sm leading-relaxed text-muted-foreground">
              {post.excerpt}
            </p>
            <TagList tags={post.tags} max={4} />
            <div className="mt-auto flex items-center justify-between pt-4">
              <MetaRow post={post} />
              <span className="story-link inline-flex items-center gap-1 font-serif text-sm text-primary">
                展卷阅读 <ArrowRight size={14} />
              </span>
            </div>
          </div>
        </>
      )}
    </Link>
  );
}
