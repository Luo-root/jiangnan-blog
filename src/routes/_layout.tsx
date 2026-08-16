import { useState, useEffect } from "react";
import { Outlet, createFileRoute } from "@tanstack/react-router";
import { Navbar } from "@/components/layout/navbar";
import { SearchDialog } from "@/components/search-dialog";
import { ThemeScenery } from "@/components/theme-scenery";
import { ScrollProgress } from "@/components/scroll-progress";

export const Route = createFileRoute("/_layout")({
  component: LayoutComponent,
});

function LayoutComponent() {
  const [searchOpen, setSearchOpen] = useState(false);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <div className="flex min-h-screen flex-col">
      <Navbar onSearchOpen={() => setSearchOpen(true)} />
      <main className="flex-1">
        <Outlet />
      </main>
      <footer className="relative overflow-hidden pt-16">
        <ThemeScenery className="absolute inset-x-0 bottom-0 h-56 opacity-55" />
        <div className="relative mx-auto flex max-w-6xl flex-col items-center gap-4 px-4 pb-14 sm:px-6">
          {/* 题跋：昼咏春日，夜咏江月 */}
          <p className="font-calligraphy text-lg tracking-[0.2em] text-ink-3 dark:hidden">阳春布德泽，万物生光辉</p>
          <p className="hidden font-calligraphy text-lg tracking-[0.2em] text-ink-2 dark:block">星垂平野阔，月涌大江流</p>
          <p className="font-mono text-xs tracking-widest text-muted-foreground">
            静识 · OBSIDIAN DRIVEN · 用 MARKDOWN 连接思想
          </p>
        </div>
      </footer>
      <SearchDialog open={searchOpen} onOpenChange={setSearchOpen} />
      <ScrollProgress />
    </div>
  );
}
