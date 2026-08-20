# ============================================================
#  Workbase MCP -> VPS one-click deploy
#
#  Usage (PowerShell):
#      $env:BLOG_VPS = "ubuntu@<VPS_IP>"
#      $env:BLOG_SSH_KEY = "C:/path/to/key.pem"
#      .\deploy\mcp\deploy-vps.ps1
#
#  Prereq:
#   1) linux amd64 binary already built at
#        server/mcp/.workbase/workbase-mcp-linux-amd64
#   2) live config at deploy/mcp/config.vps.yaml
#        (copy from config.vps.example.yaml and fill admin_auth.pass_hash;
#         Token 不在 yaml，登录 webUI 签发；config.vps.yaml is gitignored)
# ============================================================

$ErrorActionPreference = 'Stop'

$Vps = $env:BLOG_VPS
if (-not $Vps) { throw "Set `$env:BLOG_VPS first, e.g. `$env:BLOG_VPS = 'ubuntu@<VPS_IP>'" }

$Key = $env:BLOG_SSH_KEY
if (-not $Key) { throw "Set `$env:BLOG_SSH_KEY first, e.g. `$env:BLOG_SSH_KEY = 'C:/path/to/key.pem'" }

$Repo = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$Mcp = Join-Path $Repo 'server\mcp'
$Bin = Join-Path $Mcp '.workbase\workbase-mcp-linux-amd64'
$Cfg = Join-Path $PSScriptRoot 'config.vps.yaml'
$Stage = Join-Path $Mcp '.workbase\vps-deploy-stage'

if (-not (Test-Path $Bin)) {
  throw "missing linux binary: $Bin (build with GOOS=linux GOARCH=amd64 first)"
}
if (-not (Test-Path $Cfg)) {
  throw "missing $Cfg — copy config.vps.example.yaml and fill real hashes first"
}

if (Test-Path $Stage) { Remove-Item -Recurse -Force $Stage }
New-Item -ItemType Directory -Force -Path $Stage | Out-Null

Copy-Item $Bin (Join-Path $Stage 'workbase-mcp')
Copy-Item $Cfg (Join-Path $Stage 'config.yaml')
Copy-Item (Join-Path $PSScriptRoot 'rebuild-blog.sh') (Join-Path $Stage 'rebuild-blog.sh')
Copy-Item (Join-Path $PSScriptRoot 'jiangnan-workbase-mcp.service') (Join-Path $Stage 'jiangnan-workbase-mcp.service')
Copy-Item (Join-Path $PSScriptRoot 'post-receive-reindex.sh') (Join-Path $Stage 'post-receive-reindex.sh')
Copy-Item (Join-Path $PSScriptRoot 'install-vps.sh') (Join-Path $Stage 'install-vps.sh')

Write-Host '[1/3] upload stage -> /tmp/workbase-deploy'
ssh -i $Key -o IdentitiesOnly=yes $Vps 'rm -rf /tmp/workbase-deploy; mkdir -p /tmp/workbase-deploy'
scp -i $Key -o IdentitiesOnly=yes `
  (Join-Path $Stage 'workbase-mcp') `
  (Join-Path $Stage 'config.yaml') `
  (Join-Path $Stage 'rebuild-blog.sh') `
  (Join-Path $Stage 'jiangnan-workbase-mcp.service') `
  (Join-Path $Stage 'post-receive-reindex.sh') `
  (Join-Path $Stage 'install-vps.sh') `
  "${Vps}:/tmp/workbase-deploy/"

Write-Host '[2/3] run install-vps.sh'
ssh -i $Key -o IdentitiesOnly=yes $Vps 'bash /tmp/workbase-deploy/install-vps.sh /tmp/workbase-deploy'

Write-Host '[3/3] done'
