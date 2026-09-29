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

## 6. 语言策略

暂不锁死。

候选方案：

### 方案 A：核心 JVM 化

Java Core + Minecraft Adapter。

优点是游戏内调用链最短。

### 方案 B：Generator 使用 Go，Minecraft Runtime 使用 Java

Go 擅长独立工具、批量任务和并行模拟；Java 保持 Minecraft 运行时零跨进程依赖。

前提是必须有稳定、版本化的 World Model 数据格式和确定性算法规范。

第一阶段应先用基准测试决定，不以“goroutine 一定更快”作为前提。

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
