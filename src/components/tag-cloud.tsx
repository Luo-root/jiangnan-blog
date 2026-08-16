import { Link } from "@tanstack/react-router";
import { getAllTags } from "@/lib/posts";

export function TagCloud() {
  const tags = getAllTags();
  const maxCount = Math.max(...tags.map((t) => t.count), 1);

  return (
    <div className="flex flex-wrap gap-2">
      {tags.map(({ tag, count }) => {
        const ratio = count / maxCount;
        const fontSize = 0.8 + ratio * 0.45;
        return (
          <Link
            key={tag}
            to="/tags/$tag"
            params={{ tag }}
            className="tag-pill inline-flex items-baseline rounded-[3px] border border-ink-3/35 px-3 py-1 font-serif tracking-wider text-ink-2 hover:border-primary hover:bg-primary hover:text-primary-foreground"
            style={{ fontSize: `${fontSize}rem` }}
          >
            {tag}
            <span className="ml-1.5 font-mono text-xs opacity-60">{count}</span>
          </Link>
        );
      })}
    </div>
  );
}
