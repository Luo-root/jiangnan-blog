import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Moon, Sun } from "lucide-react";

const THEME_LABELS = {
  light: "朝曦",
  dark: "夜隐",
} as const;

const TRANSITION_MS = 2200;
const THEME_SWITCH_AT = TRANSITION_MS / 2; // 50% 中点：离场天体落完、主题翻转、入场天体开始升起

// 天体尺寸：与 Hero h-40 = 160px 一致
const CELESTIAL_SIZE = 160;
const CELESTIAL_HALF = CELESTIAL_SIZE / 2; // 80
// Hero 静态位置：left-32 = 128px, top-40 = 160px, navbar h-16 = 64px
// 升起终点/落山起点往右 50px（home X = 258）
const CELESTIAL_HOME_X = 128 + CELESTIAL_HALF + 50; // 258
const CELESTIAL_HOME_Y = 64 + 160 + CELESTIAL_HALF; // 304

// 落山路径（太阳和月亮共用）：home → 向右 → 右下落山
function buildSetPath(): string {
  const w = window.innerWidth;
  const h = window.innerHeight;
  const setX = 0.92 * w;
  const setY = 1.10 * h + 80; // 落点再下移 80px，落得更深
  const cx = 0.68 * w; // 控制点向右偏，先向右再下坠
  const cy = 0.18 * h;
  return `path("M ${CELESTIAL_HOME_X} ${CELESTIAL_HOME_Y} Q ${cx} ${cy} ${setX} ${setY}")`;
}

// 升起路径（太阳和月亮共用）：左下 → 向左凸 → home（整体右移 40px）
function buildRisePath(): string {
  const w = window.innerWidth;
  const h = window.innerHeight;
  const riseX = 0.06 * w + 40; // 视口左下方外，右移 40px
  const riseY = 1.05 * h;
  const cx = 0.02 * w + 40; // 控制点在起点左侧，弧线向左凸（朝画面外），右移 40px
  const cy = 0.55 * h;
  return `path("M ${riseX} ${riseY} Q ${cx} ${cy} ${CELESTIAL_HOME_X} ${CELESTIAL_HOME_Y}")`;
}

function supportsOffsetPath(): boolean {
  return typeof CSS !== "undefined" && CSS.supports("offset-path", 'path("M 0 0 L 1 1")');
}

export function ThemeToggle() {
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [isTransitioning, setIsTransitioning] = useState(false);

  useEffect(() => {
    const stored = localStorage.getItem("theme");
    const root = document.documentElement;
    const current =
      stored === "dark" ||
      (!stored && root.classList.contains("dark"))
        ? "dark"
        : "light";
    setTheme(current);
    root.classList.remove("light", "dark");
    root.classList.add(current);
    root.setAttribute("data-theme", current);
  }, []);

  const toggle = () => {
    if (isTransitioning) return;
    const next = theme === "light" ? "dark" : "light";
    const root = document.documentElement;
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const duration = reducedMotion ? 0 : TRANSITION_MS;

    setTheme(next);
    setIsTransitioning(true);

    const sky = root.querySelector<HTMLElement>(".theme-transition-sky");
    if (sky) {
      sky.classList.remove("sunset", "sunrise", "celestial-off");
      sky.classList.add(next === "dark" ? "sunset" : "sunrise");

      if (supportsOffsetPath()) {
        const sunEl = sky.querySelector<HTMLElement>(".theme-transition-sun");
        const moonEl = sky.querySelector<HTMLElement>(".theme-transition-moon");
        // 落山天体用 setPath（home→右下），升起天体用 risePath（左下→home）
        const setPath = buildSetPath();
        const risePath = buildRisePath();
        // 明→暗：太阳落山，月亮升起
        // 暗→明：月亮落山，太阳升起
        if (next === "dark") {
          if (sunEl) sunEl.style.offsetPath = setPath;
          if (moonEl) moonEl.style.offsetPath = risePath;
        } else {
          if (moonEl) moonEl.style.offsetPath = setPath;
          if (sunEl) sunEl.style.offsetPath = risePath;
        }
      } else {
        sky.classList.add("celestial-off");
      }
    }

    root.classList.add("theme-transitioning");
    root.setAttribute("data-theme", next);

    // 中点：落山天体走完 → 翻转主题 → 入场天体开始升起
    const switchTimer = window.setTimeout(() => {
      root.classList.remove("light", "dark");
      root.classList.add(next);
      localStorage.setItem("theme", next);
    }, duration === 0 ? 0 : THEME_SWITCH_AT);

    const cleanupTimer = window.setTimeout(() => {
      root.classList.remove("theme-transitioning");
      if (sky) {
        sky.classList.remove("sunset", "sunrise", "celestial-off");
        sky.querySelectorAll<HTMLElement>(".theme-transition-sun, .theme-transition-moon").forEach((el) => {
          el.style.offsetPath = "";
        });
      }
      setIsTransitioning(false);
    }, duration);

    return () => {
      window.clearTimeout(switchTimer);
      window.clearTimeout(cleanupTimer);
    };
  };

  const nextLabel = theme === "light" ? "切换至夜隐" : "切换至朝曦";

  return (
    <>
      <button
        onClick={toggle}
        aria-label={nextLabel}
        aria-busy={isTransitioning}
        title={nextLabel}
        className="theme-toggle group relative flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        <span className="sr-only">当前主题：{THEME_LABELS[theme]}</span>
        <span className="theme-icon-sun" aria-hidden="true"><Sun size={17} /></span>
        <span className="theme-icon-moon" aria-hidden="true"><Moon size={17} /></span>
      </button>
      {/* 切换动画：veil 帷幕 + horizon 地平线（全局渐变氛围）+ 天体弧线动画。
          Portal 到 body，navbar 的 backdrop-blur 会让子元素 position:fixed 退化 */}
      {createPortal(
        <div className="theme-transition-sky" aria-hidden="true">
          <div className="theme-transition-veil" />
          <div className="theme-transition-horizon" />
          <div className="theme-transition-sun" />
          <div className="theme-transition-moon" />
        </div>,
        document.body
      )}
    </>
  );
}