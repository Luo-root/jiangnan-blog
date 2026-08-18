# 遇见江楠 · Agent Workbase MCP v0.1 设计文档

> 状态：Draft v0.1  
> 日期：2026-08-17  
> 项目定位：Blog as Agent Workbase / 博客即 Agent 工作基座  
> 当前仓库：`jiangnan-blog`（后续建议演进为 `jiangnan-workbase`）

---

## 0. 一句话结论

「遇见江楠」不再只是一个 Obsidian 驱动的静态博客，而是一套以 Obsidian Vault 为事实源、以公开博客为展示层、以公网私密 MCP Server 为 Agent 接入层的个人 Agent 工作基座。

核心目标：

```text
跨设备、跨 Agent、跨工具，仍能获得一致的个人上下文、项目状态、知识索引、Skill/MCP 能力目录和可控写入体验。
```

v0.1 不追求复杂 RAG、不引入向量数据库、不做网页聊天框、不做 Agent 保姆式安装器，而是先把以下能力打稳：

```text
1. Workbase 自描述
2. Startup Context
3. 知识检索与读取
4. 项目上下文
5. Skill Registry
6. MCP Registry
7. 通用 Proposal 写入协议
8. Inbox 独立待办区
9. Git-backed 写入演进路线
10. 公网私密访问与安全边界
```

---

## 1. 背景与问题

### 1.1 当前博客已有基础

当前系统已经具备：

```text
D:\Data\工作台\                       # Obsidian Vault，内容事实源
D:\Code\Front-end\博客\              # React/Vite 静态博客代码
/home/studio/workbench                 # VPS 上的 Vault 镜像
/home/studio/app/repo                  # VPS 上的博客代码
/home/studio/app/public                # Caddy 静态服务目录
```

内容链路：

```text
本地 Obsidian Vault
  ↓ sync.ps1
VPS bare repo
  ↓ post-receive
/home/studio/workbench
  ↓ VAULT_ROOT build
/home/studio/app/public
```

前端链路：

```text
本地前端仓库
  ↓ deploy/pull.ps1
scp repo.tar.gz
  ↓ deploy-code.sh
/home/studio/app/repo
  ↓ npm run build
/home/studio/app/public
```

当前博客已支持：

- Obsidian Vault 构建时注入。
- 文章 / 项目 / 友链 / 归档多栏目。
- WikiLink 与普通 Markdown 相对链接处理。
- 反链与图谱。
- 公开部署到 VPS + Caddy。

### 1.2 新问题

普通博客解决的是：

```text
人如何阅读我的知识沉淀？
```

Agent Workbase 要解决的是：

```text
Agent 如何稳定、可信、低成本地读取我的长期上下文？
Agent 如何知道我当前在做什么？
Agent 如何复用我的 Skill / MCP 能力目录？
Agent 如何提出知识库写入，而不污染正式内容？
换设备或换 Agent 工具时，如何保持体验一致？
```

### 1.3 设计目标

v0.1 目标：

```text
任意授权 Agent 连接公网私密 MCP endpoint 后，可以：

1. 读取 Workbase manifest。
2. 获取 startup context。
3. 搜索和读取授权范围内的知识库内容。
4. 获取项目状态、决策、下一步。
5. 获取 Skill / MCP Registry。
6. 创建 proposal，或追加 inbox 待办。
7. 所有访问有权限边界和审计记录。
8. 不泄露真实 IP、私钥路径、token、secret 内容。
```

---

## 2. 非目标

v0.1 明确不做：

```text
1. 不做网页聊天框。
2. 不引入向量数据库。
3. 不直接让公网 MCP 修改正式 Vault 正文。
4. 不实现完整 OAuth 2.1 授权服务器。
5. 不做多用户系统。
6. 不做复杂安装器。
7. 不为每个 Agent 工具单独生成安装教程。
8. 不将 private / secret 内容发布到公开博客。
9. 不改变现有 Obsidian 作为内容事实源的原则。
10. 不大规模重构前端仓库结构。
```

后续可以演进，但 v0.1 不承担这些复杂度。

---

## 3. 核心原则

### 3.1 Obsidian Vault 是事实源

正式知识内容以 Markdown 文件为事实源。

```text
文章 = 文章/*.md
项目 = 项目/*.md
Skill Registry = Workbase/skills/*.md
MCP Registry = Workbase/mcps/*.md
Context Pack = Workbase/context/*.md
普通笔记 = Vault 内其它 Markdown
```

MCP Server 不维护另一份长期真相。它只做：

```text
读取、索引、暴露、生成 proposal、记录审计。
```

### 3.2 公网可访问，不等于公开可访问

MCP Server 部署在公网，方便跨设备和跨 Agent 接入。

但访问必须是私密的：

```text
HTTPS + Authorization Bearer Token + scope + audit
```

### 3.3 Agent 不是傻瓜，不需要保姆式 guide

Skill/MCP Registry 的职责不是教每个 Agent 工具怎么安装，而是提供：

```text
能力是什么
在哪里
来源是什么
风险是什么
授权要求是什么
是否公开可迁移
```

Agent 拿到来源、endpoint、transport、auth、scope 后，自行适配自己的运行时。

### 3.4 写入统一走 Proposal

不要为每类写入都创建一个工具：

```text
context.update
article.create
skill.register
mcp.register
project.patch
```

统一入口：

```text
proposal.create
```

由 proposal 内部的：

```text
target.type + operation.type + payload
```

表达不同写入目标。

inbox 不是「写入」，是独立待办：走 `inbox.append` / `inbox.update`，不经过 Proposal，也没有审批 / apply / commit（见 §16）。

### 3.5 context.startup 是派生结果

`context.startup` 不应该直接手写或直接 patch。

它应由多个 context pack 合成：

```text
startup = profile + engineering-style + security-boundaries + active-projects + recent-focus
```

修改 startup 的正确方式是修改背后的 context pack。

### 3.6 先结构化，再智能化

v0.1 不引入向量库。优先使用：

```text
frontmatter
title
tag
path
heading
WikiLink
backlink
SQLite FTS / 简单全文索引
```

个人知识库第一阶段更需要可靠结构，而不是不透明召回。

---

## 4. 总体架构

### 4.1 逻辑分层

```text
┌───────────────────────────────────────────────┐
│                  Agent Clients                 │
│ MiniMax Code / Cursor / Claude / ChatGPT / ... │
└───────────────────────┬───────────────────────┘
                        │ HTTPS + Bearer Token
                        ▼
┌───────────────────────────────────────────────┐
│          Jiangnan Workbase MCP Server          │
│ manifest / context / knowledge / project / ... │
└───────────────────────┬───────────────────────┘
                        │ read index / write proposals
                        ▼
┌───────────────────────────────────────────────┐
│             Workbase Private Store             │
│ index / proposals / inbox / audit / config     │
└───────────────────────┬───────────────────────┘
                        │ index source
                        ▼
┌───────────────────────────────────────────────┐
│              VPS Vault Mirror                  │
│           /home/studio/workbench               │
└───────────────────────┬───────────────────────┘
                        │ sync from local
                        ▼
┌───────────────────────────────────────────────┐
│              Local Obsidian Vault              │
│             D:\Data\工作台                    │
└───────────────────────────────────────────────┘
```

### 4.2 文件层建议

在本地 Vault 中新增 `Workbase/`（决策已确认：`Workbase/` 放在 `D:\Data\工作台\` 根下）：

```text
D:\Data\工作台\
├── Workbase\
│   ├── context\
│   │   ├── profile.md
│   │   ├── engineering-style.md
│   │   ├── security-boundaries.md
│   │   ├── active-projects.md
│   │   └── recent-focus.md
│   ├── skills\
│   │   └── *.md
│   ├── mcps\
│   │   └── *.md
│   └── policies\
│       └── visibility.md
├── 文章\
├── 项目\
├── 友链\
└── 部署溯源\
```

注意：`Agent Inbox/` 不放在本地 Vault。proposal 与 inbox 只存 VPS 私有区（决策已确认），且两者完全独立：

- proposal = 正式知识写入请求，经 webUI 审批后 apply 回正式 Vault 文件。
- inbox = 独立待办（todo），状态机 `pending → reviewing → done | abandoned`，done/abandoned 保留 7 天后自动删除；不转 proposal、不审批、不 apply、不 commit、不与 Obsidian 联动。

`Workbase/` 是私有工作基座目录，不作为公开博客栏目。构建时 `virtual:vault-tree` 应排除 `Workbase/`（与 `.obsidian` / `.trash` 同等处理），否则会被当成一个新栏目暴露到公开站。

在 VPS 私有区新增：

```text
/home/studio/workbase/
├── config.yaml             # 不进 Git，保存 token hash / scopes / paths
├── index/
│   ├── manifest.json
│   ├── notes.sqlite
│   ├── graph.json
│   ├── context/
│   ├── skills.json
│   └── mcps.json
├── proposals/
│   └── *.md
├── inbox/
│   └── *.md
└── audit/
    └── audit.sqlite
```

---

## 5. 命名与仓库演进

当前仓库仍为：

```text
Luo-root/jiangnan-blog
```

但产品定位已经超出 blog。后续建议演进为：

```text
Luo-root/jiangnan-workbase
```

中文名：

```text
遇见江楠 · Agent 工作基座
```

建议路线：

```text
阶段 1：不改 repo 名，先补 docs/ 与 server/mcp/。
阶段 2：MCP v0.1 跑通后，再评估 GitHub rename。
阶段 3：如需要，再演进 monorepo 结构。
```

初始目录不建议大迁移：

```text
D:\Code\Front-end\博客\
├── src\                  # 现有 web
├── server\
│   └── mcp\              # 后续 Go MCP Server
├── docs\
│   └── agent-workbase-mcp-v0.1.md
├── deploy\
│   └── mcp\              # 后续 MCP 部署脚本
└── README.md
```

---

## 6. MCP 传输与认证

### 6.1 Transport

远程 MCP 使用 Streamable HTTP。

建议 endpoint：

```text
https://mcp.<domain>/mcp
```

备案 / HTTPS 未完成前，开发验证可以使用：

```text
localhost
内网
临时隧道
测试域名
```

但正式目标是：

```text
Caddy HTTPS
  ↓
/mcp
  ↓
Go MCP Server
```

官方 MCP 2026-07-28 规范继续推进 Streamable HTTP，并将旧 HTTP+SSE 视为不建议新实现采用的路径；同时规范引入 stateless core、header-based routing、cacheable list results 等能力。设计时按 Streamable HTTP 对齐。

### 6.2 v0.1 认证

v0.1 使用简单 Bearer Token：

```http
Authorization: Bearer <WORKBASE_TOKEN>
```

要求：

```text
1. token 不放 URL query。
2. token 不写入公开仓库。
3. 服务端只存 token hash。
4. 每个 Agent 使用独立 token。
5. token 绑定 scopes。
6. audit 记录 client，不记录 token。
```

示例配置：

```yaml
clients:
  - id: minimax-code
    token_hash: sha256:...
    scopes:
      - read:context
      - read:knowledge
      - read:project
      - read:registry
      - write:proposal
      - write:inbox
  - id: readonly-agent
    token_hash: sha256:...
    scopes:
      - read:context
      - read:knowledge
      - read:project
```

### 6.3 后续 OAuth

MCP 官方授权规范基于 OAuth 2.x / OAuth 2.1 方向，HTTP-based transport 的受保护资源访问应使用 Bearer access token。后续如需要多设备 consent、动态客户端注册、token 轮换、撤销，可升级到 OAuth/OIDC。

v0.1 不实现完整 OAuth Server，避免认证系统吞掉主体复杂度。

---

## 7. 权限与 Scope

建议 v0.1 scopes：

```text
read:manifest       # 读取 workbase.manifest
read:context        # 读取 context.startup / context.get
read:knowledge      # 搜索和读取 note/article
read:project        # 读取 project list/get
read:registry       # 读取 skill/mcp registry
write:proposal      # 创建 proposal
write:inbox         # 追加 / 更新 inbox 待办
read:inbox          # 读取 inbox 待办
ops:audit           # 查看审计摘要
admin:reindex       # 触发重建索引，默认不给普通 Agent
```

工具权限矩阵：

| Tool | Required scope | v0.1 风险 |
|---|---|---|
| `workbase.manifest` | `read:manifest` | low |
| `context.startup` | `read:context` | medium |
| `context.get` | `read:context` | medium |
| `knowledge.search` | `read:knowledge` | medium |
| `knowledge.get` | `read:knowledge` | high，可能返回私密正文 |
| `project.list` | `read:project` | medium |
| `project.get` | `read:project` | high，可能返回项目状态 |
| `skill.list` | `read:registry` | low/medium |
| `skill.get` | `read:registry` | medium |
| `mcp.list` | `read:registry` | medium |
| `mcp.get` | `read:registry` | medium/high，可能含私密 endpoint |
| `proposal.create` | `write:proposal` | medium/high |
| `proposal.list` | `write:proposal` | medium |
| `proposal.get` | `write:proposal` | medium/high |
| `inbox.append` | `write:inbox` | low |
| `inbox.update` | `write:inbox` | low |
| `inbox.list` | `read:inbox` | low |
| `inbox.get` | `read:inbox` | low |
| `audit.list_recent` | `ops:audit` | medium |

---

## 8. 内容可见性模型

所有 Markdown 建议支持 frontmatter：

```yaml
visibility: public | private | secret | draft
```

语义：

| visibility | 公开博客 | 公共 Agent Index | 私密 MCP | 说明 |
|---|---:|---:|---:|---|
| `public` | yes | yes | yes | 可公开展示 |
| `private` | no | no | yes | 个人知识库，授权 Agent 可读 |
| `secret` | no | no | no by default | 默认不暴露给远程 MCP |
| `draft` | no by default | no | configurable | 草稿，按策略读取 |

缺省策略建议：

```text
现有公开博客内容如无 visibility，暂按 public 兼容。
Workbase/context 默认 private。
Workbase/skills 可 public/private。
Workbase/mcps 可 public/private。
部署溯源默认 private。
secret 必须显式标注。

（inbox 只存 VPS 私有区 `/home/studio/workbase/inbox/`，不进入 Vault，因此不参与 visibility 判定。）
```

敏感模式拦截：

```text
真实 VPS IP
token / api_key / secret
私钥路径
ssh private key block
.env 内容
Cookie / Authorization header
```

---

## 9. MCP 工具集 v0.1

最终工具集保持克制。

```text
workbase.manifest

context.startup
context.get

knowledge.search
knowledge.get

project.list
project.get

skill.list
skill.get

mcp.list
mcp.get

proposal.create
proposal.list
proposal.get

inbox.append
inbox.update
inbox.list
inbox.get

audit.list_recent
```

### 9.1 `workbase.manifest`

用途：让新 Agent 知道这个 Workbase 是什么、支持什么能力、边界是什么。

返回字段：

```json
{
  "id": "jiangnan-workbase",
  "name": "遇见江楠 · Agent Workbase",
  "version": "0.1.0",
  "description": "Blog as Agent Workbase",
  "capabilities": {
    "context": true,
    "knowledge": true,
    "project": true,
    "skill_registry": true,
    "mcp_registry": true,
    "proposal": true,
    "inbox": true,
    "direct_write": false,
    "vector_search": false
  },
  "visibility_policy": {
    "public": "可公开展示与索引",
    "private": "授权 Agent 可读",
    "secret": "默认不暴露给远程 MCP",
    "draft": "草稿，按策略读取"
  },
  "tools": ["context.startup", "knowledge.search"]
}
```

### 9.2 `context.startup`

用途：给新 Agent 一份启动上下文，让它快速进入状态。

它是派生结果，不是直接写入目标。

来源：

```text
Workbase/context/profile.md
Workbase/context/engineering-style.md
Workbase/context/security-boundaries.md
Workbase/context/active-projects.md
Workbase/context/recent-focus.md
```

返回内容：

```text
身份与语言
工程偏好
安全边界
活跃项目
当前重点
下一步
建议读取的 context packs
```

### 9.3 `context.get`

用途：读取具体 context pack。

请求：

```json
{
  "id": "engineering-style"
}
```

返回：

```json
{
  "id": "engineering-style",
  "title": "工程风格",
  "visibility": "private",
  "updated_at": "2026-08-17T17:00:00+08:00",
  "content": "...markdown...",
  "metadata": {}
}
```

### 9.4 `knowledge.search`

用途：搜索授权范围内的知识库。

请求：

```json
{
  "query": "Agent Workbase proposal",
  "scope": "all",
  "intent": "general",
  "limit": 10
}
```

返回摘要要足够让 Agent 判断是否值得 get：

```json
{
  "results": [
    {
      "id": "note_xxx",
      "title": "Agent Workbase MCP 设计",
      "path_hint": "Workbase/...",
      "type": "note",
      "visibility": "private",
      "summary": "...",
      "matched_fields": ["title", "heading", "body"],
      "score": 0.92,
      "matched_via": "ft5_fulltext + wikilink_backref",
      "signals": {
        "ft5_fulltext": 0.85,
        "wikilink_backref": 0.40,
        "frontmatter": 0.30,
        "access": 0.25,
        "recency": 0.20
      }
    }
  ]
}
```

`intent` 可选，取值 `why` / `when` / `entity` / `general`（默认 `general`），用于调整各信号排序权重：`why` 抬高反链（因果上下文）权重、`when` 抬高 frontmatter 时间权重、`entity` 抬高 title/tags/正链权重。`signals` 把 `score` 拆成可解释分，`matched_via` 标注命中了哪些信号——让 Agent 面对的是可解释的信号分解，而不是黑盒分数。其中 `access` 反映读取热度（见 §19.4）。

### 9.5 `knowledge.get`

用途：读取具体 note/article/context 源内容。

请求：

```json
{
  "id": "note_xxx",
  "include": ["content", "metadata", "links"]
}
```

返回：

```json
{
  "id": "note_xxx",
  "title": "...",
  "type": "article|note|context_pack|project|skill|mcp_server",
  "visibility": "private",
  "frontmatter": {},
  "content": "...markdown...",
  "links": {
    "forward": [],
    "backlinks": []
  },
  "base_commit": "abc123"
}
```

### 9.6 `project.list`

用途：列出项目摘要。

`list` 不只是 title，要返回可判断相关性的摘要。

```json
{
  "projects": [
    {
      "id": "jiangnan-workbase",
      "name": "遇见江楠 · Agent Workbase",
      "summary": "博客即 Agent 工作基座",
      "status": "active",
      "current_focus": "MCP v0.1 设计",
      "tags": ["blog", "agent", "mcp"]
    }
  ]
}
```

### 9.7 `project.get`

用途：读取项目完整上下文。

返回：

```text
项目定位
当前状态
当前重点
关键决策
下一步
风险
关联文章/笔记
仓库/部署信息的脱敏描述
```

### 9.8 `skill.list`

用途：列出 Skill Registry 摘要。

`list` 必须返回足够信息，不是只返回 title。

```json
{
  "skills": [
    {
      "id": "markdown-lint",
      "name": "Markdown Lint",
      "summary": "检查 Markdown fence/frontmatter/异常长代码块",
      "tags": ["markdown", "lint"],
      "visibility": "public",
      "risk": "low",
      "source": {
        "type": "github",
        "url": "https://example.com/..."
      }
    }
  ]
}
```

### 9.9 `skill.get`

用途：返回 Skill 完整定义。

不做安装器，只返回可信来源、元信息、正文。

```json
{
  "id": "markdown-lint",
  "name": "Markdown Lint",
  "summary": "...",
  "visibility": "public",
  "risk": "low",
  "source": {},
  "content_type": "text/markdown",
  "content": "# Markdown Lint\n..."
}
```

### 9.10 `mcp.list`

用途：列出 MCP Registry 摘要。

```json
{
  "servers": [
    {
      "id": "jiangnan-workbase",
      "name": "Jiangnan Workbase MCP",
      "summary": "私密个人知识基座 MCP",
      "transport": "streamable-http",
      "endpoint_hint": "https://mcp.<domain>/mcp",
      "auth": "bearer",
      "visibility": "private",
      "risk": "personal-knowledge-base"
    }
  ]
}
```

### 9.11 `mcp.get`

用途：返回 MCP Server 完整定义。

不教 Agent 怎么安装，只提供接入所需事实：

```json
{
  "id": "jiangnan-workbase",
  "transport": "streamable-http",
  "endpoint": "https://mcp.<domain>/mcp",
  "auth": {
    "type": "bearer",
    "token_hint": "Use your own environment variable. Never hardcode token."
  },
  "scopes": ["read:context", "read:knowledge", "write:proposal"],
  "tools": ["context.startup", "knowledge.search"],
  "source": {
    "repo": "https://github.com/Luo-root/jiangnan-workbase"
  }
}
```

### 9.12 `proposal.create`

用途：创建统一写入意图。

```json
{
  "kind": "context_update",
  "target": {
    "type": "context_pack",
    "id": "active-projects",
    "path": "Workbase/context/active-projects.md"
  },
  "operation": {
    "type": "patch_section",
    "section": "遇见江楠 Workbase / 当前重点"
  },
  "payload": {
    "format": "markdown",
    "content": "当前重点：固化 Proposal 写入协议。"
  },
  "reason": "记录本轮设计决策"
}
```

### 9.13 `proposal.list` / `proposal.get`

用途：Agent 查询自己创建过的 proposal 条目。

proposal 和 inbox 都只存储在 VPS 私有区：

```text
/home/studio/workbase/proposals/
/home/studio/workbase/inbox/
```

不进入本地 Obsidian Vault。用户的阅读和审批通过 webUI 完成（见 §21.5），不走 MCP 工具。

`proposal.list` 返回 pending / approved / applied / rejected / conflict 条目的摘要列表；`proposal.get` 返回单条完整内容（含 diff/preview 和 receipt）。

Agent 可以查看自己的 proposal 状态，但不能直接 approve/reject。审批仅限 webUI 操作。

### 9.14 `inbox.append`

用途：新建一条待办（todo），初始状态 `pending`。只存 VPS，不进本地 Vault。

请求：

```json
{
  "kind": "inbox_todo",
  "payload": {
    "format": "markdown",
    "content": "## 待办：排查博客搜索高亮样式\n..."
  },
  "reason": "本轮讨论遗留的跟进事项"
}
```

返回：

```json
{
  "id": "inbox_20260818_001",
  "status": "pending",
  "location": "/home/studio/workbase/inbox/2026-08-18T10-32-38.md"
}
```

约束：

```text
1. 只能 append 到 inbox 目录。
2. 文件名带时间戳（日期 + 时间），因为一天可能有多条，日期不够区分。
3. 不能覆盖已有正式 Vault 文件。
4. 必须过敏感信息检测。
5. 不进入本地 Obsidian；它是一条待办，不是知识写入。
```

### 9.15 `inbox.update` / `inbox.list` / `inbox.get`

inbox 是待办，Agent 和 WebUI 都能「创建」「编辑内容」「调整状态」。

`inbox.update` 编辑单条内容或改变状态：

```json
{
  "id": "inbox_20260818_001",
  "status": "reviewing",
  "payload": {
    "format": "markdown",
    "content": "（可选，替换正文）"
  }
}
```

状态机：

```text
pending → reviewing → done | abandoned
```

| 状态 | 含义 |
|---|---|
| `pending` | 待处理（刚创建，还没做） |
| `reviewing` | 待审核（已做完，等确认；不是 Proposal 审批，不触发 apply/commit） |
| `done` | 已完成（审核通过） |
| `abandoned` | 已废弃（不再需要处理） |

生命周期：

```text
done / abandoned 保留 7 天后自动删除。
pending / reviewing 保留到状态改变为止，不设自动删除。
```

`inbox.list` 返回摘要列表：

```json
{
  "items": [
    {
      "id": "inbox_20260818_001",
      "created_at": "2026-08-18T10:32:38+08:00",
      "updated_at": "2026-08-18T15:00:00+08:00",
      "created_by": "minimax-code",
      "summary": "待办：排查博客搜索高亮样式",
      "status": "done"
    }
  ]
}
```

`inbox.get` 返回单条完整内容：

```json
{
  "id": "inbox_20260818_001",
  "status": "done",
  "content": "...markdown..."
}
```

用户对 inbox 的浏览、编辑与状态调整走 webUI（见 §21.5），也可通过 MCP 工具操作。

### 9.16 `audit.list_recent`

用途：查看最近访问元信息。

请求：

```json
{
  "mode": "detail",
  "limit": 20
}
```

`mode` 取值：

| mode | 返回内容 | 用途 |
|---|---|---|
| `detail` | 操作名、时间、scope、目标 id（不含正文/查询原文） | 本机自查 |
| `hashed` | 操作名、时间、SHA-256 内容哈希（不含正文/查询原文） | 跨 Agent/设备边界对齐，不泄密 |

不返回 token，不返回完整私密正文。

---

## 10. Resource 设计

可选暴露 MCP resources：

```text
workbase://manifest
context://startup
context://packs/{id}
knowledge://notes/{id}
project://{id}
skill://{id}
mcp-server://{id}
proposal://{id}
inbox://{id}
```

v0.1 可以先只实现 tools，不强制 resources。

---

## 11. Prompt 设计

可选 prompts：

```text
startup-brief
project-handoff
session-summary-to-inbox
decision-record-proposal
lesson-capture-proposal
skill-register-proposal
mcp-register-proposal
```

v0.1 可以先写入文档，不急着实现。

---

## 12. Context Pack 设计

### 12.1 目录

```text
Workbase/context/
├── profile.md
├── engineering-style.md
├── security-boundaries.md
├── active-projects.md
└── recent-focus.md
```

### 12.2 frontmatter

```yaml
---
id: engineering-style
type: context_pack
title: 工程风格
visibility: private
updated: 2026-08-17
startup: true
priority: high
---
```

### 12.3 正文建议

```md
# 工程风格

## Rules

### 最简实现优先

- Why: 用户明确反感过度抽象。
- Apply when: 架构设计、MCP 工具集、部署脚本。

### 传参优于隐式全局状态

...
```

### 12.4 startup 合成规则

`context.startup` 由以下内容摘要合成：

```text
profile: 身份、语言、公开信息
engineering-style: 工作方式
security-boundaries: 安全红线
active-projects: 当前活跃项目
recent-focus: 最近重点
```

约束：

```text
1. startup 不直接落盘为事实源。
2. startup 可缓存为 index 派生产物。
3. 修改 startup 等价于修改 context pack。
4. startup 输出必须控制长度。
```

---

## 13. Skill Registry 设计

### 13.1 事实源

```text
Workbase/skills/*.md
```

### 13.2 frontmatter

```yaml
---
id: markdown-lint
kind: skill
name: Markdown Lint
summary: 检查 Markdown fence/frontmatter/异常长代码块
visibility: public
risk: low
tags:
  - markdown
  - lint
source:
  type: github
  url: https://example.com/markdown-lint-skill
license: unknown
---
```

### 13.3 正文

```md
# Markdown Lint

## Purpose

## When to use

## Inputs

## Outputs

## Safety

## Source
```

### 13.4 list/get 差异

```text
skill.list = 能力索引，返回摘要、风险、来源、标签。
skill.get = 完整 Skill 定义，返回 frontmatter + markdown 原文。
```

不提供：

```text
skill.get_install_guide
```

安装方式如有必要，写进 Skill 正文即可。

---

## 14. MCP Registry 设计

### 14.1 事实源

```text
Workbase/mcps/*.md
```

### 14.2 frontmatter

```yaml
---
id: playwright-mcp
kind: mcp_server
name: Playwright MCP
summary: 浏览器自动化 MCP Server
visibility: public
risk: browser-control
transport: stdio
source:
  type: github
  url: https://example.com/playwright-mcp
auth:
  type: none
scopes: []
---
```

私密 MCP 示例：

```yaml
---
id: jiangnan-workbase
kind: mcp_server
name: Jiangnan Workbase MCP
summary: 私密个人 Agent 工作基座
visibility: private
risk: personal-knowledge-base
transport: streamable-http
endpoint: https://mcp.<domain>/mcp
auth:
  type: bearer
scopes:
  - read:context
  - read:knowledge
  - write:proposal
---
```

### 14.3 list/get 差异

```text
mcp.list = MCP 能力索引，返回摘要、transport、auth、risk、source。
mcp.get = 完整 MCP 定义，返回 endpoint、auth、scopes、tools、正文。
```

不提供：

```text
mcp.get_connection_guide
```

Agent 自行适配。

---

## 15. Proposal 通用写入协议

### 15.1 定位

Proposal 是统一写入意图格式。

它不是某个具体领域的工具，而是跨领域 envelope：

```text
Intent + Target + Operation + Payload + Validation + Audit
```

### 15.2 为什么需要 Proposal

直接写入有风险：

```text
1. Agent 可能选错目标文件。
2. 可能污染正式知识结构。
3. 可能把临时想法写成长期决策。
4. 可能泄露敏感信息。
5. 可能和本地 Obsidian 编辑冲突。
```

Proposal 让写入变成：

```text
先表达意图
再展示 diff / preview
再人工确认
再落盘
再审计
```

### 15.3 Schema 草案

```yaml
id: prop_20260817_001
kind: note_patch
status: pending

created:
  by: minimax-code
  at: 2026-08-17T17:00:00+08:00
  reason: 记录 Agent Workbase MCP 设计决策

risk:
  level: medium
  reasons:
    - 修改长期项目上下文
  requires_approval: true

base:
  source: vault
  commit: abc123

target:
  type: note
  id: optional
  path: 部署溯源/jiangnan-workbase.md

operation:
  type: append_section
  section: Agent Workbase MCP

payload:
  format: markdown
  content: |
    ...

validation:
  checks:
    - target_exists
    - valid_markdown_fence
    - no_secret_pattern
    - visibility_allowed
```

### 15.4 target.type

v0.1 target types：

```text
note
context_pack
project
article
skill
mcp_server
```

| target.type | 用途 | 事实源 |
|---|---|---|
| `note` | 普通笔记 | Vault 任意授权 md |
| `context_pack` | 上下文包 | `Workbase/context/*.md` |
| `project` | 项目上下文 | `项目/*.md` |
| `article` | 正式文章 | `文章/*.md` |
| `skill` | Skill Registry | `Workbase/skills/*.md` |
| `mcp_server` | MCP Registry | `Workbase/mcps/*.md` |

### 15.5 operation.type

v0.1 操作类型：

```text
create_file
append
append_section
patch_section
replace_frontmatter
register_item
```

| operation.type | 用途 |
|---|---|
| `create_file` | 新建文章 / 新建 note |
| `append` | 文件末尾追加 |
| `append_section` | 追加到某个标题下 |
| `patch_section` | 替换某个标题内容 |
| `replace_frontmatter` | 修改 frontmatter |
| `register_item` | 新增 skill / mcp registry item |

暂不引入 JSON Patch / AST Patch。

### 15.6 Adapter 模型

对外只有统一 proposal。

内部按 target type 分发：

```text
ProposalService
  ├── NoteAdapter
  ├── ContextPackAdapter
  ├── ProjectAdapter
  ├── ArticleAdapter
  ├── SkillAdapter
  └── MCPServerAdapter
```

Adapter 负责：

```text
1. 校验 target 是否允许。
2. 校验 operation 是否支持。
3. 生成 preview / diff。
4. 检查敏感信息。
5. 检查 Markdown fence。
6. 检查 visibility。
7. 后续 apply 时负责落盘。
```

### 15.7 v0.1 支持矩阵

| target.type | operation | v0.1 行为 |
|---|---|---|
| `note` | `append/append_section` | 生成 proposal，不自动 apply |
| `context_pack` | `append_section/patch_section` | 生成 proposal |
| `project` | `patch_section/append_section` | 生成 proposal |
| `article` | `create_file` | 生成 proposal |
| `skill` | `register_item` | 生成 proposal |
| `mcp_server` | `register_item` | 生成 proposal |

---

## 16. Inbox 设计

### 16.1 定位

Inbox 是独立待办（todo）区，不是知识写入、不是 Proposal 中间态、不进入 Obsidian Vault。

它只负责：

```text
记录一条「要处理的事」，并跟踪它的状态。
```

不做审批、不做 apply、不做 git commit、不转 proposal、不与 Obsidian 联动。

### 16.2 适合内容

```text
对话中产生的跟进事项
临时想法（待办化）
排查中的问题
想做但未排期的任务
```

### 16.3 不适合内容

```text
正式文章正文
长期项目决策
安全策略正式条目
Skill/MCP 正式 registry item
部署配置
凭据/token/私钥
```

> 以上「不适合内容」若要进入正式知识库，应走 Proposal（§15），而不是塞进 inbox。

### 16.4 状态机

```text
pending → reviewing → done | abandoned
```

| 状态 | 含义 |
|---|---|
| `pending` | 待处理（刚创建，还没做） |
| `reviewing` | 待审核（已做完，等确认；不是 Proposal 审批，不触发 apply/commit） |
| `done` | 已完成（审核通过） |
| `abandoned` | 已废弃（不再需要处理） |

### 16.5 生命周期

```text
done / abandoned 保留 7 天后自动删除。
pending / reviewing 保留到状态改变为止，不设自动删除。
```

### 16.6 操作能力

Agent 与 WebUI 对 inbox 的操作：

```text
1. 创建（Agent: inbox.append / WebUI: 新建）
2. 编辑内容
3. 调整状态（pending → reviewing → done / abandoned）
```

工具：

```text
inbox.append   新建 pending 待办
inbox.update   编辑内容或改变状态
inbox.list     列出摘要
inbox.get      读取单条
```

不做审批、不做 apply、不做 commit、不转 proposal。

### 16.7 与 Proposal 的关系

两者定位不同，完全独立，不要混用：

```text
proposal = 正式知识写入请求（有明确 target + operation，走审批 → apply → commit）
inbox    = 独立待办（无 target，无审批，无 apply，无 commit）
```

inbox 不能「转 proposal」：如果 Agent 后续需要正式写入，应**独立创建一条新 proposal**，两者不自动关联。

proposal 走 webUI 审批流转：

```text
Agent 写入
  ↓
proposal.create（有 target + operation）
  ↓
/home/studio/workbase/proposals/
  ↓
用户 webUI 审批（同意 / 编辑后同意 / 拒绝）
  ↓
同意 → apply → git commit → rebuild + reindex
拒绝 → 标记作废
```

inbox 不走审批，只做状态流转：

```text
Agent / WebUI
  ↓
inbox.append（新建 pending）
  ↓
/home/studio/workbase/inbox/2026-08-18T10-32-38.md
  ↓
inbox.update（pending → reviewing → done / abandoned）
  ↓
7 天后自动删除（done / abandoned）
```

示例（inbox 条目）：

```yaml
kind: inbox_todo
id: inbox_20260818_001
status: pending

created:
  by: minimax-code
  at: 2026-08-18T10:32:38+08:00

payload:
  format: markdown
  content: |
    ## 待办：排查博客搜索高亮样式
    ...
```

### 16.8 inbox 只存 VPS

inbox 只落 VPS 私有区 `/home/studio/workbase/inbox/`，不进入本地 Obsidian Vault，不触发 apply / git commit / rebuild。

---

## 17. Git-backed 写入路线

### 17.1 观点

公网 MCP 可以写入正式 Vault，但写入必须是：

```text
Git-backed
base-commit aware
diff-first
approval-first（webUI）
conflict-stop
audited
```

写入不是由 Agent 直接触发，而是 Agent 提交 proposal 后，用户在 webUI 审批通过，MCP 才执行 apply 并 commit。inbox 是独立待办，不在此列——它不触发 apply / commit（见 §16）。

### 17.2 写入等级

| Level | 能力 | 阶段 |
|---|---|---|
| L0 | proposal（pending，不落正式正文） | v0.1 |
| L1 | webUI 审批 apply（Git-backed commit） | v0.1 |
| L2 | base_commit 校验 + diff preview + 冲突停止 | v0.1 |
| L3 | 低风险自动 append（免审批） | v0.2+ |

v0.1 已包含 L0+L1+L2：proposal 创建后，webUI 审批通过即 apply + commit；apply 前校验 base_commit，冲突则停止。inbox 是独立待办，不属于任何写入等级（见 §16）。

### 17.3 apply 流程（webUI 审批后）

```text
1. Agent 创建 proposal，包含 base_commit（或由服务端补记）。
2. 服务端生成 diff/preview，标记 pending。
3. 用户在 webUI 查看，可编辑表述。
4. 用户同意：
   a. 服务端检查当前 HEAD 是否等于 base_commit。
   b. 若 stale，返回 stale_base，要求重读。
   c. 若一致，apply 到 workbench。
   d. git commit 到 vault.git。
   e. rebuild + reindex。
5. 用户拒绝：标记作废，不 apply。
6. 冲突：停止，保留 conflict proposal。
```

### 17.4 冲突策略

| 情况 | 策略 |
|---|---|
| base commit 不是最新 | 返回 `stale_base`，要求重读 |
| patch 无法应用 | 返回 `patch_failed` |
| Git merge conflict | 停止，不自动合并 |
| 写入 secret 文件 | 拒绝 |
| public 文件含敏感模式 | 拒绝或要求高风险确认 |

### 17.5 禁止策略

```text
不自动解决语义冲突。
不自动覆盖本地 Obsidian 更新。
不把服务器 workbench 变成无约束第二主写入源。
```

### 17.6 Receipt 结果与幂等

每次 apply 产生一个 receipt，记录 apply 的真实结果。状态机：

```text
pending → approved → applied    （同意 + commit 成功 + 内容 hash 校验通过）
        ↘ rejected              （拒绝，作废）
        ↘ conflict              （stale_base / patch_failed / merge conflict，停止不改动）
```

`applied` 的严格定义：**git commit 成功 + 目标文件 apply 后内容 SHA-256 与预期一致**。用户在 webUI 点了「同意」只是 `approved`，不代表 `applied`。

```yaml
receipt:
  proposal_id: prop_20260817_001
  status: applied            # pending | approved | applied | rejected | conflict
  applied_at: 2026-08-17T17:05:00+08:00
  commit: def456              # 仅 applied：apply 后的 git commit
  content_sha256: "..."       # 目标文件 apply 后内容哈希
  replayed: false             # 幂等标记
```

关键语义：

```text
1. applied 需要 commit 成功 + hash 校验，二者缺一不可。
2. conflict 不改动任何文件，保留原 proposal 供重读。
3. 幂等：同一 proposal 重复 apply → 返回原 receipt，replayed=true，不重复提交。
4. 校验失败（input_invalid，如 secret 命中 / fence 不闭合）是控制结果，不是 receipt。
```

---

## 18. 同步模型

### 18.1 当前主链路（正向）

```text
本地 Obsidian Vault
  ↓ sync.ps1
VPS bare repo (vault.git)
  ↓ post-receive
/home/studio/workbench
  ↓ reindex
/home/studio/workbase/index
  ↓ MCP Server
Agent Clients
```

### 18.2 source of truth

```text
本地 Obsidian = 正式内容主写入源
VPS workbench = vault.git 的 working tree（无独立 .git，用 GIT_DIR + GIT_WORK_TREE）
公网 MCP = read + proposal + inbox（proposal 走审批，inbox 是独立待办）
```

### 18.3 reindex 触发（决策已确认）

采用 post-receive 主动触发：

```text
post-receive hook 末尾：
1. git checkout 到 workbench
2. 触发博客 build
3. 触发 MCP reindex（curl 127.0.0.1:8787/internal/reindex 或 systemctl reload）
```

不再用 server 轮询 commit hash。

### 18.4 反向同步（MCP 审批 apply 后回流）

审批通过后，MCP 在 VPS 上修改 workbench 文件。为避免下次本地 push 覆盖这些修改，apply 必须同步提交到 vault.git：

```text
用户 webUI 审批通过
  ↓
MCP 修改 /home/studio/workbench/<target>
  ↓
git --git-dir=/home/studio/vault.git \
    --work-tree=/home/studio/workbench \
    add <target> && commit -m "workbase: apply <proposal_id>"
  ↓
vault.git HEAD 前进
  ↓
本地 Obsidian 下次 git pull 时拿到该 commit（冲突走 git 正常 merge）
  ↓
rebuild + reindex
```

注意：proposal apply 的修改体系与正常文件修改不同——

| | 正常文件修改 | proposal apply |
|---|---|---|
| 触发源 | 本地 Obsidian 编辑 → sync.ps1 push | webUI 审批通过 |
| 写路径 | push → post-receive hook | MCP 直接改 workbench → 手动 git commit |
| build/reindex | post-receive 自动触发 | apply 后手动补触发 |

apply 是直接改 bare repo 的 working tree，绕过了 post-receive hook（hook 只在 push 时触发），所以 apply 后的 rebuild + reindex 必须**手动补触发**（curl `127.0.0.1:8787/internal/reindex` 或 `systemctl reload`），不能指望 post-receive 自动完成。

### 18.5 本地 sync.ps1 调整

sync.ps1 从"只 push"改为"先 pull 再 push"：

```text
git pull --rebase
git push
```

这样 MCP 在 VPS 上的 commit 会先合并回本地，再推送本地新改动。冲突由 git 正常处理。

### 18.6 冲突处理

| 情况 | 策略 |
|---|---|
| MCP apply 基于旧 base | apply 前检查 base_commit，stale 则拒绝 |
| 本地与 VPS 分叉 | git pull --rebase 正常合并 |
| 合并冲突 | git 保留冲突标记，用户手动解决 |
| MCP 不直接覆盖本地未同步内容 | 依赖 git 冲突机制兜底 |

---

## 19. Index 设计

### 19.1 不使用向量数据库

v0.x 不引入向量数据库。

原因：

```text
1. Markdown + frontmatter + WikiLink 结构已经很强。
2. 当前更需要项目状态、决策、下一步，而不是相似段落召回。
3. 向量库增加部署和调试复杂度。
4. 个人知识库错误召回风险较高。
```

### 19.2 推荐 SQLite FTS

索引可使用：

```text
SQLite + FTS5
```

数据表：

```sql
notes(id, path, title, type, visibility, updated_at, access_count, frontmatter_json, summary)
notes_fts(id, title, headings, body, tags)
links(source_id, target_id, link_type, raw)
projects(id, note_id, status, current_focus)
skills(id, note_id, risk, source_json)
mcps(id, note_id, transport, endpoint_hint, auth_json)
```

v0.1 如果不想引入 SQLite，也可先 JSON index + 简单搜索，但建议 SQLite FTS，部署仍然轻量。

### 19.3 Index 输出

```text
/home/studio/workbase/index/
├── notes.sqlite
├── manifest.json
├── graph.json
├── context-startup.json
├── skills.json
└── mcps.json
```

### 19.4 访问计数与热度

记录每个 note / project / skill / mcp 被 Agent 读取（`knowledge.get` / `project.get` / `skill.get` / `mcp.get`）的次数，作为热度信号：

```sql
notes(..., access_count INTEGER DEFAULT 0, last_access_at)
```

热度服务两个方向：

1. **排序加权**：`knowledge.search` 的 `signals` 增加 `access` 信号——读取次数越高排序越靠前（§9.4）；skill/mcp 的 `list` 也可按热度排序。
2. **冷数据清理**：长期零访问 + 低重要性的条目进入清理候选，由用户确认后删除。

v0.1 只做「计数 + 排序加权」；自动清理（GC）延后到 v0.2，且必须用户确认，不自动删除。

---

## 20. 安全设计

### 20.1 禁止泄露

绝不返回：

```text
真实私钥内容
token
密码
.env 原文
ssh private key
secret visibility 内容
未授权 private 内容
```

### 20.2 敏感模式检测

写入 proposal / inbox / public 内容前检查：

```text
Authorization: Bearer
api_key
secret
token
-----BEGIN .* PRIVATE KEY-----
Windows 用户私钥路径
公网 IP 模式（按策略）
.env 风格赋值
```

### 20.3 Audit

审计记录：

```json
{
  "time": "2026-08-17T17:00:00+08:00",
  "client_id": "minimax-code",
  "tool": "knowledge.get",
  "resource_id": "note_xxx",
  "scope": "read:knowledge",
  "result": "success",
  "content_bytes": 1200
}
```

不记录：

```text
token
完整私密正文
secret 内容
```

### 20.4 日志脱敏

所有日志中：

```text
Authorization header -> [REDACTED]
token -> [REDACTED]
private key -> [REDACTED]
```

---

## 21. 部署设计

### 21.1 进程

```text
systemd service: jiangnan-workbase-mcp
binary: /home/studio/workbase/bin/workbase-mcp
config: /home/studio/workbase/config.yaml
```

### 21.2 Caddy

```text
mcp.<domain> {
    reverse_proxy 127.0.0.1:8787
}
```

备案/HTTPS 完成前不强推正式公网 endpoint。

### 21.3 端口

```text
127.0.0.1:8787
```

公网只经 Caddy 暴露。

### 21.4 配置

`config.yaml` 不进 Git：

```yaml
server:
  listen: 127.0.0.1:8787

vault:
  root: /home/studio/workbench
  git_dir: /home/studio/vault.git

workbase:
  root: /home/studio/workbase
  index: /home/studio/workbase/index
  proposals: /home/studio/workbase/proposals
  inbox: /home/studio/workbase/inbox

admin:
  listen: 127.0.0.1:8788
  auth:
    user: admin
    pass_hash: ...

auth:
  clients: []
```

### 21.5 WebUI 后台（proposal 审批 + inbox 待办）

proposal 与 inbox 的阅读统一走一个私密 webUI，但两者操作不同。

proposal 职责（审批流转）：

```text
1. 列出 pending proposal。
2. 展示每条内容的 diff / preview。
3. 支持「同意」「编辑后同意」「拒绝」。
4. 同意 → MCP apply 修改 workbench → git commit → rebuild + reindex。
5. 拒绝 → 标记作废。
```

inbox 职责（创建 + 编辑 + 状态调整，无审批）：

```text
1. 列出 inbox 条目。
2. 支持「新建」待办（等价 inbox.append，初始 pending）。
3. 展示每条内容，支持「编辑内容」和「调整状态」（pending → reviewing → done / abandoned）。
4. inbox 自身不触发 apply / commit，done/abandoned 保留 7 天后自动删除。
```

访问方式：

```text
https://workbase.<domain>/  （Caddy 反代到 127.0.0.1:8788）
```

认证：

```text
独立 admin 账号 + 密码（不开放给 Agent）。
与 MCP Bearer token 分离。
```

v0.1 实现建议：

```text
先做极简版：一个静态 HTML + 少量 JS，通过内网 HTTP API 操作。
不集成到公开博客（博客是公开展示层，webUI 是私密审批层）。
```

---

## 22. 实现建议

### 22.1 技术栈

建议 Go：

```text
Go
mcp-go
SQLite FTS5
yaml parser
goldmark 或轻量 markdown parser
```

理由：

```text
1. 用户主力 Go。
2. 后续可复用 Eino/mcp-go 经验。
3. 单 binary 部署简单。
4. 适合 VPS 常驻服务。
```

### 22.2 目录

```text
server/mcp/
├── cmd/workbase-mcp/
├── internal/config/
├── internal/auth/
├── internal/audit/
├── internal/vault/
├── internal/indexer/
├── internal/search/
├── internal/tools/
├── internal/proposal/
├── internal/inbox/
├── internal/apply/
├── internal/admin/          # webUI 审批后台
├── internal/sanitize/
└── README.md
```

### 22.3 模块职责

| 模块 | 职责 |
|---|---|
| `config` | 读取 config.yaml |
| `auth` | Bearer token hash、scope 校验 |
| `audit` | SQLite 审计日志 |
| `vault` | 读取 Vault 文件、frontmatter、路径规范 |
| `indexer` | 构建 notes/projects/skills/mcps/context index |
| `search` | FTS 查询 |
| `tools` | MCP tool handlers |
| `proposal` | proposal schema、落盘、preview |
| `inbox` | inbox 待办落盘、读取、状态流转、7 天清理 |
| `apply` | webUI 审批后 apply + git commit + reindex |
| `admin` | webUI 后台 HTTP API（proposal 审批 + inbox 编辑/状态调整） |
| `sanitize` | 敏感信息检测与脱敏 |

---

## 23. 验证与验收

### 23.1 v0.1 验收标准

```text
1. 未带 token 请求 MCP 返回 401。
2. 带只读 token 可调用 manifest/context/search/get。
3. context.startup 返回当前用户工作风格与活跃项目摘要。
4. knowledge.search 可检索 Vault 镜像内容。
5. project.get 可返回项目当前重点、决策、下一步。
6. skill.list/get 从 Workbase/skills/*.md 生成。
7. mcp.list/get 从 Workbase/mcps/*.md 生成。
8. proposal.create 可创建 typed proposal。
9. inbox.append 可新建 pending 待办；inbox.update 可改状态；inbox.list/get 可读回。
10. webUI：proposal 可审批（同意/拒绝/编辑）；inbox 可创建、编辑、调整状态（pending → reviewing → done / abandoned）。
11. 同意 → apply + git commit → rebuild + reindex。
12. secret visibility 内容默认不返回。
13. 返回内容不包含真实 IP、私钥路径、token。
14. 所有 tool call 有审计记录。
15. sync.ps1 推送后 VPS reindex 可更新 MCP index。
16. MCP apply 后本地 git pull 可同步回该修改。
```

### 23.2 文档验收

```text
1. docs/agent-workbase-mcp-v0.1.md 存在。
2. README 更新项目定位。
3. 不含真实 VPS IP / 私钥路径。
4. 说明 v0.1 不使用向量数据库。
5. 说明 Proposal target/operation schema。
```

---

## 24. Milestones

### M0：设计固化

产物：

```text
docs/agent-workbase-mcp-v0.1.md
README 定位更新
Workbase 目录规范
```

### M1：Vault Schema 与示例内容

产物：

```text
Workbase/context/*.md
Workbase/skills/example.md
Workbase/mcps/example.md
vite.config.ts 排除 Workbase/ 栏目
SCHEMA.md 更新
```

### M2：Indexer MVP

产物：

```text
server/mcp index command
notes/projects/skills/mcps/context index
SQLite FTS 或 JSON index
```

### M3：MCP Read-only Server

产物：

```text
workbase.manifest
context.startup/context.get
knowledge.search/get
project.list/get
skill.list/get
mcp.list/get
Bearer auth
audit
```

### M4：Proposal + Inbox + WebUI 审批

产物：

```text
proposal.create/list/get
inbox.append/update/list/get
proposal markdown 落盘
webUI 后台（proposal 审批：同意/拒绝/编辑；inbox：创建 + 编辑 + 状态调整）
同意 → apply + git commit
敏感信息检查
```

### M5：VPS 部署

产物：

```text
systemd service
Caddy reverse proxy
post-receive reindex
部署文档
```

### M6：双向同步加固

产物：

```text
base_commit 校验（已完成 L2）
冲突检测与 conflict proposal
sync.ps1 pull --rebase + push
本地 Obsidian 回流验证
```

---

## 25. 风险与对策

| 风险 | 对策 |
|---|---|
| 私密内容泄露 | visibility + scope + sanitize + audit |
| token 泄露 | 不入库、hash 存储、独立 token、可撤销 |
| 双主写入冲突 | v0.1 proposal，v0.2 Git-backed base_commit |
| Agent 乱写知识库 | typed proposal + adapter + approval |
| Inbox 堆积 | 状态流转 + done/abandoned 7 天自动删除 |
| 向量库复杂化 | v0.x 不引入 |
| Registry 双事实源 | Skill/MCP 用 Markdown frontmatter 作为事实源 |
| startup 与 context 不一致 | startup 仅派生，不直接写 |
| 公开仓库误提交部署信息 | 文档和脚本使用占位符与 env |

---

## 26. 待确认问题

已确认（2026-08-17）：

```text
1. Workbase/ 放在 D:\Data\工作台\ 根下。
2. inbox 落点：只落 VPS 私有区 /home/studio/workbase/inbox/。
3. reindex 触发：post-receive 主动触发。
4. proposal/inbox 只存 VPS；proposal 走 webUI 审批后 apply，inbox 是独立待办（pending → reviewing → done/abandoned，7 天删除，不审批，不转 proposal）。
```

仍待确认：

```text
1. 后续仓库是否 rename 为 jiangnan-workbase？
2. Context Pack 初始需要哪些文件？
3. Skill/MCP Registry 哪些条目先作为样例？
4. webUI admin 的认证方式（单账号密码 / OIDC / 其他）？
5. MCP apply 后是否需要在本地 Obsidian 内做二次确认，还是直接 git pull 即可？
6. token scope 是否需要区分不同 Agent 工具？
```

---

## 27. 初始推荐决策

为了最小可行但结构完整，推荐初始决策如下：

```text
1. 项目定位改为 Blog as Agent Workbase。
2. 暂不 rename repo，先补 docs 与 server/mcp。
3. MCP transport 使用 Streamable HTTP。
4. v0.1 auth 使用 Bearer Token + scopes。
5. v0.x 不使用向量数据库。
6. Obsidian Vault 是正式内容事实源。
7. context.startup 是派生结果，不直接写。
8. Skill/MCP Registry 用 Markdown + frontmatter。
9. 正式写入走 proposal.create；待办走 inbox.append/update，两者完全独立。
10. proposal/inbox 只存 VPS；proposal 走 webUI 审批后 Git-backed apply，inbox 是独立待办（pending → reviewing → done/abandoned，7 天删除，不审批）。
11. Workbase/ 放本地 Vault 根，但构建时排除出公开栏目。
12. inbox 落 /home/studio/workbase/inbox/，不进本地 Vault，不转 proposal。
13. reindex 由 post-receive 主动触发。
14. MCP apply 后 commit 到 vault.git，本地 git pull 回流。
15. sync.ps1 改为先 pull 再 push。
```

---

## 28. 参考资料

- MCP Streamable HTTP transport, 2026-07-28 specification: https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/basic/transports/streamable-http.mdx
- MCP Authorization specification: https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
- MCP 2026-07-28 release notes: https://blog.modelcontextprotocol.io/posts/2026-07-28/
- 参考项目 Personal Agent Foundation: https://github.com/DongLiStudio/personal-agent-foundation
- 参考项目 mnemon（LLM-supervised 记忆层）: https://github.com/mnemon-dev/mnemon
- MAGMA（四图记忆模型）: https://arxiv.org/abs/2601.03236

### 28.1 参考 mnemon 的吸收

`mnemon-dev/mnemon` 是 LLM-supervised 记忆层（四图模型 + `remember`/`link`/`recall` 三原语 + 单二进制 SQLite）。它与 Workbase 范式不同——它把记忆抽成结构化 Insight/Edge 图，我们坚持 Markdown 事实源 + WikiLink 反链。因此**不采纳四图存储引擎**，但吸收其检索层方法论、生命周期与写入安全语义，映射到现有 FTS5 + 反链 + Proposal/Git-apply：

| # | 吸收点 | mnemon 原设计 | 落到 Workbase |
|---|---|---|---|
| 1 | Signal transparency | recall 结果暴露 keyword/entity/similarity/graph 信号分解 | `knowledge.search` 返回 `signals` + `matched_via`（§9.4） |
| 2 | Intent-aware retrieval | WHY/WHEN/ENTITY/GENERAL 意图自适应权重 | `knowledge.search` 加可选 `intent` 参数（§9.4） |
| 3 | Built-in dedup | remember 内建 diff，重复跳过 / 冲突替换 | Proposal apply 前做内容重叠检测（§17.6 前置） |
| 4 | Privacy-safe receipts | receipt 只输出操作名 + SHA-256，不泄原文 | `audit.list_recent` 加 `mode: hashed`（§9.16） |
| 5 | Receipt 状态机 + 幂等 | accepted/rejected/input_invalid/replayed 严格区分 | Proposal 加 receipt + 幂等语义（§17.6） |
| 6 | Named stores 隔离 | MNEMON_STORE 独立库 | 已由 scope + project 覆盖，仅确认方向 |

不采纳（理由）：

| 不采纳点 | 理由 |
|---|---|
| 四图模型（temporal/entity/causal/semantic 多类型边） | 违背 Markdown 事实源 + 最简实现，需引入抽取器 |
| 内嵌 embedding（Ollama nomic-embed-text） | 已定 v0.x 不引入向量库，FTS5 + 反链足够 |
| Agency 本地 daemon + peer exchange + CAS artifact | 个人工作台过度，Git-backed apply 已覆盖受控写入 |
| 多 runtime 保姆式 setup | 已否决，Agent 自行适配 Skill/MCP |

---

## 29. 下一步执行建议

下一步建议按顺序做：

```text
1. 更新 README：从个人博客升级为 Agent Workbase。
2. 同步 SCHEMA.md：补充 Workbase/ 目录与 visibility 规范。
3. vite.config.ts：构建排除 Workbase/（避免作为公开栏目）。
4. 在 Vault 新建 Workbase/context 初始文件。
5. 在 Vault 新建 Workbase/skills 与 Workbase/mcps 样例。
6. 实现 server/mcp indexer MVP。
7. 实现 read-only MCP tools。
8. 实现 proposal.create + inbox.append/update。
9. 实现 webUI 审批后台（同意/拒绝/编辑 + apply + commit）。
10. post-receive 增加 reindex 触发。
11. sync.ps1 改为先 pull 再 push。
12. 部署到 VPS 内网端口，Caddy 反代。
13. 等 HTTPS 条件满足后开放公网私密接入。
```
