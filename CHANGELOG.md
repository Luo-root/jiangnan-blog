# Changelog

本仓库按 SemVer 发版。

## 0.1.1 — 计划

功能验证收尾 + 使用中打磨。不换协议、不拆仓。计划见 [`docs/v0.1.1.md`](docs/v0.1.1.md)。路线见 [`docs/roadmap.md`](docs/roadmap.md)。

## 0.1.0 — 2026-08-21

Blog as Agent Workbase 第一版。

### 公开博客
- Obsidian Vault（`D:/Data/工作台`）为事实源；朝曦 / 夜隐双主题
- 栏目：文章 / 项目 / 友链 / 归档 / 图谱
- 部署：内容 `sync.ps1`、代码 `pull.ps1`，两条链路

### Agent Workbase MCP
- 20 个 MCP 工具（含 `proposal.update`）
- Git-backed 3-way apply；Inbox 看板 + 评论；Token SQLite
- 热度艾宾浩斯 `Hot()`；审计 SQLite；后台 WebUI（shadcn/ui）
- `workbase.identity` 返回可粘贴 `agent_prompt`（从 Vault 即时拼操作环）
- 仓库：`Luo-root/jiangnan-blog-agent-workbase`
- Go module：`github.com/Luo-root/jiangnan-blog-agent-workbase/mcp`

契约：`docs/agent-workbase-mcp-v0.1.md` + `SCHEMA.md`。
