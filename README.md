# 遇见江楠 · Agent Workbase

> Blog as Agent Workbase / 博客即 Agent 工作基座

水墨国风 × 科技感的个人博客，内容用 Obsidian 写，构建产物纯静态；同时以公网私密 MCP Server 作为 Agent 接入层，提供跨设备、跨 Agent 的一致上下文、知识检索、项目状态、Skill/MCP Registry 与受控写入。

## 三层定位

```text
Obsidian Vault（D:\Data\工作台）  = 知识事实源
公开博客（本仓库构建产物）        = 展示层
公网私密 MCP Server（server/mcp/） = Agent 接入层
```

详细设计见 [`docs/agent-workbase-mcp-v0.1.md`](docs/agent-workbase-mcp-v0.1.md)。

检索层使用 SQLite FTS + frontmatter / WikiLink / 可解释信号，**不引入向量数据库**。

## 技术栈

- **框架**：React 19 + TypeScript + Vite 7
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

构建（开发期用 `pnpm vite build`，会先让 TanStack Router 生成 `routeTree.gen.ts`）：

```bash
pnpm vite build
# 产物在 dist/
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
deploy/              # VPS 部署脚本（见下文「部署」）
```

## 数据流

博客内容**不在仓库内**，从外部 Obsidian Vault 读取：

- Vault 根目录通过 `VAULT_ROOT` 环境变量配置
- 一级目录 = 栏目：`文章/`、`项目/`、`友链/`
- 各栏目对应 `src/lib/<栏目>.ts` 解析器（plugin 风格加载）

构建期 Vite 通过虚拟模块 `virtual:vault-tree` 把整个 Vault 注入；dev server 走 `/vault/*` 中间件直读磁盘，build 走 `generateBundle` emit 到 `dist/vault/`。

## 双主题

- **朝曦（亮）**：石青 / 冰川青 / 朱砂暖锚
- **夜隐（暗）**：子夜玄青 / 黛蓝 / 冰川青

主题切换有日/月弧线穿越动画 + 暮光/晨曦帷幕过渡。

## 部署

VPS：`ubuntu@<VPS_IP>`（自有 IP，**不要提交到公开仓库**），Caddy 静态站点，部署目录 `/home/studio/app`。

> 部署细节涉及具体 IP，**不要提交到 GitHub**。`deploy/pull.ps1` / `deploy/deploy-code.sh` 里的 IP 走环境变量（`$env:BLOG_VPS`）+ 私钥路径（`$env:BLOG_SSH_KEY`），不在源码里硬编码。

**关键设计：内容/代码分离，两条独立推送链路**。

| 改什么 | 本地命令 | 推到哪 | 谁来 build |
|---|---|---|---|
| **前端代码**（本仓库） | `.\deploy\pull.ps1` | `scp repo.tar.gz` → VPS | VPS `deploy-code.sh` 跑 `npm run build` |
| **博客内容**（Obsidian 工作台） | `D:\Data\工作台\sync.ps1 "msg"` | `git push vps main` → bare repo | `post-receive` 钩子自动跑 `npm run build` |

**为什么分离**：内容频繁更新（每天改文章），用 Git push 走 SSH 触发 post-receive 钩子最轻量；代码更新频率低，但 VPS 到 GitHub 网络不稳定，**不能依赖 VPS `git pull`**，所以走本地 tar + scp。

### 日常：改完前端代码

```powershell
cd D:\Code\Front-end\博客
.\deploy\pull.ps1
```

做了什么：
1. `tar` 打包（排除 `node_modules / dist / .git / .backup`）
2. `scp` 到 VPS `/home/studio/app/repo.tar.gz`
3. `ssh` 跑 VPS 上 `/home/studio/app/deploy-code.sh`：解压 → 软链复用旧 `node_modules` → `npm run build` → `rsync` 到 public

**先 commit + push 到 GitHub 再 deploy**：

```powershell
git add -A
git commit -m "..."
git push origin main
.\deploy\pull.ps1
```

### 日常：改完博客内容

```powershell
cd D:\Data\工作台
.\sync.ps1 "修复了一个 typo"
```

VPS `post-receive` 钩子自动：
1. `git checkout` 内容到 `/home/studio/workbench`
2. `cd /home/studio/app/repo && VAULT_ROOT=/home/studio/workbench npm run build`
3. `rsync dist/ → public/`

部署日志：`ssh ubuntu@<VPS_IP> "tail -30 /home/studio/app/deploy.log"`

### 首次部署

详见 `deploy/README.md`：
- VPS 裸机初始化（`setup-vps.sh`）
- Caddy 配置（`caddyfile`）
- 部署脚本用法（`pull.ps1` / `deploy-code.sh`）
- 工作台 bare repo 初始化（`D:\Data\工作台\deploy-vps.sh`）

## Agent Workbase MCP

公网私密 Agent 接入层，代码在 `server/mcp/`。

```text
Obsidian Vault（D:\Data\工作台）  = 知识事实源
公开博客（本仓库构建产物）        = 展示层
MCP Server（server/mcp/）         = Agent 接入层
```

同一份 vault，两套扫描：博客走 Vite `virtual:vault-tree`（只收 public 且非草稿）；MCP 走 indexer 写 SQLite。不共享同一份 index。

| 文档 | 路径 |
|---|---|
| 设计（why / 验收） | [`docs/agent-workbase-mcp-v0.1.md`](docs/agent-workbase-mcp-v0.1.md) |
| 字段契约 | [`SCHEMA.md`](SCHEMA.md) |
| 实现说明 | [`server/mcp/README.md`](server/mcp/README.md) |
| 配置模板 | [`server/mcp/config.example.yaml`](server/mcp/config.example.yaml) |

关键约束（写代码以契约为准，不要抄现有 `internal/`）：

- 所有 MCP 工具都必须带 Bearer token
- 描述性字段从 `Workbase/` 即时读，Go 不硬编码文案
- Agent 改正式内容只走 `proposal.create`；Inbox 是待办，不审批
- Token 在 `{runtime}/auth.sqlite`，不进 `config.yaml`
- 8 个标准 scope；`admin:reindex` 不是 Agent scope
- 敏感过滤默认关；授权 Agent 读完整原文
- reindex 只走本机 `POST /internal/reindex`；Caddy 必须排除 `/internal*`
- `notes.id` = vault 相对路径（正斜杠，含 `.md`）；入库 / 查询一律 ToSlash
- `knowledge.search` 已传 kind 过滤后为空 → 空结果，不回落默认；intent / scope 非法值报错
- `context.startup` 跳过 secret；各 get 对 draft 放行，secret → `secret_blocked`

骨架（config / `workbase.identity` / Token SQLite / `/internal/reindex`）已按契约落地。索引读路径、写路径、热度 / 审计仍是后续 PR；MCP 进程等那些合完再启。部署脚本在 `deploy/mcp/`。
