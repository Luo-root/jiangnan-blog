import { Link2 } from "lucide-react";
import { getBacklinkPosts } from "@/lib/posts";
import { ArticleCard } from "./article-card";

export function Backlinks({ slug }: { slug: string }) {
  const backlinks = getBacklinkPosts(slug);
  if (backlinks.length === 0) return null;

  return (
    <div className="mt-16 border-t border-border pt-8">
      <div className="mb-5 flex items-center gap-2 text-sm font-medium text-muted-foreground">
        <Link2 size={16} className="text-primary" />
        反向链接 · {backlinks.length} 篇文章引用了本文
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        {backlinks.map((post) => (
          <ArticleCard key={post.slug} post={post} variant="compact" />
        ))}
      </div>
    </div>
  );
}
