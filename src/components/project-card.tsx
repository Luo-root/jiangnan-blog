import { Link } from "@tanstack/react-router";
import { ExternalLink, Github, PlayCircle, BookOpen, Globe, Link2 } from "lucide-react";
import type { Project, ProjectLink, ProjectLinkType } from "@/lib/projects";

/** link type → 图标 + 默认文案 */
const LINK_META: Record<ProjectLinkType, { icon: typeof Github; label: string; className: string }> = {
  repo:  { icon: Github,      label: "仓库", className: "bg-foreground/8 text-foreground hover:bg-foreground/14" },
  demo:  { icon: ExternalLink, label: "演示", className: "bg-primary/12 text-primary hover:bg-primary/20" },
  video: { icon: PlayCircle,   label: "视频", className: "bg-rose-500/10 text-rose-400 hover:bg-rose-500/18" },
  docs:  { icon: BookOpen,     label: "文档", className: "bg-sky-500/10 text-sky-400 hover:bg-sky-500/18" },
  site:  { icon: Globe,        label: "官网", className: "bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/18" },
  other: { icon: Link2,        label: "链接", className: "bg-secondary/60 text-secondary-foreground hover:bg-secondary/80" },
};

export function ProjectLinkBadge({ link, stopPropagation = true }: { link: ProjectLink; stopPropagation?: boolean }) {
  const meta = LINK_META[link.type] || LINK_META.other;
  const Icon = meta.icon;
  return (
    <a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={stopPropagation ? (e) => e.stopPropagation() : undefined}
      className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 font-mono text-xs transition-all hover:gap-2 ${meta.className}`}
    >
      <Icon size={13} /> {link.label || meta.label}
    </a>
  );
}

/** 国风墨色封面：和文章卡片共用一套色板 */
const INK_COVERS = [
  "from-[#3d5a73] to-[#1f3245]",
  "from-[#3c423c] to-[#1c211c]",
  "from-[#6e4a36] to-[#38241a]",
  "from-[#7d6428] to-[#42351a]",
  "from-[#503c52] to-[#2a1f2c]",
  "from-[#3f5d4e] to-[#1f3027]",
];

const STATUS_STYLE: Record<string, string> = {
  "维护中": "bg-emerald-500/12 text-emerald-400",
  "进行中": "bg-sky-500/12 text-sky-400",
  "已归档": "bg-muted text-muted-foreground",
};

function ProjectCover({ name, cover }: { name: string; cover: string }) {
  if (cover) {
    return (
      <img
        src={cover}
        alt={name}
        loading="lazy"
        className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
      />
    );
  }
  const hash = name.charCodeAt(0) % INK_COVERS.length;
  return (
    <div className={`relative flex h-full w-full items-center justify-center overflow-hidden bg-gradient-to-br ${INK_COVERS[hash]}`}>
      <div className="absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 70% 60% at 30% 25%, rgba(255,255,255,0.08), transparent 65%), radial-gradient(ellipse 60% 55% at 75% 80%, rgba(0,0,0,0.24), transparent 70%)",
        }}
      />
      <span className="relative font-serif text-6xl font-bold text-white/85 drop-shadow-lg">
        {name.charAt(0) || "·"}
      </span>
    </div>
  );
}

/**
 * 项目卡片
 * - 整卡可点击 → 跳详情页
 * - 「链接」是真外链 `<a target="_blank">`，stopPropagation 避免冒泡
 */
export function ProjectCard({ project }: { project: Project }) {
  return (
    <Link
      to="/projects/$slug"
      params={{ slug: project.slug }}
      className="group flex flex-col overflow-hidden rounded-2xl border border-border/40 bg-card/60 shadow-[0_6px_22px_-16px_rgb(43_124_166_/_0.22),0_1px_0_0_rgb(143_194_223_/_0.08)_inset] transition-all hover:shadow-[0_18px_40px_-22px_rgb(43_124_166_/_0.4),0_0_0_1px_rgb(43_124_166_/_0.2)] hover:-translate-y-1"
    >
      <div className="relative aspect-[16/10] w-full overflow-hidden">
        <ProjectCover name={project.name} cover={project.cover} />
        {project.status && (
          <span className={`absolute right-2 top-2 rounded-full px-2 py-0.5 font-mono text-[11px] ${STATUS_STYLE[project.status] || "bg-muted text-muted-foreground"}`}>
            {project.status}
          </span>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-2.5 p-5">
        <h3 className="font-serif text-lg font-bold leading-snug tracking-wide text-foreground transition-colors group-hover:text-primary">
          {project.name}
        </h3>
        {project.summary && (
          <p className="line-clamp-2 text-sm leading-relaxed text-muted-foreground">
            {project.summary}
          </p>
        )}

        {project.stack.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {project.stack.map((tech) => (
              <span
                key={tech}
                className="rounded-full bg-secondary/60 px-2 py-0.5 font-mono text-[11px] text-secondary-foreground"
              >
                {tech}
              </span>
            ))}
          </div>
        )}

        {project.links.length > 0 && (
          <div className="mt-auto flex flex-wrap gap-2 pt-1">
            {project.links.map((link, i) => (
              <ProjectLinkBadge key={`${link.url}-${i}`} link={link} />
            ))}
          </div>
        )}
      </div>
    </Link>
  );
}