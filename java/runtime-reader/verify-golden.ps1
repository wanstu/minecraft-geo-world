param(
    [string]$Model = "",
    [string]$Golden = "",
    [switch]$All
)

$ErrorActionPreference = "Stop"
$repo = (Resolve-Path (Join-Path $PSScriptRoot "..\.." )).Path
if ($All -and (-not [string]::IsNullOrWhiteSpace($Model) -or -not [string]::IsNullOrWhiteSpace($Golden))) {
    throw "-All cannot be combined with -Model or -Golden."
}
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

if ($All) {
    $fixtures = @(Get-ChildItem (Join-Path $repo "internal\goldenvectors\testdata") -Filter "runtime-*.json" | Sort-Object Name)
    if ($fixtures.Count -eq 0) { throw "No committed runtime golden fixtures found." }
    foreach ($fixture in $fixtures) {
        $modelPath = [System.IO.Path]::ChangeExtension($fixture.FullName, ".geoworld")
        if (-not (Test-Path -LiteralPath $modelPath)) { throw "Missing golden model: $modelPath" }
        Write-Host "Verifying $($fixture.Name)"
        & java -cp $out geoworld.runtime.GoldenVerifier $modelPath $fixture.FullName
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    Write-Host "Golden fixture verification passed: $($fixtures.Count)/$($fixtures.Count) models"
    exit 0
}

& java -cp $out geoworld.runtime.GoldenVerifier $Model $Golden
exit $LASTEXITCODE
