# GeoWorld 群系采样优化与岩壁矿脉出露（2026-10-11）

## 反馈及目标
用户实机截图（Seed -563659577384754347）在高山 Y306 看不到矿脉，且此前真实 Java 客户端日志显示区块生成主要耗时为群系计算（4736区块均值：biomes 18.960ms、voxel 7.539ms、shoreline 3.341ms、minerals 0.108ms）。本次不修改任何原版方块/群系/生物/实体注册，不恢复天然噪声洞穴。

## 群系处理
`Minecraft26_3BiomeRealizer` 原流程在同一 (x,z) 从模型重复取样地形及气候（`biomeId` 读一次、`surfaceBiomeId` 又读一次）；区块 `GeoWorldShoreline.ChunkPlan` 本就含该坐标的完整 `RuntimeSample`，却没有复用。

新增 `biomeIdFromSample(model,x,y,z,sample)` 并供 `GeoWorldChunkGenerator.buildTerrain`、`getBaseColumn` 及模型性能基准使用，同时普通 3D `biomeId` 自身的 sample/climate 只读一次。所有原有逻辑（包括 biome coverage 保底、洞穴候选、临海、生态渐变和 56 群系）不变。

正式回归 `fingerprintGeoWorldBiomeDecisions` 采样每颗种子 24,576 个三维点：Seed 123456 指纹 `9f3de994862521fb`、Seed -563659577384754347 指纹 `71bfd1e1968b4db6`，优化前后完全一致。并逐点比较普通解析与传入 sample 的新 API。

开发阶段模型独立基准 48 区块、3 轮预热：群系阶段 0.103ms/区块，旧历史单次基准 2.190ms（不同 JVM/负载、不应当视为可信的定量加速比）。**仍需看新版 Minecraft 客户端 latest.log 的真实区块耗时，并同种子同区域对照。**

## 裸露矿石
旧矿脉只有地下曲线脉，偶然相交地表时受 Y、地形与 1/16 微区筛选影响，超过 Y240 的岩石山顶几乎不会出矿。

保留原有地下矿脉与全部 8 类主世界矿石及紫水晶晶洞，仅增加**稀疏、连续且由地貌决定的浅层岩石矿脉**：
- 按 64x64 格地理锚点，根据模型地表真实高程生成自然小矿脉；大约每 12 个合格山区源格择取一个，候选只允许 Mountain/Foothills 且非水域、陆地。
- Y215 以上只会选择铁/绿宝石；较低区域还可能选择煤或铜。Iron/Emerald 可延伸到 Y319；不允许高山直接出现钻石/红石/金矿等随机点状矿石。
- 真实 voxel 方块必须是 STONE、地表/岩壁外露且附近无水体，不覆盖 GRASS/SNOW/SAND/任何水或自定义结构。矿脉沿空间连续曲线生成，不是任意散点。
- 对应测试种子高山 X1680..2700、Z-2950..-1650：计划出露矿体位置 5495，Y240以上 4926，岩石近表可选 2757，其中顶部岩石表面实际相交候选 692；例如 `2011 297 -2914`。跨区块一致性测试 PASS。此数是 **计划位置**，并非现场逐块验证结果。

## 回归与安装
Gradle `check` 含 `fingerprintGeoWorldBiomeDecisions`、`verifyGeoWorldSurfaceOutcrops`、`verifyGeoWorldMinerals`、`verifyGeoWorldWaterOwnership`、`auditGeoWorldBiomeCoverage` 等。
- 标准模型未改，Engine cache 无需升级；但已生成区块的石头不会自动改成矿石。
- 使用 `java/neoforge-mod/build.cmd` 部署到 HMCL 26.3，注意游戏可能占用旧 JAR；若部署失败不能声称成功。
- 优先使用新建 GeoWorld 世界或未生成区块观察矿脉。
