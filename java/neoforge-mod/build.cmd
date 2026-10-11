@echo off
setlocal
pushd "%~dp0"

set "GEOWORLD_JDK25=D:\tools\jdk\jdk-25_windows-x64_bin\jdk-25.0.4.1"
if not defined GEOWORLD_HMCL_MODS set "GEOWORLD_HMCL_MODS=D:\games\hmcl\.minecraft\versions\26.3\mods"
set "GEOWORLD_HMCL_JAR=GeoWorld-NeoForge-26.3-0.1.0-dev.jar"

if exist "%GEOWORLD_JDK25%\bin\java.exe" goto use_jdk
if defined JAVA_HOME if exist "%JAVA_HOME%\bin\java.exe" goto run_gradle

echo GeoWorld NeoForge 26.3 requires Java 25.
echo Expected local JDK: %GEOWORLD_JDK25%
echo Or set JAVA_HOME to a Java 25 JDK.
popd
exit /b 1

:use_jdk
set "JAVA_HOME=%GEOWORLD_JDK25%"
set "PATH=%JAVA_HOME%\bin;%PATH%"

:run_gradle
echo JAVA_HOME=%JAVA_HOME%
call gradlew.bat build --no-daemon -Pgeoworld.devBuild=true %*
if errorlevel 1 goto build_failed

set "GEOWORLD_BUILT_JAR="
for /f "delims=" %%F in ('dir /b /a-d /o-d "build\libs\geoworld-*.jar" 2^>nul') do if not defined GEOWORLD_BUILT_JAR set "GEOWORLD_BUILT_JAR=build\libs\%%F"
if not defined GEOWORLD_BUILT_JAR (
    echo GeoWorld build succeeded, but no mod jar was found in build\libs.
    popd
    exit /b 1
)

if not exist "%GEOWORLD_HMCL_MODS%" mkdir "%GEOWORLD_HMCL_MODS%"
copy /y "%GEOWORLD_BUILT_JAR%" "%GEOWORLD_HMCL_MODS%\%GEOWORLD_HMCL_JAR%" >nul
if errorlevel 1 (
    echo Failed to deploy GeoWorld mod to HMCL:
    echo   %GEOWORLD_HMCL_MODS%\%GEOWORLD_HMCL_JAR%
    echo Make sure Minecraft 26.3 is closed, then run build.cmd again.
    popd
    exit /b 1
)

echo.
echo GeoWorld mod deployed to HMCL 26.3:
echo   %GEOWORLD_HMCL_MODS%\%GEOWORLD_HMCL_JAR%
popd
exit /b 0

:build_failed
set "GEOWORLD_BUILD_EXIT=%ERRORLEVEL%"
echo GeoWorld NeoForge build failed with exit code %GEOWORLD_BUILD_EXIT%.
popd
exit /b %GEOWORLD_BUILD_EXIT%
