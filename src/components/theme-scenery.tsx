interface ThemeSceneryProps {
  className?: string;
}

/**
 * 亮主题 · 朝曦白昼人间世
 * 背景:青绿山水 + 江 + 远山 + 一条小船(1664x928 JPG)
 * 用 background-image 而不是 <img>:background-size: cover 在所有容器尺寸下都铺满,
 * 不会出现 object-fit 与容器尺寸不匹配时的黑条/留白。
 * background-position: center 70% —— 把图垂直向下偏,远山在视野中上,水在视野中部。
 */
function DawnScenery() {
  return (
    <div
      className="dawn-scenery absolute inset-0"
      aria-hidden="true"
      style={{
        backgroundImage: "url(/assets/theme/dawn-spring-mountains.jpg)",
        backgroundSize: "cover",
        backgroundPosition: "center 70%",
        backgroundRepeat: "no-repeat",
      }}
    />
  );
}

/**
 * 暗主题 · 夜隐夜阑山海间
 * 背景:夜空星河 + 远山 + 一棵白梅 + 江中月亮倒影(1664x928 JPG)
 * background-position: center 70% —— 垂直向下偏,水面/月光倒影在视野中下。
 */
function NightScenery() {
  return (
    <div
      className="night-scenery absolute inset-0"
      aria-hidden="true"
      style={{
        backgroundImage: "url(/assets/theme/night-river.jpg)",
        backgroundSize: "cover",
        backgroundPosition: "center 70%",
        backgroundRepeat: "no-repeat",
      }}
    />
  );
}

/**
 * 主题场景层容器:
 * - 亮主题:DawnScenery 整图背景
 * - 暗主题:NightScenery 整图背景
 * - 切换由 CSS `.dark .night-scenery { display: block }` 控制
 */
export function ThemeScenery({ className = "" }: ThemeSceneryProps) {
  return (
    <div className={`theme-scenery pointer-events-none select-none ${className}`} aria-hidden="true">
      <DawnScenery />
      <NightScenery />
    </div>
  );
}
