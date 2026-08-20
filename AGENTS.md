# AGENTS.md

## Dependencies
- react-markdown + remark-gfm + remark-math + rehype-katex + rehype-highlight：Markdown 渲染管线（GFM 扩展、数学公式、代码高亮）
- katex：数学公式字体与样式（CSS 在 styles.css 中 @import）
- highlight.js：代码语法高亮主题
- fuse.js：文章全文模糊搜索
- js-yaml：YAML frontmatter 解析（浏览器端纯 JS 实现）
- d3-force：知识图谱力导向布局引擎（Obsidian 图谱同款方案，alpha 衰减实现收敛后静止）

## Architecture
- 内容层：`D:/Data/工作台/`（Obsidian Vault），通过 `VAULT_ROOT` 环境变量可配。`vite.config.ts` 构建时遍历 Vault，按**一级目录（栏目）分组**产出虚拟模块 `virtual:vault-tree`（`栏目 → {相对路径 → md原文}`），排除 `.obsidian` / `.trash` / `Workbase`
- 栏目（一级目录）即内容类型：`文章`（博客正文）、`项目`（项目卡片）、`友链`（友链卡片）。每个栏目一套解析器：
  - `src/lib/posts.ts`：文章（frontmatter 解析、WikiLink→站内链接、正链/反链索引）
  - `src/lib/projects.ts`：项目（name/summary/repo/demo/stack/status 卡片字段）
  - `src/lib/friends.ts`：友链（name/url/avatar/desc）
- 数据层：`src/lib/posts.ts` 负责 frontmatter 解析、WikiLink→站内链接转换、正链/反链索引构建
- 渲染层：`src/components/markdown-renderer.tsx` 配置 remark/rehype 插件链，自定义 pre 组件加复制按钮
- 图谱层：`src/components/graph-view.tsx` 用 d3-force 力导向模拟 + Canvas 渲染；模拟收敛后自动 fit-to-view
- 附件：`virtual:vault-index`（文件名→相对路径）解析 `![[图]]`；dev 走 `/vault/*` 中间件直读磁盘，build 走 `generateBundle` emit 到 `dist/vault/`

## Content Sync（部署方向）
- 博客源码 GitHub repo：`Luo-root/jiangnan-blog`（友链申请 Issue 入口 `.../issues/new`）
- 本地：`VAULT_ROOT` 直指 `D:/Data/工作台`，直读磁盘，零拷贝
- 部署：目标为自托管 VPS。方案 = 工作台作 Git 仓库 → 服务器 bare repo + post-receive hook 触发 `VAULT_ROOT=<checkout目录> npm run build`
- 项目卡片整卡 `Link` 跳详情页 `/projects/$slug`，外链按钮在详情页内

## Patterns / Constraints
- Workbase MCP 内容类型只有 `context/` / `skills/` / `mcps/`。不要再建 `Workbase/conventions/` 或 `Workbase/policies/`：工程规范写进 context pack，Git 流程写进 skill，可见性规则只放 `config.yaml` 的 `schema.visibility_policy` / `schema.visibility_default`。Workbase 下其它路径如果出现 md，indexer 当普通 private note，不是第四种 registry。
- MCP 实现以 `SCHEMA.md` + `docs/agent-workbase-mcp-v0.1.md` + `server/mcp/README.md` 为准。骨架（config / identity / Token SQLite / reindex mux）已按契约落地；索引读路径、写路径 3-way、热度 / 审计仍是后续 PR，不要把旧 `noteID()` / payload-as-ours / 默认敏感正则当规格。
- 用户要求保留 Obsidian 图谱关系与附件引用完整性：WikiLink 双向链接 + 反向链接区 + 可视化图谱
- 内容可替换：用户将 .md 文件放入 `D:/Data/工作台/文章/`（或通过 Obsidian 直接编辑）即可自动解析，刷新或重新构建可见
- 设计方向：水墨国风 × 科技感。「墨分五色」ink-1~ink-5 token（浅色=焦浓重淡清墨阶，深色=月白墨韵反转）+ 宣纸/玄黑底（oklch 双套）；国风工具类 `.seal`/`.vertical-text`/`.hairline`/`.hairline-y`/`.huawen`（回纹 mask）/`.ink-blot`（墨晕）/`.paper-grain`（纸纹）/`.dot-grid` 定义在 styles.css @layer utilities；山水剪影组件 `src/components/ink-mountains.tsx`（三层 fill-ink-* 路径，用 mask 渐隐溶入底色）；日期/标签/统计数字一律 font-mono；文章卡片去卡片化：册页式 row/compact/featured 变体，中文数字编号（壹贰叁…）
- 生图功能沙箱默认未开启：水墨素材用 CSS 渐变 + SVG（mask 回纹、feTurbulence 纸纹、贝塞尔山形）实现，主题自适应优于位图

## What Didn't Work
- ❌ `pnpm run build`（tsc --noEmit && vite build）首次失败 → routeTree.gen.ts 未更新导致路由类型不匹配 → 先运行 `npx vite build` 让 TanStack Router 插件生成路由树，再跑完整 build
- ❌ 手写物理模拟实现力导向图 → 节点持续漂移抖动永不收敛、悬停时整图跳动 → 换 d3-force（alpha 衰减到 alphaMin 自动停止，拖拽用 alphaTarget 标准模式）

## Lessons
- TypeScript 闭包内不保持 `const` 变量的 null 收窄：graph-view.tsx 中 `canvas`/`ctx` 在 useEffect 闭包内报 possibly null → 用非空断言 `canvasRef.current!` 解决
- Canvas 绘制用固定颜色无法适配明暗主题：需检测 `document.documentElement.classList.contains("dark")` 动态配色，并用 MutationObserver 监听 html class 变化触发重绘
- useEffect 依赖每次渲染新建的对象（如 tagColorMap）会导致副作用反复重跑：静态数据用 useMemo 缓存保证引用稳定
- Hero 区绝对定位的装饰元素（竖排文字 + 印章容器）按 top-1/2 垂直居中会因内容总高超出而压到下方统计卡片：定位基准要上移（top-1/3）或显式限制装饰区高度，截图验收时重点检查与卡片的交叠
