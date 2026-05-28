Param(
    [string]$Target = "all",
    [string]$Arch = "amd64"
)

$OutDir = Join-Path -Path $PSScriptRoot -ChildPath "..\bin"
if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Path $OutDir | Out-Null }

switch ($Target) {
    'all' { $targets = @('windows','darwin','linux') }
    default { $targets = @($Target) }
}

foreach ($t in $targets) {
    $ext = ''
    if ($t -eq 'windows') { $ext = '.exe' }
    $outfile = Join-Path $OutDir "hi-$t-$Arch$ext"
    Write-Host "Building $t/$Arch -> $outfile"
    $env:GOOS = $t
    $env:GOARCH = $Arch
    go build -o $outfile .
}

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Write-Host "Build finished. Binaries are in: $OutDir"
