# ============================================================
#  Frontend code -> VPS one-click deploy
#
#  Usage (PowerShell, from project root):
#      $env:BLOG_VPS = "ubuntu@<VPS_IP>"
#      $env:BLOG_SSH_KEY = "C:/path/to/key.pem"
#      .\deploy\pull.ps1
#
#  Or set in your PowerShell profile so you don't repeat it.
#
#  What it does (pure linear, no defensive checks):
#      1. tar the project (excluding node_modules / dist / .git)
#      2. scp to VPS: /home/studio/app/repo.tar.gz
#      3. ssh + bash /home/studio/app/deploy-code.sh
#
#  Prereqs (already done once):
#      - VPS has deploy-code.sh at /home/studio/app/
#      - Local ssh key configured
# ============================================================

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$Vps = $env:BLOG_VPS
if (-not $Vps) { throw "Set $env:BLOG_VPS first, e.g. `$env:BLOG_VPS = 'ubuntu@<VPS_IP>'" }

$Key = $env:BLOG_SSH_KEY
if (-not $Key) { throw "Set $env:BLOG_SSH_KEY first, e.g. `$env:BLOG_SSH_KEY = 'C:/path/to/key.pem'" }

$Tar = "deploy/repo.tar.gz"
$RemoteTar = "/home/studio/app/repo.tar.gz"

Write-Host "[1/3] pack repo (exclude node_modules / dist / .git) ..."
tar --exclude=node_modules --exclude=dist --exclude=.git `
    --exclude=deploy/repo.tar.gz `
    --exclude=deploy/blog.tar.gz `
    --exclude=deploy/jiangnan-blog.tar.gz `
    --exclude=deploy/node22.tar.xz `
    --exclude=deploy/workbench.tar.gz `
    --exclude=deploy/jiangnan-blog.b64 `
    --exclude=.backup `
    --exclude=dist.tar.gz `
    -czf $Tar .
if ($LASTEXITCODE -ne 0) { throw "tar failed" }

Write-Host "[2/3] scp -> $Vps"
scp -i $Key -o IdentitiesOnly=yes $Tar "${Vps}:${RemoteTar}"
if ($LASTEXITCODE -ne 0) { throw "scp failed" }

Write-Host "[3/3] ssh + bash deploy-code.sh"
ssh -i $Key -o IdentitiesOnly=yes $Vps "bash /home/studio/app/deploy-code.sh"
if ($LASTEXITCODE -ne 0) { throw "deploy failed" }

Write-Host "[OK] deployed."

