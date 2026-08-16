import { Link, createFileRoute } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { getPostsByTag } from "@/lib/posts";
import { ArticleCard } from "@/components/article-card";

export const Route = createFileRoute("/_layout/tags_/$tag")({
  component: TagPage,
});

function TagPage() {
  const { tag } = Route.useParams();
  const posts = getPostsByTag(tag);

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <Link
        to="/posts"
        className="reveal mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft size={14} /> 返回文章列表
      </Link>

      <div className="reveal" data-reveal-delay="50">
        <div className="flex items-center gap-3">
          <span className="rounded-full bg-primary/10 px-3 py-1 text-sm font-medium text-primary">
            #{tag}
          </span>
          <h1 className="font-serif text-3xl font-bold">{tag}</h1>
        </div>
        <p className="mt-2 text-muted-foreground">共 {posts.length} 篇文章</p>
      </div>

      {posts.length === 0 ? (
        <div className="py-20 text-center text-muted-foreground">该标签下暂无文章</div>
      ) : (
        <div className="mt-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {posts.map((post, i) => (
            <div key={post.slug} className="reveal" data-reveal-delay={`${i * 80}`}>
              <ArticleCard post={post} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
