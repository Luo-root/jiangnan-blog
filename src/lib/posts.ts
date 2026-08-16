import { load as yamlLoad } from "js-yaml";
import vaultIndex from "virtual:vault-index";
import vaultTree from "virtual:vault-tree";

export interface PostMeta {
  slug: string;
  title: string;
  date: string;
  tags: string[];
  cover: string;
  excerpt: string;
  draft: boolean;
}

export interface Post extends PostMeta {
  content: string;
  rawContent: string;
  readingTime: number;
  links: string[];
  backlinks: string[];
  wordCount: number;
}

// ---------------------------------------------------------------------------
// 数据源：直接扫描本地 Obsidian Vault（构建时静态打包）
// - 内容来自虚拟模块 virtual:vault-tree（vite.config.ts 按一级目录遍历生成）
// - 排除 .obsidian / .trash 配置目录（虚拟模块生成时已过滤）
// - slug 使用栏目内相对路径，目录分隔符 '/' → '__'，避免同名文件冲突
// - 文章栏目固定为「文章」目录；后续栏目（项目/友链）由各自解析器消费
// ---------------------------------------------------------------------------

const POSTS_SECTION = "文章";

function parseFrontmatter(raw: string): { meta: Record<string, unknown>; body: string } {
  const fmRegex = /^---\n([\s\S]*?)\n---\n?/;
  const match = raw.match(fmRegex);
  if (!match) return { meta: {}, body: raw };
  try {
    const meta = yamlLoad(match[1]) as Record<string, unknown>;
    return { meta, body: raw.slice(match[0].length) };
  } catch {
    return { meta: {}, body: raw.slice(match[0].length) };
  }
}

/** 相对路径（无 .md）→ slug（'/' → '__'） */
function toSlug(relPathNoExt: string): string {
  return relPathNoExt.split("/").join("__");
}

/** 文件名（去 .md）→ slug 反查键 */
function fileNameOf(relPathNoExt: string): string {
  const name = relPathNoExt.split("/").pop() || relPathNoExt;
  return name.replace(/\.md$/, "");
}

export function slugifyHeading(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\u4e00-\u9fa5\s-]/g, "")
    .trim()
    .replace(/\s+/g, "-");
}

function processWikiLinks(
  body: string,
  slugSet: Set<string>,
  titleToSlug: Map<string, string>,
  imageNameToRel: Record<string, string>,
  noteDir: string
): { content: string; links: string[] } {
  const links: string[] = [];

  // 1) Obsidian 图片嵌入 ![[xxx.png|alt]] → markdown 图片 /vault/<相对路径>
  let content = body.replace(/!\[\[([^\]]+)\]\]/g, (_full, inner: string) => {
    let target = inner;
    let alt = inner;
    if (inner.includes("|")) {
      const [t, d] = inner.split("|");
      target = t.trim();
      alt = d.trim();
    }
    const name = target.split(/[#^]/)[0].trim();
    // 优先笔记同目录，其次全局文件名索引
    let rel = noteDir && imageNameToRel[`${noteDir}/${name}`] ? imageNameToRel[`${noteDir}/${name}`] : imageNameToRel[name];
    if (!rel && name.includes("/")) rel = imageNameToRel[name.split("/").pop() || ""];
    if (rel) {
      return `![${alt}](/vault/${encodeURI(rel)})`;
    }
    return `![${alt}](${name})`;
  });

  // 2) WikiLink 双链 [[target|display]] → 站内链接
  const wikiLinkRegex = /\[\[([^\]]+)\]\]/g;
  content = content.replace(wikiLinkRegex, (_full, inner: string) => {
    let target = inner;
    let display = inner;
    if (inner.includes("|")) {
      const [t, d] = inner.split("|");
      target = t.trim();
      display = d.trim();
    }
    // 去掉锚点/块引用
    const base = target.split(/[#^]/)[0].trim();
    // 匹配顺序：完整路径 slug → 文件名 → 标题
    let slug = toSlug(base.replace(/\.md$/, ""));
    if (!slugSet.has(slug)) {
      const byName = titleToSlug.get(fileNameOf(base));
      if (byName) slug = byName;
    }
    if (!slugSet.has(slug)) {
      const byTitle = titleToSlug.get(base);
      if (byTitle) slug = byTitle;
    }
    if (slugSet.has(slug)) {
      if (!links.includes(slug)) links.push(slug);
      return `[${display}](/posts/${slug})`;
    }
    return display;
  });

  // 3) 普通 markdown 相对路径图片 ![](images/x.png) → /vault/<笔记目录>/images/x.png
  content = content.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (m, alt: string, src: string) => {
    const s = src.trim();
    if (/^(https?:|data:|blob:|\/)/i.test(s)) return m;
    const cleaned = s.replace(/^\.\//, "");
    const rel = noteDir ? `${noteDir}/${cleaned}` : cleaned;
    return `![${alt}](/vault/${encodeURI(rel)})`;
  });

  return { content, links };
}

function buildPosts(): Post[] {
  const rawPosts: { slug: string; rel: string; meta: Record<string, unknown>; body: string }[] = [];
  const slugSet = new Set<string>();
  const titleToSlug = new Map<string, string>();

  for (const [relPath, raw] of Object.entries(vaultTree[POSTS_SECTION] ?? {})) {
    const rel = relPath.replace(/\.md$/, "");
    const slug = toSlug(rel);
    const { meta, body } = parseFrontmatter(raw);
    const title = (meta.title as string) || fileNameOf(rel);
    rawPosts.push({ slug, rel, meta, body });
    slugSet.add(slug);
    titleToSlug.set(fileNameOf(rel), slug);
    if (title !== fileNameOf(rel)) titleToSlug.set(title, slug);
  }

  const posts: Post[] = rawPosts
    .filter(({ meta }) => !meta.draft)
    .map(({ slug, rel, meta, body }) => {
      const noteDir = rel.includes("/") ? rel.split("/").slice(0, -1).join("/") : "";
      const { content, links } = processWikiLinks(body, slugSet, titleToSlug, vaultIndex, noteDir);
      const title = (meta.title as string) || fileNameOf(rel);
      const cleanText = body.replace(/[#*`~\-\[\]()!]/g, "").replace(/\s+/g, "");
      const wordCount = cleanText.length;
      return {
        slug,
        title,
        date: (meta.date as string) || "",
        tags: (meta.tags as string[]) || [],
        cover: (meta.cover as string) || "",
        excerpt: (meta.excerpt as string) || "",
        draft: false,
        content,
        rawContent: body,
        readingTime: Math.max(1, Math.ceil(wordCount / 400)),
        links,
        backlinks: [],
        wordCount,
      };
    });

  for (const post of posts) {
    for (const targetSlug of post.links) {
      const target = posts.find((p) => p.slug === targetSlug);
      if (target && !target.backlinks.includes(post.slug)) {
        target.backlinks.push(post.slug);
      }
    }
  }

  posts.sort((a, b) => b.date.localeCompare(a.date));
  return posts;
}

const allPosts = buildPosts();

export function getAllPosts(): Post[] {
  return allPosts;
}

export function getPostBySlug(slug: string): Post | undefined {
  return allPosts.find((p) => p.slug === slug);
}

export function getPostsByTag(tag: string): Post[] {
  return allPosts.filter((p) => p.tags.includes(tag));
}

export function getAllTags(): { tag: string; count: number }[] {
  const tagMap = new Map<string, number>();
  for (const post of allPosts) {
    for (const tag of post.tags) {
      tagMap.set(tag, (tagMap.get(tag) || 0) + 1);
    }
  }
  return Array.from(tagMap.entries())
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count);
}

export function getBacklinkPosts(slug: string): Post[] {
  return allPosts
    .filter((p) => p.links.includes(slug))
    .sort((a, b) => b.date.localeCompare(a.date));
}

export function getStats() {
  return {
    posts: allPosts.length,
    tags: getAllTags().length,
    links: allPosts.reduce((sum, p) => sum + p.links.length, 0),
  };
}

export interface GraphNode {
  id: string;
  label: string;
  tag: string;
  degree: number;
}

export interface GraphEdge {
  source: string;
  target: string;
}

export function getGraphData(): { nodes: GraphNode[]; edges: GraphEdge[] } {
  const nodes: GraphNode[] = allPosts.map((p) => ({
    id: p.slug,
    label: p.title,
    tag: p.tags[0] || "未分类",
    degree: p.backlinks.length + p.links.length,
  }));
  const edges: GraphEdge[] = [];
  const slugSet = new Set(allPosts.map((p) => p.slug));
  for (const post of allPosts) {
    for (const target of post.links) {
      if (slugSet.has(target)) {
        edges.push({ source: post.slug, target });
      }
    }
  }
  return { nodes, edges };
}

export function extractHeadings(content: string): { id: string; text: string; level: number }[] {
  const headings: { id: string; text: string; level: number }[] = [];
  const regex = /^(#{2,3})\s+(.+)$/gm;
  let match;
  while ((match = regex.exec(content)) !== null) {
    const level = match[1].length;
    const text = match[2].trim();
    headings.push({ id: slugifyHeading(text), text, level });
  }
  return headings;
}
