# GeoWorld 架构草案

## 1. 分层

```text
                       GeoWorld Model
                              │
                 ┌────────────┴────────────┐
                 │                         │
          GeoWorld Generator        GeoWorld Runtime
                 │                         │
        可视化 / 预生成 / 导出       Minecraft Common
                                           │
                            ┌──────────────┼──────────────┐
                            │              │              │
                         NeoForge        Fabric         Forge
```

## 2. 核心领域模块

```text
world/
  topology
  coordinates
  seed
  region

tectonics/
  plates
  boundaries
  uplift
  rifting
  volcanism

terrain/
  elevation
  erosion
  sediment
  coastline

atmosphere/
  latitude
  pressure
  prevailing-wind

ocean/
  basin
  currents
  sea-temperature

climate/
  temperature
  precipitation
  monsoon
  aridity

hydrology/
  drainage
  river
  lake
  wetland

ecology/
  climate-classification
  ecoregion
  biome-mapping
```

## 3. 数据尺度

不应该以 16×16 Chunk 为全球模拟基本单位。

建议至少三层尺度：

### Global Grid

负责板块、海陆、风场、洋流等低分辨率全球模型。

### Macro Tile

负责区域降水、水文、生态与局部地貌。

一个 Tile 可以覆盖数百到数千个 Minecraft Chunk。

### Chunk Detail

真正进入 Minecraft 后才生成：

- 方块
- 洞穴
- 植被
- 装饰
- 结构

## 4. 核心数据接口

GeoWorld 的长期稳定接口应当是“数据模型”，而不是 Loader API。

示例：

```text
GeoCell
- elevation
- plateId
- crustType
- crustAge
- temperatureAnnual
- precipitationAnnual
- precipitationSeasonality
- monsoonStrength
- windSummer
- windWinter
- oceanCurrent
- aridity
- drainageBasin
- ecoRegion
```

Minecraft Adapter 再做：

```text
GeoCell → Terrain
GeoCell → Biome
GeoCell → Surface Rules
GeoCell → Features
GeoCell → Structures
```

## 5. Generator 与 Runtime 的关系

两者必须消费同一种确定性世界模型。

同一 Seed + 同一配置：

- Generator 预生成结果
- Runtime 按需生成结果

应当在宏观地理上保持一致。

## 6. 当前语言与验证路径

当前算法已经使用 Go。Phase 6 的首个跨语言验证采用：

Go Generator -> 版本化 World Model -> Java reader/sampler -> 一个 Minecraft Loader。

先验证模型消费与连续采样，不立即把全部全球模拟重写为 Java。需记录解析配置、schema/generator version、坐标/单位、字段编码、哈希及浮点/取整规则；使用跨语言 golden vectors 验证。

游戏内自主生成宏观模型的最终方案（JVM 模拟、预生成模型等）仍待性能和接入结果决定。外部导入世界不代表已经完成玩家在游戏内只输入 Seed 的产品流程。

首次游戏验证提前到完整 Generator UI 和批量世界导出之前。调试中间场与 Runtime 持久化字段分开，避免全部 Lab 数组成为常驻模型。

## 7. 世界配置不可变性

世界生成参数必须绑定到具体 World Model，并与世界存档一起持久化。

建议：

```text
level.dat / GeoWorld metadata
        ↓
GeoWorldWorldConfig
        ↓
WorldModel version + seed + generation parameters
```

Runtime 启动已有世界时：

1. 从世界存档读取 GeoWorldWorldConfig。
2. 校验 World Model / Schema 版本。
3. 使用该世界自己的配置生成后续 Chunk。
4. 不读取“当前全局默认值”覆盖已有世界。

全局设置只代表“下一次创建世界时的默认值”。

如果未来需要改变已存在世界的生成规则，必须进入显式迁移流程，并明确告知可能需要重生成区块；不能通过普通设置页静默改变。

## 8. 跨阶段契约与一致性

- Crust 表示地壳基底，不能等同实际陆海。
- 世界范围/原点以 blocks 表示，网格是节点采样精度，海平面显式共享。
- 任何有空间作用范围的阶段都必须携带 GridGeometry；历史 Cell 默认值统一先通过 128 blocks/node 参考尺度翻译为物理距离，再按当前节点间距采样，不能让网格精度改变地理尺度。
- 外海淹水需连通判定，封闭低地不自动变海洋。
- 宏观模型与最终采样地形允许尺度不同，但最终海岸、河床、水面、湖泊和群系必须一致。
- 统一生成入口必须验证依赖与一致性，而非仅组装多个 Model。
- 完整实施和验收见 PHASE6.md。


## 9. Phase 6.2 当前统一生成入口

当前正式生成因果链：

```text
generator.Config
  ├─ Seed
  ├─ GridGeometry
  ├─ SeaLevelNormalized
  └─ stage configs
        ↓
GenerateWorld(ctx, config, progress)
        ↓
Phase1 → Climate → Hydrology → Erosion → Landform → Geomorph
       → Ecology → MCProfile → Terrain
       → Reconcile → CoastTerrain → Waterbed → WorldSample
       → RuntimeProfile
        ↓
WorldModel
  ├─ resolved Config
  ├─ shared Geometry / Surface
  ├─ raw / applied ground layers
  ├─ Runtime Ground/Water sampler
  └─ Runtime biome/profile resolver
```

当前原则：

- Seed / Geometry / SeaLevel 只有顶层一份正式真相源。
- 阶段级重复字段若显式填写，只能用于兼容性检查，不能覆盖顶层共享值。
- `WorldModel.Config` 保存每个阶段最终实际使用的参数，为 Phase 6.3 manifest / persistence 准备。
- 当前 WorldModel 暂时挂完整阶段模型，方便 Phase 6.2 做跨阶段一致性校正；这不是未来持久化格式承诺。
- 正式跨阶段审计逐步改用 GenerateWorld；单阶段 Lab 仍可直接调用模块用于算法实验。
- Context 取消当前位于阶段边界。需要阶段内部即时取消时，context 必须进入相应算法循环。

Phase 6.2 已完成 Terrain / Water reconciliation、WorldSample 与 RuntimeProfile 的最终消费链。下一步架构对象是 Phase 6.3 的最小可持久化 World Model schema：只保存重建 Runtime 所需的数据、解析配置、版本和校验信息，不把全部 Lab 暂态数组直接序列化。


## 10. Terrain / Water Reconciliation 分层

Phase 6.2 不再把“最终高度”视为唯一水体几何。当前 reconciliation 明确分三层：

```text
Broad Terrain control points
  └─ CoastTargetElevation
       负责外海连通 / 宏观海陆身份

Local channel / basin bed
  ├─ RiverBedTarget
  └─ LakeBedCeiling
       负责河床纵剖面 / 湖盆容水

Water surface datum
  ├─ Ocean = shared SeaLevel
  ├─ Lake  = Lake.SurfaceElevation
  └─ River = center depth × RiverWeight + receiver backwater floor
```

`internal/reconcile` 在宏观节点层把 raw Terrain、Macro Surface、Hydrology 转换成上述约束。Phase 6.2 第三切片已经让 broad coast target 被连续 Terrain sampler 真正消费：`WorldModel.Terrain` 保留 raw Terrain，`WorldModel.CoastTerrain` 挂载 smoothstep correction overlay；correction 命中宏观目标且在 Macro Cell 边界的一阶导趋零，不覆盖原始诊断。

当前两个正式 Seed 的 applied CoastTerrain 已达到 Surface mismatch=0、node error=0，并通过 Chunk / Macro slope 连续性护栏。RiverBedTarget / LakeBedCeiling 已由 carve-only Waterbed 实际消费；Waterbed Ground 再由 WorldSample 叠加独立 Ocean/Lake/River 水面。宽域地形、局部床面与水面仍保持三个不同几何对象。


## 11. CoastTerrain continuous broad-surface layer

Phase 6.2 第三切片后，Terrain 层明确区分：

```text
Raw Terrain
  └─ Phase 4 continuous terrain，保留历史诊断
        +
CoastCorrection macro grid
  └─ smoothstep-bilinear continuous overlay
        ↓
CoastTerrain
  └─ 已满足宏观海岸 broad-surface control points
        ↓
Waterbed Ground
  └─ carve-only River/Lake local beds
        ↓
WorldSample Runtime
  └─ Ground + Ocean/Lake/River water surface
```

CoastCorrection 使用 interpolating smoothstep-bilinear，而不是普通 bilinear 或 B-Spline：普通 bilinear 会给 correction 带来 Macro boundary slope kink；B-Spline 则不能保证 correction 在宏观控制点精确命中。当前 sampler 同时满足 exact target、convex hull 与 correction boundary derivative≈0。

GenerateWorld 保留 Raw Terrain 和 CoastTerrain 两份模型，Reconciliation 在 CoastTerrain 应用后重新采样控制点，并重新构建 LakeBed / RiverBed 约束。这样局部水道约束始终建立在真实 broad base 上，不会使用过期高度场。


## 12. Local Waterbed layer

Phase 6.2 第四切片后，最终地形链进一步分层：

```text
Raw Terrain
   + broad CoastCorrection
        ↓
CoastTerrain
   + local carve-only Waterbed
        ↓
Applied Ground
   + WorldSample water semantics
        ↓
Runtime Ground/Water sample
```

Waterbed 不是第二张 broad heightmap。它只在河道 corridor 与显式 lake basin 附近查询局部几何：
- River：由 RiverBedTarget、实际 CoastTerrain profile、RiverStrength→half-width 共同定义。
- Lake：由 LakeBedCeiling、Lake.SurfaceElevation 和局部 raster footprint 定义。
- 所有床面操作 carve-only，不能抬高 broad terrain。
- River segment 按有向 DAG 构建，并传播 actual sampled outlet height，使汇流口不同 corridor 的重叠不会形成深坑后回升。
- 查询通过 nearest-macro-node bucket 缩小候选集合，Runtime 不需要每个 block 扫描全局河网。

Broad coast correction 另外增加 homogeneous Macro Cell interior refinement。节点 exact-hit 只是必要条件；四角同类的 Cell 内部也必须留在正确 sea-level 一侧。混合 coastline Cell 和 EnclosedBelowSea 不使用该强制规则。

当前 Applied ground surface 已在两个正式 Seed 上满足：
- macro applied coast mismatch = 0
- homogeneous coast interior violations = 0
- river centerline upslope samples = 0
- explicit lake bed violations = 0
- chunk-boundary continuity guard 通过

Phase 6.2 第五切片已冻结当前最小 Runtime Ground/Water 查询接口；Waterbed 仍只负责 Ground 与 receiver metadata，水面由 WorldSample 独立求解。正式两个 Seed 已通过 Runtime strict-final。


## 13. Runtime Ground / Water sample

Phase 6.2 第五切片后，正式 Runtime-facing 查询由 `internal/worldsample` 提供。它不重新生成宏观模型，而是组合已应用的 Ground 与水面语义。

```text
WorldSample.Sample(x,z)
  ├─ GroundHeight
  ├─ WaterKind
  ├─ WaterSurfaceDefined
  ├─ WaterSurfaceElevation
  ├─ WaterDepth
  ├─ OceanEligibility
  └─ RiverWeight
```

分层约束：
- Raw Terrain：保留算法原始连续地形和历史诊断。
- CoastTerrain：只负责 broad coast correction。
- Waterbed Ground：只负责 carve-only river/lake local bed；不能用水位抬高 Ground。
- WorldSample：最后叠加 Ocean / Lake / River 水面，不回写 Ground。

水体语义：
- Ocean = shared SeaLevel，但必须满足 continuous ocean eligibility 且 Ground below sea。
- EnclosedBelowSea 默认不进海；只有 Hydrology 显式 Lake 才产生内陆水面。
- Lake = Hydrology Lake.SurfaceElevation。
- River 基础水深 = center depth × RiverWeight；RiverWeight 向 bank 平滑衰减。
- River 若进入 Lake/Ocean，receiver datum 作为 backwater floor，只提高 channel 内 WaterDepth。
- Receiver detection 同时使用 Hydrology FlowTo 与 river segment / square lake raster footprint 的空间相交，保证几何判定与实际 lake sampler 一致。

`geoworld-consistency-audit -strict-final` 直接验收这一层；Raw Terrain 的历史 coast/river mismatch 不再决定 final-world pass/fail。


## 14. Runtime Profile resolver

Phase 6.2 第六切片把最终 Runtime 链补成：

WorldSample Ground/Water + Macro Ecology / MCProfile → RuntimeProfile.Sample(x,z) → WaterKind + Ground/Water + Biome/Terrain/Surface/Vegetation Profile

RuntimeProfile 不重新计算气候或生态。它解决的是连续坐标与宏观离散标签之间的语义选择：

- Ocean：从当前 Macro Cell 的 ocean corners 选择权重最高的 profile。
- River / Lake / dry land：从 land corners 选择权重最高的 terrestrial profile。
- corner 权重使用与连续采样一致的 smoothstep-bilinear 局部权重，但离散标签本身不插值。
- SemanticFallback 表示目标语义侧没有可用 corner；SemanticMismatch 表示最终选择仍与 Runtime water side 冲突。
- strict-final 要求两个字段均为 0，并要求 Unknown=0。

这样可以同时满足：
- mixed coast 的真实 coastline crossing 决定 Ocean/land 语义；
- 河流/湖泊保留周边 terrestrial climate/ecology，而不是被错误变成 Ocean biome；
- Runtime/Profile 的最终消费接口不需要重新跑全球模型。

Phase 6.2 到此完成。Phase 6.3 将在此 Runtime 契约上定义持久化最小字段，而不是序列化所有中间 Lab 数据。

## 15. Phase 6.3 persisted Runtime model

Phase 6.3 不持久化整个 generator.WorldModel。首版文件只保存可以重建最终 Runtime 的最小有向依赖：

ResolvedConfig + GridGeometry + SeaLevel
  + Macro Surface
  + CoastTerrain scalar grids + height correction
  + Waterbed RiverSegments / LakeNodes
  + minimal Ecology / MCProfile
        ↓
Load
        ↓
rebuild Waterbed spatial buckets
        ↓
WorldSample.Build
        ↓
RuntimeProfile.Build

V1 envelope：
- gzip+JSON，便于 Go/Java 都有成熟标准库处理。
- manifest 独立记录 magic、schemaVersion、generatorVersion、Seed、Geometry、SeaLevel、payload byte count、SHA-256。
- payload checksum 通过后才允许恢复 Runtime。
- schemaVersion 解释字段布局；generatorVersion 解释生成语义。二者职责分开。

保存发布策略：
- 同目录临时文件完整写入并 fsync。
- create-only V1 使用 hard-link publish；目标存在则失败，因此不会出现部分目标文件，也不会破坏已有完整文件。
- 覆盖已有模型属于后续独立 replace protocol，必须明确 crash recovery / rollback，不采用先删除旧目标的方案。

当前 640×400 正式 Seed 的 V1 文件约 7.63 MB，重开 Runtime 484 个固定样点逐字段一致。