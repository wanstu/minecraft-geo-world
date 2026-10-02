param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GeneratorArgs
)

$ErrorActionPreference = "Stop"
$repo = (Resolve-Path (Join-Path $PSScriptRoot "..\.." )).Path
$out = Join-Path $repo "output\java-minecraft-save-writer\classes"
Remove-Item $out -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $out | Out-Null

$sources = @()
$sources += Get-ChildItem (Join-Path $repo "java\runtime-reader\src\main\java") -Recurse -Filter *.java | ForEach-Object { $_.FullName }
$sources += Get-ChildItem (Join-Path $PSScriptRoot "src\main\java") -Recurse -Filter *.java | ForEach-Object { $_.FullName }

& javac --release 21 -encoding UTF-8 -d $out $sources
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Push-Location $repo
try {
    & java -cp $out geoworld.minecraft.v26_3.PreviewWorldGenerator @GeneratorArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
