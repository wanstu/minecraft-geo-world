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
