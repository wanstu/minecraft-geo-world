param(
    [string]$Model = "",
    [string]$Golden = ""
)

$ErrorActionPreference = "Stop"
$repo = (Resolve-Path (Join-Path $PSScriptRoot "..\.." )).Path
if ([string]::IsNullOrWhiteSpace($Model)) {
    $Model = Join-Path $repo "internal\goldenvectors\testdata\runtime-v1.geoworld"
}
if ([string]::IsNullOrWhiteSpace($Golden)) {
    $Golden = Join-Path $repo "internal\goldenvectors\testdata\runtime-v1.json"
}

$out = Join-Path $repo "output\java-runtime-reader\classes"
Remove-Item $out -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $out | Out-Null

$sources = Get-ChildItem (Join-Path $PSScriptRoot "src\main\java") -Recurse -Filter *.java |
    ForEach-Object { $_.FullName }

& javac --release 21 -encoding UTF-8 -d $out $sources
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& java -cp $out geoworld.runtime.GoldenVerifier $Model $Golden
exit $LASTEXITCODE
