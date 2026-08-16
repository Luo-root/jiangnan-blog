import { createFileRoute } from "@tanstack/react-router";
import { GraphView } from "@/components/graph-view";
import { DawnClouds, NightWaves } from "@/components/ornaments";

export const Route = createFileRoute("/_layout/graph")({
  component: GraphPage,
});

function GraphPage() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <div className="reveal mb-6">
        <div className="flex items-center gap-4">
          <h1 className="font-calligraphy text-4xl tracking-wide sm:text-5xl">览星图谱</h1>
          <span className="seal mt-1 h-8 w-8 text-xs">星</span>
        </div>
        <p className="dawn-copy mt-2 font-mono text-xs tracking-[0.3em] text-ink-3">STAR ATLAS · 日行山轨</p>
        <p className="night-copy mt-2 font-mono text-xs tracking-[0.3em] text-ink-4">STAR ATLAS · 月照寒江</p>
        <p className="dawn-copy mt-3 max-w-2xl leading-loose text-muted-foreground">
          文章为峰，双链为径。白昼观之，是春山宣纸上的一幅墨星图——群峰错落，路径分明。
          拖拽星辰、滚轮缩放，点击星名即可前往对应文章。
        </p>
        <p className="night-copy mt-3 max-w-2xl leading-loose text-muted-foreground">
          文章为星，双链为河。入夜观之，是寒江之上倒映的真实星空——星光沉静，水脉相连。
          拖拽星辰、滚轮缩放，点击星名即可前往对应文章。
        </p>
        <DawnClouds className="dawn-only mt-5 h-10 w-72 text-ink-4/50" />
        <NightWaves className="night-only mt-5 h-9 w-72 text-ink-4/60" />
      </div>
      <div className="reveal" data-reveal-delay="100">
        <GraphView />
      </div>
    </div>
  );
}
