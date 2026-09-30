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
        ↓
WorldModel
  ├─ resolved Config
  ├─ shared Geometry / Surface
  └─ current stage models
```

当前原则：

- Seed / Geometry / SeaLevel 只有顶层一份正式真相源。
- 阶段级重复字段若显式填写，只能用于兼容性检查，不能覆盖顶层共享值。
- `WorldModel.Config` 保存每个阶段最终实际使用的参数，为 Phase 6.3 manifest / persistence 准备。
- 当前 WorldModel 暂时挂完整阶段模型，方便 Phase 6.2 做跨阶段一致性校正；这不是未来持久化格式承诺。
- 正式跨阶段审计逐步改用 GenerateWorld；单阶段 Lab 仍可直接调用模块用于算法实验。
- Context 取消当前位于阶段边界。需要阶段内部即时取消时，context 必须进入相应算法循环。

下一步架构对象将是最终 Terrain / Water reconciliation 数据：它必须同时引用最终连续 Terrain、宏观 Surface、Hydrology Flow/Lake 信息，并产出可被 Ecology/Profile/Runtime 一致消费的最终水陆语义。


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
  └─ River = 后续由河宽/流量/床面规则求解
```

`internal/reconcile` 在宏观节点层把 raw Terrain、Macro Surface、Hydrology 转换成上述约束。Phase 6.2 第三切片已经让 broad coast target 被连续 Terrain sampler 真正消费：`WorldModel.Terrain` 保留 raw Terrain，`WorldModel.CoastTerrain` 挂载 smoothstep correction overlay；correction 命中宏观目标且在 Macro Cell 边界的一阶导趋零，不覆盖原始诊断。

当前两个正式 Seed 的 applied CoastTerrain 已达到 Surface mismatch=0、node error=0，并通过 Chunk / Macro slope 连续性护栏。RiverBedTarget 在 CoastTerrain 上重新求解后仍能达到 target upslope=0，但它和 LakeBedCeiling 仍只是**待应用的局部床面约束**。下一层必须以 carve-only channel/basin overlay 消费这些 target；不能把 river/lake bed correction 混进 broad coast surface。


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
River/Lake local bed overlays（下一切片）
        ↓
Final Runtime Terrain（尚未命名/冻结）
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
Applied ground surface
   + Runtime Water Surface sampler（下一切片）
        ↓
Runtime terrain/water sample
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

仍未冻结为 Runtime 最终接口，因为 River 只有 bed / width，还没有 water depth / water-surface rule。下一层应返回统一样本：
```text
groundHeight
waterKind
waterSurfaceDefined
waterSurfaceElevation
```
其中 Ocean / Lake datum 已明确，River 必须在下一切片定义。
