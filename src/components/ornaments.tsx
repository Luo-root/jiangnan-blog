/**
 * 主题装饰组件库：把意象从背景 SVG 带进正文区。
 * 明（朝曦）：云纹 DawnClouds
 * 暗（夜隐）：波纹 NightWaves
 * 均以 currentColor 为主色，尺寸由 className 控制，动画复用 styles.css 的 keyframes。
 */

interface OrnamentProps {
  className?: string;
}

/* —— 明 · 祥云纹横带：三朵卷云，缓慢漂移 —— */
export function DawnClouds({ className = "" }: OrnamentProps) {
  return (
    <svg className={className} viewBox="0 0 600 48" fill="none" aria-hidden="true" preserveAspectRatio="xMidYMid meet">
      <g className="dawn-cloud" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
        <path d="M24 34c-8 0-12-8-6-13s14-2 14 4c3-9 19-10 22-1 8-3 16 2 14 10" />
        <path d="M38 34h34" strokeDasharray="1 7" />
      </g>
      <g className="dawn-cloud dawn-cloud--slow" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.75">
        <path d="M250 26c-6 0-9-7-4-10s11-2 11 3c2-7 15-8 17-1 6-2 12 2 10 8" />
        <path d="M262 34h52" strokeDasharray="1 7" />
      </g>
      <g className="dawn-cloud" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.6" style={{ animationDelay: "2.4s" }}>
        <path d="M480 36c-7 0-10-7-5-11s12-1 12 4c3-8 16-8 18 0 7-3 13 1 12 7" />
        <path d="M494 36h40" strokeDasharray="1 7" />
      </g>
    </svg>
  );
}

/* —— 暗 · 江面波纹横带：三道水纹 + 一枚涟漪 —— */
export function NightWaves({ className = "" }: OrnamentProps) {
  return (
    <svg className={className} viewBox="0 0 600 48" fill="none" aria-hidden="true" preserveAspectRatio="xMidYMid meet">
      <g stroke="currentColor" strokeLinecap="round">
        <path d="M0 14c40-8 80 8 120 0s80 8 120 0 80 8 120 0 80 8 120 0 80 8 120 0" strokeWidth="1.6" strokeOpacity="0.55" strokeDasharray="3 9" />
        <path d="M0 26c50-7 90 7 140 0s90 7 140 0 90 7 140 0 90 7 140 0" strokeWidth="1.4" strokeOpacity="0.38" strokeDasharray="2 11" />
        <path d="M0 38c60-6 100 6 160 0s100 6 160 0 100 6 160 0" strokeWidth="1.2" strokeOpacity="0.25" strokeDasharray="2 13" />
      </g>
      <circle className="night-ripple" cx="300" cy="26" r="10" stroke="currentColor" strokeWidth="1.4" strokeOpacity="0.6" fill="none" />
    </svg>
  );
}
