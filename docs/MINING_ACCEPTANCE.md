# 矿区材料与实机验收

更新时间：2026-11-07。当前结论：**原生方块放置回归与 1.21.1 NeoForge + Create 实机验收均已通过**。

## 1. 本轮改动

新增 1.21.1 版本族的 GeoWorldMiningMaterialVerifier，直接调用 GeoWorldV26MineralVeins.populate，写入真正的 Minecraft ProtoChunk，再逐块读取结果。NeoForge 用与现有要塞检查相同的空 ModList 启动方式；Fabric 直接运行同一验收器。两个目标均已纳入 check。

本轮只增加验收器、构建检查和只读坐标工具，没有修改生成实现、字段、API / 生成规则 / 模型 / 缓存版本。Query API v1 的 86 个字段保持原契约。

## 2. 自动验收结果

| 目标 | JDK | 执行 | 结果 |
|---|---|---|---|
| 1.21.1 NeoForge | 21 | check jar | 通过，4m49s |
| 1.21.1 Fabric | 21 | check jar | 通过，2m55s |

两个加载器结果一致：

- 注册表包含 8 类原版矿物和 1 个受控测试家族；同时覆盖 c:ores 与 forge:ores、普通石与深板岩宿主。
- 35 个矿区、140 次主矿/伴生矿及宿主材料映射；测试家族分别出现在 2 个主矿区和 1 个伴生矿区。
- 核对 86,762 个实际区块矿石方块，其中 44,420 个属于测试家族；计数包含 3 个启用矿堆区块和 1 个关闭矿堆区块。
- 核对 815 个矿堆方块，其中 147 个位于水面下；3 个矿堆中心实际保留 5～8 格高度。
- 12 个保护样点（非岩石、空气、水）保持不变，基岩也保持不变。
- 关闭矿堆开关后，地表以上保留空气/水。
- 移除测试家族后，3 个原有主矿/伴生矿分配保留；地下保留宿主，地表不生成替代矿堆。
- 只有矿物字典标签、没有合格宿主的家族不会被当成可放置矿物。

测试家族使用原版下界石英矿和下界金矿作为受控方块，仅在独立测试 JVM 中绑定标签。这验证了“标签 → 冻结矿物家族 → 实际方块”的流程，**不等同于实际加载 Create 后的游戏验收**。没有给 GeoWorld 增加内容模组依赖。

结构化证据：[mining-materials-20261107.json](../acceptance/evidence/mining-materials-20261107.json)。完整日志在 .tmp/mining-material-20261107/。本轮新增检查仅覆盖 1.21.1 两个加载器；其他发布目标仍以 v0.1.3 的既有 CI 记录为准。

复跑（在仓库根目录、JDK 21）：

    java/adapters/1.21.1-neoforge/gradlew.bat -p java/adapters/1.21.1-neoforge verifyGeoWorldMiningMaterials --no-daemon
    java/adapters/1.21.1-fabric/gradlew.bat -p java/adapters/1.21.1-fabric verifyGeoWorldMiningMaterials --no-daemon

## 3. 已读取的真实存档

1.21.1 NeoForge 的 Create 6.0.10 标签列出 create:zinc_ore 与 create:deepslate_zinc_ore；先前实际启动日志也记录了 9 个可用家族（含 zinc）。

| 存档 | Seed | 地表矿堆开关 | 只读模型检查 |
|---|---|---|---|
| 987654321 | 987654321 | 关闭 | 有锌主矿、锌伴生矿和水下矿区；文件字节未变 |
| live-world | -4250254937638407070 | 关闭 | 有锌伴生矿；文件字节未变 |

这两个世界不能用来验证开启矿堆后的地表效果。不要修改冻结参数，也不要把旧区块没有矿堆判为新生成逻辑失败。

已准备私人验收包：

    output/mining-material-acceptance/GeoWorld-0.1.3-dev-mining-acceptance-mc1.21.1-neoforge.jar

SHA-256：70d850ee89f9f03af580c05bd7af2fc8599f97fdd05305cb72bee10e4d154d7b。

已经比较当前安装包与验收包的 7 个关键类（矿物目录、矿脉与矿堆、矿区规划、地形填充、模型读取、模型生成和引擎桥接），字节全部一致。**实机验证可继续使用当前安装包**，本轮没有替换 HMCL 的 JAR。

## 4. 实机验收结果

2026-11-07 已使用 **1.21.1 NeoForge + Create** 创建 Seed `123456` 的新 GeoWorld 世界，锁定模型确认 `surfaceOrePile=true`、规则 `geoworld-rules-v1.1.16`、矿区数 35。实际模型给出的验收点为：

- 锌主矿：`/tp @s 18443 4 9289`
- 锌伴生矿：`/tp @s -5350 15 9569`
- 水下矿区（钻石 + 金）：`/tp @s 961 29 2839`

用户已逐项实机确认：模组矿物正常、主矿/伴生矿对应正常、矿堆堆积形态正常、中心高度 5～8 格正常、水下矿堆正常。**本项人工验收通过。**

随后将新建世界的 `GeoWorldCreationConfig.defaults()` 改为默认开启地表原矿堆；旧序列化配置缺失 `surface_ore_pile` 字段以及兼容构造仍保持关闭语义，不改变旧世界行为。

## 5. 读取实际模型中的验收坐标

只读工具：[MiningTargets.java](../java/verification-tools/MiningTargets.java)。它输出锁定模型中的候选位置与传送命令，不访问游戏区块，不触发生成。

JDK 21+，在仓库根目录运行：

    java --class-path "<GeoWorld JAR>" --source 21 java/verification-tools/MiningTargets.java "<存档/data/geoworld/world.geoworld>" zinc

输出包含 surfaceOrePile、Seed、规则版本，以及 mod-primary / mod-companion / water-floor 候选点。主矿/伴生矿候选仅在模型已冻结对应家族时返回；没有任何矿区分配的旧模型返回空列表。工具使用内部模型类，不属于对外稳定 Query API。
