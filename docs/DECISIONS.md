# GeoWorld 关键设计决定

> 记录“为什么这样做”。这些决定不是临时聊天结论，后续修改时应明确更新原因。

## D-001：GeoWorld 同时是工具与 Runtime

状态：已确定

GeoWorld Core / World Model 是核心。

外部形态：

- Generator：离线生成、分析、导出。
- Runtime：Minecraft 按需生成。

原因：世界模拟和 Minecraft 接入属于不同问题，不应绑死。

---

## D-002：不绑定单一 Mod Loader

状态：已确定

Loader 只是 Minecraft Adapter。

目标结构：

```text
GeoWorld Model
      ↓
Minecraft Common
 ┌────┼────┐
Neo Fabric Forge
```

---

## D-003：V1 不做球形世界

状态：已确定

Minecraft 保持无限平面坐标。

中央有限区域承担完整地理模拟；外围使用无限特殊海洋。

原因：

- 避免球面投影、极点、首尾接缝复杂度。
- 板块 / 气候模拟仍然可以在有限平面中成立。
- 保持 Minecraft 原始坐标和区块模型。

---

## D-004：外围世界规则

状态：已确定

- 东 / 西：Infinite Storm Ocean。
- 北 / 南：Infinite Frozen Ocean。
- 四角：两种环境自然叠加。
- 外围不是硬 World Border，玩家仍能继续探索。
- 主地理区边缘必须留海洋缓冲，避免大陆直接撞“世界尽头”。

---

## D-005：GeoWorld 生成气候，不模拟实时天气

状态：已确定

GeoWorld 可以计算夏 / 冬甚至 12 个月的气候统计值，但 V1 只输出长期气候和生态。

不负责：

- 四季玩法
- 当前月份
- 每日天气
- 实时区域风暴

未来独立季节 / 天气 Mod 可读取 GeoWorld API。

---

## D-006：世界生成配置创建后不可修改

状态：已确定

影响以下内容的参数只能在创建新世界时设置：

- Seed
- 世界地理规模
- 板块参数
- 海陆比例
- 气候模型
- 水文模型
- 世界外围规则

创建后写入世界自己的 GeoWorld metadata。

已有世界继续生成新区块时必须读取该世界自己的配置，而不是全局默认值。

原因：防止旧区块 / 新区块发生不可逆割裂。

---

## D-007：大陆与板块必须解耦

状态：已实现

错误模型：

```text
一个板块 = 一块大陆 / 一片海洋
```

正确方向：

- 大陆可以跨多个板块。
- 板块可以同时包含海陆地壳。
- 海岸线不等于板块边界。

---

## D-008：噪声不能决定宏观地理

状态：已确定

噪声可以用于：

- 海岸细节
- 山体细节
- 局部高度变化
- 方块级自然感

噪声不负责：

- 大陆位置
- 山系主轴
- 海沟
- 洋中脊
- 主河流骨架
- 气候区

宏观结构必须来自可解释的地理因果链。

---

## D-009：板块构造应该产生“带”，不是一格线

状态：已实现第一版

真实造山带、裂谷、海沟、岛弧都有宽度。

因此边界分类只作为构造源，后续要生成影响场：

```text
boundary source
      ↓
distance falloff
      ↓
tectonic influence band
      ↓
elevation / trench / ridge / volcanism
```

---

## D-010：海洋地壳年龄决定海洋俯冲倾向

状态：已实现

洋中脊生成年轻海洋地壳。

远离洋中脊：

- 地壳逐渐变老。
- 温度降低。
- 密度增加。

海洋-海洋汇聚时，优先让较老的一侧俯冲。

这将用于决定：

- 海沟侧
- 岛弧侧
- 俯冲方向


---

## D-011：构造边界是“源”，地貌必须形成有宽度的影响带

状态：已实现第一版

不再直接把边界 Cell 当成山脉 / 海沟本身。

流程：

```text
Plate Boundary
      ↓
Tectonic Feature Source
      ↓
Distance Falloff
      ↓
Tectonic Influence Band
      ↓
Elevation Field
```

不同构造类型使用不同半径和幅度：

- Collision Orogen：宽、强抬升
- Volcanic Arc：中等宽度抬升
- Island Arc：海洋侧局部抬升
- Trench：窄而强的下沉
- Mid-Ocean Ridge：较宽弱抬升
- Continental Rift：较宽下沉
- Transform Fault：窄且弱

带宽和幅度已经升级为由相对板块速度、地壳厚度、地壳年龄等共同决定；后续仍会继续校准物理关系。


---

## D-012：板块自然化采用连续坐标扭曲，不做逐 Cell 随机扰动

状态：已实现第一版

原始 Voronoi 的优点是：

- 板块连续
- 拓扑简单
- 计算便宜
- 易于解释

缺点是边界过于笔直、几何感明显。

V1 自然化方案：

```text
World Coordinate
      ↓
Low-frequency Smooth Domain Warp
      ↓
Warped Coordinate
      ↓
Nearest Plate Seed
```

不采用：

```text
每个 Cell 独立随机修改 plate id
```

原因是后者容易产生：

- 孤立 Cell
- 板块碎片
- 锯齿边界
- 不稳定的构造带

原始 `RawPlateID` 保留用于调试与算法比较，实际构造计算使用自然化后的 `PlateID`。


---

## D-013：构造强度不能只由“边界类型”决定

状态：已实现第一版

同样是大陆碰撞，慢速轻微挤压与高速强烈碰撞不应产生一样的山脉。

因此 GeoWorld 使用：

```text
Relative Plate Velocity
        ↓
Boundary Strength
        +
Crust Thickness / Ocean Age
        ↓
Tectonic Intensity
        ↓
Landform Width + Elevation Amplitude
```

这让地貌类型和地貌强度分离：

- Feature 决定“是什么”。
- Intensity 决定“有多强”。


---

## D-014：基础海陆高程与构造影响必须分层

状态：已实现第一版

不能把“大陆架、深海盆地”和“山脉、海沟”混成一个公式直接生成。

GeoWorld 采用：

```text
Base Geography
  ├─ Coast Distance
  ├─ Continental Crust Thickness
  └─ Ocean Crust Age
          ↓
     Base Elevation
          +
Tectonic Influence
          ↓
   Final Macro Elevation
```

原因：

- 大陆架 / 大陆坡主要是大陆边缘几何，不应依赖是否正好有板块边界。
- 山脉 / 海沟属于构造异常，应叠加在基础地形之上。
- 后续气候、水文和 Minecraft 细化可以分别读取 BaseElevation、TectonicInfluence 和最终 Elevation。


---

## D-015：丘陵属于 Relief / 侵蚀结果，不属于宏观随机噪声

状态：已确定

GeoWorld 不采用：

```text
noise > threshold → hills
```

丘陵应来自中等尺度起伏：

```text
Tectonic Background
+ Distance to Mountain Belt
+ River Incision
+ Precipitation
+ Terrain Age
+ Erosion Resistance Approximation
        ↓
Relief
        ↓
Hills / Foothills / Valleys / Rolling Plains
```

因此 Phase 1 只建立宏观高程骨架。

最终丘陵放到气候与水文之后的侵蚀 / Relief 阶段生成。

Elevation 与 Relief 必须分离：

- Elevation：这里整体有多高。
- Relief：周围一定尺度内起伏有多强。

高海拔 + 低 Relief 可以是高原；中等海拔 + 中 Relief 才可能是丘陵。

---

## D-016：Minecraft Chunk 不是地形数学边界

状态：已确定

最终地形必须定义为连续世界坐标函数：

```text
height = H(worldX, worldZ)
```

Chunk 只负责采样其覆盖的世界坐标。

禁止：

```text
generateChunk(cx, cz) → 独立随机地形
```

原因：

- 独立 Chunk 生成容易产生接缝。
- 世界坐标采样天然保证相邻 Chunk 使用同一函数。
- 相同坐标无论从哪个 Chunk 请求，都得到相同结果。

最终高度建议分层：

```text
BaseElevation
+ TectonicRelief
+ ErosionRelief
+ RegionalDetail
+ LocalDetail
= H(worldX, worldZ)
```

宏观 Grid 只是采样点，不是方块高度台阶。

从宏观场到 Minecraft 坐标时必须使用连续插值；初期可采用双线性，正式方案优先考虑受限 Hermite / 单调 cubic 等不会产生明显 overshoot 的平滑方法。

---

## D-017：地貌边界使用连续权重混合

状态：已确定

平原、丘陵、山麓、山地等不能按分类边界直接切换地形函数。

应使用连续权重：

```text
Terrain =
  Plain * plainWeight
+ Hills * hillsWeight
+ Mountain * mountainWeight
```

权重随位置连续变化且总和归一化。

这样可以避免：

- 群系墙
- 地形突然变陡
- 宏观 Cell 边缘台阶
- Chunk 边界视觉割裂


---

## D-018：外围海洋缓冲是硬约束，不只是概率偏置

状态：已实现

仅靠“越靠近边缘越不容易生成大陆”的软偏置，极端 Seed 仍可能让陆地碰到模拟边界。

因此 V1 同时采用：

```text
Hard Ocean Guard Band
+
Soft Ocean Edge Bias
```

默认 `EdgeOceanBufferFraction = 0.04`。

硬缓冲保证外围一定为海洋；软偏置保证缓冲区内侧的大陆轮廓仍然自然退入海洋，而不是沿矩形边界被切断。

大陆 Cell 数从合法候选区中精确选取，因此目标 OceanFraction 不因硬缓冲而漂移。


---

## D-019：水文填洼表面与真实地形必须分离

状态：已实现

水文算法需要一个保证可排水的高程表面，但最终世界又必须保留真实洼地和湖盆。

因此同时保留：

```text
OriginalElevation
FilledElevation
FillDepth
```

其中：

- OriginalElevation：真实宏观地形，后续侵蚀 / Runtime 继续使用。
- FilledElevation：仅用于 Flow Direction / Flow Accumulation。
- FillDepth：记录为了建立排水出口临时抬高了多少。

禁止：

```text
OriginalElevation = FilledElevation
```

否则会永久抹平湖盆、内流洼地和未来可形成湖泊的地貌。

LakePotential 应从 FillDepth 与水量供应继续推导，而不是把 Priority Flood 结果直接当最终地形。


---

## D-020：Lake SpillOutlet 是潜在溢流口，不等于实际常年出流

状态：已确定

Priority Flood 为每个封闭洼地建立“达到某个水位后可以从哪里溢出”的拓扑。

因此 Lake 中保存：

```text
SpillOutlet
```

而不是语义更强的：

```text
PermanentOutlet
```

当前水文阶段尚未完整计算：

- 湖面蒸发。
- 地下渗透。
- 土壤蓄水。
- 长期入流 / 出流平衡。

所以不能仅因为存在 SpillOutlet，就断言这个湖实际会持续外流。

未来水量平衡可以在不修改排水拓扑的情况下分类：

- Exorheic / 外流湖。
- Endorheic / 内流湖。
- Seasonal / 季节湖。
- Saline tendency / 盐湖倾向。


---

## D-021：TerrainMaturity 是侵蚀成熟度代理，不是实际地质年龄

状态：已确定

Phase 4 当前没有模拟大陆数亿年的真实演化历史。

因此使用：

```text
TerrainMaturityProxy
```

表达“构造活动较弱、长期削低作用更充分”的倾向。

第一版：

```text
TerrainMaturityProxy
≈ 1 - TectonicSupport
```

禁止把这个字段解释为：

- 年份。
- 百万年。
- 岩石真实年龄。

未来若 GeoWorld 加入显式地质历史，可将该代理替换为真正的地形 / 岩性年龄模型。


---

## D-022：宏观网格分辨率不能隐式改变地貌物理尺度

状态：已实现第一步

GeoWorld 的 Macro Grid 主要承担：

- 世界模型采样。
- Generator 预计算。
- Runtime 插值数据源。

因此：

```text
240 × 150
640 × 400
1024 × 640
```

可以具有不同计算精度，但不应该仅因为 Cell 数变化就把：

- 山系变宽数倍。
- Relief 观察尺度变大数倍。
- 河谷扩散范围变大数倍。
- 山麓覆盖半张大陆。

Phase 4.2 实测发现固定 Cell 半径会造成：

```text
same seed
640×400 mountain ≈ 15%
240×150 mountain ≈ 33%
```

因此所有代表空间尺度的默认 Kernel / Radius 应：

1. 根据宏观网格 / 世界物理尺度解析。
2. 不直接把固定 Cell 数当成物理距离。
3. 显式实验参数可以覆盖默认解析值。

Phase 4 当前已处理：

- ReliefRadius。
- MountainInfluence distance。
- ValleySpread radius / decay。

后续继续审查 Phase 1 / Phase 2 中仍以 Cell 为单位的空间参数，并逐步迁移到统一的 GeoScale / WorldMetric 抽象。


---

## D-023：Regional Terrain 使用世界坐标连续函数，默认采用 Cubic B-Spline

状态：已实现第一版

Phase 4.3 将最终区域地形定义为：

```text
H(worldX, worldZ)
```

而不是：

```text
H(chunkX, chunkZ, localX, localZ)
```

Chunk 只负责查询世界坐标。

为了同时满足：

- 高度连续。
- 坡度连续。
- 不 overshoot。
- 不出现宏观 Cell 台阶。

Regional Terrain 默认使用 Uniform Cubic B-Spline 对宏观控制场采样。

选择 B-Spline 而不是普通 Catmull-Rom 的主要原因：

- B-Spline 基函数非负。
- 权重和为 1。
- 不会在山脊 / 高原附近产生超过控制点范围的插值尖峰。
- 网格内部可达到 C2 连续。

代价：

- 不保证曲线精确穿过每个宏观节点。

GeoWorld 接受这一点，因为宏观 Grid Node 本来就是世界模型控制点，而不是必须保真的方块级高度样本。

Regional Detail 也必须：

- 只读取绝对世界坐标 + Seed。
- 不能读取 Chunk-local 随机状态。
- 振幅受连续 LandformWeight 控制。


---

## D-024：河漫滩、冲积扇与峡谷必须来自水文因果，不保证每个 Seed 都出现

状态：已实现第一版

GeoWorld 不采用：

```text
random point
→ generate canyon / alluvial fan
```

这些地貌必须从已经存在的河流和地形背景推导。

### 河漫滩 / 宽谷底

主要条件：

```text
真实河流
+ 高汇流量
+ 低坡度
+ 低 Relief
→ Floodplain / Wide Valley Floor
```

### 冲积扇

核心触发条件：

```text
upstream mountain / steep memory
+ river
+ abrupt slope reduction
+ sediment supply
→ Alluvial Fan Core
→ downstream directional spreading
```

冲积扇的展开方向必须来自实际 FlowTo，而不是随机角度。

### 峡谷

主要条件：

```text
river
+ high accumulation
+ high Relief
+ steep slope
+ existing incision
→ Canyon
```

峡谷必须叠加在已有河谷因果链上，不能独立挖槽。

### 世界多样性

不要求每个 Seed 必须包含：

- 强峡谷。
- 强冲积扇。
- 同样数量的特殊地貌。

某个世界完全可以：

- 山地较少，因此冲积扇很弱。
- 河流多为平缓大河，因此峡谷很少。

验收只要求：

1. 特殊地貌不会大面积泛滥。
2. 固定多 Seed 验收批次中相关机制不能整体失效。
3. 各字段保持连续、确定、可解释。
4. 接入最终 Terrain 后不得破坏 Chunk / Macro Grid 连续性。


---

## D-025：宏观归一化高程与米制海拔必须分离

状态：已实现第一版

GeoWorld 的宏观 Elevation 是世界形态控制值，不是线性物理米数。

禁止默认：

```text
normalized 0.5..1.0
→ linear 0..6000m
```

原因：

- 普通大陆内部会被解释成过高海拔。
- 气候 lapse rate 会把大量中纬度陆地降温成寒带。
- 后续生态会系统性偏向苔原 / 冰原。

当前气候层单独使用可校准转换：

```text
physicalElevation =
  normalizedHeight ^ ElevationCurvePower
  × MaxLandElevationM
```

默认：

`ElevationCurvePower = 1.70`

端点保持：

- sea level → 0m
- normalized top → MaxLandElevationM

---

## D-026：纬度温度端点与中纬度曲线必须独立校准

状态：已实现第一版

只规定：

- EquatorTempC
- PolarTempC

还不足以定义合理的全球温度场。

旧曲线虽然端点正确，但会把中纬度算得过冷。

因此加入：

`LatitudeTempCurvePower`

默认：

`3.50`

目标：

- 赤道仍保持热带温度。
- 高纬 / 极区仍显著寒冷。
- 中纬度先缓慢降温，再向高纬明显下降。

该参数属于长期气候模型参数，必须写入世界自己的生成配置版本。

---

## D-027：Eco Region 与 Minecraft Biome 分离

状态：已实现 Eco Region 第一版

GeoWorld 不采用：

```text
climate threshold
→ Minecraft biome
```

而采用：

```text
Climate
+ Hydrology
+ Terrain
→ Continuous Ecology
→ Eco Region
→ Minecraft Biome / Surface / Vegetation
```

原因：

- 同一个温带森林可以出现在平原、丘陵、河漫滩和山地。
- Minecraft Biome 是渲染 / 游戏世界分类，Eco Region 是长期生态语义。
- 分离后未来可以替换 Minecraft Adapter，而不改核心生态模型。

Eco Region 代表长期稳定生态，不代表实时天气或当前季节。

---

## D-028：Minecraft Core Profile 与 Loader Registry 分离

状态：已实现第一版

GeoWorld Core 不直接返回具体 Minecraft Registry ID。

Core 返回：

```text
BiomeFamily
TerrainVariant
SurfaceProfile
VegetationProfile
```

原因：

1. Minecraft Registry ID 属于 Adapter / Loader 层。
2. 同一个长期生态区需要根据地形和水文产生不同世界表现。
3. 未来可以在不修改地理 / 生态核心的情况下使用 Vanilla Biome、GeoWorld 自定义 Biome 或第三方 Biome。
4. Minecraft 版本变化不应迫使 Core 改写生态模型。

---

## D-029：特殊河流地貌在 Minecraft Profile 映射时必须保留

状态：已实现第一版

河漫滩、冲积平原、冲积扇和峡谷是前面水文 / 地貌阶段已经推导出的覆盖地貌。

禁止重新把它们和普通平原直接竞争后抹掉。

因此达到明确门槛后：

```text
Canyon
Alluvial Fan
Floodplain
Alluvial Plain
```

优先成为 TerrainVariant。

只有不存在明确覆盖地貌时，才由 Plain / Hill / Foothill / Mountain / Plateau 竞争基础 TerrainVariant。

---

## D-030：地壳、实际海陆与水体必须分离

状态：宏观第一版已实现并接入；最终连续海岸/水体尚待 6.2

Continental/Oceanic 是构造基底；LandMask 是某一地形阶段的实际非海洋范围；湖泊/河流另有水体身份。不能由 crust == Continental 重建 LandMask。

外海从有限模型边界的海平面淹水节点向内部连通。海平面以下但与外海隔绝的低地保留为封闭洼地，是否形成湖泊由水文决定。宏观阶段共享同一结果；最终连续地形还必须再次协调海岸与水体，不能把宏观修正当作全部修复。

Phase 1 的 OceanFraction 当前控制 Oceanic 地壳面积。修正后须分别报告地壳面积比例与实际海洋比例；未来目标海洋面积需单独校准，不能继续声称二者恒等。

## D-031：世界范围与计算精度分离

状态：GridGeometry 与旧采样器校验已实现；旧算法参数迁移待 6.1b

世界范围与原点使用 Minecraft blocks；网格 Width/Height 表示节点数。节点间距 = 范围长度 / (节点数 - 1)。物理米制海拔、归一化地形控制值和 Minecraft Y 必须显式转换，不混用。

首版契约支持 X/Z 分别计算节点间距；现有只有一个 CellSize 的旧采样器必须通过校验或迁移后才能消费非等距网格。所有 Cells 参数逐项迁移，记录参考默认尺度；定义结构体本身不等于已经实现分辨率稳定性。

## D-032：最终水文一致性是独立验收门槛

状态：已确定，待 Phase 6.2 实现

气候/水文使用宏观地形可以保留，但后续平滑与细节不能使最终河床、水面、湖面和生态相互冲突。需要明确河床纵剖面和水面约束，或最终地形上的水文校正。不可把 FilledElevation 直接覆盖真实地形，也不可隐藏逆向连线来通过审计。

分类占比、内部置信度、Chunk epsilon 连续性分别只是对应机制的护栏，不能代替完整世界验收。

## D-033：跨语言与游戏验证前移

状态：已确定验证路径，尚未选择首个 Minecraft 版本/Loader

先用现有 Go Generator 导出模型，由 Java 读取/采样并接一个 Loader 验证；暂不重写全部全球模拟。模型格式带版本、已解析配置、哈希/数值规则与 golden vectors。正式格式冻结前完成跨语言验证。

游戏内自主生成宏观模型的策略保留为后续显式决定。完整 UI、多格式导出、批量世界预生成、多个 Loader 不作为首次游戏验证的前置条件。

## D-034：进度入口与历史记录分开

状态：已执行文档整理

STATUS.md 只描述当前基线、进行中、未解决及下一步。原有逐阶段状态全文归档到 PROGRESS.md；PHASE 文档保存实现细节与实验。所有历史数字标注语境，不因算法变化被删除或当作当前结果。每个开发切片在验证后同步状态并提交。


## D-035：旧 Cell 空间参数必须通过共享参考物理尺度迁移

状态：Phase 6.1b 当前范围已完成；后续新增空间参数继续强制执行

历史算法大量使用 Cell / iteration 表达作用范围。Phase 6.1b 规定：

1. 旧默认参数先解释为旧 128 blocks/node 基线下的物理距离，而不是直接把字段名从 Cells 改成 Blocks。
2. 共享 world.DefaultMacroSpacingBlocks = 128 只用于翻译历史默认值，不规定未来世界必须使用 128 blocks/node。
3. 每个阶段必须接收或保存 GridGeometry，运行时按实际节点间距把物理距离解析成采样偏移。
4. 同一物理世界范围改变 Width / Height 时，已迁移机制的作用距离、坡度量纲和传播距离不得系统漂移。
5. Legacy 手写输入可以暂时回退到 128 blocks/node 以保护旧测试/工具，但正式 adapter 必须传真实 Geometry；统一 Generator 后应集中解析兼容默认值。
6. 对传播算法，迭代次数和每步距离要分开考虑：仅按节点数增加迭代次数不等于物理尺度正确。

当前已按此原则迁移：
- Phase 1 CoastDistance、构造影响半径及相关海岸/海盆阈值。
- Climate OceanDistance / CoastalTempDecay、SST 传播、水汽传播、默认 MoistureIterations 与 Orographic Lift 梯度。
- Erosion Slope / ReliefRadius / LocalRelief。
- Hydrology FillEpsilon、D8 slope、LocalRunoff 物理面积量纲、MinLakeAreaBlocks2。
- Geomorph 上游陡坡/山地记忆衰减与冲积扇扩散范围。
- Landform MountainDistance、MountainInfluence、ValleyRadius、ValleyDecay。
- Phase 1 reference-world plate/continent geometry、edge buffer 与固定节点频率。
- Terrain GridGeometry / ScalarGrid sampling / world Origin；CellSizeBlocks / Origin Config 仅作受控 legacy 兼容。

Phase 6.1b 最终残余审计已完成。剩余 `*Cells` 使用点属于 blocks→当前采样网格的运行时换算、128-block reference 翻译、测试/统计或 legacy 兼容，而不是新的固定节点生成尺度。该决策继续约束 6.2 之后的所有新增空间参数。


## D-036：正式生成入口只有一个共享 Seed / Geometry / SeaLevel 真相源

状态：Phase 6.2 第一切片已实施

正式世界生成必须经过 `internal/generator.GenerateWorld`。Seed、GridGeometry、SeaLevelNormalized 是顶层共享世界契约，不允许阶段级配置静默覆盖。

规则：
1. 顶层 Config 解析一次共享默认值与合法范围；阶段配置只描述阶段自身参数。
2. Climate / Hydrology / Erosion / Landform / Geomorph 若显式提供 SeaLevel，只允许与顶层值一致，否则拒绝。
3. Terrain 的 seed / CellSize / Origin legacy 字段不能成为第二套真相源；正式路径从顶层 Seed / Geometry 派生。
4. Phase 1 正式入口使用 error-returning `GenerateChecked`；legacy `Generate` 仅作兼容 wrapper。
5. WorldModel 必须保存各阶段实际解析后的 Config，后续持久化使用 resolved config，而不是调用方稀疏输入。
6. Consistency Audit 等正式跨阶段验证优先消费统一 Generator，避免多个 CLI 复制调度顺序后逐渐分叉。
7. 当前 context 取消只保证阶段边界可取消；若未来阶段耗时需要更细粒度，必须把 context 继续传入阶段内部，而不是声称已经支持即时取消。

该决定不等于 6.2 水体一致性已完成；它只是为最终海岸、湖面、河床校正建立唯一因果链。


## D-037：海岸控制点、河床纵剖面与水面必须分层约束

状态：约束模型、broad coast application、river/lake bed overlay、Runtime Ground/Water sampler 与 Runtime Profile 均已实施；Phase 6.2 已收口

最终水体协调不能把所有问题都塞进同一个“修正后高度”数组。

规定：
1. `CoastTargetElevation` 是宽域 Terrain 的宏观控制点目标，用于保持外海连通和宏观海陆身份；它可以 raise / carve broad surface control point。
2. 合法 `EnclosedBelowSea` 不因海岸修复被强行抬到海面以上；其是否成为湖泊由 Hydrology 决定。
3. `RiverBedTarget` 是局部 channel bed 纵剖面，必须独立于 broad Terrain control point；修复逆坡时只能 carve，不通过抬高河床制造假下坡。
4. Ocean / Lake 水面是独立 water-surface datum：Ocean=共享 SeaLevel，Lake=Hydrology Lake.SurfaceElevation。
5. 显式 Lake 生成 `LakeBedCeiling`，最终湖床必须低于水面并保留 clearance；当前违反点必须显式暴露，不能把水画在地表上方后忽略。
6. River 水面由 Runtime 层独立求解：基础水深来自 center depth × RiverWeight；若下游进入 Lake/Ocean，则 receiver datum 作为 backwater floor，只增加河道内水深，不抬高 Ground。
7. Constraint target 为 0 mismatch/upslope 只证明约束可解；Phase 6.2 第三/四/五切片已经依次把 CoastTarget、River/Lake Waterbed、Runtime Water Surface 真正应用。`-strict-final` 现在检查最终 Runtime Ocean/Lake/River 语义，Raw Terrain 历史 mismatch 只保留诊断意义。
8. Broad correction 必须保留 raw Terrain 作为诊断基线，不允许覆盖原始 806/608、72/126 等问题数字后声称问题消失；WorldModel 需明确区分 raw Terrain 与 applied CoastTerrain。

原因：宽域地形、河床和水面属于三个不同几何对象。混在一个高度场里会导致修河时整片地形被拉低、修海岸时河谷被填平，或为了通过测试而隐藏真实逆坡。


## D-038：Broad coast correction 使用命中控制点且边界导数归零的连续 overlay

状态：Phase 6.2 第三切片已实施

海岸 broad correction 既必须精确命中 reconciliation 的宏观控制点，又不能因为 correction 本身在 Macro Cell 边界制造新的坡度折线。

因此当前选择 `smoothstep-bilinear correction`：
1. 四角 correction 值按双线性组合，但局部 tx/tz 先经过 smoothstep。
2. 它是 interpolating sampler，宏观节点 correction 精确命中，不像 B-Spline 会把控制点当近似控制值。
3. 权重保持非负且和为 1，修正值不会超出邻域 correction 凸包。
4. smoothstep 在 0/1 端点导数为 0，因此 correction 在每个 Macro Cell 边界两侧一阶导都趋近 0；原 Terrain 的连续坡度承担边界导数，overlay 不额外制造 kink。
5. Raw Terrain 不被覆盖；GenerateWorld 另建 CoastTerrain，再叠加 carve-only Waterbed Ground，最后由 WorldSample 返回独立 Ground + Water。strict-final 验证最终 Runtime，而不是把任一中间高度场冒充最终世界。
6. 海岸 correction 应用后必须重新计算 river/lake bed 约束，因为局部床面依赖 broad terrain，不能复用 raw Terrain 上的旧 target。

该策略用于 sparse broad-surface correction；它不适合作为河道宽度/局部 channel 横断面的最终 sampler，River/Lake overlay 仍需独立几何规则。


## D-039：Waterbed 必须按河网拓扑传播实际出口高度

状态：Phase 6.2 第四切片已实施

RiverBedTarget 的宏观节点单调不等于连续 local channel 必然单调。局部 profile 会采样真实 CoastTerrain；如果上游 segment 中间遇到更低 broad terrain，carve-only profile 会把实际出口压到低于宏观 endpoint target。若下游 segment 仍从旧 target 开始，相邻 corridor 在汇流处叠加后会出现“额外深坑 → 影响衰减 → 向下游回升”。

因此：
1. Waterbed 以 Hydrology/Reconciliation 的有向河网 DAG 顺序构建 segment。
2. 每个 segment 的实际 sampled outlet height 必须传播到 downstream node 的 effective bed。
3. 下游 segment 从所有上游实际 outlet 的最低约束开始，不能只读原始 macro RiverBedTarget。
4. Profile 始终 carve-only；传播只能降低 downstream start，不能抬高任何床面。
5. 正式验收必须沿**实际最终 Waterbed centerline**进行高密度非节点采样；只检查宏观 From/To 节点不够。
6. River corridor 横向边缘使用 smoothstep 回到 broad terrain，保证局部 carve 连续且不制造硬 bank seam。

正式 Seed 在传播前出现 425 / 813 个连续上坡采样步；传播后均为 0。

---

## D-040：Broad coast 必须验证同类 Macro Cell 内部，不只命中四角节点

状态：Phase 6.2 第四切片已实施

Smoothstep correction 精确命中宏观控制点，但 Raw Terrain 本身是连续 C2 场，四角都在海平面正确一侧并不能证明 Cell 内部没有跨回错误一侧。

规则：
1. 四角均为外海的 Cell，内部抽样必须保持低于 shared SeaLevel safety margin。
2. 四角均为正常陆地且不含 EnclosedBelowSea 的 Cell，内部抽样必须保持高于 sea safety margin。
3. 混合海岸 Cell 不做同类约束，其真实 coastline crossing 由连续 Terrain + sea datum 决定。
4. 含 EnclosedBelowSea 的 Cell 不强制抬高，避免破坏封闭低地语义。
5. 发现同类 Cell 内部缺口时，把最大需要的增量/下切量回灌给该 Cell 四个 broad correction 控制点；smoothstep-bilinear 权重和为 1，因此连续场内部获得至少同等安全修正。
6. refinement 后重新定义 CoastTargetElevation 与统计，不允许文档仍引用 refine 前控制点数量。

该护栏首次运行在 Seed 731894205 找到 160 个正常陆地内部采样点低于海平面，最大 deficit 0.0061282；refinement 后两个正式 Seed 同类 Cell 内部违规均为 0。


## D-041：Runtime 水面必须独立于 Ground，并显式处理受纳水体回水

状态：Phase 6.2 第五切片已实施

最终世界不能再用“一个高度值”同时代表地表和水面。正式 Runtime 查询必须分别返回 Ground 与 Water。

规则：
1. `WorldSample` 是 Runtime-facing 最终采样层；至少返回 GroundHeight、WaterKind、WaterSurfaceDefined、WaterSurfaceElevation、WaterDepth。
2. Ocean 水面固定使用 shared SeaLevel，但必须同时满足 continuous ocean eligibility 与 Ground below sea；宏观 `EnclosedBelowSea` 不因高度低就自动成为海洋。
3. 显式 Lake 使用 Hydrology `Lake.SurfaceElevation`；Lake datum 与 lake bed 分离，Ground 必须真实低于水面后才暴露水体。
4. River 基础水深由 `RiverCenterDepthNormalized × RiverWeight` 给出；RiverWeight 同时承担横向 bank taper，不允许把固定水深扩散到整个 broad surface。
5. 若河段最终进入 Lake/Ocean，receiver datum 是该河段的 backwater floor；它只能增加 channel 内 WaterDepth，不能抬高 Waterbed Ground。
6. Receiver detection 不能只依赖 Hydrology FlowTo。实际 Runtime lake footprint 是 square raster footprint，因此 river segment 与该 square 的几何交叠也必须算作 receiver；不能用不一致的 Euclidean circle 近似。
7. River→Lake/Ocean 过渡必须沿最终 Runtime water surface 做非节点连续采样；河床单调并不能自动证明水面单调。
8. Raw Terrain、CoastTerrain、Waterbed Ground、WorldSample Runtime 是四个不同层级。历史 Raw mismatch 数字必须保留，但 final-world pass/fail 只能看 applied Ground/Water。
9. `geoworld-consistency-audit -strict-final` 必须对 Runtime Ocean/Lake/River 语义真实失败/通过，不允许硬编码“未完成”或通过。

正式两个 Seed 在该规则下均达到：
- Runtime Ocean node violations = 0
- Enclosed ocean leaks = 0
- Lake node violations = 0
- River undefined/dry steps = 0
- River water-surface upslope steps = 0


## D-042：Runtime Profile 选择必须服从最终水陆语义，但局部河湖不重写宏观生态

状态：Phase 6.2 第六切片已实施

Ecology / MCProfile 是宏观气候与地貌分类，不应因为最终 block 坐标落进一条局部河道或一个湖泊就重跑整套 Climate/Ecology；但连续 mixed coast 也不能直接使用最近宏观节点，导致 Ocean water 上出现 terrestrial biome/profile。

规则：
1. Runtime Profile 是独立解析层，消费最终 WorldSample 与已有宏观 Ecology / MCProfile。
2. Ocean 坐标必须从所在 Macro Cell 的 ocean corners 中选择 profile；结果必须是 ocean biome family、TerrainOcean、VegetationNone。
3. River / Lake / dry land 坐标从 land corners 选择 terrestrial profile；局部河湖的 WaterKind 独立携带，不能把其生态强制改成 Ocean。
4. 离散 profile 不做数值插值；当前使用 smoothstep-bilinear corner weights，只在目标语义侧选择权重最大的 corner，避免发明不存在的离散中间标签。
5. 若所在 Cell 找不到目标语义侧 corner，必须暴露 SemanticFallback；正式 Runtime audit 要求 fallback=0。
6. SemanticMismatch / Unknown 必须进入 strict-final。
7. mixed coast、外围边界和角落必须使用连续世界坐标审计，不能只验证宏观节点。

两个正式 Seed 已达到 fallback=0、semantic mismatch=0、Unknown=0；外围 8304 个采样点/Seed 全部为 Ocean 且 Profile 一致。

## D-043：持久化只保存 Runtime 最小模型，并把版本/校验作为文件契约

状态：Phase 6.3 第一切片已实施

WorldModel 内存对象用于生成与调试，不等于长期存档 schema。持久化格式必须只保存重建 Runtime 所需的稳定字段。

规则：
1. V1 不序列化 Phase1→Geomorph 全部中间模型；保存 ResolvedConfig、Macro Surface、applied CoastTerrain grids、Waterbed river/lake geometry、最小 Ecology/MCProfile。
2. 文件 manifest 必须含 schemaVersion 与 generatorVersion；任一未知版本都拒绝加载，不能套用当前默认值猜测旧模型。
3. payload 必须有 SHA-256 校验；损坏模型在构建 Runtime 前拒绝。
4. CoastTerrain 采样网格与 Terrain config 是 Runtime 连续高度的正式持久化输入；Waterbed spatial buckets 属于可重建索引，不进入文件。
5. WorldSample 与 RuntimeProfile 是可由持久化字段重建的 Runtime 层，不重复保存派生 cache。
6. V1 Save 是 create-only。目标已存在时必须失败并保留原文件；在定义可靠的 replace/rollback 语义之前，不允许先删除旧文件。
7. 当前外层 gzip+JSON 优先换取跨语言可读性与版本可控性；若未来切换 binary field encoding 必须升级 schemaVersion，不得静默改变 V1。

正式 640×400 文件约 7.63 MB，重开固定 484 样点 0 mismatch。

---

## D-044：首次 Minecraft 验证先走 Vanilla 26.3 Save，Loader 后置

状态：Phase 6.4 第一切片已实施

跨语言 Runtime 已经有稳定 golden 后，首次真实游戏验证不应立刻引入 Fabric / NeoForge / Forge。先输出标准 Vanilla 存档，可以把“GeoWorld 模型是否正确映射为 Minecraft Chunk”与“Loader 生命周期/注入点是否正确”两个问题拆开。

规则：
1. Phase 6.4 首个目标版本固定为 Minecraft Java Edition 26.3；DataVersion=5023，level.dat storage version=19133。版本相关 Chunk/NBT 逻辑必须隔离在 `v26_3` adapter，不污染通用 GeoWorld Java Runtime reader。
2. 先实现标准 NBT + Anvil Region + `level.dat` + `data/world_gen_settings.dat`，生成 Vanilla Preview save；Loader 只有在客户端真实进入 Preview 并完成第一轮地形/水体/群系验收后才开始。
3. 26.3 Overworld 高度映射采用**边界语义**：GeoWorld 0.0→Y=-64，0.5→Y=63，1.0→Y=320；方块地面满足 `y < groundBoundary`，水满足 `groundBoundary <= y < waterBoundary`。Ocean surface=0.5 因而最高水方块为 y=62，first available Y=63。
4. Preview 第一版允许有限半径预生成，范围外继续使用 vanilla 26.3 normal worldgen；边界接缝属于明确已知限制，不通过隐藏扩展预生成范围冒充解决。
5. Mojang 官方 server jar 仅用于开发期 reference verifier，必须校验固定 SHA-1，不进入 Git，也不成为 writer/runtime 依赖。
6. 官方 `LevelVersion/LevelSettings/WorldGenSettings/SerializableChunkData` 全部能解析只证明格式契约成立；**不能把 codec 通过冒充成 Minecraft 客户端真实进入世界已验收**。
7. 首次客户端验收优先观察世界列表识别、出生点、存档损坏/升级提示、地形高度、Ocean/River/Lake、Biome 与可见表层；洞穴、矿物、结构、完整植被与多 Loader 不属于这一切片的完成条件。

当前默认 Preview（Seed 731894205，radius=8）已经由 Mojang 26.3 官方 codec 解析 `level.dat` / `world_gen_settings.dat`，并通过 289/289 Chunk；客户端真实进入世界仍待验收。
