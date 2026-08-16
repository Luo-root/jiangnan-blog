import { Link, createFileRoute } from "@tanstack/react-router";
import { ArrowLeft, Calendar } from "lucide-react";
import { ProjectLinkBadge } from "@/components/project-card";
import { getProjectBySlug } from "@/lib/projects";
import { MarkdownRenderer } from "@/components/markdown-renderer";
import { ThemeScenery } from "@/components/theme-scenery";

export const Route = createFileRoute("/_layout/projects_/$slug")({
  component: ProjectDetail,
});

function formatDate(date: string): string {
  if (!date) return "";
  const d = new Date(date);
  if (isNaN(d.getTime())) return date;
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

function ProjectDetail() {
  const { slug } = Route.useParams();
  const project = getProjectBySlug(slug);

  if (!project) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-20 text-center">
        <p className="text-muted-foreground">项目不存在</p>
        <Link to="/projects" className="mt-4 inline-block text-primary hover:underline">
          返回项目列表
        </Link>
      </div>
    );
  }

  return (
    <article className="post-detail relative mx-auto max-w-6xl overflow-hidden px-4 py-12 sm:px-6">
      <ThemeScenery className="post-scenery absolute -inset-x-24 top-0 h-[360px] opacity-20" />
      <Link
        to="/projects"
        className="reveal mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft size={14} /> 返回项目列表
      </Link>

      <div className="reveal relative" data-reveal-delay="50">
        <div className="dawn-only mb-5 flex items-center gap-3 font-mono text-[10px] tracking-[0.3em] text-ink-4">
          <img src="/assets/theme/sun.png" alt="" className="h-4 w-auto" />
          <span>WORK NOTE / SUNLIT EDITION</span>
        </div>
        <div className="night-copy mb-5 flex items-center gap-3 font-mono text-[10px] tracking-[0.3em] text-ink-4">
          <img src="/assets/theme/moon.png" alt="" className="h-4 w-auto" />
          <span>WORK NOTE / MOONLIT EDITION</span>
        </div>

        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="font-calligraphy text-4xl leading-snug tracking-wide sm:text-5xl">{project.name}</h1>
          </div>
          <span className="seal mt-1 h-9 w-9 shrink-0 text-sm">{project.name.charAt(0)}</span>
        </div>

        {project.summary && (
          <p className="mt-4 max-w-2xl text-sm leading-relaxed text-muted-foreground">{project.summary}</p>
        )}

        <div className="mt-4 flex flex-wrap items-center gap-3">
          {project.date && (
            <span className="flex items-center gap-1 font-mono text-xs tracking-wider text-muted-foreground">
              <Calendar size={14} /> {formatDate(project.date)}
            </span>
          )}
          {project.status && (
            <span className="rounded-full bg-secondary/60 px-2.5 py-0.5 font-mono text-xs text-secondary-foreground">
              {project.status}
            </span>
          )}
        </div>

        {project.stack.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {project.stack.map((tech) => (
              <span key={tech} className="rounded-full bg-card/60 px-2.5 py-0.5 font-mono text-xs text-secondary-foreground">
                {tech}
              </span>
            ))}
          </div>
        )}

        {project.links.length > 0 && (
          <div className="mt-5 flex flex-wrap gap-2">
            {project.links.map((link, i) => (
              <ProjectLinkBadge key={`${link.url}-${i}`} link={link} stopPropagation={false} />
            ))}
          </div>
        )}

        <div className="dawn-only mt-10 h-px bg-gradient-to-r from-transparent via-primary/30 to-transparent" />
        <div className="night-only mt-10 h-px bg-gradient-to-r from-transparent via-night-moon/30 to-transparent" />
      </div>

      <div className="reveal mt-10" data-reveal-delay="100">
        <MarkdownRenderer content={project.content} />
      </div>
    </article>
  );
}
