interface InkMountainsProps {
  className?: string;
}

/**
 * 水墨山水剪影：三层远山，用「墨分五色」的淡/重/浓三色叠加。
 * 浅色模式为墨山，深色模式反转为月下白山的「夜观山水」意境。
 */
export function InkMountains({ className = "" }: InkMountainsProps) {
  return (
    <svg
      viewBox="0 0 1440 220"
      preserveAspectRatio="none"
      aria-hidden="true"
      className={className}
    >
      {/* 远山 · 淡墨 */}
      <path
        className="fill-ink-4/45"
        d="M0 150 C 130 105, 260 155, 390 120 C 520 85, 650 145, 780 112 C 910 79, 1040 138, 1170 108 C 1300 78, 1380 118, 1440 100 L1440 220 L0 220 Z"
      />
      {/* 中山 · 重墨 */}
      <path
        className="fill-ink-3/50"
        d="M0 178 C 150 138, 300 186, 470 152 C 640 118, 790 180, 950 148 C 1110 116, 1270 168, 1440 138 L1440 220 L0 220 Z"
      />
      {/* 近山 · 浓墨 */}
      <path
        className="fill-ink-2/80"
        d="M0 198 C 210 168, 410 206, 630 182 C 850 158, 1070 202, 1280 174 C 1350 164, 1400 176, 1440 170 L1440 220 L0 220 Z"
      />
    </svg>
  );
}
