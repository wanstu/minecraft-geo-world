# GeoWorld R8.1h — v1.1.26 双 Fabric 完整构建验收

日期：2026-11-12（dadm / Linux）  
基线提交：`fa49b91b4a1c622f6191cc61342b443d67d57b98`  
工作分支：`adm/geoworld-r81f-spawn-height-20261112`  
新世界生成规则：`geoworld-rules-v1.1.26`

**阶段结论：本轮两个 Fabric 目标的 `clean check build` 均通过；结合此前已通过的 1.21.1 NeoForge，当前 v1.1.26 本地真实完整构建进度为 3/9。仍不具备整版发布资格。**

## 本轮实际运行结果

| Minecraft | Loader | JDK | Gradle 命令 | 实测结果 |
|---|---|---|---|---|
| 1.21.1 | Fabric | 21 | `clean check build` | **BUILD SUCCESSFUL in 1m 38s**；17/17 项执行成功 |
| 26.3 | Fabric | 25 | `clean check build` | **BUILD SUCCESSFUL in 3m 54s**；41/41 项执行成功 |
| 1.21.1 | NeoForge | 21 | R8.1g 上轮 `clean check build` | **通过**；5m15s、45 项 Gradle 可执行任务，其中 1 项使用缓存 |

两项新 Fabric 构建均使用 `--offline --no-daemon --max-workers=1`，并限制 Gradle Java 内存，均为**本地完整构建**，不是仅编译成功，也不是静态配置检查。

### 1.21.1 Fabric 验证

- Java 与六平台原生引擎编译成功；`verifyGeoWorldAdapter` 通过。
- 生物群系指纹：seed 123456，**24,576 个点一致**。
- 矿物/矿区材料：9 个材料族、35 个矿区，映射和矿堆边界验证通过。
- 模型 Query：7 个历史模型，**34/34 契约**通过；Query 生命周期及并发快照验证通过。
- `check`、`remapJar`、`assemble`、`build` 全部执行通过。

### 26.3 Fabric 验证

- Java、六套原生引擎和跨平台 Tool 产物构建成功；`verifyGeoWorldAdapter` 通过。
- 生物群系 24,576 点指纹通过；矿区及矿脉体积、连通性通过。
- 模型 Query **34/34**、生命周期、结构适配/快照/要塞预览、出生点通过。
- **特殊地貌验证通过**：本次 seed=123456 包括 2 个天坑及其他海岛和火山坑；未再出现曾经的过时平底断言失败。
- 海岸、河岸、湿地、群系表面、WebUI 启停与动态端口回归通过。
- `check`、`assemble`、`build` 全部执行通过。

### 独立 JAR 内容检查

| 本地模块 | SHA256（前 16 位） | 文件字节数 | 验证 |
|---|---|---:|---|
| 1.21.1 Fabric | `0d45cb7b3d37ed1c` | 9,206,178 | Fabric 元数据、六个原生引擎均齐全 |
| 26.3 Fabric | `f9ce8d72b44aceac` | 29,163,663 | Fabric 元数据、六个原生引擎均齐全 |

进一步实际读取两个 JAR 中的 `GeoWorldEngineBridge.class` 常量，确认**编译产物主动要求 `geoworld-rules-v1.1.26`，且不包含原 `v1.1.25` 的引擎握手常量**。不是只在源码中更改了版本号。

**重要：** 上述两个 JAR 的本地默认 Gradle `mod_version` 依旧是 `0.1.4`，文件名均为 `geoworld-0.1.4.jar`。这是未发布开发分支的本地构建文件，**不能当作历史已发布的 0.1.4，也不是可交付的正式 v0.1.5 RC 安装包**。

## v1.1.26 九目标验收矩阵

**本地完整构建 PASS：3/9。**

- PASS：1.21.1 NeoForge、1.21.1 Fabric、26.3 Fabric。
- RESOURCE_BLOCKED：26.3 NeoForge。先前在线初始化首次 Minecraft 反编译导致 dadm 本地 8 GB 内存与 2 GB Swap 不足，已安全停止；本轮未重复执行该高风险任务。不能记为源码编译失败，也不能记为通过。
- NOT_FULLY_CHECKED：26.2 Fabric/NeoForge、26.1.2 Fabric/NeoForge、1.20.1 Fabric。本地没有完成对应新规则完整构建；1.20.1 的 JDK17 在 dadm 上仍未安装。
- 正式 **GitHub Windows 九目标 CI：未运行**，客户端游戏相机地貌观感验收：未完成。

之前 v1.1.25 的两个 Fabric 构建的历史通过记录，现已被本轮 v1.1.26 两次**新的实测通过**替换；没有把不同源码版本混为同一次验收。

## 证据与验收门槛

- `docs/acceptance/r81h_fabric_full_builds_20261112.json`：两份完整构建的任务统计、日志 SHA256、JAR SHA256、编译后新规则字节码判定。
- `docs/acceptance/r81h_fabric1211_v126_build_evidence.txt`、`r81h_fabric263_v126_build_evidence.txt`：构建任务和关键 PASS 行摘录；附原始完整日志 SHA256。
- `docs/acceptance/r81h_fabric1211_v126_jar_inspection.json`、`r81h_fabric263_v126_jar_inspection.json`：Fabric Loader 元数据、原生引擎、JAR 完整性。
- `docs/acceptance/r81h_current_rules_local_matrix.json`：9 个目标的**当前**版本化状态，明确区分 PASS/BLOCKED/NOT CHECKED。
- `scripts/test_r81h_fabric_full_build_evidence.py`：自动保证不能将 3/9 写成 9/9，不能将旧版构建、源码编译或无官方 Windows CI 的状态冒充发布资格。

本轮只执行构建与验收，**没有修改河口水文算法、天坑坡面、旧世界存档或新玩法内容**。

**发布资格继续 HOLD：未合并 master、未推送、未创建 tag、未发布任何 JAR。** 下一个可控验收阶段优先检查 **26.2 Fabric**（先检查离线依赖和构建环境；不把未运行目标标记成功）。正式 Windows CI 应在有授权并具备完整分支后运行。
