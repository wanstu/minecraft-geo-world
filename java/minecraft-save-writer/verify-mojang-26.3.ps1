param(
    [string]$ServerJar = "",
    [string]$World = "",
    [int]$Radius = 8,
    [ValidateSet("any","qa","survival")]
    [string]$Mode = "any"
)

$ErrorActionPreference = "Stop"
$repo = (Resolve-Path (Join-Path $PSScriptRoot "..\.." )).Path
if ([string]::IsNullOrWhiteSpace($ServerJar)) {
    $ServerJar = Join-Path $repo "output\minecraft-26.3-reference\server.jar"
}
if ([string]::IsNullOrWhiteSpace($World)) {
    $World = Join-Path $repo "output\minecraft-26.3-preview\GeoWorld-Preview"
}
$ServerJar = (Resolve-Path $ServerJar).Path
$World = (Resolve-Path $World).Path
if ($Radius -lt 0 -or $Radius -gt 31) { throw "Radius must be in 0..31" }

$expectedServerSha1 = "33680f5f2ac32864d6d7cf5e56a705fdb3e05f4c"
$actualServerSha1 = (Get-FileHash -Algorithm SHA1 $ServerJar).Hash.ToLowerInvariant()
if ($actualServerSha1 -ne $expectedServerSha1) {
    throw "Unexpected 26.3 server jar SHA-1: got=$actualServerSha1 want=$expectedServerSha1"
}

$cache = Join-Path $repo "output\mojang-26.3-verifier\server"
Remove-Item $cache -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $cache | Out-Null

Push-Location $cache
try {
    & jar xf $ServerJar
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} finally {
    Pop-Location
}

$version = Get-Content (Join-Path $cache "version.json") -Raw | ConvertFrom-Json
if ($version.id -ne "26.3" -or [int]$version.world_version -ne 5023) {
    throw "Expected official Minecraft 26.3 server jar (DataVersion 5023), got id=$($version.id) world_version=$($version.world_version)"
}

$versionLine = Get-Content (Join-Path $cache "META-INF\versions.list") |
    Where-Object { $_ -match "`t26\.3`t" } |
    Select-Object -First 1
if (-not $versionLine) { throw "26.3 inner server jar entry not found" }
$parts = $versionLine -split "`t"
$inner = Join-Path (Join-Path $cache "META-INF\versions") $parts[2]
if (-not (Test-Path $inner)) { throw "Inner server jar not found: $inner" }
$libraries = Get-ChildItem (Join-Path $cache "META-INF\libraries") -Recurse -Filter *.jar |
    ForEach-Object { $_.FullName }

$mainOut = Join-Path $repo "output\java-minecraft-save-writer\classes"
Remove-Item $mainOut -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $mainOut | Out-Null
$mainSources = @()
$mainSources += Get-ChildItem (Join-Path $repo "java\runtime-reader\src\main\java") -Recurse -Filter *.java | ForEach-Object { $_.FullName }
$mainSources += Get-ChildItem (Join-Path $repo "java\minecraft-v26_3-common\src\main\java") -Recurse -Filter *.java | ForEach-Object { $_.FullName }
$mainSources += Get-ChildItem (Join-Path $PSScriptRoot "src\main\java") -Recurse -Filter *.java | ForEach-Object { $_.FullName }
& javac --release 21 -encoding UTF-8 -d $mainOut $mainSources
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$referenceOut = Join-Path $repo "output\mojang-26.3-verifier\classes"
Remove-Item $referenceOut -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $referenceOut | Out-Null
$compileCp = [string]::Join(';', @($inner, $mainOut) + $libraries)
$referenceSource = Join-Path $PSScriptRoot "src\reference\java\geoworld\minecraft\reference\Mojang26_3Verifier.java"
& javac --release 25 -encoding UTF-8 -cp $compileCp -d $referenceOut $referenceSource
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$runCp = [string]::Join(';', @($referenceOut, $inner, $mainOut) + $libraries)
Push-Location $cache
try {
    & java -cp $runCp geoworld.minecraft.reference.Mojang26_3Verifier $World $Radius $Mode
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
