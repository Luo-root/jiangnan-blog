# 部署脚本说明

本目录的脚本用于把**前端代码**推送到 VPS（IP 走环境变量 `$env:BLOG_VPS`，**不要硬编码**）。

**博客内容（Obsidian）走另一条链路**，由 `D:\Data\工作台\sync.ps1` 推到 VPS bare repo，详见 `D:\Data\工作台\README.md` 和 `D:\Data\工作台\部署溯源\jiangnan-blog.md`。

## 文件清单

| 文件 | 在哪运行 | 作用 |
|---|---|---|
| `setup-vps.sh` | VPS 一次性 | 裸机初始化：装 git/caddy，建工作台 bare repo + post-receive 钩子 |
| `caddyfile` | VPS `/etc/caddy/Caddyfile` | Caddy 配置：域名 + 反代 + SPA fallback |
| `blog-update.sh` | （已废弃） | 旧版 cron 5 分钟拉取方案，已被 `deploy-code.sh` 取代 |
| `deploy-code.sh` | VPS（`pull.ps1` 触发） | 解压 `repo.tar.gz` → 复用 `node_modules` → `npm run build` → `rsync` |
| `pull.ps1` | 本地 PowerShell | 一键：`tar` + `scp` + `ssh bash deploy-code.sh` |

## 日常：推送代码

```powershell
cd D:\Code\Front-end\博客
git add -A
git commit -m "..."
git push origin main
.\deploy\pull.ps1
```

`pull.ps1` 做了什么：

1. `tar` 打包当前目录（排除 `node_modules` / `dist` / `.git` / `.backup`），产物 `deploy/repo.tar.gz`
2. `scp deploy/repo.tar.gz $BLOG_VPS:/home/studio/app/repo.tar.gz`
3. `ssh ... "bash /home/studio/app/deploy-code.sh"`：
   - 把当前 `repo/` 改名为 `repo.old.<ts>` 备份
   - 解压 `repo.tar.gz` 到 `repo/`
   - 软链 `repo.old.<ts>/node_modules` → `repo/node_modules`（**省 33s 重装**）
   - `VAULT_ROOT=/home/studio/workbench npm run build`
   - `rsync -a --delete repo/dist/ /home/studio/app/public/`
   - 清理超过 2 个的 `repo.old.*` 备份

完成后访问 VPS 临时 IP 验证（备案通过后改用域名）。

## VPS 端关键路径

```
/home/studio/
├── app/
│   ├── repo/              # 当前前端代码（来自 repo.tar.gz 解压）
│   ├── repo.old.*/        # 历史版本（最多保留 2 个，自动清理）
│   ├── repo.tar.gz        # 本次上传的 tar（解压后保留，清理旧 backup 时顺带删）
│   ├── public/            # Caddy root（rsync 同步自 repo/dist/）
│   ├── public.bak.*/      # Caddyfile 配置里 rsync --backup 留的历史目录
│   ├── public.old.*/      # 早期手动 rsync 残留（待清理）
│   ├── deploy.log         # 工作台 post-receive 钩子日志（内容自动部署）
│   ├── build/             # 旧版 build 输出（已废弃）
│   └── deploy-code.sh     # VPS 端代码部署脚本（pull.ps1 触发）
├── vault.git/             # 工作台 bare repo（接 sync.ps1 push）
├── workbench/             # 工作台 checkout（无 .git，纯内容）
└── sub2api-deploy/        # 其他项目残留
```

## 关键决策

- **VPS 不直连 GitHub**（网络不稳定），所以前端代码走 `本地 tar → scp`，不走 `git pull`
- **不复用 rsync --delete 远程内联命令**（安全策略拦），改用 VPS 端 .sh 脚本
- **保留 2 个历史版本** `repo.old.*`，build 失败可手动 `mv repo.old.<ts> repo && rsync` 回滚
- **post-receive 钩子和 pull.ps1 是两套独立链路**，互不依赖

## 故障排查

| 现象 | 排查 |
|---|---|
| `pull.ps1` 报 `scp failed` | 检查 `$env:BLOG_SSH_KEY` 指向的私钥是否存在；VPS 安全组是否放行 22 |
| 部署后页面没更新 | `ssh ubuntu@... "tail -30 /home/studio/app/deploy.log"` 看 build 日志；检查 `rsync` 后 `/home/studio/app/public/index.html` mtime |
| `node_modules` 软链失效 | VPS 上 `ls -la /home/studio/app/repo/node_modules`；手动 `npm ci` 重装 |
| 想强制全新 build | `rm /home/studio/app/repo/node_modules` 后再跑 `pull.ps1`（会触发 VPS `npm install` 兜底） |
| 回滚到上一个版本 | `ssh ... "mv repo repo.broken && mv repo.old.<ts> repo && rsync -a --delete repo/dist/ /home/studio/app/public/"` |
