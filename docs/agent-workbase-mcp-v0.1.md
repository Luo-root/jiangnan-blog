# 遇见江楠 · Agent Workbase MCP 设计文档

> 状态：v0.1 完整版（不分 v0.1/v0.2 阶段，v0.1 即完整 v1.0 设计的第一版实施）
> 日期：2026-08-19
> 项目定位：Blog as Agent Workbase / 博客即 Agent 工作基座
> 仓库：`Luo-root/jiangnan-blog`（暂不 rename）
> 文档分层：设计文档（why / what / 验收）/ `SCHEMA.md`（API + 字段 + 状态机 + 算法）/ `D:/Data/工作台/Workbase/`（运行时自描述数据）

---

## 0. 顶层原则

### 0.1 单一事实源原则

MCP 任何对外可见的描述性字段**必须**从 `D:/Data/工作台/Workbase/` 即时读取。Workbase 自身是 vault 长期记忆的一部分，修改走 Obsidian 编辑 + `sync.ps1` 推送；**不允许** Go 代码持有描述性字符串字面量。

适用范围：

- `workbase.identity` 的 `workbase.name` / `workbase.description` / `workbase.getting_started` / `workbase.critical_rules` / `workbase.see_also`
- `context.startup` 合成来源
- `skill.list` / `mcp.list` / `project.list` 的条目数据（与 workbase.identity.workbase.tools 是同一来源）
- 任何 Agent 可读的"是什么 / 能做什么 / 边界在哪"类信息

不属于本原则的范围（属于协议层标识符或实现细节）：

- MCP 协议层 `id` / `version`（manifest 自身的协议标识符，不是 Workbase 的描述）
- HTTP 路由、方法名
- 错误码、状态机名
- 信号权重的默认值（实现常量，可被 config 覆盖）

**违反本原则 = 后续修改必须改 Go 重编译 = 违反"长期记忆可独立维护"**。验收时核对：修改 `Workbase/mcps/jiangnan-workbase.md` 后立即反映到 `workbase.identity`，不重启进程。

### 0.2 内容统一分发原则

公开博客和后台管理读**同一份 vault**，扫描各做各的（不是共享同一份 index 文件）：

```text
D:/Data/工作台/  ←  Obsidian Vault（事实源）
        ↓                              ↓
  Vite `virtual:vault-tree`      MCP indexer
  （构建时扫盘）                  （写 SQLite notes）
        ↓                              ↓
  公开博客 dist/                 后台 / MCP HTTP API
```

- 公开博客 = Vite 构建时扫 vault，只收 `visibility=public` 且非草稿
- 后台 / MCP = indexer 扫同一份 vault 写 SQLite，private / secret 按 scope 可读
- **不**让博客读 `workbase/index`，也**不**让后台另开一套 vault 路径

### 0.3 文档分层

| 层级 | 路径 | 职责 | 修改频率 |
|---|---|---|---|
| 设计文档 | `docs/agent-workbase-mcp-v0.1.md` | why / what / 验收口径 | 重大设计变更 |
| API/数据契约 | `SCHEMA.md` | 字段映射、状态机、表结构、算法公式、frontmatter schema | 字段/状态变更 |
| 自描述数据 | `D:/Data/工作台/Workbase/` | 运行时数据（manifest 文案、context pack、registry） | 日常编辑 |

**同一信息只在一处定义**。例如 `visibility` 字段的取值表（`public` / `private` / `secret` / `draft`）的事实源 = `config.yaml` 的 `schema.visibility_policy` / `schema.visibility_default`（Go 启动加载到 `cfg.Schema.*`，运行时**不**再读盘）。`SCHEMA.md §3` 是**给人看的说明**，不重复数据——改本表后**必须**同步改 `config.yaml`。

字段映射的修改流程：

1. 改 `SCHEMA.md` 对应小节
2. 同步改 `server/mcp/internal/...` 中对应代码
3. 部署
4. **两步缺一不可**（设计文档 + 验收 §23.1.35 强制）

### 0.4 不分版本原则

v0.1 = 完整版。**不预留 v0.2 演进项**。任何确定要做的事都直接做：

- 3-way merge（无冲突自动 apply）
- 艾宾浩斯遗忘曲线热度
- 字段映射在 SCHEMA.md 细粒度
- 管理后台 TS 工程化
- 视觉规范统一
- 配置文件可配项（权重、retention_days、half_life_days）

理由：分版本 = 给自己留借口不做 = 验收时缩水。不接受的对写成「不做」；接受的对写成当前行为。不要标「下一版候选」。

---

## 1. 背景与问题

### 1.1 当前博客已有基础

```text
D:/Data/工作台/                     # Obsidian Vault，内容事实源
D:/Code/Front-end/博客/            # React/Vite 静态博客代码
/home/studio/workbench               # VPS 上的 Vault 镜像
/home/studio/app/repo                # VPS 上的博客代码
/home/studio/app/public              # Caddy 静态服务目录
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

已支持：Obsidian Vault 构建时注入、文章/项目/友链/归档多栏目、WikiLink 与相对链接、反链与图谱、公开部署到 VPS + Caddy。

### 1.2 新问题

普通博客解决的是"人如何阅读我的知识沉淀"。Agent Workbase 要解决的是：

```text
Agent 如何稳定、可信、低成本地读取我的长期上下文？
Agent 如何知道我当前在做什么？
Agent 如何复用我的 Skill / MCP 能力目录？
Agent 如何提出知识库写入，而不污染正式内容？
换设备或换 Agent 工具时，如何保持体验一致？
```

### 1.3 设计目标

v0.1 目标：任意授权 Agent 连接公网私密 MCP endpoint 后，可以读取 Workbase 自描述（workbase.identity）、获取 startup context、搜索和读取授权范围内的知识库内容、获取项目状态/决策/下一步、获取 Skill/MCP Registry、创建 proposal 或追加 inbox 待办；所有访问有权限边界和审计记录；不泄露真实 IP/私钥路径/token/secret。

---

## 2. 非目标

```text
1. 不做网页聊天框
2. 不引入向量数据库
3. 不让 Agent 绕过 Proposal 直接修改正式 Vault
4. 不实现完整 OAuth 2.1 授权服务器
5. 不做多用户系统
6. 不做复杂安装器
7. 不为每个 Agent 工具单独生成安装教程
8. 不将 private / secret 内容发布到公开博客
9. 不改变 Obsidian 作为内容事实源的原则
10. 不预留 v0.2（v0.1 即完整版）
```

---

## 3. 核心原则

### 3.1 Obsidian Vault 是事实源

正式知识内容以 Markdown 文件为事实源：

```text
文章      = 文章/*.md
项目      = 项目/*.md
友链      = 友链/*.md
Skill     = Workbase/skills/*.md
MCP       = Workbase/mcps/*.md
Context   = Workbase/context/*.md
普通笔记  = Vault 内其它 .md（不含 `Workbase/context|skills|mcps/`；Workbase 下误放的其它 md 也按 note）
```

MCP Server 不维护另一份长期真相，只做读取、索引、暴露、生成 proposal、记录审计。

### 3.2 公网可访问 ≠ 公开可访问

MCP Server 部署在公网方便跨设备跨 Agent 接入，但访问必须是私密的：

```text
HTTPS + Authorization Bearer Token + scope + audit
```

### 3.3 Agent 不是傻瓜

Skill/MCP Registry 的职责不是教每个 Agent 工具怎么安装，而是提供：能力是什么、在哪里、来源是什么、风险是什么、授权要求是什么、是否公开可迁移。Agent 拿到来源、endpoint、transport、auth、scope 后自行适配。

### 3.4 写入统一走 Proposal

不创建 `context.update` / `article.create` / `skill.register` / `mcp.register` / `project.patch` 等单点写入工具，统一入口 `proposal.create`，由内部 `target.type + operation.type + payload` 表达不同写入目标。

Inbox 不是写入，是独立待办：走 `inbox.append` / `inbox.update`，不经过 Proposal、不审批、不 apply、不 commit。

### 3.5 context.startup 是派生结果

`context.startup` 不直接手写或 patch，由多个 context pack 按 `priority` 合成。修改 startup 的正确方式是修改背后的 context pack。startup 输出**不缓存**，每次调用即时合成（命中即记录访问热度）。

合成跳过 `visibility=secret`（即使标了 `startup: true`）。draft 照收。`context.get` 对 draft 放行，secret 默认 `secret_blocked`。和 project / skill / mcp 同一套。

### 3.6 先结构化，再智能化

v0.1 不引入向量库，优先使用 frontmatter / title / tag / path / heading / WikiLink / backlink / SQLite FTS。个人知识库第一阶段更需要可靠结构，而不是不透明召回。

### 3.7 MCP 描述性字段从 vault 即时读

详见 §0.1。补充：每次调用即时读磁盘，**不缓存**。文件读取失败 → 返回 MCP error，**不**静默 fallback 到硬编码字符串。

### 3.8 base_commit 不匹配时尝试 3-way merge

Agent 写长期记忆不是"我有 token 所以我可以改"，而是"用户审批后才改"。3-way merge 取代简单 abort。`ours` 必须是完整文件，不能是 payload 片段（详见 §17.3）：

- 先在 `base_commit` 的目标文件上施加 operation，得到完整 ours
- 目标文件在 `base_commit` 之后无任何修改 → 完整 ours 直接落盘
- 目标文件有修改但无冲突 → 3-way（base=文件@base, other=文件@HEAD, ours=完整文件）后 apply
- 3-way 产生冲突 → 状态 `conflict`，返回冲突区段
- frontmatter 内部冲突 → 一律 conflict（结构化字段不自动合并）
- `create_file` / `register_item`：HEAD 已存在 → conflict；不存在 → 直接落盘，不走 merge

工具实现用 `git merge-file`；frontmatter 冲突的判定 = YAML 解析 + 字段级 diff。

### 3.9 视觉规范统一

公开博客（朝曦/夜隐主题）和后台管理（明亮专业主题）共享**同一套设计 token**：

- 颜色、字体、间距、圆角、阴影、动效
- 主题切换 = 切换 token，**不**切换组件结构
- 后台 = 公开博客设计系统的 admin skin

不存在"后台独立设计语言"。视觉规范在 `server/mcp/admin/src/styles/tokens.css`（与博客 `src/styles.css` 同源）。

---

## 4. 总体架构

### 4.1 逻辑分层

```text
┌───────────────────────────────────────────────┐
│               Agent Clients                    │
│  MiniMax Code / Cursor / Claude / ChatGPT / ...│
└───────────────────────┬───────────────────────┘
                        │ HTTPS + Bearer Token
                        ▼
┌───────────────────────────────────────────────┐
│          Jiangnan Workbase MCP Server          │
│  identity / context / knowledge / project /    │
│  skill / mcp / proposal / inbox / audit        │
└───────────────────────┬───────────────────────┘
                        │ read index / write proposals
                        ▼
┌───────────────────────────────────────────────┐
│             Workbase Private Store             │
│  index / proposals / inbox / audit / config    │
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
│             D:/Data/工作台/                    │
└───────────────────────────────────────────────┘
```

### 4.2 Vault 目录

```text
D:/Data/工作台/
├── Workbase/                # Agent 工作基座私有层（构建排除）
│   ├── context/             # context pack
│   ├── skills/              # Skill Registry
│   └── mcps/                # MCP Registry
├── 文章/                    # 博客正文
├── 项目/                    # 项目卡片
├── 友链/                    # 友链卡片
└── 部署溯源/                # 部署记录（默认 private）
```

VPS 私有区：

```text
/home/studio/workbase/
├── config.yaml              # 不进 Git：admin pass_hash / paths / grace_period_hours（token 全部在 SQLite）
├── index/                   # SQLite + JSON
├── proposals/               # *.md
├── inbox/                   # *.md
└── audit/                   # audit.sqlite
```

### 4.3 内容统一分发

同一份 vault，两套扫描（详见 §0.2）：

```text
                 D:/Data/工作台/
                    ↓            ↓
          Vite 扫盘构建      MCP indexer → SQLite
          (public 且非草稿)   (public/private/secret by scope)
```

---

## 5. 命名与仓库演进

**不重命名**仓库。`Luo-root/jiangnan-blog` 继续使用。`blog` 不等于"博客文章"——它是"个人对外发布的所有内容"的统称，包括博客、项目展示、友链、归档，以及未来可能的视频栏目。

中文名："遇见江楠 · Agent 工作基座"。

---

## 6. MCP 传输与认证

### 6.1 Transport

Streamable HTTP（按 MCP 2025-11-25 规范）。Endpoint：`https://mcp.<domain>/mcp`，Caddy HTTPS 反代到 `127.0.0.1:8787`。

### 6.2 Bearer Token

```http
Authorization: Bearer <WORKBASE_TOKEN>
```

要求：

1. token 不放 URL query
2. token 不写入公开仓库
3. 服务端存 token hash 到 SQLite `auth_tokens` 表（**不**存 config.yaml，也不存明文）
4. 每个 Agent 使用独立 token
5. token 绑定 scopes
6. audit 记录 client，不记录 token

### 6.3 后续 OAuth

MCP 官方授权规范基于 OAuth 2.x / 2.1。后续如需多设备 consent、动态客户端注册、token 轮换、撤销，可升级 OAuth/OIDC。v0.1 不实现完整 OAuth Server。

### 6.4 Token 生命周期

Token 全流程走 **webUI 自助 + SQLite 存储**，**不**走 SSH / `config.yaml` 改文件 / 重启。

#### 6.4.1 存储位置：SQLite `auth_tokens` 表

```sql
CREATE TABLE auth_tokens (
  id            INTEGER PRIMARY KEY,
  name          TEXT    NOT NULL,           -- = client_id。同名可有多行（active + grace），audit 不碎
  token_hash    TEXT    NOT NULL,           -- SHA-256(明文 token)
  scopes        TEXT    NOT NULL,           -- JSON array，如 ["read:context","read:knowledge"]
  status        TEXT    NOT NULL DEFAULT 'active',  -- active | grace | revoked
  grace_until   TIMESTAMP,                 -- 仅 status=grace 有值：旧 token 灰度截止时间
  description   TEXT,                      -- 用户填的描述
  created_at    TIMESTAMP NOT NULL,
  created_by    TEXT    NOT NULL,          -- admin user id
  last_used_at  TIMESTAMP,
  use_count     INTEGER DEFAULT 0
);

CREATE UNIQUE INDEX idx_auth_tokens_active_name ON auth_tokens(name) WHERE status='active';
CREATE INDEX idx_auth_tokens_hash ON auth_tokens(token_hash);
CREATE INDEX idx_auth_tokens_status ON auth_tokens(status);
```

**`config.yaml` 不再含 `auth.clients[]`**。所有 token / scope 信息都来自 SQLite。

#### 6.4.2 webUI 创建流程

```
登录 webUI → Settings → Token 管理 → 点"创建 Token"
  ↓
表单：
  名称 (name = client_id)*: [_________________]
  描述:                     [_________________]
  权限 (scopes)（来自 SCHEMA.md §2.1，仅 8 个标准 scope）:
   ☑ read:context       (默认勾选)
   ☑ read:knowledge     (默认勾选)
   ☐ read:project
   ☐ read:registry
   ☐ read:inbox
   ☐ write:proposal
   ☐ write:inbox
   ☐ ops:audit
   # 没有 admin:reindex。重建索引不是 Agent scope，Token UI 不展示、不签发。
  [取消]                          [创建]
  ↓
后端：
  1. 校验：没有 status='active' 的同名行 + scope 合法（必须在 SCHEMA §2.1 表内）
     # 不是整列 name 唯一。grace / revoked 同名行可以在。
     # SELECT WHERE name=? 会把轮换后的再签发误拒。
  2. 生成明文 token = base64(crypto/rand 32 bytes)  (32 字节随机 = 43 字符 base64)
  3. token_hash = SHA-256(明文)    # 不含 sha256: 前缀
  4. INSERT INTO auth_tokens (name, token_hash, scopes, created_at, created_by, status='active')
  5. 同步 upsert 该行进 tokenCache（签发必须立刻能用，不能等 5s reload）
  6. 返回明文 token **仅一次**（不存明文到任何地方）
  ↓
弹窗（模态，关之前必须点「我已保存」或复制成功）：
  ┌─ 你的 Token（只展示一次，请立即复制保存）─────────┐
  │  Name: minimax-code                              │
  │  说明: 给 MiniMax Code 用的只读+提案 token        │
  │  Token: <base64 32 字节，仅展示这一次>            │
  │  Scopes: read:context, read:knowledge, ...        │
  │  [复制到剪贴板]  [我已保存]                       │
  └──────────────────────────────────────────────────┘
  复制成功 → 按钮文案改成「已复制」，短暂态，不是静默。
  签发 / 轮换 / 撤销的成功、失败、错误 → **sonner toast（bottom-right）**，写清原因（重名 / scope 非法），不要页顶错误条，不要右上角自写列表，不要只 reload。明文弹窗用 Dialog；轮换 / 撤销二次确认用 AlertDialog，不用 window.confirm。
  ↓
关弹窗 / 刷新页面 → 列表显示 **active / grace** 的 name + **description** + scopes + status + 创建时间 + last_used_at + use_count
                    **永远不再展示明文**。`revoked` **不进列表**（作废即从界面消失）
签发 / 轮换 / 撤销都要有 toast 反馈。轮换点下去之前二次确认（文案带 name，§6.4.4）。撤销只要确认「作废该 token」，**不要**再做一个输入 name 的展示层（§6.4.5）。列表里的 description 就是创建时填的那一行，不能只进库不渲染。
```

**`config.yaml` 不再含 `auth.clients[]`**——所有 Token 都在 SQLite `auth_tokens` 表，零重启生效。唯一保留的 auth 字段是 `auth.grace_period_hours`（轮换 / 撤销灰度时长）。

#### 6.4.3 运行时使用

HTTP middleware 流程：

```go
// server/mcp/internal/auth/middleware.go
func Authenticate(req *http.Request) (*AuthContext, error) {
    token := extractBearer(req)
    hash := sha256hex(token)

    // 认证只读内存 cache。
    // 签发：SQLite INSERT 后必须同步 upsert 新 hash。
    // 轮换：SQLite 旧行改 grace 后必须同步改/删旧 hash 的 cache，再 upsert 新 hash。
    //        只 upsert 新行会让旧 token 再活 ≤5s 的 active。
    // reload（每 5s）只给「别的副本 / 崩溃恢复」用，不能挡新 token。
    // 撤销：可以等 ≤5s；签发 / 轮换改旧 cache 不行。
    row := tokenCache.lookup(hash)
    if row == nil {
        return nil, ErrUnauthorized  // HTTP 401
    }

    // 状态检查
    switch row.Status {
    case "active":
        // 通过
    case "grace":
        if time.Now().After(row.GraceUntil) {
            return nil, ErrUnauthorized  // 灰度过期
        }
        // 通过（旧 token 仍可用直到 grace_until）
    case "revoked":
        return nil, ErrUnauthorized
    default:
        return nil, ErrInternal
    }

    return &AuthContext{
        ClientID: row.Name,
        Scopes:   row.Scopes,
    }, nil
}

// 异步更新 last_used_at + use_count
go func() {
    db.Exec("UPDATE auth_tokens SET last_used_at=?, use_count=use_count+1 WHERE id=?", time.Now(), row.ID)
}()
```

#### 6.4.4 轮换（零重启）

```
webUI → Token 列表 → 选中 token → 点"轮换"
  ↓
弹窗确认：轮换将立刻作废旧明文（grace 默认 0）。确认文案必须带 token name。
  - 灰度期 = config.auth.grace_period_hours，**默认 0 = 无灰度**（安全感优先）
  - 配 N = 旧 token 在 grace_until 之前仍可用
  ↓
后端（顺序不能反，部分唯一索引要求先腾出名额）：
  1. 生成新明文 token
  2. UPDATE 旧行: status='grace', grace_until=now+grace_period_hours
     # name 不变，audit 的 client_id 不碎
     # grace_period_hours=0 → grace_until=now（已过期，Authenticate 立刻 401）
  3. 同步改旧 hash 的 cache：
     - grace_period_hours > 0 → upsert 旧 hash 为 status='grace' + grace_until
     - grace_period_hours = 0 → 直接从 cache 删旧 hash（跟撤销同一套）
     # 只 upsert 新行不够。Authenticate 对 cache 里仍是 active 的旧 hash 不看 grace_until，
     # 会把旧 token 再当没轮换过的新 token 放行，identity 也会报 active。
  4. INSERT 新行: 同名 name，status='active'，新 token_hash
     # 必须在步骤 2 之后。先 INSERT 会撞 UNIQUE INDEX ... WHERE status='active'
  5. 同步 upsert 新行进 tokenCache（签发必须立刻能用）
  6. 返回新明文（仅一次）
  ↓
旧 token：步骤 3 之后立刻按 grace 规则生效（默认 0 = 立刻 401，不是「再活 5s 的 active」）
新 token：步骤 5 之后立刻可用（用户复制完马上填进 Agent 不能 401）
```

**灰度可配**：`config.yaml` 的 `auth.grace_period_hours`（**默认 0**）。设为 N = 灰度 N 小时；设 0 = 无灰度。轮换必须同步改旧 cache；撤销也可以同步删，5s reload 只是兜底。**签发同步写 cache**。

**零重启**：SQLite + 同步写 cache。reload 只给崩溃恢复 / 多副本。

#### 6.4.5 撤销（零重启）

```
webUI → Token 列表 → 选中 token → 点"撤销"
  ↓
确认弹窗：作废该 token（文案带 name 即可）。**不要**再要求输入 name——撤销 = 删掉作废，不是再展示一条记录。
  ↓
后端：
  1. UPDATE auth_tokens SET status='revoked' WHERE id=?
     # 行留在 SQLite（name 可再签发、audit 的 client_id 不碎），不是物理 DELETE
  2. 同步从 tokenCache 删掉该 hash（撤销也可以同步，5s reload 只是兜底）
  ↓
立刻从列表消失。最多再活到下次 reload（SLA ≤ 5s）；同步删了就立刻失效。
webUI `GET /api/auth_tokens` **不返回** `status=revoked` 的行。
```

**零重启**：同上。

#### 6.4.6 审计保证

audit 记录 `client_id`（即 `auth_tokens.name`），**不**记录：
- 明文 token
- token hash（hash 是认证凭据，不入审计）
- grace_until / last_used_at 等

实施位置：`server/mcp/internal/auth/middleware.go` 的 `Authenticate()` 写 ctx → `server/mcp/internal/audit/audit.go` 写库。

### 6.5 占位符约定（公开仓库 + 文档）

- `WORKBASE_TOKEN_<NAME>`：文档里示意 token 时的占位符
- `REPLACE_WITH_SHA256_HEX`：hash 占位符（**不带** `sha256:` 前缀，统一 hex 字符串）
- `REPLACE_WITH_*`：config.example.yaml 的占位符
- 公开仓库 / 设计文档 / commit message / PR description **绝不**出现：
  - 真实 token 原文
  - 真实 token hash
  - 真实 IP / 私钥路径 / 真实密码
  - `secrets.local.txt` / `config.yaml`（真实值）的内容

---

## 7. 权限与 Scope

完整 scope 列表和工具权限矩阵在 `SCHEMA.md §2` 维护。设计原则：

- 工具注册时声明所需 scope（`srv.AddTool` metadata）
- HTTP 层校验 token 是否含该 scope
- tool middleware 记录 audit
- scope 列表的事实源 = `server/mcp/internal/tools/tools.go` 的 `toolScopes` map
- 与 `SCHEMA.md §2` 表格**双向校对**（代码改了必须同步文档，反之亦然）

修改 scope 列表必须同步：

1. `SCHEMA.md §2` 表格
2. `server/mcp/internal/tools/tools.go` 的 `toolScopes` map
3. 部署后 token 重新签发（如果新增了 scope）

---

## 8. 可见性模型

完整可见性策略表、缺省规则、敏感模式检测在 `SCHEMA.md §3 + §21` 维护。设计原则：

- `visibility` 字段在 frontmatter 中是单一事实源
- 取值固定：`public` / `private` / `secret` / `draft`
- 缺省规则：按文件所在一级目录查表（见 `SCHEMA.md §3.2`）
- 运行时权威 = `config.yaml` 的 `schema.visibility_policy` / `schema.visibility_default`（启动加载到 `cfg.Schema.*`，运行时**不**再读盘）
- Go 代码**不持有** visibility 字符串字面量；所有值从 `config.yaml` 加载
- `SCHEMA.md §3` 是**给人看的说明**，不是 Go 解析输入——改 `SCHEMA.md §3` 后**必须**同步改 `config.yaml` 的 `schema.visibility_*`
- **不缓存的例外**：`workbase.identity` 的 `name` / `description` / `getting_started` / `critical_rules` / `see_also` 每次调用即时读 vault md（见 §9.1），改完要立即可见

### 8.1 配置加载流程（config.yaml + SCHEMA.md 双文件）

**目标**：Go 代码从单一 YAML 文件读取所有配置 + 策略，**不**开额外文件、**不**解析 Markdown、**不**写硬编码字符串字面量。

#### 8.1.1 文件分工

| 文件 | 用途 | 读取方 |
|---|---|---|
| `config.yaml` | **所有结构化数据**（部署配置 + 策略 + 默认值 + 状态机 + 权重） | Go 启动时 `LoadConfig()` |
| `config.example.yaml` | **公开模板**（含 `REPLACE_WITH_*` 占位符，入库） | 人复制起点 |
| `SCHEMA.md` | **API 文档**（why / what / 字段说明 / 验收口径） | 人读，**不**被 Go 解析 |

**`config.yaml` 是事实源**。`SCHEMA.md` 里出现的所有结构化字段说明**指向** `config.yaml` 的对应字段，不重复数据。

**同步约束**：人改 `config.yaml` 后**同时**改 `SCHEMA.md` 里的对应字段说明（手工保持一致，类似设计文档 vs 代码字段的双改流程 §0.3）。

**不**单开 `schema.yaml` / `policies.yaml` / `weights.yaml` 等多个文件——文件越多部署越乱。**一个 config.yaml 含所有结构化数据**。

#### 8.1.2 config.yaml 的 `schema` 块结构

```yaml
# server/mcp/config.yaml
# 真实值不入 Git。结构化数据全在这一个文件的 schema 块。
# 改字段说明 → 同步改 SCHEMA.md

server:
  listen: 127.0.0.1:8787

admin:
  listen: 127.0.0.1:8788
  session_ttl: 3600
  login_rate_limit: 5

vault:
  root: /home/studio/workbench
  git_dir: /home/studio/vault.git

workbase:
  root: /home/studio/workbench/Workbase    # Vault 内 Registry 源（事实源）
  runtime: /home/studio/workbase           # 进程运行时私有区，自动拼接 index/proposals/inbox/audit
  rebuild_cmd: /home/studio/workbase/bin/rebuild-blog.sh

# ============ 可调参数（顶层直读）============
inbox:
  retention_days: 7            # done/abandoned 保留天数

index:
  access:
    half_life_days: 7         # 艾宾浩斯半衰期
    min_score: 0.001          # 低于此值不参与 Hot 排序

knowledge:
  search:
    weights:                  # 留空 = 用代码内 const 默认值
      title:            5.0
      tags:             4.0
      frontmatter:      3.0
      section:          2.0
      fulltext:         1.5
      wikilink_backref: 2.0
      access:           1.0
      recency:          0.5
    intent_bias:
      why:     { frontmatter: 1.3, section: 1.3 }
      when:    { recency: 1.5 }
      entity:  { tags: 1.3 }
      general: {}

audit:
  retention_days: 90          # 审计日志保留天数
  recent_limit: 100           # audit.list_recent 默认返回条数

# ============ Admin 鉴权（单账号）============
# 个人工作台只会有一个 admin 账号；凭证直接写在 config.yaml
admin_auth:
  user: REPLACE_WITH_ADMIN_USER
  pass_hash: REPLACE_WITH_SHA256_HEX_ADMIN    # SHA-256(password)，不含 sha256: 前缀

# ============ schema 块：枚举 / 状态机 / 策略（结构化，不重复）============
schema:
  # 可见性策略（4 档）—— 启动加载到 cfg.Schema.VisibilityPolicy
  visibility_policy:
    public:  "可公开展示与索引"
    private: "授权 Agent 可读"
    secret:  "默认不暴露给远程 MCP"
    draft:   "草稿。授权 MCP 可读；search scope=all / 各 list / context.startup 收录；公开博客不发布"

  # 缺省 visibility（按一级目录）—— 启动加载
  visibility_default:
    文章:               public
    项目:               public
    友链:               public
    部署溯源:           private
    Workbase/context:   private
    Workbase/skills:    private
    Workbase/mcps:      private
    default:            private     # 新增一级目录默认 private（安全开箱即用）

  # 敏感模式。默认 [] = 关闭。个人台要把 Skill / MCP / 文章完整给授权 Agent。
  sensitive_patterns: []

  # 审计最小字段集
  audit_min_fields: [ts, tool, client_id, scopes, args_digest, result_status, duration_ms]

  # 审计 result_status 取值
  audit_result_status: [success, error, unauthorized, forbidden]

  # Proposal 状态机（§17）
  proposal_states: [pending, approved, applied, rejected, conflict]
  proposal_state_transitions:
    pending:  [approved, rejected]   # 创建校验失败是控制层拒绝，不进 conflict
    approved: [applied, conflict]    # 3-way / commit 只在 approved 之后发生
    applied:  []
    rejected: []
    conflict: [approved]             # 可救回：编辑 payload 后重走 approved；默认换新 base

  # Proposal target / operation 类型
  proposal_target_types:    [note, context_pack, project, article, skill, mcp_server]
  proposal_operation_types: [create_file, append, append_section, patch_section, register_item]

  # Inbox 状态机（SCHEMA §17.2）—— pending 可直接 done/abandoned（看板拖拽一步到位）
  inbox_states: [pending, reviewing, done, abandoned]
  inbox_state_transitions:
    pending:   [reviewing, done, abandoned]
    reviewing: [done, abandoned]
    done:      []
    abandoned: []

  # 自动生成 ID 前缀。只给 proposal / inbox。
  # notes.id = vault 相对路径（正斜杠，含 .md），不用前缀。入库 / 查询一律 ToSlash。
  id_prefixes:
    proposal:     "prop"
    inbox:        "inbox"
```

> **schema 块只放结构化定义**（枚举 / 状态机 / 可见性 / 敏感模式）。**不**放可调参数——可调参数走顶层 `inbox` / `index` / `knowledge` / `audit`，缺省用代码 const，**一份数字只写一次**。

#### 8.1.3 Go 实现（单 LoadConfig + 单 Config struct）

```go
// server/mcp/internal/config/config.go
package config

import (
    "fmt"
    "os"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Server     ServerConfig     `yaml:"server"`
    Admin      AdminConfig      `yaml:"admin"`        // listen / session_ttl / login_rate_limit
    AdminAuth  AdminAuthConfig  `yaml:"admin_auth"`   // 单账号 user / pass_hash
    Auth       AuthConfig       `yaml:"auth"`         // grace_period_hours（Token 主体在 SQLite，不在 yaml）
    Vault      VaultConfig      `yaml:"vault"`
    Workbase   WorkbaseConfig   `yaml:"workbase"`
    Inbox      InboxConfig      `yaml:"inbox"`        // retention_days
    Index      IndexConfig      `yaml:"index"`        // access.{half_life_days,min_score}
    Knowledge  KnowledgeConfig  `yaml:"knowledge"`    // search.weights / search.intent_bias
    Audit      AuditConfig      `yaml:"audit"`        // retention_days / recent_limit
    Schema     Schema           `yaml:"schema"`       // 仅结构化（枚举/状态机/visibility/敏感）
}

type AdminAuthConfig struct {
    User     string `yaml:"user"`
    PassHash string `yaml:"pass_hash"`
}

type AuthConfig struct {
    GracePeriodHours int `yaml:"grace_period_hours"`  // 0 = 无灰度。轮换同步改/删旧 cache；撤销 SLA ≤5s
}

type Schema struct {
    VisibilityPolicy          map[string]string          `yaml:"visibility_policy"`
    VisibilityDefault         map[string]string          `yaml:"visibility_default"`
    SensitivePatterns         []string                   `yaml:"sensitive_patterns"`
    AuditMinFields            []string                   `yaml:"audit_min_fields"`
    AuditResultStatus         []string                   `yaml:"audit_result_status"`
    ProposalStates            []string                   `yaml:"proposal_states"`
    ProposalStateTransitions  map[string][]string        `yaml:"proposal_state_transitions"`
    ProposalTargetTypes       []string                   `yaml:"proposal_target_types"`
    ProposalOperationTypes    []string                   `yaml:"proposal_operation_types"`
    InboxStates               []string                   `yaml:"inbox_states"`
    InboxStateTransitions     map[string][]string        `yaml:"inbox_state_transitions"`
    IdPrefixes                map[string]string          `yaml:"id_prefixes"`
}

var cfg *Config

// LoadConfig 启动时调用一次，解析 config.yaml
func LoadConfig(path string) error {
    raw, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("read config.yaml: %w", err)
    }

    c := &Config{}
    if err := yaml.Unmarshal(raw, c); err != nil {
        return fmt.Errorf("parse config.yaml: %w", err)
    }

    if len(c.Schema.VisibilityPolicy) == 0 {
        return fmt.Errorf("config.yaml: schema.visibility_policy 不能为空")
    }
    if c.AdminAuth.User == "" || c.AdminAuth.PassHash == "" {
        return fmt.Errorf("config.yaml: admin_auth.user / pass_hash 必填")
    }

    cfg = c
    return nil
}

func GetConfig() *Config { return cfg }
```

#### 8.1.4 为什么 config.yaml 一个文件含所有结构化数据

| 方案 | 文件数 | 优点 | 缺点 |
|---|---|---|---|
| 多个文件（schema + policies + weights） | 3+ | 关注分离 | 部署要管 3+ 文件；改一个忘了同步另几个 |
| **单 config.yaml** | 1 | **一个文件含所有；改一处一眼看到全貌** | 文件稍大（但仍是 KB 级） |

**结论**：单文件部署友好。**关注分离 = 文件级，粒度按需**——一个文件 1-2KB 是合理。

#### 8.1.5 调用栈

```text
main()
  ↓ LoadConfig("./config.yaml")              // 启动时一次（YAML 解析，含 schema 块）
  ↓ srv := tools.NewServer(cfg)
  ↓ srv.Start(":8787")
  ↓
请求进来
  ↓ auth.Middleware
  ↓ → tokenCache.lookup(hash)                    // 只读内存 cache，不每请求查 SQLite
  ↓ → cfg.Schema.GetVisibilityPolicy()           // 从内存 Config（YAML 加载）
  ↓ → tools.HandleWorkbaseIdentity()             // 读 vault md 即时 + cfg
```

#### 8.1.6 修改生效

- 改 `config.yaml`（任何字段）→ 重启 MCP（`systemctl restart jiangnan-workbase-mcp`）→ 重新 `LoadConfig()` → 内存 Config 替换
- **不**支持热更新，也**不**吃 SIGHUP。Go 默认不处理 reload；写成 `reload` 等于没重启
- 例外：`workbase.identity.workbase.*` 字段每次调用即时读 vault md（§9.1）
- Token 签发 / 轮换 / 撤销走 SQLite + 同步改 cache，**不用**重启

#### 8.1.7 单元测试

- 写 fixture `config.yaml` → 测 `LoadConfig()` 所有字段正确
- 改 fixture 一行 → 期望值变 → 确认 LoadConfig 重新 parse
- 故意写无效 YAML → 测 `LoadConfig()` 返回 error
- 故意缺必填字段（如 `schema.visibility_policy`）→ 测 `LoadConfig()` 返回 error
- 测 `GetConfig()` 返回非 nil + 字段正确


---

## 9. MCP 工具集

20 个工具，每个的字段映射（请求/返回/状态码/错误）在 `SCHEMA.md §4-§20` 维护（一工具一节，§12 = `proposal.update`，§21 是敏感模式，§22-§27 是结构化 / 状态机 / 算法 / 落点 / 修改流程）。本节列用途和关键设计。

```text
workbase.identity          SCHEMA.md §4
context.startup           SCHEMA.md §5
context.get               SCHEMA.md §6
knowledge.search          SCHEMA.md §7
knowledge.get             SCHEMA.md §8
project.list              SCHEMA.md §9
project.get               SCHEMA.md §9.3
skill.list                SCHEMA.md §10
skill.get                 SCHEMA.md §10.3
mcp.list                  SCHEMA.md §11
mcp.get                   SCHEMA.md §11.3
proposal.create           SCHEMA.md §13
proposal.list             SCHEMA.md §14
proposal.get              SCHEMA.md §15
proposal.update           SCHEMA.md §12
inbox.append              SCHEMA.md §16
inbox.update              SCHEMA.md §17
inbox.list                SCHEMA.md §18
inbox.get                 SCHEMA.md §19
audit.list_recent         SCHEMA.md §20
```

### 9.1 workbase.identity

让新 Agent 知道 **Workbase 是什么** + **自己能做什么**。一次调用两个块都拿到。

**所有 token 调用都需要**（MCP 协议无公开 endpoint）。任意有效 token 即可调，**不**要求额外 scope。

#### 9.1.1 Workbase 描述性字段（从 vault 即时读）

| 字段 | 数据源 | 读取方式 |
|---|---|---|
| `id` | Go 常量 | MCP 协议层标识符（`jiangnan-workbase`） |
| `name` | `Workbase/mcps/jiangnan-workbase.md` frontmatter `name` | YAML 解析 |
| `version` | Go 常量 | MCP 协议层版本（`0.1.0`） |
| `description` | `Workbase/mcps/jiangnan-workbase.md` frontmatter `summary` | YAML 解析 |
| `capabilities` | 运行时聚合 | `srv.AddTool` 注册（哪些工具存在） |
| `tools` | 运行时聚合 | `toolScopes` keys |
| `visibility_policy` | `config.yaml` `schema.visibility_policy` | YAML 解析（结构化数据，不走 Vault） |
| `getting_started` | `Workbase/mcps/jiangnan-workbase.md` `## Purpose` | frontmatter 后正文 |
| `critical_rules` | `Workbase/mcps/jiangnan-workbase.md` `## Security` | `-` 列表解析 |
| `see_also` | `Workbase/mcps/jiangnan-workbase.md` `## Source` | 链接解析 |

**读取策略**：

- 每次调用即时读磁盘，**不缓存**
- 任一文件读取/解析失败 → 返回 MCP error，**不**静默 fallback 到硬编码

修改 Workbase 描述 = 修改 vault md + `sync.ps1` 推送 → 不重新编译 Go。

#### 9.1.2 Auth 字段（当前 token 元数据）

| 字段 | 类型 | 说明 |
|---|---|---|
| `client_id` | string | token 名称（即 `auth_tokens.name`） |
| `scopes` | []string | 该 token 拥有的 scope 列表 |
| `status` | enum | `active` / `grace`。`revoked` 过不了 middleware，identity 返回不了 |
| `created_at` | RFC3339 | token 创建时间 |
| `last_used_at` | RFC3339，可选 | 从未用过 → 省略或 `null`。不要空串 / 零时间 |
| `use_count` | int | 新 token = 0。本次 +1 是认证后异步写，本响应可能还是旧值 |
| `allowed_tools` | []string | 由 `scopes` × `toolScopes` 推导出的可调用工具列表（前端友好） |

**不返回**（敏感信息）：

- token 原文
- token hash
- `grace_until`
- 撤销者 / 创建者真实身份

#### 9.1.3 完整响应示例

```json
{
  "workbase": {
    "id": "jiangnan-workbase",
    "name": "Jiangnan Workbase MCP",
    "version": "0.1.0",
    "description": "私密个人 Agent 工作基座",
    "capabilities": {
      "context": true, "knowledge": true, "project": true,
      "skill_registry": true, "mcp_registry": true,
      "proposal": true, "inbox": true,
      "direct_write": false, "vector_search": false
    },
    "tools": [
      "workbase.identity", "context.startup", "context.get",
      "knowledge.search", "knowledge.get",
      "project.list", "project.get",
      "skill.list", "skill.get",
      "mcp.list", "mcp.get",
      "proposal.create", "proposal.list", "proposal.get", "proposal.update",
      "inbox.append", "inbox.update", "inbox.list", "inbox.get",
      "audit.list_recent"
    ],
    "visibility_policy": {
      "public": "可公开展示与索引",
      "private": "授权 Agent 可读",
      "secret": "默认不暴露给远程 MCP",
      "draft": "草稿。授权 MCP 可读；search scope=all / 各 list / context.startup 收录；公开博客不发布"
    },
    "getting_started": "...",
    "critical_rules": ["HTTPS", "visibility", "token+scope", "audit"],
    "see_also": ["https://github.com/Luo-root/jiangnan-blog"]
  },
  "auth": {
    "client_id": "minimax-code",
    "scopes": ["read:context", "read:knowledge", "write:proposal", "write:inbox", "read:inbox"],
    "status": "active",
    "created_at": "2026-08-18T22:00:00+08:00",
    "last_used_at": "2026-08-19T14:00:00+08:00",
    "use_count": 1234,
    "allowed_tools": [
      "workbase.identity", "context.startup", "context.get",
      "knowledge.search", "knowledge.get",
      "proposal.create", "proposal.list", "proposal.get", "proposal.update",
      "inbox.append", "inbox.update", "inbox.list", "inbox.get"
    ]
  }
}
```

**注意**：`workbase.tools` 是 Workbase 提供的所有工具；`auth.allowed_tools` 是当前 token 可调的工具子集。两者关系：`auth.allowed_tools ⊆ workbase.tools`。

#### 9.1.4 Agent 典型用法

```text
1. Agent 连上 MCP（initialize）
2. 调 workbase.identity
   → 知道 Workbase 是什么（workbase 块）
   → 知道自己能干啥（auth.allowed_tools）
3. 按 allowed_tools 调用对应工具
4. 遇到 401 / 403 时再调 workbase.identity 看 scope 是否变化
```

#### 9.1.5 实施位置

- `server/mcp/internal/tools/identity.go`（新文件）
- `server/mcp/internal/auth/middleware.go` 的 `Authenticate()` 返回 `AuthContext{ClientID, Scopes, Status}` → 注入请求 ctx
- `toolScopes` 中 `workbase.identity` 的 `RequiredScope` 设为空字符串（任意有效 token）


### 9.2 context.startup

给新 Agent 一份启动上下文，快速进入状态。**派生结果**，不直接手写或 patch。来源：`Workbase/context/*.md` 中 `startup: true` **且** `visibility ≠ secret` 的 pack，按 `priority` 排序后合成。secret 永不进合成；draft 照收。

### 9.3 context.get

读取具体 context pack。请求 `{id}`，返回 frontmatter + 正文（去 frontmatter）。draft 放行（有 `read:context` 即可）；secret 默认 `secret_blocked`。

### 9.4 knowledge.search

搜索授权范围内的知识库。**信号权重**可配（config 覆盖 + 代码默认 fallback）：

```yaml
# config.yaml
knowledge:
  search:
    weights:                  # 可选覆盖
      title: 5.0              # 命中 title 时加 5.0 分
      tags: 4.0               # 命中 tags 时加 4.0 分
      frontmatter: 3.0        # 命中 frontmatter 任意字段时加 3.0 分
      section: 2.0            # 命中 heading 段时加 2.0 分
      fulltext: 1.5           # 正文 FTS5 命中时加 1.5 分
      wikilink_backref: 2.0   # WikiLink 反链命中时加 2.0 分
      access: 1.0             # 艾宾浩斯热度（见 §25）
      recency: 0.5            # 时间衰减
```

**score 计算**（**绝对分制**，不归一化）：
- `score = Σ (命中信号的 weight)`
- `0` = 无任何命中（**门禁失败，不入结果**）
- 越大 = 越相关
- 物理意义 = "这条结果有多匹配查询"（简单可解释）
- 不同查询的 score **不可直接比较**（每次查询的满分不同）
- 排序用 score 降序即可

例：
- 仅 frontmatter 命中 → `score = 3.0`
- title (5.0) + fulltext (1.5) 命中 → `score = 6.5`
- 仅 access / recency / wikilink_backref 命中 → `score = 0`，不入结果
- title (5.0) + recency (0.5) 命中 → `score = 5.5`（recency 不算门禁字段，但不影响 score）

**intent 调整**（事实源 = `config.yaml` 的 `knowledge.search.intent_bias`，绝对倍率）：

| intent | 调整 |
|---|---|
| `why` | `frontmatter` weight × 1.3 + `section` weight × 1.3 |
| `when` | `recency` weight × 1.5 |
| `entity` | `tags` weight × 1.3 |
| `general` | 无调整 |

默认值在 `server/mcp/internal/search/weights.go` 内 const。逻辑 = 标准 fallback：cfg 有值用 cfg，没值用 const。

**命中门禁与排序分离**：

- 命中门禁：至少一个 `title` / `tags` / `frontmatter` / `section` / `fulltext` 命中才入结果
- 排序信号：上面全部 + `wikilink_backref` / `access` / `recency`

`intent` 参数（`why` / `when` / `entity` / `general`）调整权重偏向。`access` 信号使用艾宾浩斯曲线（见 §25）。

**visibility 过滤**：`scope=all` = public + private + **draft**。`scope=public` / `private` 各只出对应档。**secret 永远不进 search**（含摘要）。secret 只走 `knowledge.get` 显式 id，且默认 `secret_blocked`。`visibility=draft` 有 `read:knowledge` 就能 get。Vite 继续跳过 `draft: true` 或 `visibility: draft`。

**kind**：缺省才用 `["note","article"]`。已传则只保留这两个，其它值丢掉；**过滤后为空 → 空结果，不回落默认**。`kind=["project"]` 不是默认的 note+article。想搜 project 走 `project.list`。

**不静默回落**：`intent` / `scope` 非法值 → `invalid_argument`，不回落 `general` / `all`。详见 `SCHEMA.md §7.7`。

**空结果兜底**（用户搜索无结果时）：

```json
// 任一门禁字段未命中 / 全部结果 score=0
{
  "results": [],
  "message": "未查询到相关内容",
  "suggestions": [
    "缩短关键词：去掉修饰词（'的'/'一个'/'关于'）",
    "改用更通用的词：例如 'kubernetes' 替代 'k8s pod 调度'",
    "检查 scope 权限：当前 token scope 是否包含 read:knowledge",
    "检查 visibility：public 内容只能搜到 public 知识"
  ],
  "query_echo": "Agent Workbase proposal",
  "executed_signals": ["title", "tags", "frontmatter", "section", "fulltext"]
}
```

**不**返回 MCP `error`（"未查询到" ≠ "查询失败"）。错误码统一留给：
- `invalid_argument`（参数缺失 / 类型错）
- `internal_error`（SQLite / FS 故障）
- `unauthorized`（token 无效 / scope 不足）

### 9.5 knowledge.get

按 `id` 读取笔记正文。`id` = `notes.id` = vault 相对路径（正斜杠，含 `.md`）；请求里的 `\` 先 `filepath.ToSlash` 再查。返回 `id` / `title` / `path_hint` / `visibility` / `updated_at` / `body` / `frontmatter` / `forward_links` / `backlinks` / `base_commit`。

- `visibility=secret` 默认 `secret_blocked`，**不**返回内容
- `visibility=draft` 放行（有 `read:knowledge` 即可）
- 授权范围内的正文**原样返回**（敏感模式默认关，见 §20.2 / SCHEMA §21）
- `base_commit` 字段供 proposal 写入使用

### 9.6 project.list / 9.7 project.get

读 `项目/*.md`。`get` 返回项目当前重点、决策、下一步（按 frontmatter 字段映射）。list 出 public + private + draft，不出 secret。get 对 draft 放行；secret 默认 `secret_blocked`。

### 9.8 skill.list / 9.9 skill.get

读 `Workbase/skills/*.md`。`list` 返回摘要 + 风险 + 来源 + 标签；`get` 返回 frontmatter + 完整正文。list / get 对 draft / secret 的规则与 project 相同（scope 换成 `read:registry`）。

### 9.10 mcp.list / 9.11 mcp.get

读 `Workbase/mcps/*.md`。`list` 返回摘要 + transport + auth + risk + source；`get` 返回 endpoint + auth + scopes + tools + 正文。list / get 对 draft / secret 的规则与 project 相同（scope 换成 `read:registry`）。

### 9.12 proposal.create

创建 proposal。客户端可传可选 `expected_base`（通常是刚读到的 `knowledge.get.base_commit`）：

- 不传 → 服务端读当前 HEAD 写入 `base_commit`
- 有传且等于当前 HEAD → 用这个值
- 有传但对不上 → 拒绝，`stale_base`，提示「你读的已经不是最新」

请求 schema 在 `SCHEMA.md §13`。枚举合法 ≠ 一定受理：`target.type` × `operation.type` 必须在 §15.7 矩阵内，否则 `operation_not_supported`。

### 9.13 proposal.list / 9.14 proposal.get

按 status / created_by / time 范围列出 / 读取单条。`get` 返回完整 proposal + receipt（含 commit / content_sha256 / replayed）+ `comments`（时间正序）。

### 9.15 proposal.update

Agent 或 webUI 改 **尚未落盘** 的 proposal。字段映射 `SCHEMA.md §12`。这是第 20 个 MCP 工具，不是预留空节。

只允许 `pending` / `conflict`。可改 `reason` / `target` / `operation` / `payload`，可追加一条 `comment`。评论 **不** 改 status。`applied` / `rejected` / `approved`（正在 apply）一律 `invalid status transition`。

小问题：人在 webUI 直接改 payload 再批准。大问题：人留评论，Agent 用本工具改 payload 并回复，人再批准。

### 9.16 inbox.append / 9.17 inbox.update

新建 pending 待办 / 编辑内容或改变状态 / 追加评论。不进 Vault，不触发 apply / commit。敏感模式开启且命中 → 响应 `warnings`，照样创建/更新。`warnings` **不**写进 `{id}.md`。

`comment` 可选。有值则 append 一条，不能改历史评论。评论不改 status。

### 9.18 inbox.list / 9.19 inbox.get

按 status / created_by / time 列出 / 读取单条。`get` 每次读再扫一遍 content 填 `warnings`，并返回 `comments`。`list` 不返回 `warnings` / `comments` 全文（看板只要摘要；`comment_count` 可带）。`done` / `abandoned` 超过 `retention_days` 不返回（已自动删除）。

### 9.20 audit.list_recent

按 time 范围 / client_id / tool 过滤，返回审计摘要。详细 schema 在 `SCHEMA.md §20`。MCP 本工具只用 `since`（从某时起到现在）。webUI `GET /api/audit/recent` 额外收 `until`，见 §21.5.5。

---

## 10. Skill Registry

事实源：`Workbase/skills/*.md`。frontmatter schema 在 `SCHEMA.md §10.1`。正文固定小节：`# Name` / `## Purpose` / `## When to use` / `## Inputs` / `## Outputs` / `## Procedure` / `## Safety` / `## Source`。

不提供 `skill.get_install_guide`，安装方式如有必要写进 Skill 正文。

---

## 11. MCP Registry

事实源：`Workbase/mcps/*.md`。frontmatter schema 在 `SCHEMA.md §11`。`mcp.list` / `mcp.get` 差异在 `SCHEMA.md §11.2-§11.4`。

不提供 `mcp.get_connection_guide`，Agent 自行适配。

---

## 12. Project Registry

事实源：`项目/*.md`。frontmatter schema 在 `SCHEMA.md §9`。字段映射在 `SCHEMA.md §9.2-§9.3`。

---

## 13. Knowledge Note

事实源 = indexer 标成 `note` 或 `article` 的 md：

- `文章/**/*.md` → `kind=article`
- vault 里其它未被排除的 md（含 `部署溯源/`；以及 `Workbase/` 下除 `context/` `skills/` `mcps/` 以外、有人误放的 md）→ `kind=note`

**`notes.id` = vault 相对路径**（正斜杠，含 `.md`，如 `文章/foo.md`）。跨目录唯一。不要只拿文件名。`knowledge.search` / `knowledge.get` 的 `id` 跟这一条走。skill / mcp / context 对外仍用 frontmatter `id`，和 notes PK 是两套。

**路径归一**：入库和查询一律 `filepath.ToSlash`。请求里的 `\` 先归一再查。Windows 上 `filepath.Rel` 会吐反斜杠，不归一则 get 404、WikiLink 对不上。旧 `vault.go:noteID()` 还 `TrimSuffix(rel, ".md")`——那是旧代码，不要当规格。

WikiLink 解析：先按完整相对路径（去 `.md`），再按文件名唯一匹配。重名只记 `raw`，不建边——避免 `文章/foo.md` 和 `部署溯源/foo.md` 撞车。`links.source_id` / `target_id` 都是 `notes.id`。

**不进 knowledge**（indexer 直接跳过或标成别的 kind）：

- `友链/`：跳过，不入库
- `项目/`：`kind=project`，只走 `project.*`
- `Workbase/context|skills|mcps/`：各自 kind，只走对应工具

`knowledge.search` 默认 `kind=["note","article"]`，`scope=all` 含 `draft`。已传 `kind` 只保留 `note`/`article`；过滤后为空 → 空结果，不回落默认。响应 `kind` 原样返回，**不要**固定写成 `note`。没有 `article.list` / `article.get`。

`knowledge.get` 只返回 `kind ∈ {note, article}`。其它 kind（即使 `notes.id` 对得上）→ `note not found`，不泄露是 skill。`visibility=draft` 放行；`secret` 默认 `secret_blocked`。

frontmatter 见 `SCHEMA.md §22`，indexer 表见 `SCHEMA.md §23`。

---

## 14. Workbase 目录分类

`Workbase/` 下文件按 `kind` 字段或所在一级目录分类（fallback）：

| 工具 | 能访问的 kind | 数据源 |
|---|---|---|
| `workbase.identity` | mcp_server | `Workbase/mcps/jiangnan-workbase.md` |
| `context.startup/get` | context_pack | `Workbase/context/*.md` |
| `knowledge.search/get` | `note` + `article` | `文章/` 以及未被排除的普通 md。默认 search 两者都收。get 其它 kind → `note not found` |
| `project.list/get` | project | `项目/*.md` |
| `skill.list/get` | skill | `Workbase/skills/*.md` |
| `mcp.list/get` | mcp_server | `Workbase/mcps/*.md`（除自己） |

**`Workbase/` 在公开博客构建时排除**（`vite.config.ts`），但 MCP 索引**包含**——按 kind 隔离访问。

---

## 15. Proposal 通用写入协议

### 15.1 定位

Proposal = 统一写入意图格式。跨领域 envelope：`Intent + Target + Operation + Payload + Validation + Audit`。

### 15.2 为什么需要

直接写入的风险：Agent 选错目标 / 污染结构 / 临时想法当长期决策 / 泄露敏感 / 与本地 Obsidian 冲突。Proposal 让写入变成：表达意图 → 展示 diff → 人工确认 → 落盘 → 审计。

### 15.3 Schema

```yaml
id: prop_20260817_001
kind: note                      # 服务端从 target.type 抄，客户端不传
status: pending
base_commit: abc123             # 服务端写入：expected_base 对得上才用，否则当前 HEAD

created:
  by: minimax-code
  at: 2026-08-17T17:00:00+08:00
  reason: 记录 Agent Workbase MCP 设计决策

risk:
  level: medium
  reasons:
    - 修改长期项目上下文
  requires_approval: true

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

validation:                     # 服务端结果，不是请求字段
  warnings: []                  # 敏感模式开启时的命中提示；默认关则为空
```

### 15.4 target.type

```text
note             Vault 任意授权 md
context_pack     Workbase/context/*.md
project          项目/*.md
article          文章/*.md
skill            Workbase/skills/*.md
mcp_server       Workbase/mcps/*.md
```

### 15.5 operation.type

```text
create_file          新建**文件**（路径必须在 vault.root 下，且文件还不存在）
append               文件末尾追加（文件必须已存在）
append_section       追加到指定标题下（标题不存在 → 先建标题再追加）
patch_section        替换指定标题内容（标题不存在 → conflict）
register_item        新增 registry item（skill / mcp，同 create_file）
```

`create_file` / `register_item` **不是**「建空文件夹」操作。`target.path` 必须指向一个文件（通常 `.md`）。父目录不存在时 apply 用 `MkdirAll` 建父目录，这是落盘副作用，枚举里没有 `mkdir` / `create_dir`。不要用 `create_file` 去建一个没有正文的目录。

不引入 JSON Patch / AST Patch。不引入 `replace_frontmatter`（枚举里没有就不收）。

### 15.6 Adapter 模型

对外只有统一 proposal。内部按 target type 分发：NoteAdapter / ContextPackAdapter / ProjectAdapter / ArticleAdapter / SkillAdapter / MCPServerAdapter。

Adapter 负责：校验 target、校验 operation、生成 preview / diff、Markdown fence 检查、visibility 检查、apply 落盘。敏感 regex 默认关；开了只记 warning。

### 15.7 受理矩阵

枚举 = 矩阵。yaml 里的 `proposal_operation_types` 就是当前能发的，不再搞「合法但不能用」。

| target.type | operation | apply 方式 |
|---|---|---|
| `note` | `append` / `append_section` | 先施加再 3-way |
| `context_pack` | `append_section` / `patch_section` | 先施加再 3-way |
| `project` | `patch_section` / `append_section` | 先施加再 3-way |
| `article` | `create_file` | 新文件：落盘前再看一眼路径；已存在 → conflict；不存在 → 直接落盘 |
| `skill` | `register_item` | 同 `create_file` |
| `mcp_server` | `register_item` | 同 `create_file` |

`create_file` / `register_item` 没有 ancestor，不走 3-way。并发两个 approved 同路径：第二个落盘前再 stat 一次，已存在 → conflict，禁止互相覆盖。

---

## 16. Inbox 设计

### 16.1 定位

Inbox = 独立待办（todo），**不是**知识写入、**不是** Proposal 中间态、**不**进入 Obsidian Vault。只负责记录"要处理的事"并跟踪状态。

不做审批、不做 apply、不做 git commit、不转 proposal、不与 Obsidian 联动。

### 16.2 适合 / 不适合

适合：对话中产生的跟进事项、临时想法（待办化）、排查中的问题、想做但未排期的任务。

不适合：正式文章正文、长期项目决策、安全策略正式条目、Skill/MCP 正式 registry item、部署配置、凭据/token/私钥。

以上"不适合"若要进入正式知识库，应走 Proposal（§15），而不是塞进 inbox。

### 16.3 状态机

```text
pending → reviewing → done | abandoned
pending → done | abandoned              # 看板拖拽一步到位，跳过 reviewing
```

| 状态 | 含义 |
|---|---|
| `pending` | 待处理（刚创建，还没做） |
| `reviewing` | 待审核（已做完，等确认；不是 Proposal 审批，不触发 apply/commit） |
| `done` | 已完成（审核通过） |
| `abandoned` | 已废弃（不再需要处理） |

状态机 **不可逆**：`done` / `abandoned` 不能拖回 `pending` / `reviewing`；`reviewing` 不能拖回 `pending`。看板拖到非法列必须拒绝并提示，卡片留在原列。终态条目仍可改正文 / 追加评论，直到 retention 删除。

### 16.4 卡片：先预览再编辑

点开卡片默认 **预览**：正文走 Markdown 渲染（GFM），方便人读。点「编辑」才切 textarea，用来纠正 Agent 写偏的内容。保存后回到预览。不要一打开就是源码框。

四列必须有视觉差，不能只靠标题文字：

| 列 | 语义色 |
|---|---|
| `pending` | 墨灰 / 未开始 |
| `reviewing` | 琥珀 / 等人看 |
| `done` | 绿 / 通过 |
| `abandoned` | 浅灰 / 废弃 |

### 16.5 评论线程（人审 ↔ Agent）

Inbox **不是** Proposal，没有批准 / apply。但 `reviewing` 的本意就是「Agent 做完了，等人看」。人审完有两条路：

- 效果够 → 拖到 `done`
- 效果不够 → **不改状态**，留一条评论，让 Agent 按评论继续改 `content`，Agent 也可以回复这条评论

评论规则：

```text
评论不改 status
评论只追加，不改、不删历史
author_type = human | agent
webUI 登录用户 → human；MCP token 调用 → agent（created_by = token name）
敏感模式开了：评论 body 命中只 warning，照样写入
```

落盘：`{id}.md` frontmatter 的 `comments:` 数组。正文仍是任务内容，评论不进 Markdown body。形状见 `SCHEMA.md §12.4`（Inbox / Proposal 共用 Comment 对象）。

### 16.6 生命周期

`done` / `abandoned` 保留 `retention_days` 天后自动删除。`pending` / `reviewing` 保留到状态改变为止，不设自动删除。

`retention_days` 可配：

```yaml
# config.yaml
inbox:
  retention_days: 7           # 默认 7
```

逻辑 = 标准 fallback：cfg 有值用 cfg，没值用 const。

### 16.7 操作

```text
inbox.append   新建 pending。敏感命中 → 响应 warnings，照样创建。warnings 不落盘
inbox.update   编辑内容 / 改变状态 / 追加评论。评论不改 status。同上
inbox.list     列出摘要。不返回 warnings / 评论全文
inbox.get      读取单条（含 comments）。每次读 rescan content 填 warnings
```

### 16.8 与 Proposal 的关系

两者定位不同，完全独立：

```text
proposal = 正式知识写入请求（target + operation，审批 → 3-way apply → commit）
inbox    = 独立待办（无 target，无审批，无 apply，无 commit）
```

inbox 不能"转 proposal"：如果后续需要正式写入，应独立创建新 proposal，两者不自动关联。

proposal 流转：

```text
Agent 写入
  ↓
proposal.create（target + operation，可选 expected_base）
  ↓
/home/studio/workbase/proposals/
  ↓
用户 webUI 审批（同意 / 编辑后同意 / 拒绝）
  ↓
同意 → 施加 operation 得完整 ours → 3-way → git commit（成功 = applied）
      → reindex + rebuild（副作用，失败仍 applied）
拒绝 → rejected
冲突 → conflict（仅 approved 之后），proposal 保留，救回默认换新 base
```

inbox 流转：

```text
Agent / WebUI
  ↓
inbox.append（新建 pending，id = inbox_YYYYMMDD_HHMMSS_fff）
  ↓
/home/studio/workbase/inbox/{id}.md     # 文件名与 id 同一套
  ↓
inbox.update
  pending → reviewing → done | abandoned
  pending → done | abandoned              # 看板拖拽一步到位，§16.3 已有
  ↓
retention_days 后自动删除（done / abandoned）
```

### 16.9 inbox 只存 VPS

inbox 只落 VPS 私有区 `/home/studio/workbase/inbox/`，不进入本地 Obsidian Vault，不触发 apply / git commit / rebuild。

---

## 17. Git-backed 写入

### 17.1 观点

公网 MCP 可以写入正式 Vault，但写入必须是：Git-backed / base-commit aware / diff-first / approval-first（webUI）/ conflict-stop / audited。

Inbox 是独立待办，不在此列。

### 17.2 写入能力

当前全部具备，不是分期：

```text
proposal（pending，不落正式正文）
webUI 审批 apply（Git-backed commit）
expected_base 校验 + diff preview + 完整 ours 后再 3-way + 冲突停止
```

### 17.3 apply 流程（webUI 审批后）

`git merge-file` 要的是三份**完整文件**。payload 经常是半句话，不能当 `ours`。正确模型是两步：

```text
1. Agent 创建 proposal（可选 expected_base，见 §9.12）。
2. 服务端生成 diff / preview，标记 pending。
3. 用户在 webUI 查看，可编辑表述。
4. 用户同意 → 进入 approved，然后：

   A. 构造完整 ours（这一步必须先做）
      - 已有文件：在 base_commit 的目标文件上，把 operation 施加一遍
        → 得到完整文件 ours
      - create_file / register_item：
        落盘前再 stat 一次路径（防两个 approved 并发都看见「不存在」）
        已存在 → conflict，停止
        不存在 → 直接落盘，不走 merge，跳到 C
      - append_section 目标标题不存在 → 先建标题再追加
      - patch_section 目标标题不存在 → conflict

   B. 3-way（仅已有文件）
      - HEAD = base_commit → ours 直接落盘，不走 merge
      - HEAD ≠ base_commit：
        base  = 文件 @ base_commit
        other = 文件 @ HEAD
        ours  = A 得到的完整文件
        无冲突 → 用 merge 结果
        文本冲突或 frontmatter 内部冲突 → conflict，停止
        merge 工具失败 → conflict，停止

   C. 落盘 + git commit
      失败 → 状态 conflict（文件未进 HEAD，可重试）
      成功 → 状态 applied（见下，rebuild 失败不再改这个状态）

   D. reindex + rebuild（副作用，不参与状态机）
      失败 → 仍然 applied；单独重跑，禁止再 apply

5. 用户拒绝：标记 rejected。
6. conflict 救回：编辑 payload 后重新 approved。
   默认重读当前 HEAD 作为新 base_commit。
```

### 17.4 冲突策略

| 情况 | 策略 |
|---|---|
| 目标文件在 base_commit 后无修改 | 完整 ours 直接落盘 |
| 目标文件有修改但无冲突 | 3-way（完整 ours vs HEAD）后 apply |
| 3-way 文本冲突 | conflict，保留 proposal |
| 3-way frontmatter 内部冲突 | conflict（结构化字段不自动合并） |
| create_file / register_item 且路径已存在（含落盘前再 stat） | conflict，禁止互相覆盖 |
| create_file / register_item 且路径不存在 | 直接落盘，不走 merge |
| append_section 目标标题不存在 | 先建标题再追加 |
| patch_section 目标标题不存在 | conflict |
| 写入 `visibility=secret` 文件 | 拒绝（控制层，不写 receipt）。跟敏感 regex 无关 |
| 命中 `sensitive_patterns` | 默认关。开了也只在 validation 记 warning，不拒 |

### 17.5 禁止策略

```text
不自动解决语义冲突
不自动覆盖本地 Obsidian 更新
不把服务器 workbench 变成无约束第二主写入源
不自动合并 frontmatter 字段（即使语义上能合并）
```

### 17.6 Receipt

```text
pending → approved → applied    (终态)
        ↘ rejected              (终态)
approved → conflict → approved  (可救回；默认换新 base)
```

没有 `pending → conflict`。创建校验失败是控制层拒绝，不写 receipt。

`applied` 严格定义：完整 ours 构造成功 +（如需）3-way 成功 + git commit 成功 + 目标文件 SHA-256 校验通过。**不含** reindex / rebuild。用户点「同意」只是 `approved`。

**`applied` / `rejected` 是终态。** 不能再改 payload、不能再批准、不能再 apply。Vault 里那次 commit 已经发生（或明确拒绝）；要再改就开一条新 proposal。webUI 对这两态只读：预览 + 评论只读 + 无「保存 / 批准 / 拒绝」。

`conflict` 是**暂停态**。救回时默认重读当前 HEAD 当新 `base_commit`（旧 base 大概率过时）。也可以明确「只改 payload、不换 base」——再冲突就再停。`pending` / `conflict` 可以：人直接改 payload；或人留评论，Agent 用 `proposal.update` 改完再回复。

### 17.7 详情页：先预览变更，再表单

点开 proposal **先看变更**，不要一进就是一堆输入框。

1. 顶栏：id / status 色标 / created_by / created_at / base_commit
2. 主区：红绿 Diff Viewer（`§21.5.4`）。`create_file` 的 before 为空文件
3. 折叠「元数据」：reason / target.path / operation / section
4. `pending` / `conflict` 才展开可编辑表单；编辑后即时重算 diff
5. 底部：评论线程（与 Inbox 同一套 `comments` 形状）

`diff` 字段（proposal.get）给人/Agent 读文本预览；webUI 用 before/after 做红绿对照，两者都要有。

```yaml
receipt:
  proposal_id: prop_20260817_001
  status: applied            # pending | approved | applied | rejected | conflict
  applied_at: 2026-08-17T17:05:00+08:00
  commit: def456              # 仅 applied：apply 后的 git commit
  content_sha256: "..."       # 目标文件 apply 后内容哈希
  base_commit: abc123         # 原始 base_commit
  merge_strategy: none | three_way    # 应用策略
  replayed: false             # 幂等标记
```

关键语义：

1. applied = 完整 ours +（如需）3-way + git commit + hash 校验。reindex / rebuild 不算进状态
2. conflict 不改动任何文件，保留原 proposal 供重读
3. 幂等：同一 proposal 已 applied 再点同意 → 返回原 receipt，replayed=true。禁止再 commit
4. 校验失败（secret 命中 / fence 不闭合 / 矩阵外 operation）是控制层拒绝，不是 receipt

---

## 18. 同步模型

### 18.1 正向

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

### 18.3 reindex 触发

post-receive hook 末尾主动触发：git checkout 到 workbench → 触发博客 build → 触发 MCP reindex。

reindex 只绑一处：同进程 `POST /internal/reindex`，和 MCP 协议 mux 分路，**不要**让 mcp-go 把这条 POST 吃成协议错误。

保护不靠 `RemoteAddr == 127.0.0.1`（反代会改掉对端地址）：

```text
1. Caddy 公开反代排除 /internal*     ← 公网打不到
2. 进程 bind 127.0.0.1:8787          ← 公网网卡听不到
3. hook / apply 用本机 127.0.0.1 直打，不带 Bearer
```

webUI 走同一条内部 HTTP。不要再写 8788，也不要给 Token 勾选 `admin:reindex`。

### 18.4 反向同步（MCP 审批 apply 后回流）

```text
用户 webUI 审批通过
  ↓
MCP 修改 /home/studio/workbench/<target>（含可能的 3-way merge 结果）
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

注意：apply 是直接改 bare repo 的 working tree，绕过 post-receive hook（hook 只在 push 时触发），所以 apply 后的 rebuild + reindex 必须**手动补触发**，不能指望 post-receive 自动完成。

**手动补触发的具体命令**（副作用失败时用，proposal 已经是 `applied`）：

```bash
# 1. MCP reindex：只绑 8787 loopback，无 Bearer
curl -X POST http://127.0.0.1:8787/internal/reindex

# 2. 博客 rebuild
/home/studio/workbase/bin/rebuild-blog.sh
```

**实施位置**：`server/mcp/internal/apply/apply.go` 拆成两段，不要挤进一个状态：

```text
commitAndMarkApplied():
  1. 落盘 + git commit + SHA-256 校验
  2. 失败 → conflict（文件未进 HEAD，可重试）
  3. 成功 → 状态 applied，写 receipt

runSideEffects():
  4. POST http://127.0.0.1:8787/internal/reindex   # 用 cfg.Server.Listen，不是 AdminListen
  5. exec cfg.Workbase.RebuildCmd
  6. 失败 → 仍然 applied；记日志；用户手动重跑上面两条。禁止再 apply
```

| 阶段 | 失败 | 状态 |
|---|---|---|
| 落盘 + `git commit` | 失败 | `conflict`（文件未进 HEAD，可重试） |
| commit 已成功，reindex / rebuild 失败 | 失败 | **已经 `applied`**，副作用单独重跑，禁止再 apply |

### 18.5 本地 sync.ps1 调整

sync.ps1 从"只 push"改为"先 pull 再 push"：

```text
git pull --rebase
git push
```

MCP 在 VPS 上的 commit 会先合并回本地，再推送本地新改动。冲突由 git 正常处理。

### 18.6 冲突处理

| 情况 | 策略 |
|---|---|
| MCP apply 基于旧 base | 3-way merge（无冲突 apply / 有冲突 conflict） |
| 本地与 VPS 分叉 | git pull --rebase 正常合并 |
| 合并冲突 | git 保留冲突标记，用户手动解决 |
| MCP 不直接覆盖本地未同步内容 | 依赖 git 冲突机制兜底 |

---

## 19. Index 设计

### 19.1 不使用向量数据库

v0.1-v1.0 不引入向量数据库。Markdown + frontmatter + WikiLink 结构已经很强；当前更需要项目状态、决策、下一步，而不是相似段落召回；向量库增加部署调试复杂度；个人知识库错误召回风险较高。

### 19.2 SQLite

三套库，不是一张表。Token 不进 `notes`。表结构以 `SCHEMA.md §23` 为准。

```text
{runtime}/index/notes.sqlite
  notes / notes_fts / links / backlinks     vault 镜像；kind 字段隔离类型
{runtime}/auth.sqlite
  auth_tokens                               Agent Token；不进 Vault / Git
{runtime}/audit/audit.sqlite
  audit_log                                 审计；不进 Vault / Git
```

`notes.kind` 取值：`note` / `context_pack` / `project` / `skill` / `mcp_server` / `article`。不开分表。这是当前完整设计。

### 19.3 Index 输出

索引就是 SQLite，不开 JSON sidecar 当分表。调试 dump 可以临时写，不进契约、不进 Git。

```text
{runtime}/index/notes.sqlite
{runtime}/auth.sqlite
{runtime}/audit/audit.sqlite
```

### 19.4 访问计数与热度算法

**艾宾浩斯遗忘曲线**：

```text
score = access_count * exp(-elapsed_days / HALF_LIFE_DAYS)
```

- `HALF_LIFE_DAYS` 默认 7（来自 `config.yaml` 的 `index.access.half_life_days`）
- `elapsed_days` = `now - last_access_at`
- `Hot()` 排序按 score 降序
- `score < min_score`（默认 `0.001`）不进榜；`score = 0`（完全未访问）不进榜
- 事实源 = `config.yaml` 的 `index.access.min_score`

**配置**：

```yaml
# config.yaml
index:
  access:
    half_life_days: 7         # 默认 7
    min_score: 0.001           # 低于此值不参与 Hot 排序
```

`Hot()` **实时计算**（不预存），保证 `last_access_at` 变化立即反映。

服务两个方向：

1. **排序加权**：`knowledge.search` 的 `signals.access` 使用本算法
2. **冷数据清理候选**：长期低 score 条目进入清理候选，由用户确认后删除。当前只做排序，不自动清理。
3. **webUI 热度榜**：`/workspace/access` 读 `Hot()`。空榜是合法状态，不是 bug

**谁会加 `access_count`**：只有 MCP 的 `knowledge.get` / `context.get` / `project.get` / `skill.get` / `mcp.get` 命中时 `Hit()`。下面这些 **不加**：

- `knowledge.search` / 后台 `/api/knowledge/search`
- 后台 `/api/knowledge?id=`（管理员预览）
- inbox / proposal / audit / identity / list 类工具
- 公开博客页面浏览

所以刚部署、Agent 还没 `get` 过任何条目时，热度页必须写清：「暂无访问记录。热度只统计 MCP get，不统计后台浏览。」不要画假条。

---

## 20. 安全设计

### 20.1 未授权访问 vs 授权读全文

挡未授权的是 **token + scope + visibility**，不是正则：

- 没 token / scope 不够 → 401 / 403
- `visibility=secret` 默认不进 search / list / `context.startup`；get 默认 `secret_blocked`
- `visibility=private` 只给有对应 scope 的 token
- `visibility=draft` 进 `scope=all` search、各 list、`context.startup`；有对应 scope 就能 get。Vite 跳过

授权 Agent 读 Skill / MCP / 文章拿**完整原文**（含 endpoint、配置说明、部署笔记）。不要把 `password: 用户自备`、`Go 1.25.0.1`、`8.8.8.8` 打成 `[REDACTED]`。

真正的密钥不该写进 vault md。写了也原样返回给授权调用方——这是用户自己的选择。

日志 / audit 仍不写 token 原文、不写 token hash（审计字段约束，跟敏感开关无关）。

### 20.2 敏感模式检测

**默认关。** `schema.sensitive_patterns: []` = 写入不检测、读出不打码。个人工作台默认开会误伤部署笔记和 Skill/MCP 配置。

可配：往 yaml 列表加 regex 才开启。开启后也**只警告、不拒绝、读出不打码**。详见 `SCHEMA.md §21`。

### 20.3 Audit

**最小字段集**（每条 audit 记录必含）：

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `ts` | RFC3339 | yes | 工具调用时间 |
| `tool` | string | yes | 工具名 |
| `client_id` | string | yes | Agent 标识（来自 SQLite `auth_tokens.name`，**不是** token 本身） |
| `scopes` | []string | yes | 实际授予的 scope 列表 |
| `args_digest` | string | yes | 参数的 SHA-256（不存原文） |
| `result_status` | enum | yes | `success` / `error` / `unauthorized` / `forbidden` |
| `duration_ms` | int | yes | 执行时长 |
| `error` | string | no | 错误信息（不含敏感数据） |
| `target_path` | string | no | apply/proposal 类的目标文件 |
| `commit` | string | no | apply 类的 git commit hash |
| `base_commit` | string | no | apply 类的 base_commit |

详细 schema 在 `SCHEMA.md §20`。`audit.list_recent` 一次返回最近 N 条，默认 100。

**client_id 获取流程**（请求时）：

1. HTTP middleware 从 `Authorization: Bearer <token>` 解析 token
2. `tokenCache.lookup(SHA-256(token))`——**只读内存 cache**，不每请求查 SQLite
3. 命中行的 `name` 字段 → 注入请求 context 的 `client_id`
4. tool handler 从 ctx 取 `client_id`，写入 audit
5. 整条链路**不存** token 原文 / token hash 到 audit

`tokenCache` 由签发 / 轮换同步 upsert、撤销同步删。每 5s reload 只给崩溃恢复。实施位置：`server/mcp/internal/auth/middleware.go` 的 `Authenticate()` 返回 `AuthContext{ClientID, Scopes}`。

### 20.4 日志脱敏

所有日志中：

```text
Authorization header → [REDACTED]
token                → [REDACTED]
private key          → [REDACTED]
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

公开反代 **必须排除** `/internal*`。整站 `reverse_proxy 8787` 会把 reindex 暴露到公网，而且对端地址变成 Caddy，`RemoteAddr == 127.0.0.1` 判断失效。

```text
mcp.<domain> {
    handle /internal* {
        respond 404
    }
    reverse_proxy 127.0.0.1:8787
}

workbase.<domain> {
    reverse_proxy 127.0.0.1:8788
}
```

8787 同进程 mux：

```text
POST /internal/reindex  → 内部 handler（hook / apply 副作用 / webUI）
其余                    → mcp-go Streamable HTTP
```

不要让 mcp-go 把 `/internal/reindex` 吃成协议错误。

### 21.3 端口

```text
127.0.0.1:8787   MCP HTTP
127.0.0.1:8788   Admin HTTP。进程 bind loopback；公开入口靠 Caddy。不要用 RemoteAddr 判断来源（反代会改掉对端地址）。
```

### 21.4 配置

真实 `config.yaml` **不进 Git**（含 admin `pass_hash` / 路径 / 灰度配置）。**Token 全部在 SQLite `auth_tokens` 表**，不在 yaml 里。**`server/mcp/config.example.yaml` 入库作为模板**（仅含字段结构 + `REPLACE_WITH_*` 占位符，部署时按需替换）。完整配置 schema 在 `SCHEMA.md §1`：

```yaml
server:
  listen: 127.0.0.1:8787

vault:
  root: /home/studio/workbench
  git_dir: /home/studio/vault.git

workbase:
  root: /home/studio/workbench/Workbase    # Vault 内 Registry 源（事实源）
  runtime: /home/studio/workbase           # 进程运行时私有区

admin:
  listen: 127.0.0.1:8788
  session_ttl: 3600          # session 过期秒
  login_rate_limit: 5        # 每分钟最多失败次数

# Token 灰度（撤销 / 轮换时旧 token 的宽限期），唯一保留的 auth 字段
# Token 主体 = SQLite auth_tokens 表（§6.4），不在 yaml
auth:
  grace_period_hours: 0       # 0 = 无灰度。轮换同步改/删旧 cache；撤销 SLA ≤5s。N = 灰度 N 小时

# 单账号 admin 凭证（个人工作台只一个）
admin_auth:
  user: REPLACE_WITH_ADMIN_USER
  pass_hash: REPLACE_WITH_SHA256_HEX_ADMIN    # SHA-256(password)，不含 sha256: 前缀

inbox:
  retention_days: 7          # done/abandoned 保留天数

index:
  access:
    half_life_days: 7        # 艾宾浩斯半衰期
    min_score: 0.001

knowledge:
  search:
    weights:                 # 留空 = 用代码内 const 默认值
      title: 5.0
      tags: 4.0
      frontmatter: 3.0
      section: 2.0
      fulltext: 1.5
      wikilink_backref: 2.0
      access: 1.0
      recency: 0.5
```

`schema` 块（visibility / 状态机 / 敏感模式 / 枚举）不在这里再抄一遍。权威副本 = `SCHEMA.md §1` 与 `server/mcp/config.example.yaml`。`sensitive_patterns` 默认 `[]`。

### 21.5 WebUI 后台

**技术栈**：React + Vite + Tailwind + TypeScript。交互控件走 **shadcn/ui**（Radix + Tailwind），与博客同源。不要自写 toast / 日期框 / Dialog / Select。详见 §21.5.10。

```text
server/mcp/admin/
├── src/
│   ├── main.tsx
│   ├── routes/
│   │   ├── login.tsx
│   │   ├── workspace/
│   │   │   ├── inbox.tsx
│   │   │   ├── proposals.tsx
│   │   │   ├── proposal/$id.tsx        # 详情 + diff viewer
│   │   │   ├── access.tsx              # 访问热度
│   │   │   ├── audit.tsx               # 审计日志
│   │   │   └── search.tsx              # 知识搜索（不走 MCP）
│   │   └── settings/
│   │       ├── token.tsx               # Token 管理
│   │       ├── system.tsx              # System 健康
│   │       ├── git.tsx                 # Git 变更
│   │       └── templates.tsx           # 模板
│   ├── components/
│   │   ├── ui/                         # shadcn/ui（从博客同源复制，admin 独立构建）
│   │   ├── toast.tsx                   # 薄封装 sonner，禁止自写 toast 列表
│   │   ├── date-time-picker.tsx        # Calendar + Popover + 时分，禁止 datetime-local
│   │   ├── comments.tsx                # 领域：评论线程
│   │   ├── markdown.tsx                # 领域：轻量 GFM 预览
│   │   └── diff-viewer.tsx             # 领域：红绿对比
│   ├── lib/
│   │   ├── api.ts                      # 后端 HTTP 客户端
│   │   ├── auth.ts                     # session 管理
│   │   ├── utils.ts                    # cn()
│   │   └── templates.ts
│   └── styles.css                      # admin skin token + shadcn CSS 变量
├── package.json
├── vite.config.ts
└── tsconfig.json
```

**登录**：独立 session token（不用浏览器弹窗 Basic Auth）。

- `POST /api/admin/login` → 颁发 session token（短期，1h）+ refresh token
- 凭证 = `config.yaml` 的 `admin_auth.user` + `pass_hash`（SHA-256 原文比对，**无盐**。单账号个人台可接受，不是通用口令方案）
- 失败限流：5 次/分钟（可配，`admin.login_rate_limit`）

**视觉规范**：与博客共享 `tokens.css`，后台 = admin skin（明亮专业）。

#### 21.5.1 路由分组（workspace / settings）

```text
/                     → 重定向到 /workspace/inbox（已登录）或 /login
/login                → 登录页

# workspace = 日常内容
/workspace/inbox                  → 看板
/workspace/proposal               → Proposal 列表
/workspace/proposal/$id           → Proposal 详情 + diff viewer
/workspace/access                 → 访问热度
/workspace/audit                  → 审计日志
/workspace/search                 → 知识搜索

# settings = 系统管理
/settings/token                   → Token 管理
/settings/system                  → System 健康
/settings/git                     → Git 变更
/settings/templates               → 模板
```

#### 21.5.2 功能模块清单

| 模块 | 路由 | 主要功能 | 关键交互 |
|---|---|---|---|
| 登录 | `/login` | 登录 / 限流 | 表单提交，错误提示 |
| 看板 | `/workspace/inbox` | Inbox 四列拖拽 + 预览/编辑 + 评论 | 非法拖拽拒绝；列有色差；评论不改状态 |
| Proposal 列表 | `/workspace/proposal` | 列出所有 proposal | 状态过滤，时间排序；状态色标 |
| Proposal 详情 | `/workspace/proposal/$id` | **先红绿 diff**，再表单；评论线程 | 终态只读；实时 diff 重算 |
| 访问热度 | `/workspace/access` | 艾宾浩斯热度榜 | 空榜说明文案，不造假条 |
| 审计日志 | `/workspace/audit` | 过滤列表 | client_id / tool / **since–until 区间** |
| 知识搜索 | `/workspace/search` | webUI 直搜 vault | 关键词可选；kind/visibility/tag 单独就能搜；清除清全部条件 |
| Token 管理 | `/settings/token` | 创建/列表/撤销/轮换 | 一次性明文弹窗 + 复制；轮换二次确认；撤销确认即作废、列表不再展示；操作结果用 toast |
| System 健康 | `/settings/system` | 进程/端口/磁盘/SQLite | 进入页开始轮询；呼吸灯；刷新有进行态 |
| Git 变更 | `/settings/git` | workbench HEAD 历史 + diff | 左侧提交树；右侧占满剩余宽度 |
| 模板 | `/settings/templates` | inbox / proposal / token 模板 | CRUD；在对应创建表单里选用 |

#### 21.5.3 知识搜索 Workspace（具体展示形式）

**路由**：`/workspace/search`

后台直扫 `notes` 表，**不过** MCP `knowledge.search` 的 `kind` 门禁。管理员能跨 kind 搜（含 `article` / `project` / `skill` / `mcp` / `context`）。这和 Agent 默认 `["note","article"]` 不是同一套——写明，不要做成「后台也只能搜 note」。

后台搜索和 MCP `knowledge.search` **不是同一套门禁**：

- MCP：`query` 必填；`kind` 只认 note/article（见 §9.4）
- 后台：`q` **不是**必填。`kind` / `visibility` / `tag` 任一有值就可以列出。全空（无关键词也无过滤）才提示「输入关键词或选一个过滤条件」。不要把搜索按钮绑死在 `q.trim()`
- `[清除]` 清关键词 **和** kind / visibility / tag / 排序（排序回到 `score`），并清结果。不是只清输入框

**完整 UI**：

```text
┌─────────────────────────────────────────────────────────────────┐
│  [🔍 输入关键词（可空）________________]  [搜索]  [清除]        │
│                                                                  │
│  过滤器:                                                         │
│   kind:       [全部▾]  (note / article / project / skill / mcp / context) │
│   visibility: [全部▾]  (public / private / secret / draft)               │
│   tag:        [______________]                                  │
│   排序:      [score▾]  (score / recency / access / hot)        │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  结果: 12 条 (耗时 0.18s)                                          │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ 1. Agent Workbase MCP 设计                             │    │
│  │    路径: Workbase/mcps/jiangnan-workbase.md            │    │
│  │    可见性: private  热度: ●●●○○                        │    │
│  │    命中信号: [title 5.0] [fulltext 1.5] [access 1.0]   │    │
│  │    score: 7.5                                            │    │
│  │    摘要: 私密个人 Agent 工作基座。提供上下文、知识...    │    │
│  │    [展开]   [跳到 MCP 详情]                             │    │
│  └────────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ 2. knowledge.search 设计                                │    │
│  │    ...                                                   │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  [加载更多...]                                                   │
└─────────────────────────────────────────────────────────────────┘
```

**[展开]** 后的左右对比视图（红绿 diff）：

```text
┌──────────────────────────┬──────────────────────────┐
│  原文命中片段（左）        │  你的 query（右）          │
├──────────────────────────┼──────────────────────────┤
│  ... 私密个人 [Agent] ... │  Agent Workbase ...      │  ← title 命中（红）
│  ... 提供 [上下文] ...    │                          │
│  ... 知识检索项目状态 ... │  knowledge search        │  ← fulltext 命中（黄）
│  ...                      │                          │
└──────────────────────────┴──────────────────────────┘
```

**0 结果兜底视图**：

```text
┌────────────────────────────────────────────────────────┐
│  ⚠ 未查询到相关内容                                      │
│  query: "Agent Workbase proposal"                      │
│  执行信号: [title] [tags] [frontmatter] [section] [ft] │
│                                                          │
│  建议:                                                   │
│   • 缩短关键词：去掉修饰词（'的'/'一个'/'关于'）          │
│   • 改用更通用的词                                       │
│   • 检查 scope 权限：你的 token scope 是否含 read:knowledge│
│   • 检查 visibility：public 内容只能搜到 public 知识       │
│                                                          │
│  [返回首页]  [改用更通用的词: "MCP"]                      │
└────────────────────────────────────────────────────────┘
```

#### 21.5.4 Diff Viewer（红绿对比）

**位置**：`/workspace/proposal/$id` 详情页的 diff 区

**对比维度**（同 git diff 视觉）：

```text
┌──────────────┬──────────────────────────────────────────────┐
│  原文         │  变更后                                       │
│  （红=删）     │  （绿=增）                                    │
├──────────────┼──────────────────────────────────────────────┤
│  # 工程风格    │  # 工程风格                                   │
│               │  +                                            │
│               │  + ## 5. MCP 集成（新增）                     │
│  ## 4. 其他   │  - ## 4. 其他                                 │
│  ... 旧内容    │  ... 旧内容                                   │
│               │  + ## 6. 后续规划（新增）                     │
└──────────────┴──────────────────────────────────────────────┘
```

**实时重算**：用户编辑 proposal 内容后 → 立即重算 diff → 更新视图（不刷新整页）。

**禁止**：Diff Viewer 不集成到公开博客（博客 = 公开展示层，webUI = 私密审批层）。

#### 21.5.5 审计时间过滤

`GET /api/audit/recent` 的时间参数是 **区间**，不是单个时刻：

| 参数 | 含义 |
|---|---|
| `since` | 下限（含）。空 = 不限起点 |
| `until` | 上限（含）。空 = 不限终点 |
| `tool` / `client_id` / `result_status` | 等值过滤 |

webUI 放两个 **日期时间选择器**（shadcn Calendar + Popover + 时分，§21.5.10），标签写成「从」「到」，**不要**一个无标签的时间控件，**不要**原生 `datetime-local`。值为空就不传该参数。有值才转 RFC3339（按浏览器本地时区）。解析失败 → toast「请输入有效的日期和时间」，**不要**把坏字符串塞给后端再 500。

MCP `audit.list_recent` 继续只用 `since`（Agent 常用「从某时起到现在」）。后台多一个 `until`，不强迫 MCP 改默认。

#### 21.5.6 System 健康

健康页 **进入就开始轮询**，默认 15s，离开页面停。只靠手动刷新 = 人走开就不知道挂了。

| 信号 | 行为 |
|---|---|
| `ok=true` | 状态点呼吸灯（绿，慢闪），文案「健康」 |
| `ok=false` / 请求失败 | 状态点红，停止呼吸，文案「异常」+ 错误 |
| 正在请求 | 刷新按钮 disabled + 「检查中…」；uptime 旁显示本次采样时间 |
| 手动刷新 | 立刻打一次，成功后短暂「已更新」 |

不要让人猜「点了没有」：按钮态、采样时间、灯，三件套都要有。健康检查是观测，不是装饰数字。

#### 21.5.7 Git 变更

左侧是 **提交节点列表**（竖线 + 圆点，线性 HEAD 历史；v0.1 工作台是单线，不画多分支拓扑）。每条：短 hash（墨色、可读）+ subject（正文色）+ author / date（次级，但对比度不能淡到像禁用）。

右侧 **占满剩余宽度**：选中后展示完整 `git show` 元数据 + 红绿 diff，高度跟主栏走，不要在大片空白里塞一条窄预览再自己出滚动条。没选中时右侧空态写「选一条提交看 diff」。

中文路径按 UTF-8 显示，不要把 `文章/` 渲染成 octal escape。

#### 21.5.8 模板用途

模板不是独立玩具。`kind` 三选一，用在对应创建表单：

| kind | 用在哪 | 预填什么 |
|---|---|---|
| `inbox` | 待办看板「新建待办」 | title / content / tags |
| `proposal` | 后台创建 / Agent 可参考的 payload 骨架 | target.type / operation / section / payload / reason |
| `token` | Token 签发表单 | name 建议值 / description / scopes |

创建 Inbox / Token 时表单顶部有「从模板填入」下拉。选中后覆盖空字段；已填的字段不静默覆盖（避免把正在写的内容冲掉）。模板页自己仍做 CRUD。

没有「用模板直接创建一条待办并跳过表单」的隐藏通道——人还是要点一次创建。

#### 21.5.9 阅读层级

后台跟博客共用 token，但管理页必须让人 **扫得动**：

- 页标题、卡片标题、主数字：加粗，用 `ink-1`，不要全员 `text-xs text-ink-3`
- 次级说明才用 `ink-3`
- 必填字段标签带明确标记（「必填」或 `*`），选填写「选填」
- 主按钮（签发 / 批准 / 保存）实心；危险（撤销 / 拒绝 / 废弃）用破坏色 + 确认
- 状态用色标，不靠一段灰字
- 成功 / 失败 / 错误用 **消息弹窗（toast）**，不要做成单页面的页顶错误条。文案写清发生了什么，自动消失；错误可手动关掉。toast 用 sonner，默认 **bottom-right**（在下面，不压左侧导航）；禁止右上角 / 页顶

#### 21.5.10 UI 组件库（不要造轮子）

后台已经能用，但原生 `<input>` / `<select>` / 自写 toast / 自写 modal 看起来像半成品。交互控件用仓库里已经有的 **shadcn/ui**，不要再手写一套，也不要另引 Ant Design / MUI / Naive。

**选 shadcn，不选 Ant Design / MUI**

| 选项 | 结论 |
|---|---|
| shadcn/ui + Radix + Tailwind | **用这个。** 博客 `src/components/ui/` 已经有 Calendar / Dialog / Select / Sonner。admin 是独立 Vite 工程，把需要的组件 **复制进** `server/mcp/admin/src/components/ui/`，自己的 `package.json` 装 radix / sonner / react-day-picker / lucide。不从博客 `src/` 跨包 import（两套构建、两套 token 入口） |
| Ant Design / MUI | **不做。** 自带设计语言，会跟 ink token 打架；后台要明亮专业 skin，不是第二套 Ant 蓝 |
| 自写 toast / datetime-local / confirm() | **不做。** 日期框、弹层、下拉、确认框都有现成封装 |

**交互控件映射**（页面交互用左边，禁止继续用右边）

| 场景 | 用 | 不用 |
|---|---|---|
| 主按钮 / 次按钮 / 危险按钮 | `Button` variant=`default` / `outline` / `destructive` | 手写 `className="rounded-lg bg-primary …"` |
| 单行输入 / 多行 | `Input` / `Textarea` | 裸 `<input>` / `<textarea>`（登录页也换） |
| 下拉（kind / visibility / 模板 / 排序） | `Select` | 原生 `<select>` |
| 多选 scope | `Checkbox` | 原生 checkbox 无 label 样式 |
| 模态（明文 token、新建待办、详情） | `Dialog` | 自写 `modal.tsx` 遮罩 |
| 二次确认（轮换 / 撤销 / 批准 / 拒绝） | `AlertDialog` | `window.confirm` |
| 操作反馈 | **sonner** `toast.success` / `toast.error` | 自写右上角列表；页顶错误条 |
| 审计「从 / 到」 | `DateTimePicker` = Calendar + Popover + 时分 `Input` | 原生 `datetime-local` |
| 状态色标 | `Badge` | 手写 `rounded-full bg-*` 当唯一色标实现（语义色仍按 §16.4 / Proposal 状态） |
| 卡片容器 | `Card` | 可选；看板列可以继续用现有色差容器 |
| 审计表 | `Table` | 手写 `<table>` 无 sticky/hover |
| 图标 | lucide-react | 临时 emoji / 纯 CSS 圆点代替可点控件 |

**toast 位置**：默认 **bottom-right**。在下面，不挡左侧导航，也不弹到右上角。成功 / info 约 3–4s 自动关；错误可手动关。登录页 401/429 仍可表单旁提示（未进 ToastProvider 的会话页）。

**日期时间**：两个独立选择器（从 / 到），不是一个无标签控件。日历选日 + 时分输入；空值不传 query；有值 → 浏览器本地时区 RFC3339。坏值停前端，toast「请输入有效的日期和时间」。API 字段仍是 `since` / `until` 字符串，见 `SCHEMA.md §20.1`。

**仍自写（领域，不是轮子）**

- Inbox 四列看板 + HTML5 拖拽（状态机在契约里，没有现成「不可逆四列」）
- Proposal / Git 红绿 Diff Viewer
- Inbox 轻量 GFM 预览（admin 不引博客那条 remark/rehype/katex 管线）
- Comment 线程布局（形状仍是 SCHEMA §12.4）

**依赖边界**：只给 `server/mcp/admin/package.json` 加 shadcn 运行时（`sonner`、`react-day-picker`、`lucide-react`、对应 `@radix-ui/*`、`class-variance-authority`、`clsx`、`tailwind-merge`）。不要把博客的 `react-markdown` / katex / fuse 打进 admin bundle。不要给 Go 加 UI 依赖。

### 21.6 可扩展性原则

**目标**：后续加新功能 = 加新 route + 加新 API 端点 + 加新后端 module，**不动现有**。

#### 21.6.1 路由命名

```text
{group}/{resource}        路由
{group}    ∈ workspace | settings | future
{resource} ∈ inbox | proposal | access | audit | search | token | system | git | templates | future
```

具体见 §21.5.1 路由表。

#### 21.6.2 API 端点命名

```text
/api/{resource}/{action}              RESTful
/api/{resource}/$id/{action}          针对单条
```

**资源列表**（v0.1）：

| 资源 | 端点 |
|---|---|
| `auth_tokens` | `POST /api/auth_tokens` / `GET /api/auth_tokens`（**不返回** revoked） / `POST /api/auth_tokens/$id/revoke` / `POST /api/auth_tokens/$id/rotate` |
| `inbox` | `POST /api/inbox` / `GET /api/inbox` / `PUT /api/inbox/$id`（更新 content / status / 追加 comment）/ `DELETE /api/inbox/$id` |
| `proposals` | `GET /api/proposals` / `GET /api/proposals/$id` / `PATCH /api/proposals/$id`（pending/conflict 改 payload + 可选 comment）/ `PUT /api/proposals/$id`（approve / reject） |
| `audit` | `GET /api/audit/recent?limit=&since=&until=&tool=&client_id=&result_status=` |
| `system` | `GET /api/system/health` |
| `git` | `GET /api/git/history?limit=` / `GET /api/git/diff/$commit` |
| `knowledge` | `GET /api/knowledge/search?q=&kind=&visibility=&tag=&limit=` / `GET /api/knowledge?id=`。`id` 走 query，因为 `notes.id` 是路径（含 `/`），不能塞进 URL path。kind 含 `article`。后台直扫 notes 表，不过 MCP kind 门禁。`q` 可选，见 §21.5.3 |
| `templates` | `GET /api/templates` / `POST /api/templates` / `POST /api/templates/$id` |

#### 21.6.3 后端 module 划分

```text
server/mcp/internal/
├── auth/                # 鉴权（Bearer + Admin session）
├── audit/               # 审计日志
├── vault/               # vault 读取 / frontmatter
├── index/               # 索引构建 + 访问热度
├── search/              # FTS5 搜索
├── tools/               # MCP tool handlers（含 workbase.identity）
├── proposal/            # proposal schema / 落盘 / preview
├── inbox/               # inbox 待办
├── apply/               # 3-way merge + git commit
├── sanitize/            # 敏感检测
├── manifest/            # workbase.identity 从 vault 读
├── admin/               # admin HTTP API（auth + inbox + proposal + audit + token + system + git + search + templates）
│   ├── auth.go          # session token
│   ├── inbox.go         # /api/inbox/*
│   ├── proposals.go     # /api/proposals/*
│   ├── audit.go         # /api/audit/*
│   ├── tokens.go        # /api/auth_tokens/*
│   ├── system.go        # /api/system/*
│   ├── git.go           # /api/git/*
│   ├── knowledge.go     # /api/knowledge/*
│   └── templates.go     # /api/templates/*
├── merge/               # 3-way merge 工具
└── config/              # config.yaml + SCHEMA.md 加载
```

**依赖最小化**：

- `auth/` → `config/`，`vault/`（仅读 path）
- `audit/` → `auth/`（client_id）
- `tools/` → `auth/`（注入 ctx）+ `audit/`（写审计）+ `proposal/`/`inbox/`（业务）
- `admin/` → `auth/`（session）+ 业务 module（CRUD）

**新加模块流程**：

1. `internal/<name>/` 建新 package
2. `admin/<name>.go` 加 HTTP 路由
3. `admin/src/routes/{group}/{name}.tsx` 加前端页面
4. 在 §21.5.1 路由表 + §21.6.2 API 端点表 加新条目
5. **不动现有任何文件**（除非加新依赖）

---

## 22. 实现建议

### 22.1 技术栈

```text
# MCP Server
Go 1.25+
mcp-go
SQLite FTS5
yaml parser
git merge-file (text 3-way merge)

# Admin WebUI
React 19+
TypeScript 5+
Vite 7+
Tailwind CSS 4+
shadcn/ui（Radix primitives + sonner + react-day-picker）
自写路由（admin 体积小，不引 TanStack Router）
```

### 22.2 目录

```text
server/mcp/
├── cmd/workbase-mcp/
├── internal/
│   ├── config/
│   ├── auth/
│   ├── audit/
│   ├── vault/                # vault 读取、frontmatter、path 规范
│   ├── index/                # indexer + access 热度
│   ├── search/               # FTS 查询 + 权重 + 信号
│   ├── tools/                # MCP tool handlers
│   ├── proposal/             # proposal schema / 落盘 / preview
│   ├── inbox/                # inbox 待办 / 状态流转 / retention
│   ├── apply/                # webUI 审批后 3-way apply + git commit + reindex
│   ├── admin/                # admin HTTP API
│   ├── sanitize/             # 敏感模式检测 + warning（不打码、不拒绝）
│   ├── merge/                # 3-way merge 工具
│   └── manifest/             # workbase.identity 从 vault 读取
├── admin/                    # 独立 TS 项目（§21.5）
├── tests/                    # 验收脚本
└── README.md
```

### 22.3 模块职责

| 模块 | 职责 |
|---|---|
| `config` | 读 config.yaml + 默认值 fallback |
| `auth` | Bearer token hash、scope 校验、admin session |
| `audit` | SQLite 审计日志 + 最小字段集 |
| `vault` | 读 vault 文件、frontmatter、path 规范、kind 分类 |
| `index` | 扫 vault 写 `notes` 单表（`kind` 隔离）+ 访问热度 |
| `search` | FTS 查询 + 权重 + 信号 + 命中门禁 + 排序 |
| `tools` | MCP tool handlers |
| `proposal` | proposal schema / 落盘 / preview / base_commit |
| `inbox` | inbox 待办落盘 / 读取 / 状态流转 / retention |
| `apply` | webUI 审批后 3-way apply + git commit + reindex |
| `admin` | admin HTTP API（auth + inbox + proposal + access） |
| `sanitize` | 敏感模式检测 + warning。读出不打码，写入不拒绝 |
| `merge` | 3-way merge 工具 + frontmatter 冲突检测 |
| `manifest` | workbase.identity 从 vault 读取 |

---

## 23. 验证与验收

### 23.1 验收标准

v0.1 完整验收（**不分版本**）：

```text
 1. 未带 token 请求 MCP 返回 401。
 2. 带只读 token 可调用 workbase.identity / context / search / get。
 3. workbase.identity.workbase 描述性字段从 vault 即时读取（不重启进程）。
 4. 修改 Workbase/mcps/jiangnan-workbase.md 后立即反映到 identity.workbase。
 5. identity.workbase.visibility_policy 从 config.yaml schema.visibility_policy 读取（改 config 后重启生效）。
 6. workbase.identity.auth 反映当前 token 实时状态（status / last_used_at / use_count）。轮换后旧 token 立刻按 grace 规则（默认 0 = 401），identity 不会再报 active。
 7. context.startup 返回当前用户工作风格与活跃项目摘要。secret pack 即使标了 startup 也不进合成。context.get 对 draft 放行，secret → `secret_blocked`。
 8. knowledge.search 默认可检索 `note` + `article`（含 `文章/`）。`scope=all` 含 `draft`。不返回 project / skill / mcp / context / 友链 / secret。已传 kind 过滤后为空 → 空结果，不回落默认。intent / scope 非法值 → `invalid_argument`。
 9. knowledge.search 权重：config 有用 config，config 无用代码默认。
10. knowledge.search 命中门禁与排序信号分离。
11. project.get 可返回项目当前重点、决策、下一步。
12. skill.list / get 从 Workbase/skills/*.md 生成。
13. mcp.list / get 从 Workbase/mcps/*.md 生成。
14. Workbase kind 分类：context / skill / mcp / note / article / project 隔离。knowledge.search 默认只出 note+article。knowledge.get 其它 kind → `note not found`。notes.id = vault 相对路径（正斜杠，含 `.md`）。入库 / 查询一律 ToSlash。
15. proposal.create：可选 `expected_base`；对不上拒绝；不传才用当前 HEAD。
16. proposal 状态机：`pending → approved → applied`（终态）/ `rejected`（终态）/ `approved → conflict → approved`（暂停态）。没有 `pending → conflict`。
17. 同意 → 先在 base 文件上施加 operation 得到完整 ours，再 3-way（base=文件@base_commit, other=文件@HEAD, ours=完整文件）。payload 片段不当 ours。
18. 落盘 + git commit + SHA-256 校验通过 → `applied`。reindex / rebuild 失败仍是 `applied`，单独重跑。
19. 3-way / commit 失败 → `conflict`，proposal 保留，返回冲突区段。救回默认换新 base。
20. frontmatter 内部冲突 → 一律 conflict（不自动合并结构化字段）。
21. 幂等：重复 apply 同一 proposal 返回原 receipt。
22. inbox.append 可新建 pending；inbox.update 可改状态 / 改正文 / 追加评论；inbox.list / get 可读回（get 含 comments，list 只含 comment_count）。敏感命中只 warning，不拒绝。warnings 不落盘，get 每次 rescan。
23. inbox retention_days 可配，config 有用 config，config 无用代码默认。
24. inbox done / abandoned 超过 retention_days 自动删除。
25. webUI：proposal 可审批（同意 / 拒绝 / 编辑）；inbox 可创建 / 预览 / 编辑 / 调整状态。Inbox 与 pending/conflict 的 proposal 有评论线程（人审 ↔ Agent 回复）；评论不改状态。`applied` / `rejected` 只读，不能再 apply。
26. webUI 独立登录页，session token 不用 Basic Auth。
27. 访问热度按艾宾浩斯曲线 score = count * exp(-elapsed/half_life) 排序。只统计 MCP get；后台浏览不加 count。空榜合法。
28. half_life_days 可配，config 有用 config，config 无用代码默认。
29. 审计最小字段集齐全（ts / tool / client_id / scopes / args_digest / result_status / duration_ms）。
30. secret visibility 内容默认不返回（search / list / startup / get）。draft 进 search `scope=all`、各 list、`context.startup`；有对应 scope 就能 get。
31. 审计 / 日志不含 token 原文、token hash、私钥块。授权 Agent 读 Skill / MCP / 文章拿完整原文。公开博客不发布 private / secret。
32. 所有 tool call 有审计记录。
33. sync.ps1 推送后 VPS reindex 可更新 MCP index。
34. MCP apply 后本地 git pull 可同步回该修改。
35. 字段映射：每个工具的请求 / 返回 schema 在 SCHEMA.md 有独立小节。
36. 可见性策略：config.yaml 的 schema 块改后 `systemctl restart` 生效（不热更新，不吃 SIGHUP）。
37. 视觉规范：后台与博客共享同一套 token。
38. 文档分层：设计文档 / SCHEMA.md / config.yaml（schema 块）/ Workbase 职责清晰，不重复。
39. `proposal.update`（SCHEMA §12）：`pending` / `conflict` 可改字段并追加评论；`applied` / `rejected` / `approved` 拒绝。详情页先红绿 diff，再折叠表单。终态只读。
40. Inbox 评论线程（SCHEMA §12.4 共用 Comment）+ 状态不可逆（`reviewing → pending` 非法；终态不能拖回）+ 卡片先预览再编辑。评论不改 status。非法拖拽拒绝并留原列。
41. 后台知识搜索 `q` 可选；`kind` / `visibility` / `tag` 任一有值即可列出；全空才提示。`[清除]` 清全部条件 + 结果。MCP `query` 仍必填。后台 get 不加 `access_count`。
42. 后台审计 `GET /api/audit/recent` 支持 `since`–`until` 区间。MCP `audit.list_recent` 只用 `since`。日期时间用 shadcn Calendar + Popover + 时分，不用 `datetime-local`。转 RFC3339 失败停前端，toast「请输入有效的日期和时间」。
43. Token 列表渲染 `description`（只列 active / grace；revoked 不展示）。签发 / 轮换明文只弹一次 + 复制成功态；轮换二次确认带 name；撤销确认即作废，不要输入 name。成功 / 失败 / 错误用 sonner toast（bottom-right），不用页顶错误条、不用右上角自写列表。
44. System 进页 15s 轮询，离开停止。绿呼吸灯 / 红异常；刷新有「检查中…」+ 采样时间。
45. Git 左侧线性提交树（不画多分支）；右侧占满剩余宽度。中文路径 UTF-8，不要 octal escape。
46. 模板 `kind` 三选一 `inbox` / `proposal` / `token`，只预填空字段，不跳过创建确认。
47. 后台交互控件用 shadcn/ui（Button / Input / Select / Dialog / AlertDialog / Calendar / Table / Badge / sonner）。不引 Ant Design / MUI。不自写 toast 列表、不自写 modal 遮罩、不用 `datetime-local` / `window.confirm`。领域组件（看板拖拽 / Diff / 轻量 Markdown / 评论线程）仍自写。
```

### 23.2 文档验收

```text
1. docs/agent-workbase-mcp-v0.1.md 重写完成。
2. SCHEMA.md 新建完成（含内容格式 + MCP 字段映射 + 审计 + 热度 + 状态机）。20 个工具各有独立小节（§12 = `proposal.update`）。§20.1 日期控件 = Calendar + Popover；§23.2 后台 UI 控件表。
3. README 更新项目定位；`server/mcp/README.md` 工具列表为 20，含 `proposal.update`。
4. 不含真实 VPS IP / 私钥路径。
5. v0.1 不使用向量数据库说明保留。
6. Proposal target / operation schema 保留。
```

---

## 24. Milestones

不分阶段。v0.1 = 完整版。完成后直接转 v1.0 维护期。

---

## 25. 风险与对策

| 风险 | 对策 |
|---|---|
| 私密内容泄露 | visibility + scope + audit（sanitize 只 warning，不挡） |
| token 泄露 | 不入库、hash、独立、可撤销 |
| 硬编码描述绕过 Proposal | §0.1 单一事实源 + 验收 §23.1.3–5 |
| 双主写入冲突 | 3-way merge + frontmatter 冲突即停 |
| Agent 乱写知识库 | typed proposal + adapter + approval |
| Inbox 堆积 | 状态流转 + retention_days 可配 |
| 向量库复杂化 | v0.1 不引入 |
| Registry 双事实源 | Workbase/*.md 作为事实源 |
| startup 与 context 不一致 | startup 仅派生 |
| 公开仓库误提交部署信息 | env + 占位符 |
| 视觉规范分裂 | §0.2 内容统一分发 + §3.9 视觉规范统一 |
| 后台简陋 | §21.5 独立 TS 项目 + §21.5.10 shadcn/ui，不造轮子 |
| 热度算法误判 | 艾宾浩斯曲线 + 实时计算 + 人工可覆盖 |
| 字段映射不同步 | 文档分层 §0.3 + 双改流程 |
| 配置文件缺默认值 | 标准 fallback：cfg → const |

---

## 26. 已确认决策（2026-08-19）

```text
 1. Workbase/ 放在 D:/Data/工作台/ 根下。
 2. inbox 落点：VPS 私有区 /home/studio/workbase/inbox/。
 3. reindex 触发：post-receive 主动触发。
 4. proposal / inbox 只存 VPS；proposal 审批后 3-way apply；inbox 是独立待办。
 5. 不引入向量数据库。
 6. SCHEMA.md 放项目内（D:/Code/Front-end/博客/SCHEMA.md），每工具一张细粒度表。
 7. 字段映射修改 = 同步改 SCHEMA.md + 对应 Go 代码（必须双改）。
 8. 后台 = React + Vite + Tailwind + TS，与博客前端一致。
 9. 后台登录 = 独立 session token，不用浏览器弹窗 Basic Auth。
10. 视觉规范 = 公开博客和后台共享一套设计 token。
11. 内容统一分发 = 同一份 vault，两套扫描（Vite 构建 vs MCP indexer），不是共享同一份 index。
12. 权重 / retention_days / half_life_days 等可配项 = config 可选覆盖 + 代码默认值。
13. 热度算法 = 艾宾浩斯曲线，v0.1 立即实施。
14. base_commit = 可选 expected_base 校验；3-way 用完整 ours，不是 payload 片段；无冲突 auto-apply，有冲突 conflict。
15. frontmatter 内部冲突 = 一律 conflict，不自动合并。
16. 文档分层 = 设计文档 / SCHEMA.md / Workbase 不重复。
17. 不分版本（v0.1 = 完整版）。
18. 不重命名仓库（继续 jiangnan-blog）。
19. 后台 UX：Inbox 评论 + 先预览再编辑 + 四列色差 + 状态不可逆；Proposal 终态只读 + 先 diff 再表单 + `proposal.update`；审计 since–until；后台搜索 q 可选；Token description / 轮换二次确认 / 撤销即从列表消失 / 明文弹窗；操作反馈用 toast 不用页顶错误条；System 15s 轮询 + 呼吸灯；Git 提交树 + 右侧占满；模板 kind=inbox|proposal|token。
20. 后台交互控件 = shadcn/ui（与博客同源）。toast = sonner、默认 bottom-right。日期 = Calendar + Popover + 时分，不用 datetime-local。不引 Ant Design / MUI。领域组件（看板 / Diff / 轻量 Markdown / 评论）仍自写。
```

---

## 27. 参考资料

- MCP Streamable HTTP transport, 2025-11-25 specification
- MCP Authorization specification
- MCP 2025-11-25 release notes
- 参考项目 Personal Agent Foundation
- 参考项目 mnemon（LLM-supervised 记忆层）
- MAGMA（四图记忆模型）

### 27.1 参考 mnemon 的吸收

| # | 吸收点 | 落到 Workbase |
|---|---|---|
| 1 | Signal transparency | `knowledge.search` 返回 `signals` + `matched_via` |
| 2 | Intent-aware retrieval | `knowledge.search` 加 `intent` 参数 |
| 3 | Built-in dedup | Proposal apply 前做内容重叠检测 |
| 4 | Privacy-safe receipts | `audit.list_recent` 含 `args_digest`（不存原文） |
| 5 | Receipt 状态机 + 幂等 | Proposal receipt + replayed 标记 |
| 6 | Named stores 隔离 | 由 scope + project 覆盖 |

不采纳：四图模型（违背 Markdown 事实源）、内嵌 embedding（不引入向量库）、本地 daemon + peer exchange（个人工作台过度）、多 runtime 保姆式 setup（已否决）。
