import { useState } from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { BookOpen, FileText, FolderGit2, Share2, Users, Search, Menu, X } from "lucide-react";
import { ThemeToggle } from "./theme-toggle";

interface NavbarProps {
  onSearchOpen: () => void;
}

const NAV_ITEMS = [
  { to: "/", label: "首页", icon: BookOpen },
  { to: "/posts", label: "文章", icon: FileText },
  { to: "/projects", label: "项目", icon: FolderGit2 },
  { to: "/friends", label: "友链", icon: Users },
  { to: "/graph", label: "图谱", icon: Share2 },
] as const;

export function Navbar({ onSearchOpen }: NavbarProps) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <header className="sticky top-0 z-40 bg-background/70 backdrop-blur-xl">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
        <Link to="/" className="flex items-center gap-2.5" onClick={() => setMobileOpen(false)}>
          <img src="/assets/theme/logo.jpg" alt="遇见江楠" className="h-9 w-9 rounded-full ring-1 ring-border/60" />
          <div className="flex flex-col leading-none">
            <span className="font-calligraphy text-lg font-bold tracking-widest">遇见江楠</span>
            <span className="dawn-copy mt-0.5 font-mono text-[10px] tracking-[0.3em] text-ink-4">朝曦入山</span>
            <span className="night-copy mt-0.5 font-mono text-[10px] tracking-[0.3em] text-ink-4">夜隐卷序</span>
          </div>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {NAV_ITEMS.map((item) => {
            const active = pathname === item.to || (item.to !== "/" && pathname.startsWith(item.to));
            return (
              <Link
                key={item.to}
                to={item.to}
                className={`nav-link flex items-center gap-1.5 rounded-full px-3.5 py-2 text-sm ${
                  active
                    ? "nav-link-active bg-foreground/8 text-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <item.icon size={15} />
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="flex items-center gap-1">
          <button
            onClick={onSearchOpen}
            aria-label="搜索"
            className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground"
          >
            <Search size={18} />
          </button>
          <ThemeToggle />
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            aria-label="菜单"
            className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground hover:text-foreground md:hidden"
          >
            {mobileOpen ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </div>

      {mobileOpen && (
        <nav className="flex flex-col gap-1 px-4 py-3 md:hidden">
          {NAV_ITEMS.map((item) => {
            const active = pathname === item.to || (item.to !== "/" && pathname.startsWith(item.to));
            return (
              <Link
                key={item.to}
                to={item.to}
                onClick={() => setMobileOpen(false)}
                className={`nav-link flex items-center gap-2 rounded-lg px-3 py-2.5 text-sm ${
                  active ? "nav-link-active bg-foreground/8 font-medium" : "text-muted-foreground"
                }`}
              >
                <item.icon size={16} />
                {item.label}
              </Link>
            );
          })}
        </nav>
      )}
    </header>
  );
}
