# GeoWorld 区块加载性能：测量与第一轮低风险优化（2026-10-11）

## 现象与边界
用户反馈客户端大片黑白空白区、树木/地表先后显现和新区块长时间不可见。截图只能证实“加载/显示滞后”，无法仅靠图片区分服务端区块生成、磁盘 IO、客户端渲染/光照、Java GC 和移动速度。

## 模型侧可重复基准

Gradle 任务：`./gradlew.bat benchmarkGeoWorldChunkPlanning`，使用 Seed 123456 默认 GeoWorld 模型，48 个固定区块位置，丢弃3轮预热，只计量：
- `GeoWorldShoreline.planChunk`
- 每个区块256列 `Minecraft26_3BiomeRealizer.biomeId`
- `GeoWorldMineralVeins.plan`

第一次修改前测量（每区块均值）：shoreline=3.059ms、biomes=2.190ms、minerals=0.159ms。
修改群系保底空间桶后的第二次测量：shoreline=4.364ms、biomes=3.622ms、minerals=0.295ms。两次都运行在独立 Gradle JVM，不保证 CPU/JIT/GC/背景工作负载一致，因此**不能据此声称加载加快或变慢**；需结合客户端端到端计时。

## 修改

- RareBiomeCoverage：建立256方块空间桶，BiomeRealizer调用保底群系时只检查附近桶，不再每个方块无条件扫描全部56个保底中心。桶覆盖候选圆的整个包围盒，保留原半径和Y有效区间。
- BlockAt：普通岩层超过浅层深度3之后直接选 stone/deepslate，只有恶地在50格内需要按Y计算分层陶瓦；此前数万石块仍会反复查询群系材质。
- MineralVeins：全部随机参数照原顺序抽取，只有完全远离目标区块的矿脉提前跳过体素化循环，矿脉形状和空间坐标不变。
- `GeoWorldChunkPerformance` 已调整为开发版自动启用（`build.cmd` 内的 `-Pgeoworld.devBuild=true` 生成构建资源）；正式版普通 Gradle 构建默认关闭。显式 JVM 参数 `-Dgeoworld.profileChunks=true/false` 始终优先，便于手动开关。只统计 GeoWorld 生成器的区块，不影响普通世界。日志写入 Minecraft 的 `latest.log`，使用统一的 `GeoWorld generated` 前缀，首次累计 32 区块，再在 128 区块倍数检查是否输出，且两条日志至少间隔 30 秒，避免刷屏。日志提供 `model, shoreline, biomes, voxel, minerals, geodes` 的平均 ms/区块。
- 增加 `verifyGeoWorldChunkPerformanceMode` 到 `check`：分别对开发/正式构建产物中的嵌入标识校验默认计时状态，杜绝后续 release 意外默认开启。
- 新增非常稀少的矿脉裸露：仅当已有矿脉恰好接触 Mountain/Foothills 的天然石头表面时，按固定 8×8 岩面微区 1/16 的概率允许煤/铁/铜/绿宝石出露；不能穿透雪、草、泥、海水及岸线。地下矿脉原分布不变，矿脉总量可能因露头略增。
- 不增加洞穴密度，不启用新的地下水洞穴模型，不改变 surface/water ownership 或群系覆盖。

## 验证

必须通过 NeoForge build/check：BiomePreset 56/56、BiomeCoverage 56/56、Mineral 8类/16变体和跨区块一致性、Shoreline、WaterOwnership、Engine、Spawn。测试通过只说明内容没有明显回归，不等于客户端加载速度已验收。

## 客户端验收

1. 相同 Seed/相同世界设置，独立新建世界两次；记录加载世界至完整显示画面的秒数。
2. 关闭着色器（如有），固定视距/模拟距离，对比慢速飞行、快速飞行中区块填充速度。确认是否客户端卡顿还是模型生成慢。
3. 使用最新版 `build.cmd` 部署开发版即可自动生成日志，无需修改 HMCL JVM 参数。进入 GeoWorld 新生成至少约 32 区块后，检查 `D:\\games\\hmcl\\.minecraft\\versions\\26.3\\logs\\latest.log` 中的 `GeoWorld generated ...`（游戏重启时也可能已轮转到 `2026-...log.gz`）。如需临时关闭，仅追加 `-Dgeoworld.profileChunks=false`。如需在正式版手动打开，使用 `-Dgeoworld.profileChunks=true`。若 `voxel` 高优先检查方块逐层写入/Heightmap；若 `shoreline` 高考虑分块缓存；若生成耗时低但可见区块慢，检查光照与渲染。
4. 原版洞穴雕刻器目前在装饰配置中“保留”，但 GeoWorld 的实际自定义区块是否有稳定的天然洞穴**尚未作专项空洞审计**。不应把“缺少洞穴”误认为性能优化成功。后续遵循 docs/UNDERGROUND_WATER_AND_SPARSE_CAVES.md 的低密度方案。
