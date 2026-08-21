# Changelog

本仓库按 SemVer 发版。`v0.1.0` 的 tag / GitHub Release **尚未打**，本文档先把准备态写清。

## 0.1.0 — 准备中

Blog as Agent Workbase 第一版完整实施。

### 公开博客
- Obsidian Vault（`D:/Data/工作台`）为事实源；朝曦 / 夜隐双主题
- 栏目：文章 / 项目 / 友链 / 归档 / 图谱
- 部署：内容 `sync.ps1`、代码 `pull.ps1`，两条链路

### Agent Workbase MCP
- 20 个 MCP 工具（含 `proposal.update`）
- Git-backed 3-way apply；Inbox 看板 + 评论；Token SQLite
- 热度艾宾浩斯 `Hot()`；审计 SQLite；后台 WebUI（shadcn/ui）
- 仓库：`Luo-root/jiangnan-blog-agent-workbase`
- Go module：`github.com/Luo-root/jiangnan-blog-agent-workbase/mcp`

契约：`docs/agent-workbase-mcp-v0.1.md` + `SCHEMA.md`。
