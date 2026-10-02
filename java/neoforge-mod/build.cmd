@echo off
setlocal

set "GEOWORLD_JDK25=D:\tools\jdk\jdk-25_windows-x64_bin\jdk-25.0.4.1"
if exist "%GEOWORLD_JDK25%\bin\java.exe" goto use_jdk

if defined JAVA_HOME if exist "%JAVA_HOME%\bin\java.exe" goto run_gradle

echo GeoWorld NeoForge 26.3 requires Java 25.
echo Expected local JDK: %GEOWORLD_JDK25%
echo Or set JAVA_HOME to a Java 25 JDK.
exit /b 1

:use_jdk
set "JAVA_HOME=%GEOWORLD_JDK25%"
set "PATH=%JAVA_HOME%\bin;%PATH%"

:run_gradle
echo JAVA_HOME=%JAVA_HOME%
call gradlew.bat build --no-daemon %*
exit /b %ERRORLEVEL%
