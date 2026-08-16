# 遇见江楠 · Obsidian 驱动静态博客

水墨国风 × 科技感的个人博客，内容用 Obsidian 写，构建产物纯静态、可部署到任何静态托管。

## 技术栈

- **框架**：React 19 + TypeScript + Vite
- **路由**：TanStack Router（文件式路由，类型安全）
- **样式**：Tailwind CSS v4 + design token（`src/styles.css`）
- **Markdown**：react-markdown + remark-gfm + remark-math + rehype-katex + rehype-highlight
- **搜索**：fuse.js（文章全文模糊搜索）
- **图谱**：d3-force 力导向 + Canvas 渲染
- **包管理**：pnpm

## 启动

```bash
pnpm install
pnpm dev          # http://localhost:3015
```

构建：

```bash
pnpm vite build   # 产物在 dist/
```

## 项目结构

```
src/
  components/        # UI 组件
  content/posts/     # 内置示例文章（生产实际数据从外部 Vault 加载）
  hooks/             # 全局钩子（滚动变量、响应式等）
  lib/               # 业务逻辑：posts / projects / friends / 搜索 / 解析器
  routes/            # TanStack Router 文件式路由
  styles.css         # 全局样式 + design token
  main.tsx           # 应用入口

public/assets/       # 静态资源（主题图、字体等）
```

## 数据流

博客内容**不在仓库内**，从外部 Obsidian Vault 读取：

- Vault 根目录通过 `VAULT_ROOT` 环境变量配置
- 一级目录 = 栏目：`文章/`、`项目/`、`友链/`
- 各栏目对应 `src/lib/<栏目>.ts` 解析器（plugin 风格加载）

## 双主题

- **朝曦（亮）**：石青 / 冰川青 / 朱砂暖锚
- **夜隐（暗）**：子夜玄青 / 黛蓝 / 冰川青

主题切换有日/月弧线穿越动画 + 暮光/晨曦帷幕过渡。

## 部署

VPS 部署方案：本地 push → GitHub → VPS `git pull` → `npm ci` → `npm run build` → nginx。

具体脚本见同目录的 `deploy/` 文件夹（待补充）。
