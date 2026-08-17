# ============================================================
#  Frontend code -> VPS one-click deploy
#
#  Usage (PowerShell, from project root):
#      .\deploy\pull.ps1
#
#  What it does (pure linear, no defensive checks):
#      1. tar the project (excluding node_modules / dist / .git)
#      2. scp to VPS: /home/studio/app/repo.tar.gz
#      3. ssh + bash /home/studio/app/deploy-code.sh
#
#  Prereqs (already done once):
#      - VPS has deploy-code.sh at /home/studio/app/
#      - Local ssh key at C:/Users/LUOYN/.ssh/studio.pem
# ============================================================

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$Vps = "ubuntu@49.232.38.216"
$Key = "C:/Users/LUOYN/.ssh/studio.pem"
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

Write-Host "[OK] deployed. Visit http://49.232.38.216/ to verify."
