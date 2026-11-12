# GeoWorld R8.1k — v1.1.26 九目标本地完整构建验收

日期：2026-11-12；dadm / Debian Linux  
源规则：`geoworld-rules-v1.1.26`  
基线提交：`b2cb831251d45049b1298c895f884c0e383b98a4`  
开发分支：`adm/geoworld-r81f-spawn-height-20261112`  
本轮变更范围：只修正 **26.3 NeoForge WebUI 自动恢复测试的端口断言**，并补齐构建环境、证据、回归测试；**未改地貌/水文/旧存档算法**。

## 结论

**当前规则 v1.1.26 的九个 Minecraft+Loader 目标，本地真实 `clean check build` 已全部完成：9/9 PASS。** 全部是 Linux dadm 上的本地 Gradle 构建，且每一个目标都有对应源版本的历史实际构建日志、最终 JAR 完整性与运行时生成规则字节码核查。

**这不是正式 Windows GitHub Actions 九目标 CI，也不代表客户端实际游戏画面验收通过。** v0.1.5 RC 依旧 **HOLD**，不合并 master、不推送、不发布 JAR。

## 本轮新增通过的两个目标

| 目标 | 本地 JDK / Gradle | `clean check build` | 结果 |
|---|---|---|---|
| **26.3 NeoForge** | JDK25 / Gradle 9.2.1 | **BUILD SUCCESSFUL in 3m 25s** | 41 项：40 执行、1 缓存 |
| **1.20.1 Fabric** | **JDK17** / Gradle 8.10.2 | **BUILD SUCCESSFUL in 5m 29s** | 16 项全部执行 |

26.3 NeoForge 在**本轮第一次**完整构建中，还发生一次 `verifyGeoWorldWebUiLaunch` 测试失败。详见下节。修复测试的错误前提后，完整 `clean check build` 从头重跑，两个 WebUI 验证任务及所有 `check`、`assemble`、`build` 任务均通过。

### 26.3 NeoForge 验证结果

- Minecraft 26.3 原版构建工件经过下载、补丁、源码转换和**7,301 个源文件重编译**。反编译 **209.45 秒**；工件准备耗时 **361.92 秒**。这是与最终正常回归不同的首次构建步骤，不能把这两组耗时加到重跑的 3m25s。
- 实际生物群系覆盖：**56/56**，采样 **80,451** 点；24,576 点指纹 PASS。
- 模型 Query：七个历史存档，**34/34** 项契约 PASS，运行时生命周期 PASS。
- 水文/河岸/湿地/海岸/高山露头、模组加载器公共层隔离检查 PASS。
- 水文最终方块列规划器：**1,183 个区块、302,848 列（其中 158,884 个有水列）**，水源归属未知、危险暴露、>1 格自由落差、运行时水面丢失、河面边界不匹配、局部水位孤峰、湿地漏水 **全部为 0**。
- WebUI 的动态端口测试、看门狗进程异常退出恢复、重启和健康检查 PASS。
- 生成 JAR：**29,156,885 字节**，SHA256 `ed51b95b01091359438ee783bc12081abc49ea2326bf1a161e303477139c94f6`。NeoForge 元数据、六个原生引擎及实际 `GeoWorldEngineBridge.class` 中的 v1.1.26 规则标识均验证通过。

本轮继续使用 `systemd-run --user --scope` 为 ModDev 初次反编译和重新构建设置 **MemoryMax=5100MiB、MemorySwapMax=256MiB**，并限制 CPU 和 Gradle 并发，没有再次触发资源阻塞。构建时没有终止其他服务。

### 1.20.1 Fabric 验证结果

原先 dadm 不存在 JDK17。本轮为用户级 SDK 添加 **Eclipse Temurin 17.0.21+9**：

`/home/wanstu/sdk/jdk/17`

下载自 Adoptium 官方二进制服务，并用官方给出的 SHA256 检验归档：
`8b9b611d917e3dfd8419d3566b5949226f69b052fed7e109c1368add1dfe8711`。

- Gradle 8.10.2 / Fabric Loom 1.7.4 / Java 17 完成 `clean check build`，**16/16 项执行 PASS**。
- 编译了 1.20.1 的旧版 ChunkStatus 适配源码及共用 Runtime Reader；
- 构建了六套原生引擎，执行 `verifyGeoWorldAdapter`；
- Query 模型契约：**34/34**，生命周期及并发快照 PASS；
- `remapJar`、`check`、`assemble`、`build` PASS。
- JAR：**9,177,467 字节**，SHA256 `73c6f24c7a1aece17a8cc55a452005a0d527c619768255e30eca0594ff6f0c24`。Fabric 元数据、六平台原生引擎以及编译后 v1.1.26 规则身份均通过检查。

## WebUI 恢复测试的具体缺陷及修复

首轮 26.3 NeoForge 构建：**BUILD FAILED in 11m 25s**，唯一失败项：

`verifyGeoWorldWebUiLaunch`：

`GeoWorld WebUI watchdog should reuse the previous port when it is available: first=34821 recovered=50063`

单独执行原测试，稳定复现：`first=58769 recovered=52351`，`BUILD FAILED in 13s`。

原因不是 GeoWorld WebUI 恢复失败。原测试通过 Java 直接关闭原生 Go WebUI 进程后，HTTP 健康请求和会话连接会留下 TCP **TIME_WAIT**；此时 Java 的不允许复用地址绑定探针报告旧端口不可用。实际在 dadm 上，用 `ss` 看到旧端口 `58769` 处于 `TIME_WAIT`；Python socket 测试 `SO_REUSEADDR=false` 返回 `EADDRINUSE`，而 `SO_REUSEADDR=true` 可绑定。WebUI 本身按设计选择**另一个可用的动态端口**，启动新进程且 HTTP 服务可用。

项目支持动态端口回退，因此测试不应对**刚刚被强制关闭**的原端口作无条件复用断言。

本次**只修改** `java/neoforge-mod/src/main/java/geoworld/neoforge/client/GeoWorldWebUiLaunchVerifier.java`：

- 仍要求看门狗检测到旧进程死亡，恢复进程 PID 与旧 PID 不同；
- 验证恢复后的 URI、端口与进程状态一致；
- **新增真实 HTTP GET `/api/status` 返回 200** 的恢复后健康验证；
- 容许旧端口尚未可绑定时切换至新的动态端口；
- 原有停止后清理状态、再次启动和被占用 8788 端口的回退测试仍保留并执行。

**没有修改 WebUI 生产启动逻辑、恢复机制、网络端口选择逻辑、地貌生成、模型版本或旧世界读取。**

修正后单独运行 `verifyGeoWorldWebUiDynamicPort verifyGeoWorldWebUiLaunch`，**BUILD SUCCESSFUL in 23s**，并在最后 NeoForge 26.3 完整 `clean check build` 中再次通过。

证据：
`docs/acceptance/r81k_neoforge263_webui_timewait_repro.json`

## 九目标最终本地构建矩阵

| Minecraft | Fabric | NeoForge |
|---|---|---|
| **26.3** | PASS（R8.1h） | **PASS（R8.1k）** |
| **26.2** | PASS（R8.1i） | PASS（R8.1j） |
| **26.1.2** | PASS（R8.1i） | PASS（R8.1j） |
| **1.21.1** | PASS（R8.1h） | PASS（R8.1g） |
| **1.20.1** | **PASS（R8.1k）** | 非当前发布目标 |

**9/9 是相同地貌运行时生成规则 v1.1.26 的真实本地完整构建通过。** 所有已构建目标的 JAR 默认 Gradle 模组版本仍是 `0.1.4`，是开发分支产物，**不是正式发布的历史 0.1.4 或 v0.1.5 RC**。

## 证据、剩余发布门槛

- 本地构建完整状态：`docs/acceptance/r81k_current_rules_local_matrix.json`
- 新增两目标构建结果：`docs/acceptance/r81k_all_nine_local_full_builds.json`
- 26.3 NeoForge：`docs/acceptance/r81k_neoforge263_v126_full_build.json`、`r81k_neoforge263_v126_jar_inspection.json`、`r81k_neoforge263_v126_build_evidence.txt`
- 1.20.1 Fabric：`docs/acceptance/r81k_fabric1201_v126_full_build.json`、`r81k_fabric1201_v126_jar_inspection.json`、`r81k_fabric1201_v126_build_evidence.txt`
- WebUI 初次失败、独立复现、修正后独立和完整测试 SHA256：`docs/acceptance/r81k_neoforge263_webui_timewait_repro.json`
- 证据回归测试：`scripts/test_r81k_nine_local_builds.py`

**仍然不成立的结论：**
- 官方 GitHub Actions **Windows 九目标 CI 尚未运行**；
- Minecraft 客户端实际游戏相机视觉验收尚未完成。R8.1g 1.21.1 NeoForge 的 1517/1517 Anvil 方块列专项验收和实际游戏画面是两种不同的证据；
- 尚未完成正式发布的版本号/元数据、tag、合并、推送、发布文件；
- 因此 **v0.1.5 RC 仍保持 HOLD**，不得仅凭本地 9/9 构建通过就发布。
