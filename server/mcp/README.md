# workbase-mcp

遇见江楠 · Agent Workbase 的 MCP 服务端 + WebUI 后台。

## 技术栈

- Go 1.25+（mcp-go v0.58.0 要求；本机可用 `GOTOOLCHAIN=auto`）
- mark3labs/mcp-go（Streamable HTTP）
- yaml.v3（frontmatter 解析）
- 标准库 net/http
- 嵌入式 WebUI（Go embed + 原生 HTML/CSS/JS）

v0.1 不引入 SQLite / 向量数据库。

## 目录

```
server/mcp/
├── cmd/workbase-mcp/main.go   # 入口
├── internal/
│   ├── config/     # YAML 配置读取
│   ├── vault/      # Obsidian Vault 扫描 + frontmatter 解析
│   ├── index/      # JSON 索引 + 访问热度计数
│   ├── inbox/      # 待办管理（append/update/list/get + 7 天清理）
│   ├── proposal/   # 写入请求管理（create/list/get）
│   ├── apply/      # Proposal 真实落盘
│   ├── auth/       # Bearer Token + scope 校验
│   ├── sanitize/   # 敏感模式检测（IP/密钥/私钥/token）
│   ├── admin/      # WebUI 后台 HTTP API + 嵌入式静态页面
│   ├── audit/      # 审计日志
│   └── tools/      # MCP 工具注册
└── README.md
```

## 快速启动

```bash
cd server/mcp
go run ./cmd/workbase-mcp
```

默认配置（无需 config.yaml）：
- 从 `D:/Data/工作台` 构建索引
- inbox 存 `.workbase/inbox/`
- proposals 存 `.workbase/proposals/`
- admin WebUI 监听 `127.0.0.1:8788`

打开浏览器访问 `http://127.0.0.1:8788` 进入看板式 inbox。

## 命令行

```bash
# 仅重建索引（不启动服务）
go run ./cmd/workbase-mcp -reindex

# 指定配置文件
go run ./cmd/workbase-mcp -config /path/to/config.yaml
```

## WebUI

- **Inbox 看板**：四列（待处理 → 待审核 → 已完成 / 已废弃），拖拽卡片换状态
- **热度**：访问次数降序排序 + 条形图可视化
- **新建**：点击列下方 `+ 新建待办` 按钮

## API

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/inbox` | 列出所有待办摘要 |
| POST | `/api/inbox` | 新建 pending 待办 |
| GET | `/api/inbox/{id}` | 读取单条 |
| PUT | `/api/inbox/{id}` | 编辑内容或改状态 |
| GET | `/api/heat` | 访问热度排名 |
| GET | `/api/proposals` | 列出 proposal |
| POST | `/api/proposals` | 创建 proposal |

## 配置

`config.yaml` 示例（开发期可省略，使用默认值）：

```yaml
server:
  listen: 127.0.0.1:8787

vault:
  root: D:/Data/工作台
  git_dir: ""

workbase:
  root: D:/Data/工作台/Workbase
  index: ./.workbase/index
  proposals: ./.workbase/proposals
  inbox: ./.workbase/inbox

admin:
  listen: 127.0.0.1:8788
```

## 部署

编译为单二进制后部署到 VPS：

```bash
GOOS=linux GOARCH=amd64 go build -o workbase-mcp ./cmd/workbase-mcp
```

```bash
# VPS 上
./workbase-mcp -config /home/studio/workbase/config.yaml
```

systemd 配置参考设计文档 §21.1。

## 注意事项

- `Workbase/` 目录：**博客构建（vite.config.ts）排除**，但 **MCP 索引器包含**（skill/mcp/context registry 来源）。
- inbox 的 pending/reviewing 状态不自动删除；done/abandoned 保留 7 天后自动清理。
- access.json 在进程退出时写盘，重启后恢复热度。