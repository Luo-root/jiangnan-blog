import { createFileRoute } from "@tanstack/react-router";
import { getAllProjects } from "@/lib/projects";
import { ProjectCard } from "@/components/project-card";
import { ThemeScenery } from "@/components/theme-scenery";
import { DawnClouds, NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/projects")({
  component: ProjectsList,
});

function ProjectsList() {
  const projects = getAllProjects();

  return (
    <div className="projects-page relative mx-auto max-w-6xl overflow-hidden px-4 py-12 sm:px-6">
      <ThemeScenery className="posts-scenery absolute -inset-x-16 top-0 h-72 opacity-30" />

      <div className="reveal relative">
        <div className="dawn-copy flex items-center gap-5">
          <img src="/assets/theme/sun.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-wide">山房筑器</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-3">DAWN WORKS · 共 {projects.length} 件</p>
          </div>
        </div>
        <div className="night-copy flex items-center gap-5">
          <img src="/assets/theme/moon.png" alt="" className="h-16 w-auto" />
          <div>
            <h1 className="font-calligraphy text-4xl font-bold tracking-[0.08em]">月下筑器</h1>
            <p className="mt-1 font-mono text-xs tracking-widest text-ink-4">NIGHT WORKS · {projects.length} PROJECTS</p>
          </div>
        </div>
        <div className="dawn-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-primary/35 to-transparent" />
        <div className="night-only mt-6 h-px max-w-xl bg-gradient-to-r from-transparent via-night-moon/35 to-transparent" />
        <DawnClouds className="dawn-only mt-3 h-10 w-full max-w-xl text-ink-4/60" />
        <NightWaves className="night-only mt-3 h-9 w-full max-w-xl text-ink-4/60" />
      </div>

      {projects.length === 0 ? (
        <div className="flex flex-col items-center gap-4 py-20 text-center">
          <img src="/assets/theme/sun.png" alt="" className="dawn-only h-16 w-auto opacity-60" />
          <img src="/assets/theme/moon.png" alt="" className="night-copy h-16 w-auto opacity-60" />
          <p className="text-muted-foreground">尚无项目，在「项目」目录放一个 .md 即可</p>
        </div>
      ) : (
        <div className="mt-10 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((project, i) => (
            <div key={project.slug} className="reveal" data-reveal-delay={`${Math.min(i * 60, 400)}`}>
              <ProjectCard project={project} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}