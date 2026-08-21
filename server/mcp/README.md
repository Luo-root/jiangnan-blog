# Workbase MCP Server

公网私密 Agent 接入层。Vault 是事实源，本目录是协议实现。

**写代码前先读契约，不要抄本目录里现有 Go。**

| 读什么 | 路径 | 职责 |
|---|---|---|
| 设计 | [`../../docs/agent-workbase-mcp-v0.1.md`](../../docs/agent-workbase-mcp-v0.1.md) | why / 验收 |
| 字段 | [`../../SCHEMA.md`](../../SCHEMA.md) | API / 状态机 / 表 / 算法 |
| 模板 | [`config.example.yaml`](config.example.yaml) | 可启动的字段结构 |
| 自描述 | `D:/Data/工作台/Workbase/` | Agent 看到的文案 |

契约已按 PR 落地：`schema:` 配置、identity、Token SQLite、reindex mux、索引 SQLite 读路径、完整 ours 3-way / apply 分阶、热度 `Hot()`、审计 SQLite、webUI session 登录、audit / knowledge / system / git / templates 后台页。  
不要把旧规格当现行：默认 9 条敏感正则、payload 当 ours、`noteID()` 去 `.md`、JSONL 审计、按 count 排热度、浏览器弹窗 Basic Auth。进程等这些 PR 合进 main 后再启。

---

## 1. 同一份 vault，两套扫描

```text
D:/Data/工作台/
        ├─ Vite virtual:vault-tree   → 公开博客（只收 public 且非草稿）
        └─ MCP indexer               → SQLite notes（private/secret 按 scope 读）
```

不共享同一份 index 文件。后台也不另开一套 vault 路径。

## 2. 双根

| 字段 | 是什么 | 本地 | VPS |
|---|---|---|---|
| `vault.root` | Vault 总根 | `D:/Data/工作台` | `/home/studio/workbench` |
| `workbase.root` | Registry 源（`context/` `skills/` `mcps/`） | `D:/Data/工作台/Workbase` | `/home/studio/workbench/Workbase` |
| `workbase.runtime` | 进程私有区（index / proposals / inbox / audit / auth） | `./`（相对本目录） | `/home/studio/workbase` |

描述性字符串必须从 `workbase.root` 即时读。Go 不许硬编码「是什么 / 能做什么」。

## 3. 配置

- 开发：本目录 `config.yaml`（不进 Git）
- 模板：`config.example.yaml`
- 权威字段：`SCHEMA.md §1`
- 启动必填：`schema.visibility_policy` + `admin_auth.user` / `admin_auth.pass_hash`
- Token **不在 yaml**。只留 `auth.grace_period_hours`（默认 0）
- `schema.sensitive_patterns` 默认 `[]`（关）

复制模板：

```powershell
Copy-Item config.example.yaml config.yaml
# 填 vault.root / workbase.root / admin_auth
```

Admin 口令是 SHA-256 hex，无盐，不含 `sha256:` 前缀。单账号个人台可接受，不是通用口令方案。

## 4. 鉴权

所有 MCP 工具都必须带 Bearer token。没有公开 endpoint。

8 个可签给 Agent 的 scope（`SCHEMA.md §2.1`）：

```text
read:context / read:knowledge / read:project / read:registry
read:inbox / write:proposal / write:inbox / ops:audit
```

- `workbase.identity`：任意有效 token（合并原 `workbase.manifest` + `workbase.whoami`）
- `write:proposal` 同时覆盖 create / list / get / update
- `admin:reindex` **不是** Agent scope。Token UI 不展示、不签发

Token 在 `{runtime}/auth.sqlite` 的 `auth_tokens`：

```sql
CREATE UNIQUE INDEX idx_auth_tokens_active_name
  ON auth_tokens(name) WHERE status='active';
```

签发 / 轮换：SQLite 成功后**同步** upsert `tokenCache`，再返回明文。撤销可以等 ≤5s。  
轮换：先把 SQLite 旧行改成 `grace`，再**同步改旧 hash 的 cache**（`grace_period_hours=0` 则直接删），再 INSERT 同名新行并 upsert 新 cache。只 upsert 新行会让旧 token 再活 ≤5s 的 `active`。

## 5. 工具（20）

```text
workbase.identity
context.startup / context.get
knowledge.search / knowledge.get
project.list / project.get
skill.list / skill.get
mcp.list / mcp.get
proposal.create / proposal.list / proposal.get / proposal.update
inbox.append / inbox.update / inbox.list / inbox.get
audit.list_recent
```

`proposal.update` 只改尚未落盘的 proposal（`pending` / `conflict`）。`applied` / `rejected` 终态禁止再改、再批准、再 apply。评论线程形状见 `SCHEMA.md §12.4`，评论不改 status。

`knowledge.search` 默认 `kind=["note","article"]`（`文章/` 进 `article`）。已传 `kind` 只保留这两个；过滤后为空 → 空结果，不回落默认。`intent` / `scope` 非法值 → `invalid_argument`。`notes.id` = vault 相对路径（正斜杠，含 `.md`）；入库 / 查询一律 `filepath.ToSlash`。`scope=all` 含 `draft`。响应 `kind` 原样返回。`友链/` 不入库。即使 `scope=all` 也不出 secret（含摘要）。secret 只走对应 get 显式 id。`knowledge.get` 只返 `kind ∈ {note, article}`，其它 kind → `note not found`。

`context.startup` 跳过 secret（即使标了 `startup: true`）；draft 照收。`context.get` 对 draft 放行，secret → `secret_blocked`。

## 6. 写入

Agent 改 Vault 正式内容的唯一入口是 `proposal.create`。Inbox 是待办，不审批、不 apply、不 commit。

客户端**不要传**：

- `kind`（服务端从 `target.type` 抄）
- `validation.checks`（检查名单服务端写死）

`target.path` 必须落在 `vault.root` 下（防 `../`），**不**要求文件已存在。入库 / 校验前一律 `filepath.ToSlash`。

| operation | 文件 |
|---|---|
| `create_file` / `register_item` | 必须不存在；落盘前再 stat，已存在 → conflict |
| `append` / `append_section` / `patch_section` | 必须存在 |
| `append_section` 标题不存在 | 先建标题再追加 |
| `patch_section` 标题不存在 | conflict / `section_not_found` |

`expected_base` 可选：有传但对不上 → `stale_base`；不传才用当前 HEAD。

状态机：`pending → approved → applied` / `rejected`；`approved → conflict → approved`。  
**没有** `pending → conflict`。创建校验失败是控制层拒绝，不写 receipt。

3-way：先在 `base_commit` 文件上施加 operation 得到**完整 ours**，再 `git merge-file`。payload 片段不当 ours。`create_file` / `register_item` 无 ancestor，不走 3-way。

apply 分阶：

- 落盘 + commit 失败 → `conflict`
- commit 已成功、reindex / rebuild 失败 → 仍是 `applied`，副作用单独重跑，禁止再 apply

conflict 救回默认换新 base（重读当前 HEAD）。

## 7. 敏感模式

默认关。授权 Agent 读 Skill / MCP / 文章拿完整原文。

开启后也只警告、不拒绝、读出不 `[REDACTED]`。挡未授权的是 token + scope + visibility。

日志 / audit 仍不写 token 原文、不写 token hash（跟这个开关无关）。

## 8. 内部 reindex

`POST /internal/reindex` 和 MCP 协议同进程、mux 分路。不要让 mcp-go 把这条 POST 吃成协议错误。

保护靠三件事，**不要**靠 `RemoteAddr == 127.0.0.1`（反代会改掉对端地址）：

1. Caddy 不转发 `/internal*`
2. 进程 bind `127.0.0.1:8787`
3. hook / apply 副作用本机直打，不带 Bearer

```text
mcp.<domain> {
    handle /internal* { respond 404 }
    reverse_proxy 127.0.0.1:8787
}
```

## 9. SQLite（三套库）

```text
{runtime}/index/notes.sqlite    notes / notes_fts / links / backlinks
{runtime}/auth.sqlite           auth_tokens
{runtime}/audit/audit.sqlite    audit_log
```

索引单表 + `kind` 隔离。不开 projects / skills / mcps / contexts 分表。都不进 Vault / Git。

热度：`Hot(halfLifeDays, minScore)` 实时算 `score = access_count * exp(-elapsed_days / half_life_days)`，按 score 降序；`score < min_score` 不进榜。默认半衰期 7 天、`min_score=0.001`。调用方从 config 传入。

审计：`{runtime}/audit/audit.sqlite` 的 `audit_log`。中间件在 `next()` 之后记；unauthorized / forbidden 也写。最小字段 ts / tool / client_id / scopes（实际授予）/ args_digest / result_status / duration_ms。不存 token 原文、args 原文、secret 正文。`audit.list_recent` 用 limit / since / tool / client_id / result_status 过滤，没有 `mode=detail|hashed`。MCP 只用 `since`。webUI `GET /api/audit/recent` 额外收 `until`。

## 10. 本地跑（重构完成后）

```powershell
cd D:\Code\Front-end\博客\server\mcp
go test ./...
go vet ./...
go run ./cmd/workbase-mcp -config config.yaml
```

- MCP：`127.0.0.1:8787`
- Admin：`127.0.0.1:8788`（独立登录页 + session token，不用浏览器 Basic Auth）

改后台前端：

```powershell
cd D:\Code\Front-end\博客\server\mcp\admin
npm install
npm run build   # 产出写入 ../internal/admin/static，由 Go embed
```

VPS 部署见 [`../../deploy/mcp/`](../../deploy/mcp/) 与 [`../../deploy/README.md`](../../deploy/README.md)。  
重构完成前不要把这份旧 binary 当可用 MCP。

## 11. 改字段

双改：`SCHEMA.md` 对应小节 + `internal/` 代码。缺一不可。

commit message：`schema(<tool>): <change>`
