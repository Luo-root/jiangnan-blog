import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  forceSimulation,
  forceManyBody,
  forceLink,
  forceCollide,
  forceX,
  forceY,
  type Simulation,
  type SimulationNodeDatum,
  type SimulationLinkDatum,
} from "d3-force";
import { getGraphData, getAllTags } from "@/lib/posts";
import { Shuffle } from "lucide-react";

interface SimNode extends SimulationNodeDatum {
  id: string;
  label: string;
  tag: string;
  degree: number;
}

type SimLink = SimulationLinkDatum<SimNode>;

// 两套星官色谱：朝曦为石青/石绿/晴空（朱砂暖锚），夜隐为月白/冰川青/黛蓝
const MONO_COLOR = "#2B7CA6";

const DAWN_TAG_COLORS = [
  "#2B7CA6", "#33916C", "#8FC2DF", "#C5392B", "#4A5961",
  "#5B8FA8", "#3A7D8C", "#6FA3B8", "#2E6E5E", "#4F9E8A",
];

const NIGHT_TAG_COLORS = [
  "#8FD0CC", "#E6EDF3", "#A9C7E8", "#6B93A8", "#8FB7C9",
  "#7EA8C4", "#9BB0BD", "#5C8AA6", "#C6D5DF", "#4A7A94",
];

function nodeRadius(degree: number): number {
  return 4 + Math.min(degree * 2, 11);
}

// hex 转 rgba
function hexA(hex: string, alpha: number): string {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
}

// 确定性伪随机：星野每次渲染一致
function mulberry32(seed: number) {
  return function () {
    seed |= 0;
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

interface BgStar {
  x: number;
  y: number;
  r: number;
  a: number;
  cross: boolean;
}

export function GraphView() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const navigate = useNavigate();
  const [colorByTag, setColorByTag] = useState(true);
  const [hoverId, setHoverId] = useState<string | null>(null);

  // 静态数据：useMemo 缓存，保证 effect 不会因引用变化而重跑
  const graphData = useMemo(() => getGraphData(), []);
  const tags = useMemo(() => getAllTags(), []);
  const tagColorIndexMap = useMemo(() => {
    const m = new Map<string, number>();
    tags.forEach((t, i) => m.set(t.tag, i));
    return m;
  }, [tags]);

  const ctrlRef = useRef<{
    setColor: (v: boolean) => void;
    resetLayout: () => void;
  } | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current!;
    const ctx = canvas.getContext("2d")!;
    const parent = canvas.parentElement!;
    const dpr = window.devicePixelRatio || 1;

    let width = parent.clientWidth;
    let height = parent.clientHeight;

    // 背景星野：浅色=宣纸墨点，深色=夜空中细碎的远星
    let bgStars: BgStar[] = [];
    function genStars() {
      const rnd = mulberry32(20260805);
      bgStars = Array.from({ length: 170 }, () => ({
        x: rnd() * width,
        y: rnd() * height,
        r: 0.4 + rnd() * 1.2,
        a: 0.2 + rnd() * 0.5,
        cross: rnd() > 0.93,
      }));
    }

    function resize() {
      width = parent.clientWidth;
      height = parent.clientHeight;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      genStars();
    }
    resize();

    // ---- 初始布局：以画布中心为圆心的小范围随机散点 ----
    // 不要用大圆周分布，否则 charge force 推开后 bbox 暴涨，
    // fitToView 会先按大 bbox 算小 scale（显得图被放大），再按收敛 bbox 重算大 scale（缩回正常），
    // 出现「先放大再缩小」的视觉跳动。改成 0.05*min(w,h) 的小散点，让模拟从稳态出发。
    const nodes: SimNode[] = graphData.nodes.map(() => {
      const jitter = Math.min(width, height) * 0.05;
      return {
        x: width / 2 + (Math.random() - 0.5) * jitter,
        y: height / 2 + (Math.random() - 0.5) * jitter,
      } as SimNode & { id: string; label: string; tag: string; degree: number };
    });
    // 真实数据写入节点（覆盖 placeholder）
    graphData.nodes.forEach((n, i) => {
      nodes[i].id = n.id;
      nodes[i].label = n.label;
      nodes[i].tag = n.tag;
      nodes[i].degree = n.degree;
    });
    const links: SimLink[] = graphData.edges.map((e) => ({
      source: e.source,
      target: e.target,
    }));

    const view = { scale: 1, offsetX: 0, offsetY: 0 };
    const interaction = {
      dragging: null as SimNode | null,
      panning: false,
      lastX: 0,
      lastY: 0,
      hoverId: null as string | null,
      colorByTag: true,
    };

    // ---- d3-force 模拟：alpha 衰减至 alphaMin 后自动停止，画面彻底静止 ----
    let shouldFit = true; // 首次布局与重新布局后，收敛时自动 fit-to-view
    const simulation: Simulation<SimNode, SimLink> = forceSimulation<SimNode>(nodes)
      .force("charge", forceManyBody<SimNode>().strength(-450))
      .force(
        "link",
        forceLink<SimNode, SimLink>(links)
          .id((d) => d.id)
          .distance(150)
          .strength(0.7)
      )
      .force(
        "collide",
        forceCollide<SimNode>()
          .radius((d) => nodeRadius(d.degree) + 22)
          .strength(0.9)
      )
      .force("x", forceX<SimNode>(width / 2).strength(0.04))
      .force("y", forceY<SimNode>(height / 2).strength(0.04))
      .alphaDecay(0.05)
      .velocityDecay(0.38);

    // 预热：先静默收敛（不 draw、不触发 end 的 fitToView），
    // 避免用户进入页面时看到「节点从中心炸开 + fitToView 一次性缩放」的「放大→恢复」跳变。
    // 收敛后立即 fitToView，直接画出最终稳定态。
    simulation.stop();
    for (let i = 0; i < 300; i++) simulation.tick();
    fitToView();
    simulation.on("tick", draw);
    simulation.on("end", () => {
      if (shouldFit) {
        shouldFit = false;
        fitToView();
      }
    });

    // 收敛后计算包围盒，让图谱居中并占满画布。
    // 关键：scale 上限收紧到 1.2（不再放大到 1.8），避免收敛后视觉上突然放大。
    function fitToView() {
      let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
      for (const n of nodes) {
        if (n.x == null || n.y == null) continue;
        minX = Math.min(minX, n.x);
        maxX = Math.max(maxX, n.x);
        minY = Math.min(minY, n.y);
        maxY = Math.max(maxY, n.y);
      }
      if (!isFinite(minX) || !isFinite(minY)) return;
      const pad = 60;
      const bw = Math.max(maxX - minX, 1) + pad * 2;
      const bh = Math.max(maxY - minY, 1) + pad * 2;
      // scale 上限 1.2：节点数少时不会"放大过头"；下限保持原 min 比例
      const scale = Math.min(width / bw, height / bh, 1.2);
      const cx = (minX + maxX) / 2;
      const cy = (minY + maxY) / 2;
      view.scale = scale;
      view.offsetX = width / 2 - cx * scale;
      view.offsetY = height / 2 - cy * scale;
      draw();
    }

    // 主题感知配色：宣纸墨星图 / 夜空星象
    function palette() {
      return document.documentElement.classList.contains("dark")
        ? {
            dark: true,
            bg0: "#0C1421",
            bg1: "#16222F",
            star: "230, 237, 243",
            link: "rgba(143, 208, 204, 0.42)",
            linkDim: "rgba(122, 145, 160, 0.08)",
            linkActive: "rgba(143, 208, 204, 0.95)",
            label: "rgba(162, 182, 194, 0.9)",
            labelActive: "rgba(230, 237, 243, 0.98)",
            ring: "rgba(230, 237, 243, 0.9)",
            core: "rgba(230, 237, 243, 0.96)",
            instrument: "rgba(143, 208, 204, 0.22)",
          }
        : {
            dark: false,
            bg0: "#F6F8F5",
            bg1: "#FFFFFF",
            star: "42, 53, 59",
            link: "rgba(43, 124, 166, 0.46)",
            linkDim: "rgba(42, 53, 59, 0.10)",
            linkActive: "rgba(197, 57, 43, 0.9)",
            label: "rgba(74, 89, 97, 0.92)",
            labelActive: "rgba(42, 53, 59, 0.98)",
            ring: "rgba(197, 57, 43, 0.78)",
            core: "rgba(255, 255, 255, 0.94)",
            instrument: "rgba(43, 124, 166, 0.18)",
          };
    }

    function linkNode(v: string | SimNode | number): SimNode | undefined {
      if (typeof v === "object") return v;
      return nodes.find((n) => n.id === String(v));
    }

    function draw() {
      const pal = palette();
      ctx.clearRect(0, 0, width, height);

      // 底：夜空 / 宣纸渐变
      const bg = ctx.createLinearGradient(0, 0, 0, height);
      bg.addColorStop(0, pal.bg0);
      bg.addColorStop(1, pal.bg1);
      ctx.fillStyle = bg;
      ctx.fillRect(0, 0, width, height);

      // 星野（屏幕坐标，不随图谱平移缩放，如透过浑天仪望见的天幕）
      for (const s of bgStars) {
        ctx.fillStyle = `rgba(${pal.star}, ${s.a * (pal.dark ? 1 : 0.75)})`;
        ctx.beginPath();
        ctx.arc(s.x, s.y, s.r, 0, Math.PI * 2);
        ctx.fill();
        if (s.cross && pal.dark) {
          ctx.strokeStyle = `rgba(${pal.star}, ${s.a * 0.4})`;
          ctx.lineWidth = 0.6;
          const L = s.r * 3.2;
          ctx.beginPath();
          ctx.moveTo(s.x - L, s.y);
          ctx.lineTo(s.x + L, s.y);
          ctx.moveTo(s.x, s.y - L);
          ctx.lineTo(s.x, s.y + L);
          ctx.stroke();
        }
      }

      // 浑天仪刻度环：同心虚线圆 + 正交轴线
      const cx = width / 2;
      const cy = height / 2;
      const base = Math.min(width, height);
      ctx.strokeStyle = pal.instrument;
      ctx.lineWidth = 1;
      ctx.setLineDash([3, 6]);
      for (const rr of [0.22, 0.38, 0.55]) {
        ctx.beginPath();
        ctx.arc(cx, cy, base * rr, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.beginPath();
      ctx.moveTo(cx - base * 0.58, cy);
      ctx.lineTo(cx + base * 0.58, cy);
      ctx.moveTo(cx, cy - base * 0.58);
      ctx.lineTo(cx, cy + base * 0.58);
      ctx.stroke();
      ctx.setLineDash([]);

      ctx.save();
      ctx.translate(view.offsetX, view.offsetY);
      ctx.scale(view.scale, view.scale);

      const hovered = interaction.hoverId;
      const neighbors = new Set<string>();
      if (hovered) {
        neighbors.add(hovered);
        for (const l of links) {
          const s = linkNode(l.source);
          const t = linkNode(l.target);
          if (!s || !t) continue;
          if (s.id === hovered) neighbors.add(t.id);
          if (t.id === hovered) neighbors.add(s.id);
        }
      }

      // 星轨：星座连线（虚线）；悬停时相邻星轨实线高亮，其余淡出
      for (const l of links) {
        const s = linkNode(l.source);
        const t = linkNode(l.target);
        if (!s || !t || s.x == null || s.y == null || t.x == null || t.y == null) continue;
        const isActive = hovered != null && (s.id === hovered || t.id === hovered);
        ctx.strokeStyle = isActive
          ? pal.linkActive
          : hovered != null
            ? pal.linkDim
            : pal.link;
        ctx.lineWidth = isActive ? 1.8 : 1;
        ctx.setLineDash(isActive ? [] : [3, 5]);
        ctx.beginPath();
        ctx.moveTo(s.x, s.y);
        ctx.lineTo(t.x, t.y);
        ctx.stroke();
      }
      ctx.setLineDash([]);

      // 星辰：光晕 + 星体 + 亮芯
      for (const n of nodes) {
        if (n.x == null || n.y == null) continue;
        const darkMode = pal.dark;
        const color = interaction.colorByTag
          ? (darkMode ? NIGHT_TAG_COLORS : DAWN_TAG_COLORS)[(tagColorIndexMap.get(n.tag) || 0) % 10]
          : (darkMode ? "#E6EDF3" : MONO_COLOR);
        const isHovered = hovered === n.id;
        const isNeighbor = neighbors.has(n.id);
        const radius = nodeRadius(n.degree);

        ctx.globalAlpha = hovered ? (isNeighbor ? 1 : 0.22) : 1;

        // 光晕：径向渐变，如星芒弥散（宣纸上需更强才显光）
        const haloR = radius * (isHovered ? 4.2 : 3);
        const haloAlpha = isHovered ? (pal.dark ? 0.5 : 0.6) : pal.dark ? 0.34 : 0.5;
        const halo = ctx.createRadialGradient(n.x, n.y, 0, n.x, n.y, haloR);
        halo.addColorStop(0, hexA(color, haloAlpha));
        halo.addColorStop(0.55, hexA(color, haloAlpha * 0.35));
        halo.addColorStop(1, hexA(color, 0));
        ctx.fillStyle = halo;
        ctx.beginPath();
        ctx.arc(n.x, n.y, haloR, 0, Math.PI * 2);
        ctx.fill();

        // 星体
        ctx.fillStyle = color;
        ctx.beginPath();
        ctx.arc(n.x, n.y, radius, 0, Math.PI * 2);
        ctx.fill();

        // 星芒高光：偏移小亮点，如星光一闪
        ctx.fillStyle = pal.core;
        ctx.beginPath();
        ctx.arc(n.x - radius * 0.3, n.y - radius * 0.3, Math.max(radius * 0.28, 0.8), 0, Math.PI * 2);
        ctx.fill();

        // 悬停星：十字星芒
        if (isHovered) {
          ctx.strokeStyle = pal.ring;
          ctx.lineWidth = 1;
          const L = radius * 3.4;
          ctx.beginPath();
          ctx.moveTo(n.x - L, n.y);
          ctx.lineTo(n.x + L, n.y);
          ctx.moveTo(n.x, n.y - L);
          ctx.lineTo(n.x, n.y + L);
          ctx.stroke();
          ctx.lineWidth = 2;
          ctx.beginPath();
          ctx.arc(n.x, n.y, radius + 4, 0, Math.PI * 2);
          ctx.stroke();
        } else if (isNeighbor && hovered) {
          ctx.strokeStyle = color;
          ctx.lineWidth = 1.2;
          ctx.beginPath();
          ctx.arc(n.x, n.y, radius + 2, 0, Math.PI * 2);
          ctx.stroke();
        }

        // 星名：悬停及邻星常显；无悬停时亮星（高度数）常显
        if (isHovered || (isNeighbor && hovered) || (n.degree >= 2 && !hovered)) {
          ctx.font = '14px "Ma Shan Zheng", "Noto Serif SC", serif';
          ctx.textAlign = "center";
          ctx.textBaseline = "bottom";
          ctx.fillStyle = isHovered ? pal.labelActive : pal.label;
          ctx.fillText(n.label, n.x, n.y - radius - 9);
        }
        ctx.globalAlpha = 1;
      }
      ctx.restore();
    }

    // ---- 交互 ----
    function getMousePos(e: MouseEvent) {
      const rect = canvas.getBoundingClientRect();
      return {
        x: (e.clientX - rect.left - view.offsetX) / view.scale,
        y: (e.clientY - rect.top - view.offsetY) / view.scale,
      };
    }

    function findNode(x: number, y: number): SimNode | null {
      // 倒序遍历：优先命中后绘制（上层）的节点
      for (let i = nodes.length - 1; i >= 0; i--) {
        const n = nodes[i];
        if (n.x == null || n.y == null) continue;
        const r = nodeRadius(n.degree) + 6;
        const dx = n.x - x;
        const dy = n.y - y;
        if (dx * dx + dy * dy < r * r) return n;
      }
      return null;
    }

    function onDown(e: MouseEvent) {
      const { x, y } = getMousePos(e);
      const node = findNode(x, y);
      if (node) {
        interaction.dragging = node;
        node.fx = node.x;
        node.fy = node.y;
        simulation.alphaTarget(0.25).restart();
      } else {
        interaction.panning = true;
      }
      interaction.lastX = e.clientX;
      interaction.lastY = e.clientY;
    }

    function onMove(e: MouseEvent) {
      const { x, y } = getMousePos(e);
      if (interaction.dragging) {
        interaction.dragging.fx = x;
        interaction.dragging.fy = y;
      } else if (interaction.panning) {
        view.offsetX += e.clientX - interaction.lastX;
        view.offsetY += e.clientY - interaction.lastY;
        interaction.lastX = e.clientX;
        interaction.lastY = e.clientY;
        draw();
      } else {
        const node = findNode(x, y);
        const id = node ? node.id : null;
        if (interaction.hoverId !== id) {
          interaction.hoverId = id;
          setHoverId(id);
          draw();
        }
        canvas.style.cursor = node ? "pointer" : "default";
      }
    }

    function onUp(e: MouseEvent) {
      if (interaction.dragging) {
        const node = interaction.dragging;
        const moved =
          Math.abs(e.clientX - interaction.lastX) +
          Math.abs(e.clientY - interaction.lastY);
        node.fx = null;
        node.fy = null;
        simulation.alphaTarget(0);
        if (moved < 5) {
          navigate({ to: "/posts/$slug", params: { slug: node.id } });
        }
      }
      interaction.dragging = null;
      interaction.panning = false;
    }

    function onLeave() {
      if (interaction.dragging) {
        interaction.dragging.fx = null;
        interaction.dragging.fy = null;
        simulation.alphaTarget(0);
      }
      interaction.dragging = null;
      interaction.panning = false;
      if (interaction.hoverId) {
        interaction.hoverId = null;
        setHoverId(null);
        draw();
      }
    }

    function onWheel(e: WheelEvent) {
      e.preventDefault();
      const rect = canvas.getBoundingClientRect();
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;
      const delta = e.deltaY > 0 ? 0.9 : 1.1;
      const newScale = Math.max(0.3, Math.min(3, view.scale * delta));
      // 以鼠标位置为中心缩放
      view.offsetX = mx - ((mx - view.offsetX) / view.scale) * newScale;
      view.offsetY = my - ((my - view.offsetY) / view.scale) * newScale;
      view.scale = newScale;
      draw();
    }

    canvas.addEventListener("mousedown", onDown);
    canvas.addEventListener("mousemove", onMove);
    canvas.addEventListener("mouseup", onUp);
    canvas.addEventListener("mouseleave", onLeave);
    canvas.addEventListener("wheel", onWheel, { passive: false });

    const onResize = () => {
      resize();
      simulation.force("x", forceX<SimNode>(width / 2).strength(0.04));
      simulation.force("y", forceY<SimNode>(height / 2).strength(0.04));
      simulation.alpha(0.3).restart();
    };
    window.addEventListener("resize", onResize);

    // 主题切换时重绘（宣纸星图 / 夜空星象配色不同）
    const themeObserver = new MutationObserver(() => draw());
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });

    // 对外控制器（按钮调用）
    ctrlRef.current = {
      setColor: (v) => {
        interaction.colorByTag = v;
        draw();
      },
      resetLayout: () => {
        shouldFit = true;
        for (const n of nodes) {
          const angle = Math.random() * Math.PI * 2;
          const r = Math.min(width, height) * 0.28;
          n.x = width / 2 + Math.cos(angle) * r;
          n.y = height / 2 + Math.sin(angle) * r;
          n.vx = 0;
          n.vy = 0;
          n.fx = null;
          n.fy = null;
        }
        simulation.alpha(1).restart();
      },
    };

    return () => {
      simulation.stop();
      canvas.removeEventListener("mousedown", onDown);
      canvas.removeEventListener("mousemove", onMove);
      canvas.removeEventListener("mouseup", onUp);
      canvas.removeEventListener("mouseleave", onLeave);
      canvas.removeEventListener("wheel", onWheel);
      window.removeEventListener("resize", onResize);
      themeObserver.disconnect();
      ctrlRef.current = null;
    };
  }, [graphData, navigate, tagColorIndexMap]);

  return (
    <div className="relative h-[calc(100vh-12rem)] min-h-[400px] w-full overflow-hidden rounded-2xl border border-border bg-card">
      <canvas ref={canvasRef} className="h-full w-full" aria-label="文章知识图谱：主题切换时在宣纸星图与深空星图之间联动" />
      <div className="absolute left-4 top-4 flex flex-col gap-2">
        <button
          onClick={() => {
            setColorByTag(!colorByTag);
            ctrlRef.current?.setColor(!colorByTag);
          }}
          className="card-glow flex items-center gap-2 rounded-sm border border-border bg-background/80 px-3 py-1.5 font-serif text-xs tracking-wider backdrop-blur hover:bg-secondary"
        >
          <span
            className="h-3 w-3 rounded-full"
            style={{ background: colorByTag ? "#C5392B" : "#2B7CA6" }}
          />
          {colorByTag ? "按星宿着色" : "统一星色"}
        </button>
        <button
          onClick={() => ctrlRef.current?.resetLayout()}
          className="card-glow flex items-center gap-2 rounded-sm border border-border bg-background/80 px-3 py-1.5 font-serif text-xs tracking-wider backdrop-blur hover:bg-secondary"
        >
          <Shuffle size={12} /> 重排星轨
        </button>
      </div>
      <div className="absolute bottom-4 right-4 font-mono text-xs text-muted-foreground">
        {graphData.nodes.length} 星辰 · {graphData.edges.length} 星轨 · 拖拽移动 · 滚轮缩放 · 点击跳转
      </div>
      {hoverId && (
        <div className="absolute bottom-4 left-4 rounded-sm border border-border bg-background/80 px-3 py-1.5 font-calligraphy text-sm tracking-wider backdrop-blur">
          {graphData.nodes.find((n) => n.id === hoverId)?.label}
        </div>
      )}
    </div>
  );
}
