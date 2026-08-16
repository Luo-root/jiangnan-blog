import { useEffect, useState, useCallback } from "react";

/**
 * 右侧阅读进度指示器（纯视觉，不替换原生滚动条拖拽能力）
 * - 朝曦：轻舟沿垂直轨道随阅读进度移动
 * - 夜隐：银色月路光带随阅读进度延展
 * - 移动端隐藏
 */
export function ScrollProgress() {
  const [progress, setProgress] = useState(0);

  const update = useCallback(() => {
    const h = document.documentElement.scrollHeight - window.innerHeight;
    if (h <= 0) { setProgress(0); return; }
    setProgress(Math.min(100, Math.max(0, (window.scrollY / h) * 100)));
  }, []);

  useEffect(() => {
    window.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update, { passive: true });
    update();
    return () => {
      window.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, [update]);

  /* 容器可用高度 = 顶部留 5%、底部留 5% → 90% */
  const topPct = 5 + progress * 0.9;

  return (
    <div
      className="scroll-progress pointer-events-none fixed right-0 top-0 z-50 h-full w-4"
      aria-hidden="true"
    >
      {/* ===== 朝曦：轻舟轨道 ===== */}
      <div className="dawn-only relative h-full">
        {/* 垂直轨道线：赭石淡色 */}
        <div className="absolute right-[6px] top-[5%] h-[90%] w-px bg-[#8b6b4f]/20" />
        {/* 轻舟 */}
        <div
          className="absolute right-[3px] text-[#6b5f55] transition-[top] duration-100 ease-linear"
          style={{ top: `${topPct}%`, transform: "translateY(-50%)" }}
        >
          <svg viewBox="0 0 28 14" className="h-auto w-3.5" fill="none" aria-hidden="true">
            {/* 船身弧线 */}
            <path
              d="M2 10 Q6 2 14 2 Q22 2 26 10"
              stroke="currentColor"
              strokeWidth="1.2"
              strokeLinecap="round"
              fill="none"
            />
            {/* 篷 */}
            <path
              d="M11 6 L11 2 M11 3 L15 3 M11 4.5 L14 4.5"
              stroke="currentColor"
              strokeWidth="0.9"
              strokeLinecap="round"
              fill="none"
            />
            {/* 竿 */}
            <line x1="14" y1="2" x2="17" y2="0.5" stroke="currentColor" strokeWidth="0.7" strokeLinecap="round" />
          </svg>
        </div>
      </div>

      {/* ===== 夜隐：月路光带 ===== */}
      <div className="night-only relative h-full">
        {/* 背景暗轨 */}
        <div className="absolute right-[6px] top-[5%] h-[90%] w-px bg-[#e6edf3]/8" />
        {/* 已读光带：从顶部延展到当前位置 */}
        <div
          className="absolute right-[6px] top-[5%] w-px"
          style={{
            height: `${progress * 0.9}%`,
            background: "linear-gradient(180deg, #e6edf3 0%, #8fd0cc 60%, #e6edf3 100%)",
            boxShadow: "0 0 6px 1px rgba(212,224,240,0.45), 0 0 3px rgba(143,208,204,0.3)",
          }}
        />
        {/* 当前位置光点 */}
        <div
          className="absolute right-[4.5px]"
          style={{
            top: `${topPct}%`,
            transform: "translateY(-50%)",
            width: 4,
            height: 4,
            borderRadius: "50%",
            background: "#e6edf3",
            boxShadow: "0 0 10px 3px rgba(212,224,240,0.7), 0 0 18px 5px rgba(143,208,204,0.35)",
          }}
        />
      </div>
    </div>
  );
}