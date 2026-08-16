import { load as yamlLoad } from "js-yaml";
import vaultTree from "virtual:vault-tree";

const PROJECTS_SECTION = "项目";

/** 项目外链类型（图标 + 文案） */
export type ProjectLinkType = "repo" | "demo" | "video" | "docs" | "site" | "other";

export interface ProjectLink {
  type: ProjectLinkType;
  url: string;
  label?: string;   // 自定义文案；缺省用默认文案
}

export interface Project {
  slug: string;
  name: string;
  summary: string;
  /** 项目外链：仓库 / 演示 / 视频 / 文档 / 官网 等，自由组合 */
  links: ProjectLink[];
  stack: string[];
  status: string;
  cover: string;
  date: string;
  content: string;
  rawContent: string;
}

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

/** 栏目内相对路径 → slug（/ → __） */
function toSlug(rel: string): string {
  return rel.split("/").join("__");
}

/** frontmatter 的 `links` 字段归一化为 ProjectLink[] */
function normalizeLinks(meta: Record<string, unknown>): ProjectLink[] {
  // 新格式：links: [{type, url, label?}, ...]
  if (Array.isArray(meta.links)) {
    return (meta.links as Array<Record<string, unknown>>)
      .filter((l) => l && typeof l.url === "string")
      .map((l) => ({
        type: (l.type as ProjectLinkType) || "other",
        url: l.url as string,
        label: typeof l.label === "string" ? l.label : undefined,
      }));
  }
  // 兼容老格式：repo / demo 单字段
  const legacy: ProjectLink[] = [];
  if (typeof meta.repo === "string" && meta.repo) {
    legacy.push({ type: "repo", url: meta.repo });
  }
  if (typeof meta.demo === "string" && meta.demo) {
    legacy.push({ type: "demo", url: meta.demo });
  }
  return legacy;
}

function buildProjects(): Project[] {
  const section = vaultTree[PROJECTS_SECTION] ?? {};
  const projects: Project[] = [];

  for (const [relPath, raw] of Object.entries(section)) {
    const rel = relPath.replace(/\.md$/, "");
    const { meta, body } = parseFrontmatter(raw);

    projects.push({
      slug: toSlug(rel),
      name: (meta.name as string) || rel.split("/").pop() || rel,
      summary: (meta.summary as string) || "",
      links: normalizeLinks(meta),
      stack: Array.isArray(meta.stack) ? meta.stack as string[] : [],
      status: (meta.status as string) || "",
      cover: (meta.cover as string) || "",
      date: (meta.date as string) || "",
      content: body,
      rawContent: raw,
    });
  }

  projects.sort((a, b) => b.date.localeCompare(a.date));
  return projects;
}

const allProjects = buildProjects();

export function getAllProjects(): Project[] {
  return allProjects;
}

export function getProjectBySlug(slug: string): Project | undefined {
  return allProjects.find((p) => p.slug === slug);
}