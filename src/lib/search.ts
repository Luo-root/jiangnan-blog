import Fuse from "fuse.js";
import { getAllPosts, type Post } from "./posts";

const posts = getAllPosts();

const fuse = new Fuse(posts, {
  keys: [
    { name: "title", weight: 0.4 },
    { name: "tags", weight: 0.25 },
    { name: "excerpt", weight: 0.2 },
    { name: "rawContent", weight: 0.15 },
  ],
  includeScore: true,
  includeMatches: true,
  threshold: 0.35,
  minMatchCharLength: 1,
  ignoreLocation: true,
});

export interface SearchResult {
  slug: string;
  title: string;
  excerpt: string;
  cover: string;
  tags: string[];
  date: string;
  matchSnippet: string;
}

function buildSnippet(post: Post, query: string): string {
  const lowerQuery = query.toLowerCase();
  const content = post.rawContent;
  const idx = content.toLowerCase().indexOf(lowerQuery);
  if (idx < 0) return post.excerpt;
  const start = Math.max(0, idx - 40);
  const end = Math.min(content.length, idx + query.length + 80);
  const prefix = start > 0 ? "…" : "";
  const suffix = end < content.length ? "…" : "";
  return prefix + content.slice(start, end).replace(/[#*`~\[\]]/g, "").trim() + suffix;
}

export function searchPosts(query: string): SearchResult[] {
  if (!query.trim()) return [];
  const results = fuse.search(query).slice(0, 8);
  return results.map((r) => {
    const post = r.item;
    return {
      slug: post.slug,
      title: post.title,
      excerpt: post.excerpt,
      cover: post.cover,
      tags: post.tags,
      date: post.date,
      matchSnippet: buildSnippet(post, query),
    };
  });
}
