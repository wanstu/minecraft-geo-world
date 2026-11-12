# GeoWorld v0.1.5 RC — NeoForge 九目标验收阻塞记录

日期：2026-11-12（dadm）  
工作分支：`adm/geoworld-r81f-spawn-height-20261112`  
基线提交：`0516c3a`；规则 `geoworld-rules-v1.1.25`

**阶段结论：正式发布 HOLD。本轮没有修改地形生成算法。** 九目标本地完整构建维持 **2/9 PASS**；1.21.1 NeoForge 检出实际河面断层，26.3 NeoForge 因首次反编译的机器资源不足停止。

## 1. Minecraft 1.21.1 NeoForge：完整构建 FAILED

命令：`./gradlew --offline --no-daemon --max-workers=1 clean check build`，JDK21。总耗时 **5m 1s**，结束 `BUILD FAILED`，失败任务是 `verifyGeoWorldWaterOwnership`。

失败前，Java 编译、跨平台原生引擎、世界群系（53/53）、24,576 点群系指纹、矿物、35 个矿区、海岸线、查询契约与生命周期、要塞/结构等阶段均已执行并通过。**不能因此把整套 `check build` 标为通过。**

最终水域规划验证器基于 `GeoWorldShorelinePlanner.planChunkModel` 检查：

| 项目 | 结果 |
|---|---:|
| 抽样区块 | 1183 |
| 规划方块列 | 302848 |
| 有水方块列 | 158884 |
| 未归属水源 | 0 |
| 水向干地不安全泄出 | 0 |
| 河流水位与 Runtime 预期不一致 | 0 |
| 河流局部水面孤峰 | 0 |
| **相邻水面高度差超过 1 格** | **77（失败）** |

**证据类型：Java 最终方块列规划器的数值结果，尚不是已保存 Minecraft Anvil 文件，更不是游戏内截图。** 不能用本次结果推断 1183 个区块已经在真实世界存档里生成。

### 单点根因定位（可重复）

在种子 `123456`、规则 `v1.1.25` 模型中，相邻两个河道采样点：

| 位置 | 原始河床归一化高度 | 规划水面归一化高度 | 对应 Minecraft 水面 |
|---|---:|---:|---:|
| (10814, 2467) | 0.4999095015 | **0.5000000000** | **Y63** |
| (10815, 2467) | 0.5001826563 | **0.5061826563** | **Y66** |

这两列 `waterKind=River`，均携带 `coastalSeaReceiver=true`。前者河床略低于海平面，命中 `GeoWorldV1.sampleWorld` 中的

```java
worldSampleConfig.coastalBackwaterVersion() >= 1 &&
ground.coastalSeaReceiver() &&
ground.finalHeight() < manifest.seaLevelNormalized()
```

故被固定为海平面。后者河床略高于海平面，回落至上游 `riverBedElevation + centerDepth`，于是出现**相邻 1 格水平距离、3 格垂直水面高差**。

局部独立读取 **144×144** 的方块列规划，检查相邻湿方块：发现 **78 个 River↔River 高差超过 1** 的边缘。这个局部扫描同时检查跨区块边缘，**不能与完整验证器仅统计区块内部的 77 项等同**。两者共同证实存在成片河面水位断层。

版本边界：真实保存的 `v1.1.24` 与本次 `v1.1.25` 模型的 `WorldSample.CoastalBackwaterVersion` 都为 **1**。因此上述特定回水机制并非天坑坡面修改时新增；**但没有对同一坐标/种子完成 v1.1.24 的实际水面回放，不宣称已证明两个版本产生完全相同的水面断层**。

原始可机读证据：

`docs/acceptance/r815_rc_neoforge1211_water_ownership_failure.json`

验证脚本本身**不应放宽 `freeFallWaterFaces == 0` 的发行门槛**。真正的修复应在单独的后续水文版本中，使河流到海洋的连续水面与河床、水流方向一致，避免未连通水域被错误填水；旧版存档的 v1.1.25 语义必须锁定。修复后需对同点及完整扫描、真实 NeoForge 保存世界方块、模型 Go/Java 一致性和相关海岸/河口回归重新验收。

## 2. Minecraft 26.3 NeoForge：构建资源阻塞（不是源码失败）

`java/neoforge-mod` 的首轮 `--offline clean check build` 因本机没有 `net.neoforged.moddev:2.0.147` 插件缓存，在 Gradle 配置阶段失败。随后**在线重试**已正确解析依赖，开始 Minecraft 26.3 原始客户端/服务端下载与 `createMinecraftArtifacts` 反编译。

首次反编译期间，独立 Java 进程 RSS 升至约 **4.5 GB**，dadm 全机 **2 GB Swap 几乎用尽**，有影响其他开发进程的资源风险。已安全取消本次独立构建，进程退出后可用内存恢复；没有终止其他项目的进程，没有修改工作空间外任何文件。

**26.3 NeoForge 目前状态：RESOURCE_BLOCKED，不是源码编译失败，也不是 FULL PASS。** 后续需要资源更充足的机器（或受控反编译内存上限与已缓存工件），优先使用正式 Windows GitHub CI 验证。不能只凭 JDK25/源码结构检查给通过状态。

## 3. 九目标汇总与发布纪律

详见 `docs/acceptance/r815_rc_local_build_matrix_20261112.json`。目前：

- **2/9 完整构建 PASS**：26.3 Fabric、1.21.1 Fabric。
- **1/9 完整构建 FAIL**：1.21.1 NeoForge，河面断层 77 处。
- **1/9 本机资源阻塞**：26.3 NeoForge，首次反编译耗尽可用 Swap，已安全停止。
- **5/9 本轮未完整构建**：26.2 NeoForge/Fabric、26.1.2 NeoForge/Fabric、1.20.1 Fabric（本机尚缺 JDK17）。
- GitHub Windows 官方九目标 CI **未运行**，客户端游戏实际相机验收 **未完成**。

**没有合并 master、没有 push、没有创建发布 tag、没有发布 JAR；旧存档、旧规则和 GeoWorld 真实地形算法均未更改。** 新的水文问题应先独立修复和专项验收，再继续发布门槛，不能绕过失败断言。


## R8.1g 后续修复结果（独立新规则）

上文记录的是 **v1.1.25 的真实失败事实**，本身不得删改。后续已在独立候选
`geoworld-rules-v1.1.26` 修复相邻河面高度的突然切换，
同一 1183 区块 Java 规划器检查中 `freeFallWaterFaces=0`，1.21.1 NeoForge 的本地 `clean check build` 已通过，原缺陷位置独立新 Minecraft 世界的 Anvil 真方块 Y65/Y65。历史 v1.1.25 存档保留旧采样语义，不做迁移。

详细证据：`docs/REALISTIC_GEOMORPHOLOGY_R81G_ESTUARY_CONTINUITY.md`。
九目标仍未完整验收，正式发布仍 HOLD。
