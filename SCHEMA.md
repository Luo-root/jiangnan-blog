# 遇见江楠 · Workbase SCHEMA

> 位置：`D:/Code/Front-end/博客/SCHEMA.md`（项目内，非工作台）
> 职责：API/数据契约、字段映射、状态机、表结构、算法公式、frontmatter schema
> 与 `docs/agent-workbase-mcp-v0.1.md` 配合：设计文档讲 why / 验收口径，本文件讲 how / 字段定义
> 同一信息只在一处定义

---

## §1. 配置文件 schema

完整 `config.yaml` schema。**可调参数可选**（未配置 = 代码内 const）；**启动必填**只有两处：`schema.visibility_policy`（4 档不能空）和 `admin_auth.user` / `admin_auth.pass_hash`（没账号就起不来）。其余字段走 fallback。结构化定义（枚举 / 状态机 / visibility / 敏感模式）放 `schema` 块；可调参数放顶层 `inbox` / `index` / `knowledge` / `audit`。**一份数字只写一次**。

```yaml
# 进程与端口
server:
  listen: 127.0.0.1:8787

admin:
  listen: 127.0.0.1:8788
  session_ttl: 3600          # session 过期秒，默认 3600
  login_rate_limit: 5        # 每分钟最多失败次数，默认 5

# Vault 路径
vault:
  root: /home/studio/workbench
  git_dir: /home/studio/vault.git

# Workbase（双根）
#   root    = Vault 内 Registry 源（事实源：context/skills/mcps/*.md）
#   runtime = 进程运行时私有区，自动拼接 index/proposals/inbox/audit
workbase:
  root: /home/studio/workbench/Workbase      # Vault 内 Registry 源
  runtime: /home/studio/workbase             # 进程运行时私有区
  rebuild_cmd: /home/studio/workbase/bin/rebuild-blog.sh

# Admin 鉴权（单账号，个人工作台只一个）
admin_auth:
  user: REPLACE_WITH_ADMIN_USER
  pass_hash: REPLACE_WITH_SHA256_HEX_ADMIN    # SHA-256(password)，hex 字符串，不含 sha256: 前缀；无盐。单账号个人台可接受，不是通用口令方案

# Token 灰度（轮换 / 撤销时旧 token 的宽限期），唯一保留的 auth 字段
# Token 主体 = SQLite auth_tokens 表（设计 §6.4），不在 yaml
auth:
  grace_period_hours: 0       # 0 = 无灰度。轮换同步改/删旧 cache；撤销 SLA ≤5s。N = 灰度 N 小时

# 可调参数（顶层直读，缺省 = 代码 const）
inbox:
  retention_days: 7          # 默认 7

index:
  access:
    half_life_days: 7        # 艾宾浩斯半衰期，默认 7
    min_score: 0.001         # 低于此值不参与 Hot 排序，默认 0.001

knowledge:
  search:
    weights:                 # 可选：覆盖默认权重
      title: 5.0
      tags: 4.0
      frontmatter: 3.0
      section: 2.0
      fulltext: 1.5
      wikilink_backref: 2.0
      access: 1.0
      recency: 0.5
    intent_bias:             # 可选：intent 调整
      why:     { frontmatter: 1.3, section: 1.3 }
      when:    { recency: 1.5 }
      entity:  { tags: 1.3 }
      general: {}

audit:
  retention_days: 90         # 审计日志保留天数
  recent_limit: 100          # audit.list_recent 默认返回条数

# schema 块：枚举 / 状态机 / visibility / 敏感模式（结构化，不重复可调参数）
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
    default:            private

  # 敏感模式。默认 [] = 关闭（不检测、不拒绝、不打码）。
  # 个人工作台：Skill / MCP / 文章要完整给到授权 Agent，别默认开。
  # 要开：往列表里加 regex。行为见 §21。
  sensitive_patterns: []

  # 审计最小字段集
  audit_min_fields: [ts, tool, client_id, scopes, args_digest, result_status, duration_ms]

  # 审计 result_status 取值
  audit_result_status: [success, error, unauthorized, forbidden]

  # Proposal 状态机（§17）—— conflict 可救回到 approved
  proposal_states: [pending, approved, applied, rejected, conflict]
  proposal_state_transitions:
    pending:  [approved, rejected]   # 创建校验失败是控制层拒绝，不写 receipt，不进 conflict
    approved: [applied, conflict]    # 3-way / commit 只在 approved 之后发生
    applied:  []
    rejected: []
    conflict: [approved]

  # Proposal target / operation 类型
  proposal_target_types:    [note, context_pack, project, article, skill, mcp_server]
  proposal_operation_types: [create_file, append, append_section, patch_section, register_item]   # 枚举 = 当前能发的 = §15.7 矩阵

  # Inbox 状态机（§17.2）—— pending 可直接 done / abandoned
  inbox_states: [pending, reviewing, done, abandoned]
  inbox_state_transitions:
    pending:   [reviewing, done, abandoned]
    reviewing: [done, abandoned]
    done:      []
    abandoned: []

  # 自动生成 ID 前缀。只给 proposal / inbox。
  # notes.id = vault 相对路径（正斜杠，含 .md），不用前缀。skill / mcp / context 对外用 frontmatter id。
  id_prefixes:
    proposal:     "prop"
    inbox:        "inbox"
```

**Fallback 原则**：cfg 有值用 cfg，没值用代码内 const 默认值。代码内 const 在 `server/mcp/internal/config/defaults.go`。

**Token 永远不在 yaml**——所有 Agent Token 都在 SQLite `auth_tokens` 表，webUI 自助管理（设计 §6.4）。yaml 里只留 `auth.grace_period_hours`（轮换 / 撤销灰度时长）。

---

## §2. 权限 scope 列表

### §2.1 scope 定义

标准 scope（8 个，可签给 Agent Token）：

```text
read:context        # 读取 context.startup / context.get
read:knowledge      # 搜索和读取 note
read:project        # 读取 project list / get
read:registry       # 读取 skill / mcp registry
read:inbox          # 读取 inbox 待办
write:proposal      # 创建 / 读取 proposal
write:inbox         # 追加 / 更新 inbox 待办
ops:audit           # 查看审计摘要
```

`admin:reindex` **不是** Agent scope。Token 创建表单不展示、不签发。重建索引只走内部 HTTP `POST /internal/reindex`，和 MCP 协议同进程、mux 分路（见设计 §18.3 / §21.2）。

保护不靠 `RemoteAddr == 127.0.0.1`（Caddy 反代会改掉对端地址），靠三件事：

1. Caddy **不转发** `/internal*`（公开 `mcp.` 反代排除这条路径）
2. 进程 bind `127.0.0.1:8787`（公网网卡听不到）
3. hook / apply 副作用用本机 `http://127.0.0.1:8787/internal/reindex` 直打，**不带 Bearer**

事实源：`server/mcp/internal/tools/tools.go` 的 `toolScopes` map。与本表**双向校对**。

| Tool | Required scope | v0.1 风险 |
|---|---|---|
| `workbase.identity` | （任意有效 token） | low |
| `context.startup` | `read:context` | medium |
| `context.get` | `read:context` | medium |
| `knowledge.search` | `read:knowledge` | medium |
| `knowledge.get` | `read:knowledge` | high（可能返回私密正文） |
| `project.list` | `read:project` | medium |
| `project.get` | `read:project` | high（可能返回项目状态） |
| `skill.list` | `read:registry` | low / medium |
| `skill.get` | `read:registry` | medium |
| `mcp.list` | `read:registry` | medium |
| `mcp.get` | `read:registry` | medium / high（可能含私密 endpoint） |
| `proposal.create` | `write:proposal` | medium / high |
| `proposal.list` | `write:proposal` | medium |
| `proposal.get` | `write:proposal` | medium / high |
| `inbox.append` | `write:inbox` | low |
| `inbox.update` | `write:inbox` | low |
| `inbox.list` | `read:inbox` | low |
| `inbox.get` | `read:inbox` | low |
| `audit.list_recent` | `ops:audit` | medium |

---

## §3. 可见性策略

### §3.1 visibility 取值

| visibility | 公开博客 | 公共 Agent Index | 私密 MCP | 说明 |
|---|---:|---:|---:|---|
| `public` | yes | yes | yes | 可公开展示 |
| `private` | no | no | yes | 授权 Agent 可读 |
| `secret` | no | no | no by default | 默认不暴露给远程 MCP |
| `draft` | no | no | yes（有对应 scope） | 草稿。search `scope=all` 收录；各 list / `context.startup` 照收；对应 get 放行。Vite 跳过。secret 仍永不进 search / list / startup |

### §3.2 缺省规则

按文件所在一级目录查表：

| 一级目录 | 默认 visibility |
|---|---|
| `文章/` | `public` |
| `项目/` | `public` |
| `友链/` | `public` |
| `部署溯源/` | `private` |
| `Workbase/context/` | `private` |
| `Workbase/skills/` | `private` |
| `Workbase/mcps/` | `private` |
| 其它（普通笔记） | `private` |

`secret` **必须**显式标注，不接受缺省。

**两个独立开关，不要混用**：

| 开关 | 听谁 | 效果 |
|---|---|---|
| `draft: true`（frontmatter 布尔） | 公开博客构建（Vite） | 不发布到公开站点 |
| `visibility: draft` | MCP 可见性策略 | 授权 MCP 可读（search `scope=all` + 各 list + `context.startup` + get 放行）；公开博客不收录 |

博客构建跳过条件 = `draft: true` **或** `visibility: draft`。MCP 只认 `visibility`，不认 `draft:` 布尔。缺 `visibility` 时按本表一级目录缺省；缺 `draft:` 布尔 = 非草稿。

### §3.3 缺省规则的代码实现位置

`server/mcp/internal/vault/visibility.go` 的 `defaultVisibility(path string) string`。**运行时权威 = `config.yaml` 的 `schema.visibility_default`**（Go 启动时 `LoadConfig()` 加载到 `cfg.Schema.VisibilityDefault`，运行时**不**再读盘）。本表 (§3.2) 是**给人看的说明**，不是 Go 解析输入——改本表后**必须**同步改 `config.yaml`。

---

## §4. workbase.identity 字段映射

合并原 `workbase.manifest` + `workbase.whoami`。一次调用拿到 Workbase 元数据 + 当前 token 元数据。

### §4.1 响应字段

```json
{
  "workbase": {
    "id": "jiangnan-workbase",
    "name": "...",
    "version": "0.1.0",
    "description": "...",
    "capabilities": {...},
    "tools": [...],
    "visibility_policy": {...},
    "getting_started": "...",
    "critical_rules": [...],
    "see_also": [...]
  },
  "auth": {
    "client_id": "minimax-code",
    "scopes": [...],
    "status": "active",
    "created_at": "...",
    "last_used_at": "...",
    "use_count": 1234,
    "allowed_tools": [...]
  }
}
```

### §4.2 workbase 块（Workbase 元数据）

| 字段 | 类型 | 数据源 | 必含 | 说明 |
|---|---|---|---|---|
| `id` | string | Go 常量 | yes | MCP 协议层标识符 = `jiangnan-workbase` |
| `name` | string | vault | yes | `Workbase/mcps/jiangnan-workbase.md` frontmatter `name` |
| `version` | string | Go 常量 | yes | MCP 协议层版本 = `0.1.0` |
| `description` | string | vault | yes | 同上 `summary` |
| `capabilities` | object | runtime | yes | 见 §4.3 |
| `tools` | []string | runtime | yes | 排序后的 toolScopes keys |
| `visibility_policy` | object | config | yes | 见 §4.4；来自 `config.yaml` `schema.visibility_policy` |
| `getting_started` | string | vault | yes | `Workbase/mcps/jiangnan-workbase.md` `## Purpose` |
| `critical_rules` | []string | vault | yes | 同上 `## Security` 段 `-` 列表 |
| `see_also` | []string | vault | no | 同上 `## Source` 段链接 |

**读取策略**：

- 每次调用即时读磁盘，**不缓存**
- 任一文件读取/解析失败 → 返回 MCP error，**不**静默 fallback 到硬编码

### §4.3 capabilities

| key | true 条件 |
|---|---|
| `context` | `context.startup` 工具已注册 |
| `knowledge` | `knowledge.search` 工具已注册 |
| `project` | `project.list` 工具已注册 |
| `skill_registry` | `skill.list` 工具已注册 |
| `mcp_registry` | `mcp.list` 工具已注册 |
| `proposal` | `proposal.create` 工具已注册 |
| `inbox` | `inbox.append` 工具已注册 |
| `direct_write` | **始终 false**（v0.1 不允许 Agent 绕过 proposal 写入） |
| `vector_search` | **始终 false**（不引入向量库） |

### §4.4 visibility_policy 格式

```json
{
  "public": "可公开展示与索引",
  "private": "授权 Agent 可读",
  "secret": "默认不暴露给远程 MCP",
  "draft": "草稿。授权 MCP 可读；search scope=all / 各 list / context.startup 收录；公开博客不发布"
}
```

4 行字符串来自 `config.yaml` `schema.visibility_policy`。这是结构化配置，不进 Vault。

### §4.5 auth 块（当前 token 元数据）

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `client_id` | string | yes | token 名称（即 `auth_tokens.name`） |
| `scopes` | []string | yes | 该 token 拥有的 scope 列表 |
| `status` | enum | yes | `active` / `grace`。`revoked` 过不了 middleware（401），identity **不会**返回 `revoked` |
| `created_at` | RFC3339 | yes | token 创建时间 |
| `last_used_at` | RFC3339 | no | 最近一次使用。从未用过（表字段 NULL）→ **省略或 `null`**，不要空串 / 零时间 |
| `use_count` | int | yes | 累计使用次数。新 token = 0。本次调用的 +1 是认证后异步写，本响应可能还是旧值 |
| `allowed_tools` | []string | yes | 由 `scopes` × `toolScopes` 推导出的可调用工具列表 |

**不返回**（敏感信息）：

- token 原文
- token hash
- `grace_until`
- 撤销者 / 创建者真实身份

### §4.6 错误返回

任一 vault 文件读取/解析失败 → 返回 MCP `error`，**不**静默 fallback 到硬编码字符串。

### §4.7 scope 要求

**任意有效 token** 即可调（不要求额外 scope）。`toolScopes` 中 `workbase.identity` 的 `RequiredScope` 设为空字符串。

### §4.8 Agent 典型用法

```text
1. Agent 连上 MCP（initialize）
2. 调 workbase.identity → 知道 Workbase 是什么 + 知道 token 能干啥
3. 按 auth.allowed_tools 调用对应工具
4. 遇到 401 / 403 时再调 workbase.identity 看 scope 是否变化
```

### §4.9 关系说明

`workbase.tools` ⊇ `auth.allowed_tools`：

- `workbase.tools` = Workbase 提供的所有工具（全集）
- `auth.allowed_tools` = 当前 token 可调的工具（子集）
- 关系：`auth.allowed_tools ⊆ workbase.tools`


## §5. context.startup 字段映射

### §5.1 响应字段

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `packs` | []object | yes | 包含的 context pack 列表 |
| `packs[].id` | string | yes | context pack id |
| `packs[].title` | string | yes | frontmatter `title` |
| `packs[].priority` | string | yes | `high` / `medium` / `low` |
| `content` | string | yes | 合成后的 Markdown 正文（去 frontmatter + 按 priority 排序拼接） |

### §5.2 来源与合成

- 数据源：`Workbase/context/*.md` 中 `startup: true` **且** `visibility ≠ secret` 的 pack
- **secret 永不进合成**（即使标了 `startup: true`）。draft 照收（public / private / draft）
- 排序：按 frontmatter `priority` 降序（high > medium > low）
- 正文拼接：每个 pack 去掉 frontmatter，保留 Markdown 正文，按顺序用 `\n\n` 拼接
- 不缓存：每次调用即时合成

### §5.3 错误

无合格 pack（没有 `startup: true`，或全部是 secret）→ `packs=[]`，`content=""`（不报错）。文件读取失败 → 跳过该 pack，不影响其他。

---

## §6. context.get 字段映射

### §6.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | context pack id（`profile` / `engineering-style` 等） |

### §6.2 响应

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | 与请求一致 |
| `title` | string | yes | frontmatter `title` |
| `visibility` | string | yes | frontmatter `visibility` |
| `updated_at` | RFC3339 | yes | frontmatter `updated` |
| `content` | string | yes | 去掉 frontmatter 后的正文 |
| `metadata` | object | no | 透传 frontmatter 其它字段 |

### §6.3 错误

| 情况 | 错误 |
|---|---|
| 缺少 `id` | `required argument id missing` |
| id 不存在 | `context_pack not found: <id>` |
| `visibility=secret` | 默认返回 `secret_blocked` 错误，**不**返回内容 |
| `visibility=draft` | **放行**（有 `read:context` 即可） |

规则与 `project.get` / `skill.get` / `mcp.get` 同一套：list / startup 出 public + private + draft，不出 secret；get 对 draft 放行，secret 默认 `secret_blocked`。

---

## §7. knowledge.search 字段映射

### §7.1 请求

| 字段 | 类型 | 必含 | 默认 | 说明 |
|---|---|---|---|---|
| `query` | string | yes | - | 搜索关键词 |
| `intent` | enum | no | `general` | `why` / `when` / `entity` / `general`。非法值 → `invalid_argument`，**不**回落 `general` |
| `limit` | int | no | 10 | 返回条数 |
| `scope` | enum | no | `all` | `all` / `public` / `private`。`all` = public + private + **draft**。**secret 永远不进 search**（含摘要）。secret 只走 `knowledge.get` 显式 id，且默认 `secret_blocked`。非法值 → `invalid_argument`，**不**回落 `all` |
| `kind` | []string | no | `["note","article"]` | 只认 `note` / `article`。缺省才用默认。已传则只保留这两个，其它值丢掉；**过滤后为空 → 空结果，不回落默认**。不能用来搜 project / skill / mcp / context |

### §7.2 响应（results 元素）

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | = `notes.id` = vault 相对路径（正斜杠，含 `.md`，如 `文章/foo.md`） |
| `title` | string | yes | note 标题 |
| `path_hint` | string | yes | 与 `id` 相同（vault 相对路径） |
| `kind` | string | yes | 命中条目的真实 kind：`note` 或 `article`。**不**固定写成 `note` |
| `visibility` | string | yes | `public` / `private` / `draft`（secret 不进 search） |
| `summary` | string | yes | frontmatter `summary` 或首段 |
| `matched_fields` | []string | yes | 命中的字段列表 |
| `score` | float | yes | 总分（绝对分制，不归一化；理论无上限） |
| `matched_via` | string | yes | 信号组合描述 |
| `signals` | object | yes | 各信号分项得分 |

### §7.3 signals 字段

```text
title            float  命中 title 权重
tags             float  命中 tags 权重
frontmatter      float  命中 frontmatter 字段权重
section          float  命中 heading 段权重
fulltext         float  正文 FTS5 命中权重
wikilink_backref float  WikiLink 反向引用次数权重
access           float  艾宾浩斯热度权重
recency          float  时间衰减权重
```

### §7.4 命中门禁与排序

- **门禁**：至少一个 `title` / `tags` / `frontmatter` / `section` / `fulltext` 命中
- **排序**：所有 signals 加权求和，按 `score` 降序

### §7.5 intent 调整

`intent` 调整的事实源 = `config.yaml` 的 `knowledge.search.intent_bias`（绝对倍率，不归一化）。yaml 是权威，**本表只描述**每条 intent 的偏向：

| intent | 权重偏向（yaml 事实源） |
|---|---|
| `why` | `frontmatter` × 1.3，`section` × 1.3 |
| `when` | `recency` × 1.5 |
| `entity` | `tags` × 1.3 |
| `general` | 无调整（`{}`） |

### §7.6 权重配置

详见 §1。代码内默认权重在 `server/mcp/internal/search/weights.go`。

### §7.7 参数校验（不静默回落）

| 参数 | 缺省 / 未传 | 已传但非法 / 过滤后为空 |
|---|---|---|
| `kind` | `["note","article"]` | 只保留 `note` / `article`。过滤后为空（含显式 `[]`、只传了 `project` 等）→ `results=[]` + 空结果兜底文案，**不**回落默认，**不**报错。想搜 project 走 `project.list` |
| `intent` | `general` | 不在 `why` / `when` / `entity` / `general` → `invalid_argument`，**不**回落 `general` |
| `scope` | `all` | 不在 `all` / `public` / `private` → `invalid_argument`，**不**回落 `all` |

`kind=["article","project"]` → 丢掉 `project`，搜 `article`。`kind=["project"]` → 空结果，不是默认的 note+article。Agent 以为带了 project 却搜到文章，是假结果。

---

## §8. knowledge.get 字段映射

### §8.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | = `notes.id` = vault 相对路径（正斜杠，含 `.md`，如 `文章/foo.md`）。请求里的 `\` 先 `filepath.ToSlash` 再查 |

### §8.2 响应

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | = `notes.id` = vault 相对路径（正斜杠，含 `.md`） |
| `title` | string | yes | note 标题 |
| `path_hint` | string | yes | 与 `id` 相同 |
| `kind` | string | yes | `note` 或 `article` |
| `visibility` | string | yes | `public` / `private` / `secret` / `draft` |
| `updated_at` | RFC3339 | yes | frontmatter `updated` 或文件 mtime |
| `body` | string | yes | Markdown 正文（去 frontmatter） |
| `frontmatter` | object | yes | 解析后的 frontmatter（key-value） |
| `forward_links` | []object | yes | 链接出去的目标列表 |
| `backlinks` | []object | yes | 反向链接列表 |
| `base_commit` | string | yes | 当前 vault HEAD commit hash |

### §8.3 错误

| 情况 | 错误 |
|---|---|
| 缺少 `id` | `required argument id missing` |
| id 不存在 | `note not found: <id>` |
| `notes.kind` 不是 `note` / `article` | `note not found: <id>`（不泄露实际是 skill / mcp / project） |
| `visibility=secret` | 默认返回 `secret_blocked` 错误，**不**返回内容 |
| `visibility=draft` | **放行**（有 `read:knowledge` 即可） |

`knowledge.get` 只返回 `kind ∈ {note, article}`。Agent 拿 skill 的 frontmatter `id`（如 `markdown-lint`）来调本工具 → `note not found`，必须走 `skill.get`。

### §8.4 forward_links / backlinks 元素

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 目标 `notes.id`（vault 相对路径） |
| `title` | string | 目标 note 标题 |
| `path_hint` | string | 与 `id` 相同 |
| `context` | string | 链接出现的上下文（前后 50 字） |

---

## §9. project.list / project.get 字段映射

### §9.1 事实源

`项目/*.md`。frontmatter schema：

```yaml
---
name: 项目名                  # 必填
summary: 一句话简介            # 必填
links:                        # 可选
  - type: repo | demo | video | docs | site | other
    url: https://...
    label: 可选
stack:                        # 可选
  - Go
  - TypeScript
status: 维护中 | 进行中 | 已归档   # 可选
cover: 封面.png                # 可选
date: 2026-07-04              # 可选
visibility: public | private | secret | draft   # 可选
---
```

### §9.2 project.list 响应元素

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | 文件名 slug |
| `name` | string | yes | frontmatter `name` |
| `summary` | string | yes | frontmatter `summary` |
| `status` | string | no | frontmatter `status` |
| `stack` | []string | no | frontmatter `stack` |
| `links` | []object | no | frontmatter `links` |
| `date` | date | no | frontmatter `date` |
| `path_hint` | string | yes | `项目/<slug>.md` |
| `visibility` | string | yes | `public` / `private` / `draft`。list **不出** secret |

list 默认出 public + private + draft。secret 不进 list。`project.get` 对 draft 放行（有 `read:project` 即可）；secret 默认 `secret_blocked`。

### §9.3 project.get 额外返回

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `body` | string | yes | Markdown 正文（去 frontmatter） |
| `current_focus` | string | no | 从正文 `## 当前重点` 段提取 |
| `next_steps` | []string | no | 从正文 `## 下一步` 段 `-` 列表 |
| `decisions` | []string | no | 从正文 `## 决策` 段 `-` 列表 |
| `frontmatter` | object | yes | 完整 frontmatter |

---

## §10. skill.list / skill.get 字段映射

### §10.1 事实源

`Workbase/skills/*.md`。frontmatter schema：

```yaml
---
id: markdown-lint             # 必填
kind: skill                   # 必填
name: Markdown Lint           # 必填
summary: 一句话简介            # 必填
visibility: public | private | secret | draft
risk: low | medium | high
tags:                         # 可选
  - markdown
  - lint
source:                       # 可选
  type: github
  url: https://...
license: unknown
updated: 2026-08-18
---
```

### §10.2 skill.list 响应元素

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | frontmatter `id` |
| `name` | string | yes | frontmatter `name` |
| `summary` | string | yes | frontmatter `summary` |
| `risk` | string | no | frontmatter `risk` |
| `tags` | []string | no | frontmatter `tags` |
| `source` | object | no | frontmatter `source` |
| `visibility` | string | yes | frontmatter `visibility` |
| `path_hint` | string | yes | `Workbase/skills/<slug>.md` |

list 默认出 public + private + draft。secret 不进 list。`skill.get` 对 draft 放行（有 `read:registry` 即可）；secret 默认 `secret_blocked`。

### §10.3 skill.get 额外返回

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `body` | string | yes | Markdown 正文（去 frontmatter） |
| `frontmatter` | object | yes | 完整 frontmatter |
| `license` | string | no | frontmatter `license` |
| `updated_at` | RFC3339 | no | frontmatter `updated` |

---

## §11. mcp.list / mcp.get 字段映射

### §11.1 事实源

`Workbase/mcps/*.md`。frontmatter schema：

```yaml
---
id: jiangnan-workbase         # 必填
kind: mcp_server              # 必填
name: MCP 名称                # 必填
summary: 一句话简介            # 必填
visibility: public | private | secret | draft
risk: personal-knowledge-base | browser-control | ...
transport: streamable-http | stdio | sse
endpoint: https://...         # 私密 MCP 必填，stdio 类型不需要
auth:
  type: bearer | oauth | none | api_key
  scopes: []                  # bearer/oauth 才有
source:                       # 可选
  type: github
  url: https://...
updated: 2026-08-18
---
```

### §11.2 mcp.list 响应元素

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | frontmatter `id` |
| `name` | string | yes | frontmatter `name` |
| `summary` | string | yes | frontmatter `summary` |
| `transport` | string | yes | frontmatter `transport` |
| `auth_type` | string | yes | frontmatter `auth.type` |
| `risk` | string | no | frontmatter `risk` |
| `source` | object | no | frontmatter `source` |
| `visibility` | string | yes | frontmatter `visibility` |
| `path_hint` | string | yes | `Workbase/mcps/<slug>.md` |

list 默认出 public + private + draft。secret 不进 list。`mcp.get` 对 draft 放行（有 `read:registry` 即可）；secret 默认 `secret_blocked`。

### §11.3 mcp.get 额外返回

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `endpoint` | string | no | frontmatter `endpoint` |
| `auth` | object | yes | frontmatter `auth` 完整 |
| `body` | string | yes | Markdown 正文（去 frontmatter） |
| `frontmatter` | object | yes | 完整 frontmatter |
| `updated_at` | RFC3339 | no | frontmatter `updated` |

### §11.4 mcp.get 安全规则

授权 Agent 读 `mcp.get` / `skill.get` 拿**完整原文**（含 endpoint、配置说明）。**不**因看起来像 IP / password 字段就裁切。真正的密钥不该写进 vault md——写了也原样返回，靠 scope + visibility 挡未授权访问，不靠读出打码。

---

## §13. proposal.create 字段映射

### §13.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `target.type` | string | yes | `note` / `context_pack` / `project` / `article` / `skill` / `mcp_server`。枚举 = 当前能发的 = 设计 §15.7 矩阵 |
| `expected_base` | string | no | Agent 读到的 vault HEAD（通常来自 `knowledge.get.base_commit`）。有传且 ≠ 当前 HEAD → 拒绝，提示「你读的已经不是最新」。不传 = 用当前 HEAD |
| `target.id` | string | no | 已存在条目的对外 id。`note` / `article` = `notes.id`（正斜杠路径，含 `.md`；`\` 先 ToSlash）；`project` = 文件名 slug；`skill` / `mcp_server` / `context_pack` = frontmatter `id`。有 `target.path` 时以 path 为准 |
| `target.path` | string | yes | vault 内相对路径。入库 / 校验前一律 `filepath.ToSlash`（`\` → `/`）。必须落在 `vault.root` 下（防 `../`）；**不**要求文件已存在 |
| `operation.type` | string | yes | `create_file` / `append` / `append_section` / `patch_section` / `register_item` |
| `operation.section` | string | no | `append_section` / `patch_section` 必填 |
| `payload.format` | string | yes | 固定 `markdown` |
| `payload.content` | string | yes | 内容（`create_file` 是完整内容；`append` 是追加内容；`append_section` / `patch_section` 是段内容；`register_item` 是 frontmatter + 模板正文） |
| `reason` | string | no | 创建原因，审计用 |
| `risk.level` | string | no | `low` / `medium` / `high` |
| `risk.reasons` | []string | no | 风险原因 |

**不要传**：`kind`（服务端从 `target.type` 抄到落盘 / 响应）；`validation.checks`（检查名单服务端写死，Agent 少传一项不能跳过）。

**`base_commit` 写入规则**：客户端不直接写 `base_commit`。有 `expected_base` 且等于当前 HEAD → 存这个值；有传但对不上 → 拒绝，不创建；不传 → 服务端读当前 HEAD。这样 Agent 基于 A 写的补丁不会被记成 B。

### §13.2 响应

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | 生成的 proposal id，格式 `prop_YYYYMMDD_NNN` |
| `status` | string | yes | 初始 `pending` |
| `base_commit` | string | yes | 服务端自动记录的 vault HEAD |
| `created_at` | RFC3339 | yes | 创建时间 |
| `created_by` | string | yes | 客户端 id（从 token 解出） |
| `diff` | string | yes | preview / diff 文本 |
| `validation` | object | yes | 各项校验结果 |

### §13.3 错误

| 情况 | 错误 |
|---|---|
| 缺 `target.type` 或 `operation.type` | `required field missing` |
| `target.path` 解析后不在 `vault.root` 下（含 `../`） | `target_path_invalid` |
| `create_file` / `register_item` 且文件已存在 | `target_already_exists` |
| `append` / `append_section` / `patch_section` 且文件不存在 | `target_not_found` |
| `patch_section` 且标题不存在 | `section_not_found`（create 时也可预检；apply 时标 conflict） |
| `target.type` + `operation.type` 不在枚举 | `operation_not_supported` |
| `expected_base` ≠ 当前 HEAD | `stale_base`（提示「你读的已经不是最新」，附当前 HEAD） |
| `target.visibility=secret` + 试图写入 | `visibility_not_writable` |
| Markdown fence 不闭合 | `invalid_markdown_fence` |

敏感模式**默认关**。开了也只警告，不因命中 regex 拒绝创建（见 §21）。

### §13.4 完整示例

```json
{
  "expected_base": "abc123def456",
  "target": {
    "type": "note",
    "path": "部署溯源/jiangnan-workbase.md"
  },
  "operation": {
    "type": "append_section",
    "section": "Agent Workbase MCP"
  },
  "payload": {
    "format": "markdown",
    "content": "v0.1 实施完成。"
  },
  "reason": "记录实施完成",
  "risk": {
    "level": "medium",
    "reasons": ["修改长期项目上下文"]
  }
}
```

---

## §14. proposal.list 字段映射

### §14.1 请求

| 字段 | 类型 | 必含 | 默认 | 说明 |
|---|---|---|---|---|
| `status` | enum | no | all | `pending` / `approved` / `applied` / `rejected` / `conflict` / `all` |
| `created_by` | string | no | - | 客户端 id 过滤 |
| `since` | RFC3339 | no | - | 创建时间下限 |
| `limit` | int | no | 50 | 返回条数 |

### §14.2 响应元素

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | proposal id |
| `kind` | string | yes | 服务端从 `target.type` 抄，不是客户端传的 |
| `status` | string | yes | 当前状态 |
| `target_path` | string | yes | 目标文件路径 |
| `created_at` | RFC3339 | yes | 创建时间 |
| `created_by` | string | yes | 客户端 id |
| `reason` | string | no | 创建原因 |
| `risk_level` | string | no | 风险等级 |

---

## §15. proposal.get 字段映射

### §15.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | proposal id |

### §15.2 响应

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | proposal id |
| `kind` | string | yes | 服务端从 `target.type` 抄，不是客户端传的 |
| `status` | string | yes | 当前状态 |
| `base_commit` | string | yes | 创建时的 vault HEAD |
| `created` | object | yes | `{by, at, reason}` |
| `risk` | object | no | `{level, reasons}` |
| `target` | object | yes | 与请求一致 |
| `operation` | object | yes | 与请求一致 |
| `payload` | object | yes | 与请求一致 |
| `validation` | object | yes | 校验结果 |
| `diff` | string | yes | preview / diff 文本 |
| `receipt` | object | no | apply 后才有 |

### §15.3 receipt 字段

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `status` | string | yes | `applied` / `rejected` / `conflict` |
| `applied_at` | RFC3339 | no | 仅 `applied` |
| `commit` | string | no | 仅 `applied`：apply 后的 git commit hash |
| `content_sha256` | string | no | 仅 `applied`：目标文件 apply 后内容哈希 |
| `base_commit` | string | yes | 本次 apply 使用的祖先。conflict 救回默认换成当前 HEAD |
| `merge_strategy` | enum | no | `none` / `three_way` |
| `replayed` | bool | yes | 幂等标记 |
| `conflict_regions` | []object | no | 仅 `conflict`：冲突区段列表 |

---

## §16. inbox.append 字段映射

### §16.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `title` | string | no | 简短标题（列表展示用） |
| `content` | string | yes | Markdown 正文 |
| `tags` | []string | no | 标签 |

### §16.2 响应

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | inbox id，格式 `inbox_YYYYMMDD_HHMMSS_fff`。落盘文件名 = `{id}.md`（与 id 同一套，不是 ISO 时间戳另起一名） |
| `status` | string | yes | 初始 `pending` |
| `created_at` | RFC3339 | yes | 创建时间 |
| `created_by` | string | yes | 客户端 id |
| `title` | string | no | 透传 |
| `content` | string | yes | 透传 |
| `tags` | []string | no | 透传 |
| `warnings` | []string | no | 运行时扫描 `content` 的结果。敏感模式开启且命中时有值。**不写入 `{id}.md`**，每次读再扫。默认关则省略。**不拒绝** |

### §16.3 错误

| 情况 | 错误 |
|---|---|
| 缺 `content` | `required field content missing` |

敏感模式命中不是错误。开了也只在响应 `warnings` 记一条，照样创建（与 proposal / §21.1 同一套）。`warnings` **不**写进 `{id}.md`。

---

## §17. inbox.update 字段映射

### §17.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | inbox id |
| `title` | string | no | 修改标题 |
| `content` | string | no | 修改正文 |
| `tags` | []string | no | 修改标签 |
| `status` | enum | no | 状态变更：`pending` / `reviewing` / `done` / `abandoned` |

### §17.2 状态机

```text
pending → reviewing → done
                  ↘   abandoned
pending      → done | abandoned
```

不允许：`done` / `abandoned` → 任何状态（终态）。

### §17.3 响应

返回更新后的完整 inbox 条目（与 `inbox.get` 响应相同 schema）。

### §17.4 错误

| 情况 | 错误 |
|---|---|
| id 不存在 | `inbox not found: <id>` |
| 状态转换非法 | `invalid status transition: <from> → <to>` |

敏感模式命中不是错误。开了也只在响应 `warnings` 记一条，照样更新。`warnings` **不**写进 `{id}.md`，下次读再扫。

---

## §18. inbox.list 字段映射

### §18.1 请求

| 字段 | 类型 | 必含 | 默认 | 说明 |
|---|---|---|---|---|
| `status` | enum | no | all | `pending` / `reviewing` / `done` / `abandoned` / `all` |
| `created_by` | string | no | - | 客户端 id 过滤 |
| `since` | RFC3339 | no | - | 创建时间下限 |
| `limit` | int | no | 50 | 返回条数 |

### §18.2 响应元素

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | inbox id |
| `status` | string | yes | 当前状态 |
| `title` | string | no | 标题 |
| `summary` | string | yes | content 首行或前 80 字 |
| `created_at` | RFC3339 | yes | 创建时间 |
| `created_by` | string | yes | 客户端 id |
| `tags` | []string | no | 标签 |

列表**不**返回 `warnings`（看板只要摘要）。敏感命中看 `inbox.get` / `inbox.append` / `inbox.update`。

`done` / `abandoned` 超过 `retention_days` 不返回（已自动删除）。

---

## §19. inbox.get 字段映射

### §19.1 请求

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | inbox id |

### §19.2 响应

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `id` | string | yes | inbox id |
| `status` | string | yes | 当前状态 |
| `created` | object | yes | `{by, at}` |
| `title` | string | no | 标题 |
| `content` | string | yes | Markdown 正文 |
| `tags` | []string | no | 标签 |
| `updated_at` | RFC3339 | yes | 最后更新时间 |
| `warnings` | []string | no | 本次读时 rescan `content`。**不**从 `{id}.md` 读出来。敏感命中时有值，不拒绝 |

### §19.3 错误

| 情况 | 错误 |
|---|---|
| id 不存在 | `inbox not found: <id>`（包括已自动删除） |

---

## §20. audit.list_recent 字段映射

### §20.1 请求

| 字段 | 类型 | 必含 | 默认 | 说明 |
|---|---|---|---|---|
| `limit` | int | no | 100 | 返回条数 |
| `since` | RFC3339 | no | - | 时间下限 |
| `tool` | string | no | - | 工具名过滤 |
| `client_id` | string | no | - | 客户端过滤 |
| `result_status` | enum | no | all | `success` / `error` / `unauthorized` / `forbidden` / `all` |

### §20.2 响应元素

| 字段 | 类型 | 必含 | 说明 |
|---|---|---|---|
| `ts` | RFC3339 | yes | 工具调用时间 |
| `tool` | string | yes | 工具名 |
| `client_id` | string | yes | Agent 标识 |
| `scopes` | []string | yes | 实际授予的 scope 列表 |
| `args_digest` | string | yes | 参数的 SHA-256 哈希（不存原文） |
| `result_status` | enum | yes | 见 §20.3 |
| `duration_ms` | int | yes | 执行时长 |
| `error` | string | no | 错误信息（不含敏感数据） |
| `target_path` | string | no | apply / proposal 类才有 |
| `commit` | string | no | apply 类才有 |
| `base_commit` | string | no | apply 类才有 |

### §20.3 result_status 取值

```text
success         工具正常返回
error           工具返回 MCP error（如 not_found / invalid_argument）
unauthorized    401：token 缺失 / 无效
forbidden       403：scope 不足
```

### §20.4 不记录字段

- token 原文
- 完整私密正文
- secret 内容
- args 原文（仅存 SHA-256 digest）

---

## §21. 敏感模式检测

**默认关。** `schema.sensitive_patterns` 缺省 / 空列表 `[]` = 写入不检测、读出不打码。个人工作台要把 Skill / MCP / 文章**完整**给到授权 Agent，默认开会误伤部署笔记和配置说明。

可配：往 yaml 列表加 regex 才开启。开启后也**只警告、不拒绝、读出不打码**——真正挡未授权的是 scope + visibility，不是正则。

```yaml
schema:
  sensitive_patterns: []          # 默认。不检测
  # 要开再自己加，例如：
  # - "-----BEGIN [A-Z ]*PRIVATE KEY-----"
```

### §21.1 开启后的行为

| 动作 | 行为 |
|---|---|
| 写入（proposal / inbox） | 命中 → 响应里记一条 warning，**照样创建**。不因 regex 拒。proposal 进 `validation.warnings`；inbox 进响应 `warnings`，**不**写进 `{id}.md` |
| 读出（knowledge.get / skill.get / mcp.get / inbox.get） | **原样返回**。不 `[REDACTED]`。inbox.get 每次读再扫一遍 content |
| 日志 / audit | 仍不写 token 原文、不写 token hash（这是审计字段约束，跟本开关无关） |

没有「拒绝级」模式。Bearer / 私钥块写进 vault 是用户自己的选择；授权 Agent 要能读到完整配置。

### §21.2 不要做的事

```text
不要默认塞 9 条 regex
不要把 password: 用户自备 / Go 1.25.0.1 / 8.8.8.8 打码或拒绝
不要让 Agent 少传 validation.checks 就跳过检查（检查名单服务端写死；默认关时名单为空）
不要靠 RemoteAddr == 127.0.0.1 判断「仅 loopback」——反代会改掉对端地址
```

---

## §22. 内容格式规范（继承自工作台 SCHEMA.md）

适用于 `D:/Data/工作台/` 下所有一级目录的 markdown 文件。构建时由 `vite.config.ts` 的 `virtual:vault-tree` 扫描器读取。

### §22.1 通用规则

- **一级目录 = 栏目**，每栏目独立解析器和路由
- 文件名即 slug 来源（栏目内相对路径，`/` → `__`）
- frontmatter 用 YAML，三个 `---` 包裹
- 公开博客不发布条件：`draft: true` **或** `visibility: draft`（两个独立开关，见 §3.2）

### §22.2 排除规则

`virtual:vault-tree` 扫描时**排除**：

- `.obsidian`（Obsidian 配置）
- `.trash`（Obsidian 回收站）
- `Workbase/`（Agent 工作基座私有目录，**不作为公开博客栏目**）

### §22.3 文章（`文章/`）

```yaml
---
title: 文章标题              # 必填
date: 2026-08-15             # 必填 YYYY-MM-DD
tags:                        # 可选
  - Go
  - 云原生
cover: 封面.png               # 可选
excerpt: 一句话简介            # 可选
draft: false                 # 可选
visibility: public            # 可选，默认 public
---
```

### §22.4 项目（`项目/`）

```yaml
---
name: 项目名                  # 必填
summary: 一句话简介            # 必填
links:                        # 可选
  - type: repo | demo | video | docs | site | other
    url: https://...
    label: 可选
stack:                        # 可选
status: 维护中 | 进行中 | 已归档
cover: 封面.png
date: 2026-07-04
visibility: public
---
```

### §22.5 友链（`友链/`）

```yaml
---
name: 友站名                  # 必填
url: https://...              # 必填
avatar: https://.../head.jpg
desc: 一句话描述
---
```

### §22.6 Workbase/context

```yaml
---
id: engineering-style
kind: context_pack
title: 工程风格
visibility: private
updated: 2026-08-17
startup: true                 # 是否进入 context.startup（secret 即使标了也不进）
priority: high                # high | medium | low
---
```

### §22.7 Workbase/skills

参见 §10.1。

### §22.8 Workbase/mcps

参见 §11.1。

---

## §23. 索引表结构

Vault 镜像用一张 `notes` 主表 + `kind` 字段隔离，**不开** projects / skills / mcps / contexts 分表。这是当前完整设计，不是分期。

indexer 按一级目录写 `kind`（frontmatter `kind` 只对 Workbase 下生效）。

**`notes.id` = vault 相对路径**（正斜杠，含 `.md`，如 `文章/foo.md`、`部署溯源/bar.md`）。跨目录唯一。不要只拿文件名 slug——`文章/foo.md` 和 `部署溯源/foo.md` 不能都写成 `foo`。

**路径归一（Windows 必做）**：入库和查询一律 `filepath.ToSlash`。本地 vault 在 Windows 上，`filepath.Rel` 会吐 `文章\foo.md`；不归一，库里就是反斜杠，Agent 按文档传正斜杠 → `knowledge.get` 404，WikiLink `[[文章/foo]]` 也对不上。请求里的 `\` 先归一再查。旧实现 `vault.go:noteID()` 还 `TrimSuffix(rel, ".md")`——那是旧代码，**不要当规格**；契约是含 `.md`。

对外 id 是两套：

| 工具 | 请求 / 响应 `id` | 对应 |
|---|---|---|
| `knowledge.search` / `knowledge.get` | `notes.id` | vault 相对路径 |
| `project.list` / `project.get` | 文件名 slug | `项目/<slug>.md`。`notes.id` 仍是路径 |
| `skill.*` / `mcp.*` / `context.*` | frontmatter `id` | 如 `markdown-lint`。`notes.id` 仍是路径 |

`id_prefixes` 只给 proposal / inbox 自动生成 ID。indexer **不**用 `art` / `note` 前缀拼 PK。

| 路径 | `notes.kind` | 进哪 |
|---|---|---|
| `文章/**/*.md` | `article` | `knowledge.search` 默认收录；`knowledge.get` |
| vault 其它 `.md`（非下表排除） | `note` | 同上 |
| `项目/**/*.md` | `project` | 只走 `project.list` / `project.get`。**不**进 knowledge |
| `Workbase/context/*.md` | `context_pack` | 只走 `context.*` |
| `Workbase/skills/*.md` | `skill` | 只走 `skill.*` |
| `Workbase/mcps/*.md` | `mcp_server` | 只走 `mcp.*` / `workbase.identity` |
| `友链/**/*.md` | **不入库** | 友链是公开博客卡片，不是知识条目 |
| `Workbase/` 下除 context / skills / mcps 以外的 md | `note` | 兜底。Workbase 只有这三个子目录；不要再建 `conventions/` 或 `policies/`。误放的 md 当普通 private 笔记，不是第四种 registry |
| `.obsidian/` `.trash/` | **不入库** | — |

`knowledge.search` 默认 `kind=["note","article"]`，`scope=all` 含 `draft`。已传 `kind` 只保留 `note`/`article`；过滤后为空 → 空结果，不回落默认。没有 `article.list` / `article.get`——文章走 knowledge。`proposal.target.type=article` 只表示写入目标，和 search 默认集合是两件事。

`knowledge.get` 只返回 `kind ∈ {note, article}`。其它 kind → `note not found`，不泄露是 skill。

Token / 审计不进 `notes`：

| SQLite 文件 | 表 | 存什么 | 进 Vault / Git？ |
|---|---|---|---|
| `{runtime}/index/notes.sqlite` | `notes` / `notes_fts` / `links` / `backlinks` | vault 镜像 | 否（运行时派生） |
| `{runtime}/auth.sqlite` | `auth_tokens` | Agent Token | 否（VPS 私有区） |
| `{runtime}/audit/audit.sqlite` | `audit_log` | 审计 | 否（VPS 私有区） |

`auth_tokens` 同名可有多行（一行 active + 若干 grace/revoked）。约束是部分唯一索引，不是整列 UNIQUE：

```sql
CREATE UNIQUE INDEX idx_auth_tokens_active_name ON auth_tokens(name) WHERE status='active';
```

```sql
-- 主表
notes(
  id TEXT PRIMARY KEY,             -- = vault 相对路径（正斜杠，含 .md）。入库 filepath.ToSlash。不是 slug，不是 art_xxx
  path TEXT NOT NULL,              -- 与 id 相同（冗余，查询方便）
  kind TEXT NOT NULL,              -- note | context_pack | project | skill | mcp_server | article
  title TEXT NOT NULL,
  visibility TEXT NOT NULL,        -- public | private | secret | draft
  updated_at TEXT NOT NULL,
  access_count INTEGER DEFAULT 0,
  last_access_at TEXT,
  frontmatter_json TEXT,
  summary TEXT
)

-- 全文搜索
notes_fts(
  id TEXT PRIMARY KEY,
  title TEXT,
  headings TEXT,
  body TEXT,
  tags TEXT
)

-- 链接
-- WikiLink 解析：先按完整相对路径（去 .md），再按文件名唯一匹配。
-- 重名 → 只记 raw，不建边（避免 文章/foo.md 和 部署溯源/foo.md 撞车）。
-- source_id / target_id = notes.id = vault 相对路径。
links(
  source_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  link_type TEXT NOT NULL,         -- wikilink | markdown
  raw TEXT,
  PRIMARY KEY (source_id, target_id, link_type)
)

-- 反链（运行时构建，访问更快）
backlinks(
  source_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  context TEXT,
  PRIMARY KEY (source_id, target_id)
)
```

---

## §24. 状态机汇总

### §24.1 Proposal

```text
pending  → approved  → applied
        ↘ rejected
approved → conflict  → approved     # 3-way / commit 失败才进 conflict
applied / rejected = 终态
conflict = 暂停态（不是终态，可回到 approved 重试）
```

没有 `pending → conflict`：创建时校验失败是控制层拒绝，不写 receipt。3-way 只发生在用户点同意、进入 `approved` 之后。

终态：`applied` / `rejected`。`conflict` 是**暂停态**——用户编辑 payload 后救回 `approved`。救回时**重读当前 HEAD 作为新 `base_commit`**（旧 base 大概率已经过时）。也可以明确「只改 payload、不换 base」——再冲突就再停。默认换 base。

### §24.2 Inbox

```text
pending  → reviewing  → done
                     ↘   abandoned
pending  → done | abandoned
```

终态：`done` / `abandoned`。`done` / `abandoned` 超过 `retention_days` 自动删除。

### §24.3 Context Pack 合成（context.startup）

```text
startup = 按 priority 降序拼接 (startup=true 且 visibility≠secret 的 context pack)
```

secret 永不进合成（即使标了 `startup: true`）。draft 照收。每次调用即时合成，无状态。

---

## §25. 热度算法

```text
score = access_count * exp(-elapsed_days / HALF_LIFE_DAYS)
```

- `HALF_LIFE_DAYS` 默认 7，config 可覆盖
- `elapsed_days` = `now - last_access_at`
- 完全未访问 score = 0
- `Hot()` 实时计算，按 score 降序
- `score < min_score`（默认 0.001）不参与 Hot 排序

### §25.1 access 写入

每次 `knowledge.get` / `context.get` / `project.get` / `skill.get` / `mcp.get` 命中时：
- `access_count += 1`
- `last_access_at = now`
- 立即 fsync 持久化（不依赖优雅退出）

### §25.2 Hot 排序返回

工具返回 `signals.access` 字段使用本算法得分。

---

## §26. 文件落点

### §26.1 本地

```text
D:/Data/工作台/Workbase/         # 事实源（vault 私有层）
D:/Data/工作台/{文章,项目,友链,部署溯源}/
D:/Code/Front-end/博客/SCHEMA.md # 本文件
D:/Code/Front-end/博客/docs/agent-workbase-mcp-v0.1.md
D:/Code/Front-end/博客/server/mcp/
D:/Code/Front-end/博客/server/mcp/.workbase/  # 运行时目录（不进 Git）
```

### §26.2 VPS

```text
/home/studio/workbase/
├── config.yaml                  # 不进 Git
├── auth.sqlite                  # auth_tokens
├── index/notes.sqlite           # notes / notes_fts / links / backlinks
├── proposals/                   # *.md
├── inbox/                       # {id}.md
└── audit/audit.sqlite           # audit_log

/home/studio/vault.git/          # bare repo
/home/studio/workbench/          # working tree
```

### §26.3 不存放

- 公开仓库：token / 私钥 / 真实 IP / `config.yaml`
- 本地 vault：`Agent Inbox/` / `proposals/` / `audit/`（只存 VPS 私有区）

---

## §27. 字段映射修改流程

任何字段变更必须**双改**：

1. 改本文件 `SCHEMA.md` 对应小节
2. 改 `server/mcp/internal/...` 中对应 Go 代码
3. 部署
4. **两步缺一不可**

提交 commit message 格式：`schema(<tool>): <change>`，例如 `schema(knowledge.search): add intent=entity weight bias`。

验收 §23.1.35 强制检查（字段映射：每个工具的请求 / 返回 schema 在 SCHEMA.md 有独立小节）。
