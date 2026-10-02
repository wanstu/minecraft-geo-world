# GeoWorld NeoForge 26.3

通用 GeoWorld NeoForge MOD。目标是一份 MOD JAR 支持多个不同 GeoWorld 存档，每个存档携带自己的 `.geoworld` 模型。

当前第一切片：

- Minecraft Java 26.3
- NeoForge 26.3.0.40-beta
- Java 25 toolchain
- 直接复用 `../runtime-reader/src/main/java`，不复制 Runtime 数学
- 普通世界没有 GeoWorld binding 时完全不介入
- GeoWorld 世界从 `data/geoworld/geoworld.properties` 读取 binding
- 加载并校验 `data/geoworld/world.geoworld`
- binding / model 不一致时明确失败，不静默退回 vanilla

当前**尚未**注册 GeoWorld ChunkGenerator；这一模块的第一阶段只证明 MOD 生命周期、世界识别与模型加载链路。

## Build

在本目录直接运行：

    build.cmd

Windows 开发机优先使用 `D:\\tools\\jdk\\jdk-25_windows-x64_bin\\jdk-25.0.4.1`；若不存在，则使用已设置的 `JAVA_HOME`。

产物：

    build/libs/geoworld-0.1.0.jar
