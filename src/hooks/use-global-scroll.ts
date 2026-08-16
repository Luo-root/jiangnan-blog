/**
 * 全局滚动视差引擎
 *
 * 把 window.scrollY 写入 :root 的 --scroll-y CSS 变量;
 * 所有 .parallax-layer 用 calc(var(--scroll-y) * factor) 算 transform。
 *
 * 由 main.tsx 启动时调用一次,贯穿整个应用生命周期。
 * rAF 节流、passive 监听、写入 GPU 友好的 transform 复合属性。
 *
 * 返回 cleanup 函数(HMR 场景下可卸载)。
 */
export function initGlobalScroll(): () => void {
  if (typeof window === "undefined") return () => {};
  let raf = 0;
  const update = () => {
    raf = 0;
    document.documentElement.style.setProperty("--scroll-y", String(window.scrollY));
  };
  const onScroll = () => {
    if (!raf) raf = requestAnimationFrame(update);
  };
  update();
  window.addEventListener("scroll", onScroll, { passive: true });
  window.addEventListener("resize", onScroll, { passive: true });
  return () => {
    window.removeEventListener("scroll", onScroll);
    window.removeEventListener("resize", onScroll);
    if (raf) cancelAnimationFrame(raf);
  };
}
