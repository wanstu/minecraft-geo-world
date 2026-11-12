# GeoWorld R8.1j — v1.1.26 双 NeoForge 完整构建验收

日期：2026-11-12，dadm（Linux）  
源码基线：`1d8dbb41f750f161efdade94c330623303cda921`  
工作分支：`adm/geoworld-r81f-spawn-height-20261112`  
本轮生成规则：`geoworld-rules-v1.1.26`

**阶段结论：26.2、26.1.2 NeoForge 均已完成真实本地 `clean check build`，无源码错误、水文失败或隔离内存 OOM，JAR 内容验证通过。v1.1.26 当前本地完整构建 7/9 PASS；正式发布继续 HOLD。**

## 1. 资源隔离及首次 NeoForge 构建

dadm 只有约 7.6 GiB 物理内存、2 GiB swap，之前 26.3 NeoForge 首次反编译使用约 4.5 GB RSS，直接构建会影响其他服务。本轮没有强行再次使用无限制 JVM，而是验证 `systemd-run --user --scope`，把 Gradle 及 ModDev 外部反编译进程放入**独立 cgroup**：

```bash
systemd-run --user --scope --quiet \
  -p MemoryMax=5100M -p MemorySwapMax=256M \
  -p MemoryHigh=5000M -p CPUQuota=250% -p TasksMax=256 \
  bash -lc 'JAVA_HOME=/home/wanstu/sdk/jdk/25 \
    JAVA_TOOL_OPTIONS=-XX:ActiveProcessorCount=2 \
    ./gradlew --no-daemon --max-workers=1 clean check build'
```

*上述命令只是说明关键限制值；真实命令还设置 Gradle JVM `-Xmx1100m`、Metaspace 512 MB、低进程优先级、独立日志文件。*

已确认外部反编译器 `ConsoleDecompiler` 实际使用 `-Xmx4096m`。给 cgroup 设置上限后，两个版本的反编译、源码转换和完整原版重编译都成功，整个构建期间没有发生隔离组 OOM，其他开发进程未被终止。

| 资源/原版处理 | 26.2 NeoForge | 26.1.2 NeoForge |
|---|---:|---:|
| 原版反编译 | 214.78 秒 PASS | 215.97 秒 PASS |
| 原版重编译 | 7,055 源文件 PASS | 6,882 源文件 PASS |
| 整套 Minecraft 原版工件准备 | 371.51 秒 PASS | 363.25 秒 PASS |
| cgroup 配置内存最大值 | 5100 MiB | 5100 MiB |
| cgroup 交换空间最大值 | 256 MiB | 256 MiB |
| 隔离组 OOM | 0 | 0 |

26.2 内存采样峰值约 4,937,785,344 字节；26.1.2 在反编译期间确认峰值至少 5,198,778,368 字节。后者是**采样时已观察到的最低峰值**，没有假称取得会话全程准确最高值。

## 2. NeoForge 两目标 `clean check build` PASS

| Minecraft / Loader | JDK | 完整构建结果 | Gradle 任务 |
|---|---|---|---|
| **26.2 NeoForge** | 25 | **BUILD SUCCESSFUL in 11m 4s** | 30 个可执行：29 执行、1 UP-TO-DATE |
| **26.1.2 NeoForge** | 25 | **BUILD SUCCESSFUL in 10m 44s** | 30 个可执行：29 执行、1 UP-TO-DATE |

两项构建均真正执行 `compileJava`、原生引擎构建、模组资源与元数据、Minecraft 群系覆盖与指纹、矿物、Query 契约/生命周期、海岸与水文验证、`check`、`assemble`、`build`。

差异化真实验收事实：

- 26.2 生物群系覆盖：**55/55**；26.1.2：**54/54**。均采样 80,451 点。
- 生物群系指纹：两目标各 **24,576 点 PASS**。
- 两目标 Query：七个历史模型，**34/34 项契约 PASS**，生命周期和并发快照验证通过。
- 两目标矿物：八矿物族矿脉连通与矿石/伴生材料检查 PASS。
- 两目标海岸、河岸、湿地、地表材料和高海拔露头验证 PASS。

**关键水文：两个版本都检查 1,183 区块、302,848 方块列；原 v1.1.25 存在的 77 处“相邻水面自由落差超过 1 格”问题，在两次 v1.1.26 实际 Java 规划器检查中均为 0；其余水源归属、暴露、运行时数据缺失、局部孤峰和湿地漏水异常计数也均为 0。**

以上是各版本 Java 最终方块列规划器的测试结果，**不能冒充对应版本已经单独创建过真实 Minecraft 存档**。R8.1g 的 1.21.1 NeoForge 真实 Anvil 验收仍保留为单独证据。

## 3. JAR 元数据、原生引擎和编译规则

| 本地模块 | JAR 大小 | SHA256 |
|---|---:|---|
| 26.2 NeoForge | 9,157,023 字节 | `6b6375ca4dfcad1ee612365884810863b124fa2048d17307ba5608aa7d6adab7` |
| 26.1.2 NeoForge | 9,157,127 字节 | `a59a6d71e139469f48903c04d2be30b67d91bae725e5df2b238130994205b082` |

独立 JAR 检查确认两份 NeoForge Loader 元数据、六套 Windows/Linux/macOS x64/ARM64 原生引擎完整；读取 `GeoWorldEngineBridge.class` 后确认两个编译产物都持有 `geoworld-rules-v1.1.26`，不再以 v1.1.25 为引擎握手规则。

**注意：** 开发分支的默认 Gradle 模组版本仍为 `0.1.4`，本地输出文件名 `geoworld-0.1.4.jar` **不是历史正式 0.1.4 版本，也不是 v0.1.5 RC 发行包**；本轮没有推送/提供这些测试 JAR。

## 4. 九目标最新状态

本轮将 R8.1i 的 **5/9** 提升为 **7/9 本地完整构建 PASS**。

| Minecraft | Fabric | NeoForge |
|---|---|---|
| 26.3 | PASS | **尚未重试完整构建：上次内存阻塞** |
| 26.2 | PASS | **PASS（本轮）** |
| 26.1.2 | PASS | **PASS（本轮）** |
| 1.21.1 | PASS | PASS |
| 1.20.1 | **尚未构建：dadm 缺 JDK17** | 不在本项目正式九目标矩阵 |

本机目前已确认的七个目标都是基于同一 `geoworld-rules-v1.1.26` 源码的本地完整构建通过。26.3 NeoForge 以前的资源阻塞**没有通过这两个版本自动得到豁免**；仍应在单独资源隔离下重新构建。1.20.1 Fabric 需要先提供 JDK17 工具链。

**正式 GitHub Windows 九目标 CI 未运行；客户端实际游戏相机视觉验收未完成。发布仍 HOLD。**

## 5. 可复验产物

- `docs/acceptance/r81j_neoforge262_v126_full_build.json`
- `docs/acceptance/r81j_neoforge2612_v126_full_build.json`
- `docs/acceptance/r81j_neoforge262_v126_jar_inspection.json`
- `docs/acceptance/r81j_neoforge2612_v126_jar_inspection.json`
- `docs/acceptance/r81j_neoforge262_v126_build_evidence.txt`
- `docs/acceptance/r81j_neoforge2612_v126_build_evidence.txt`
- `docs/acceptance/r81j_two_neoforge_full_builds.json`
- **`docs/acceptance/r81j_current_rules_local_matrix.json`**：当前唯一有效的最新九目标本地构建状态。
- `scripts/test_r81j_neoforge_build_evidence.py`：保证不会将 7/9 本地通过、受阻目标或未执行官方 CI 误报为正式发布。

本轮仅新增验收证据、文档和测试，**不修改地形算法、世界数据、旧规则或原有存档，不合并、不 push、不发布 JAR**。
