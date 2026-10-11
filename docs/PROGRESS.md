# GeoWorld 历史进度档案

2026-09-29 从原 STATUS.md 完整归档。下文的“当前/下一步”属于历史语境；最新入口是 STATUS.md。所有旧指标均保留，后续算法改变不得把这些历史数字当作当前输出。

---

# GeoWorld 项目状态

> 本文件是项目进度的单一入口。每完成一个阶段性功能，应同步更新本文件。

更新时间：2026-09-29

## 当前阶段

**Phase 5 — 生态区与 Minecraft 映射**

当前子阶段：**Phase 6.1 — Generator Core / World Model 串联（准备开始）**

## 当前稳定基线

最近提交：

- `d238299` — add fluvial depositional geomorphology
- `151df07` — simplify generator prototype navigation and explanations
- `5b7a1e0` — add continuous regional terrain sampling
- `6545409` — add continuous landform weights and scale aware terrain context
- `d626e6a` — add erosion drivers valley incision and relief
- `8340ed4` — add river hierarchy seasonal flow and explicit lakes
- `bbf3890` — add hydrology basins runoff and river network
- `aca7269` — add seasonal precipitation and monsoon rainfall
- `aa94199` — add seasonal thermal contrast and monsoon winds
- `59e9d32` — add moisture transport precipitation and rain shadow
- `879cd5a` — propagate ocean thermal anomalies onto land
- `f73c13a` — add wind driven surface currents and SST transport
- `6877c54` — add phase2 temperature pressure and wind climate model
- `97796a5` — close phase1 with hard ocean buffer and audit

## 已完成

### 产品 / 架构

- GeoWorld 定位为“地理世界生成引擎”，不是单一 Mod 或单一工具。
- 双形态：
  - GeoWorld Generator：游戏外生成、检查、预生成、导出。
  - GeoWorld Runtime：Minecraft 游戏内按需生成。
- Loader 通过 Adapter 隔离，未来可支持 NeoForge / Fabric / Forge。
- V1 世界拓扑：
  - Minecraft 坐标理论无限。
  - 中央有限 GeoWorld 地理区。
  - 东西最终进入 Infinite Storm Ocean。
  - 南北最终进入 Infinite Frozen Ocean。
- GeoWorld 负责长期气候，不负责实时四季 / 天气。
- 世界生成参数只在创建新世界时生效，并写入世界存档；已有世界只读。

### Phase 1.1 — 最小构造链路

状态：完成

- Seed → 板块中心和速度
- 汇聚 / 张裂 / 转换边界
- 大陆 / 海洋地壳与板块解耦
- 海洋面积按实际 Cell 控制
- 初步构造地貌分类

### Phase 1.2 — 地壳年龄与构造影响带

状态：完成

- 海洋地壳年龄场
- 洋中脊年轻、远离后逐渐变老
- 海洋-海洋汇聚时较老一侧优先俯冲
- 海沟侧 / 岛弧侧区分
- 大陆火山弧 / 海沟区分
- 构造影响从一格边界扩展为有宽度的带
- 海洋地壳老化导致海盆逐渐加深
- 自动测试覆盖确定性、海洋面积、年龄范围、影响带宽度

### Phase 1.3 — 板块边界自然化

状态：完成第一版

- 保留原始 Voronoi 作为调试骨架。
- 实际板块归属使用平滑 Domain Warp 后的查询坐标。
- 板块边界变得弯曲和不规则。
- 不做逐 Cell 随机抖动，避免碎片。
- 当前测试 Seed Plate Warp 变化比例：6.21%。

### Phase 1.4 — 地壳与构造参数深化

状态：**完成第一版**

已实现：

1. 大陆地壳厚度场：大陆边缘较薄，内部逐渐增厚。
2. 板块相对运动速度进入 Boundary Strength。
3. Boundary Strength + 地壳厚度 + 海洋年龄共同形成 Tectonic Intensity。
4. 强碰撞形成更高、更宽的山系；弱碰撞影响更小。
5. 海沟、岛弧、裂谷、洋中脊的带宽与幅度不再完全固定。
6. Generator 新增 `crust-thickness.png` 与 `tectonic-intensity.png`。
7. 自动测试增加地壳厚度变化与构造强度变化检查。

当前测试 Seed：

- Ocean Area：68.00%
- Ocean Age：0.000 ～ 0.908
- Continental Thickness：0.440 ～ 0.825
- Tectonic Intensity：0.089 ～ 0.920
- Tectonic Influence：约 15.24% Cell

## 当前输出

- `plates-raw.png`
- `plates.png`
- `crust.png`
- `crust-thickness.png`
- `coast-distance.png`
- `base-elevation.png`
- `terrain-zones.png`
- `boundaries.png`
- `tectonics.png`
- `features.png`
- `ocean-age.png`
- `subduction.png`
- `tectonic-intensity.png`
- `tectonic-influence.png`
- `elevation.png`

## Phase 1.5 — 基础高程场

状态：**完成第一版**

已实现：

- 有符号海岸距离场：大陆为正、海洋为负。
- 大陆内部基础高程随海岸距离与大陆地壳厚度变化。
- 海洋基础高程明确分为大陆架 → 大陆坡 → 深海盆地。
- 深海盆地继续受海洋地壳年龄控制。
- 基础高程与构造影响分层：先有海陆地形，再叠加山系 / 海沟 / 裂谷等构造。
- 构造影响记录主导 Feature，用于宏观地形分类。
- 宏观地形分类：
  - Continental Interior
  - Continental Coast
  - Continental Shelf
  - Continental Slope
  - Ocean Basin
  - Mountain Belt
  - Plateau
  - Trench
  - Island Arc
  - Mid-Ocean Ridge
  - Rift Valley
- 高原判定已收紧为“厚地壳 + 广域构造抬升”，避免把普通大陆内部误判成高原。
- 新增自动测试：海岸距离正负号、大陆架/大陆坡/海盆高程顺序、核心地形分区存在性。

当前测试 Seed：

- Elevation：0.127 ～ 0.915
- Continental Shelf：3.27%
- Continental Slope：5.66%
- Ocean Basin：55.28%
- Mountain Belt：1.48%
- Plateau：0.43%
- Trench：0.31%
- Island Arc：0.10%
- Mid-Ocean Ridge：3.37%
- Rift Valley：3.86%

新增输出：

- `coast-distance.png`
- `base-elevation.png`
- `terrain-zones.png`

## Phase 1.6 — Phase 1 验收

状态：**完成**

已完成第一轮自动验收：

- 8 个固定 Seed 全部通过。
- 所有 Seed 的海洋面积保持目标值。
- 所有最终高程均无 NaN / Inf，且保持在 0..1。
- 所有测试 Seed 的地图四条模拟边界均保持海洋，未出现大陆贴边。
- 同 Seed 确定性测试继续通过。

当前性能基线（Windows / i5-11260H，单次 benchtime=1x，仅作开发参考）：

- 640 × 400：约 147 ms，约 66 MB 临时 / 模型分配。
- 1024 × 640：约 375 ms，约 168 MB 临时 / 模型分配。

注意：当前 Lab 同时保存大量调试字段和图层数据，最终 Runtime World Model 会单独设计紧凑存储，不能直接把这里的内存数字视为 Mod 常驻开销。

新增多 Seed 静态视觉验收页：

- `prototype/acceptance.html`
- Seed：1 / 42 / 2026 / 731894205
- 每个 Seed 只展示：海陆骨架、构造影响、宏观地形分区、最终高程。
- 最终高程顶格占比最高约 0.927%，目前没有出现大面积高程饱和。
- 已增加回归护栏：最终高程顶格比例不得超过 3%；构造源 Cell 的强度顶格比例不得超过 20%。

连续采样基础：

- 新增 `internal/sampling/ScalarGrid`。
- 当前使用双线性插值。
- 自动测试覆盖宏观 Cell 边界连续性与 Minecraft 16-block Chunk 边界连续性。
- 未来可在同一抽象下升级受限 Hermite / 单调 cubic。

最终收口：

- 新增 `cmd/geoworld-audit`，可批量审计多 Seed。
- 64 Seed 审计通过。
- 最小大陆距模拟边界：8 个宏观 Cell。
- 平均大陆边界缓冲：17.44 Cell。
- 边界违规：0。
- 山系占比：2.10% ～ 8.93%。
- 高原占比：0 ～ 1.70%。
- 海沟占比：1.20% ～ 3.89%。
- 洋中脊占比：2.83% ～ 10.98%。
- 裂谷占比：1.13% ～ 7.59%。
- 最终高程顶格占比最大 0.869%。
- 构造强度顶格比例最大 8.855%。
- 外围海洋缓冲从“软偏置”升级成硬约束，并仍精确保持目标海洋面积。
- `EdgeOceanBufferFraction` 已成为世界模型参数，默认 4%。

Phase 1 正式完成。

- 大陆轮廓合理。
- 山系与构造有因果关系。
- 海沟、岛弧、洋中脊位置合理。
- 大陆不贴主世界外围。
- 同 Seed / 配置完全可复现。

## 下一阶段

Phase 1 验收后进入：

**Phase 2 — 气候骨架**

纬度 → 基础温度 → 大尺度气压 / 盛行风 → 海温 → 洋流 → 降水。


## Phase 2.1 — 纬度与基础温度

状态：**完成第一版**

第一步只建立稳定的年平均温度骨架：

```text
Latitude
+ Land Elevation
+ Ocean / Land distinction
→ Annual Mean Temperature
```

暂不加入季风、洋流和降水；这些在后续子阶段逐层叠加。


## Phase 2.2 — 理想化气压与盛行风

状态：**完成第一版**

已实现：

- 赤道低压
- 副热带高压
- 副极地低压
- 极地高压
- 信风
- 西风带
- 极地东风
- 30° / 60° 连续风带过渡
- 南北半球镜像

当前测试 Seed：

- Pressure Index：0.223 ～ 0.792
- Wind Magnitude：0.048 ～ 0.956

下一步：**Phase 2.3 — Sea Surface Temperature + Wind-driven Ocean Current**。


## Phase 2.3 — 海表温度与表层洋流

状态：**完成第一版**

已实现：

- SST 纬度背景。
- 海水冻结点下限。
- 风驱表层流。
- 南北半球 Coriolis 相反偏转。
- 海岸阻挡与沿岸偏转。
- SST 平流 / 扩散 / 背景松弛。
- 陆地洋流严格为 0。

当前测试 Seed：

- SST：-1.80°C ～ 29.14°C
- Current Magnitude：0.001 ～ 0.593

下一步：**Phase 2.4 — Ocean-to-Land Temperature Correction**。


## Phase 2.4 — 洋流对陆地温度修正

状态：**完成第一版**

已实现：

- SST 相对同纬度基准的暖 / 寒异常。
- 最近海洋异常向陆地传播。
- 影响随离海距离指数衰减。
- 原有海拔降温不会被直接覆盖。

当前测试 Seed：

- Ocean Temperature Correction：-0.391°C ～ +0.289°C

下一步：**Phase 2.5 — Moisture Transport / Precipitation / Orographic Rain**。


## Phase 2.5 — 长期水汽与降水骨架

状态：**完成第一版**

已实现：

- SST 控制海洋蒸发潜力。
- 盛行风平流水汽。
- 邻域扩散。
- 网格尺寸自适应水汽迭代。
- 弱陆地蒸散再循环近似。
- 低压抬升增雨。
- 山地迎风坡地形雨。
- 水汽消耗形成背风雨影。
- 初步年降水 mm。

16 Seed 审计：

- 陆地平均降水：734 ～ 1528 mm
- <250 mm 干旱陆地：22.7% ～ 59.7%
- >1500 mm 湿润陆地：14.0% ～ 38.1%
- 6000 mm 顶格区域：最多 1.27%

Phase 2 第一版正式完成。

下一步：**Phase 3.1 — Summer/Winter Representative States + Monsoon Index**。


## Phase 3.1 — 暖季 / 冷季代表态与季风指数

状态：**完成第一版**

已实现：

- 陆地 / 海洋不同的季节温差。
- 大陆性随离海距离增强。
- 暖季 / 冷季代表温度。
- Monsoon Index。
- 暖季向陆季风异常。
- 冷季离陆季风异常。
- 季风异常叠加现有盛行风，不替换全球环流。

当前测试 Seed：

- Seasonal Amplitude：1.52 ～ 20.90°C
- Monsoon Index：最大 0.865
- 有效季风 Cell：60637

下一步：**Phase 3.2 — Warm / Cold Season Precipitation + Rainfall Seasonality**。


## Phase 3.2 — 季节性降水

状态：**完成第一版**

已实现：

- 年平均 / 暖季 / 冷季共用同一套水汽求解器。
- 暖季降水。
- 冷季降水。
- 季节年降水。
- Rainfall Seasonality。
- Monsoon Rain Ratio。
- 季节温度对水汽的弱耦合。
- 季节水汽向长期背景态回归，避免冷季数值清零。

12 Seed 审计：

- Seasonal Annual / Phase2 Annual：0.93 ～ 1.16
- 弱季风暖 / 冷季比：1.27 ～ 1.92
- 强季风暖 / 冷季比：1.68 ～ 28.19
- 强季风陆地占比：11.2% ～ 23.6%

下一步：**Phase 3.3 — Watershed / Flow Direction / Flow Accumulation / River Network**。


## Phase 3.3 — 流域与河网骨架

状态：**完成第一版**

新增独立模块：

- `internal/hydrology`
- `cmd/geoworld-hydrology-lab`
- `cmd/geoworld-hydrology-audit`
- `prototype/hydrology.html`

已实现：

- Priority Flood 水文填洼。
- Original / Filled Elevation 分离。
- FillDepth。
- D8 Flow Direction。
- 降水驱动 Local Runoff。
- Flow Accumulation。
- BasinID。
- MajorBasinCount。
- RiverStrength。
- LakePotential。
- WetlandPotential。

8 Seed 审计：

- 河流：1.72% ～ 3.15% 陆地
- 填洼：3.88% ～ 9.69%
- 湖泊候选：2.09% ～ 6.42%
- 湿地候选：1.33% ～ 4.21%
- 主要河流流域：13 ～ 38
- 最大归一化填洼深度：0.080 ～ 0.145

当前 640 × 400 Seed 731894205：

- 全部流域：1738
- 主要流域：62
- 河流：2.07%
- 湖泊候选：1.74%
- 湿地候选：1.59%

下一步：**Phase 3.4 — River Hierarchy / Lake Connectivity / Phase 3 Acceptance**。


## Phase 3.4 — 河流等级、季节河与显式湖泊

状态：**完成第一版**

已实现：

- Strahler River Order。
- 主河 / 支流层级。
- 暖季 / 冷季径流分别汇流。
- PerennialRiverPotential。
- SeasonalRiverPotential。
- 显式 LakeID。
- Lake Surface / MaxDepth。
- SpillOutlet。
- 湖泊所属 Basin 与入流水量。
- SpillOutlet 明确为“潜在溢流口”，不代表湖一定外流。

12 Seed 最终水文审计：

- 河流：1.68% ～ 3.15% 陆地
- 主要河流流域：6 ～ 38
- 最高 Strahler Order：2 ～ 3
- 季节河占优：12.9% ～ 39.6% 河流 Cell
- 显式湖泊：2 ～ 7
- 最大湖泊：76 ～ 318 宏观 Cell
- 最大湖深：0.080 ～ 0.145 normalized

当前 640 × 400 Seed 731894205：

- RiverOrder 最大：3
- Order >= 3：128 Cell
- 季节河倾向占优：861 Cell
- 常流河倾向占优：838 Cell
- 显式湖泊：14
- 最大湖泊：222 Cell

Phase 3 第一版正式完成。

下一步：**Phase 4.1 — Erosion Driver / Valley Incision / Relief**。


## Phase 4.1 — 侵蚀驱动力与河谷切割

状态：**完成第一版**

新增：

- `internal/erosion`
- `cmd/geoworld-erosion-lab`
- `cmd/geoworld-erosion-audit`
- `prototype/erosion.html`

已实现：

- Slope。
- Wetness。
- TectonicSupport。
- TerrainMaturity proxy。
- ErosionResistance proxy。
- FluvialPower。
- ValleyIncision。
- HillslopeErosion。
- TotalErosion。
- ErodedElevation。
- LocalRelief / ReliefIndex。

当前 640 × 400 Seed：

- 平均侵蚀：0.00074
- 最大侵蚀：0.01962
- 河谷切割影响：1.53% 陆地
- 高 Relief：17.39% 陆地

多 Seed 回归曾发现并修复：

- 原始低海岸 Cell 被 erosion floor 错误抬高。

当前保证：

- 侵蚀只降低地形。
- 海洋不变。
- 低于 floor 的原始陆地保持原样。

下一步：**Phase 4.2 — Hills / Foothills / Rolling Plains / Plateau continuous weights**。


## Phase 4.2 — 丘陵 / 山麓 / 高原连续权重

状态：**完成第一版**

新增：

- `internal/landform`
- `cmd/geoworld-landform-lab`
- `cmd/geoworld-landform-audit`
- `prototype/landform.html`

基础地貌连续权重：

- RollingPlainWeight
- HillWeight
- FoothillWeight
- MountainWeight
- PlateauWeight

额外 Overlay：

- ValleyCore
- ValleyWeight

当前正式 640 × 400 多 Seed 审计：

- 平原 dominant：62.80% ～ 65.64%
- 丘陵：9.63% ～ 11.73%
- 山麓：3.64% ～ 4.30%
- 山地：18.20% ～ 21.13%
- 高原：0.05% ～ 3.58%
- 混合过渡带：11.53% ～ 16.45%
- 强河谷 Overlay：0.64% ～ 0.84%

尺度问题已修复：

- 同 Seed 240×150 山地占比曾为 33.24%。
- 640×400 为 14.96%。
- Phase 4 空间核改为按网格尺度解析后，240×150 降至 12.42%。

已新增完整流水线尺度回归测试，防止预览分辨率再次改变世界地貌比例。

下一步：**Phase 4.3 — Regional Terrain Height / continuous sampling / chunk seam acceptance**。


## Phase 4.3 — Regional Terrain 合成与连续采样

状态：**完成第一版**

新增：

- `internal/terrain`
- `cmd/geoworld-terrain-lab`
- `cmd/geoworld-terrain-audit`
- `prototype/terrain.html`
- `ScalarGrid.SampleCubicBSpline`

Regional Terrain：

```text
ErodedElevation
+ continuous Landform Weights
+ world-coordinate Regional Detail
- Valley Carve
→ H(worldX, worldZ)
```

正式测试 Seed 731894205：

- Grid：640 × 400
- Cell：128 blocks
- 约 81.8k × 51.1k blocks
- Height：0.14571 ～ 0.91951
- Regional Detail：-0.03224 ～ +0.03088
- Max Valley Carve：0.00454
- Chunk epsilon jump：1.02e-7
- Macro slope mismatch：5.36e-6

已实现 smooth bounded height addition，避免 Regional Detail 把山顶硬 clamp 到 1.0。

多 Seed Terrain Audit 已加入失败护栏。

Phase 4.4 已完成，Phase 4 第一版正式收口。


## Phase 4.4 — 河谷横断面与沉积型地貌

状态：**完成第一版**

新增：

- `internal/geomorph`
- `cmd/geoworld-geomorph-lab`
- `cmd/geoworld-geomorph-audit`
- `prototype/geomorph.html`

连续字段：

- FloodplainWeight
- AlluvialPlainWeight
- AlluvialFanCore
- AlluvialFanWeight
- CanyonWeight
- ValleyFloorWidth
- DepositionPotential
- SedimentSupply
- SlopeDrop
- UpstreamSteepMemory
- UpstreamMountainMemory

真实因果关系：

```text
大河 + 平缓低地
→ 河漫滩 / 宽谷底

低坡 + 低 Relief + 长期汇流
→ 冲积平原

上游山地 + 上游陡坡
+ 河流出山口后明显变缓
+ 上游侵蚀物源
→ 山前冲积扇

强河流 + 高 Relief + 较陡坡度
→ 峡谷
```

正式 640 × 400 多 Seed 审计：

- 河漫滩：0.175% ～ 0.311% 陆地
- 冲积平原：1.141% ～ 1.923%
- 明显冲积扇：0 ～ 0.062%
- 明显峡谷：0 ～ 0.013%
- 宽谷底：0.146% ～ 0.280%
- 明显沉积区：0.098% ～ 0.150%

允许单个世界没有明显冲积扇 / 峡谷，不为凑数量强行生成。

Phase 4.4 已接入最终 Terrain：

- 河漫滩 / 冲积平原 / 宽谷底会降低局部粗糙度。
- 冲积扇产生轻微沉积抬升。
- 峡谷在已有河谷基础上额外下切。
- 新增 `deposition-lift.png` 与 `canyon-carve.png`。

接入后连续性复验仍通过：

- Chunk epsilon jump：约 1e-7
- Macro slope mismatch：约 1e-6

## Phase 4 总结

状态：**完成第一版**

当前地形因果链：

```text
Tectonics
+ Climate
+ Hydrology
→ Erosion
→ Relief
→ Landform Weights
→ Fluvial / Depositional Geomorphology
→ Continuous Terrain H(worldX, worldZ)
```

下一步：**Phase 5.1 — Climate / Hydrology / Terrain → Eco Region 分类**。

## Phase 5.1 — 长期生态区分类

状态：**完成第一版**

新增：

- `internal/ecology`
- `cmd/geoworld-ecology-lab`
- `cmd/geoworld-ecology-audit`
- `prototype/ecology.html`
- `docs/PHASE5.md`

连续生态字段：

- Moisture / Aridity
- Tree / Grass / Shrub / Bare Ground
- Snow
- Soil Moisture
- Vegetation Density
- Wetland Ecology
- Alpine Ecology

有限 Eco Region：

- Warm / Temperate / Cold / Frozen Ocean
- Ice Sheet
- Tundra
- Alpine
- Boreal / Temperate / Subtropical Forest
- Temperate Grassland
- Temperate / Hot Desert
- Mediterranean Shrubland
- Savanna
- Tropical Monsoon Forest
- Tropical Rainforest
- Wetland

Phase 5.1 接入真实世界时反向发现并修复 Phase 2 温度标尺问题：

- normalized elevation 不再线性映射到 0..6000m。
- 新增 ElevationCurvePower，默认 1.70。
- 新增 LatitudeTempCurvePower，默认 3.50。
- 中纬度不再被系统性算得过冷。

正式 640×400 多 Seed 生态审计：

- 冷地生态：约 5% ～ 17%
- 森林：约 27% ～ 37%
- 干旱生态：约 15% ～ 27%
- 草原 / 灌丛：约 27% ～ 40%
- 热带生态：约 39% ～ 46%
- 低置信度：约 1.4% ～ 2.1%

全量气候 / 季风 / 水文回归仍通过。

Phase 5.1 后续目标：**Eco Region → Minecraft Biome / Surface / Vegetation Mapping**；已在 Phase 5.2 完成第一版。

## Phase 5.2 — Minecraft 生成 Profile

状态：**完成第一版**

新增：

- `internal/mcprofile`
- `cmd/geoworld-mcprofile-lab`
- `cmd/geoworld-mcprofile-audit`
- `prototype/mcprofile.html`

输出：

- Biome Family
- Terrain Variant
- Surface Profile
- Vegetation Profile
- Profile Confidence

重要边界：

- Core 不绑定具体 Loader。
- Core 不写死 Minecraft Registry ID。
- Loader Adapter 后续负责 Vanilla / GeoWorld 自定义 / 第三方群系映射。

正式 640×400 多 Seed 审计：

- Unknown = 0
- Profile confidence：0.692 ～ 0.755
- 低置信度：0.09% ～ 0.24%
- 平原类：54% ～ 70%
- 丘陵 / 山麓：12% ～ 19%
- 山地 / 高原：15% ～ 27%
- 河流特殊地貌：1.7% ～ 2.6%
- 森林植被：20% ～ 37%
- 开放植被：25% ～ 44%
- 荒漠稀疏植被：15% ～ 33%
- 寒冷 / 高山稀疏植被：4% ～ 20%

Phase 5 完成第一版。

下一步：**Phase 6.1 — 把当前算法模块串成正式 Generator Core / World Model，并开始真实参数驱动生成。**

---

## 2026-09-29 — 评审后基线与 Phase 6.1a

- 4a5f685：Phase 5.2 及调整后的路线正式提交。
- 开发分支：agent/world-model-foundation。
- 完成统一坐标契约、宏观海陆连通判定与全链路共享；详见 PHASE6.md 的当前结果。
- 全量测试、4 Seed Profile、2 Seed 宏观一致性审计通过。
- 最终连续海岸和河网逆坡仍未解决，未将其标成已完成。
- 下一步：6.1b 旧空间参数迁移。


## 2026-09-30 — Phase 6.1b 第一切片：Phase 1 物理尺度迁移

基线：7de68a4，分支 agent/world-model-foundation。

本切片完成：
- Phase 1 Config 增加 OriginX / OriginZ / SpanXBlocks / SpanZBlocks。
- 旧默认仍按 128 blocks/node 解析，现有 640×400 世界物理范围不变。
- Phase 1 World 保存 GridGeometry，并明确要求当前宏观网格 X/Z 等距。
- CoastDistance 从 Cell 单位迁移为 blocks。
- 大陆内部、陆架、陆坡、海盆、海岸/高原等依赖 CoastDistance 的阈值全部按旧 128-block 参考尺度换算。
- Tectonic feature band 由固定 Cell 扩散改为物理半径 + 当前网格重采样。
- 新增 128 与 64 blocks/node 的固定世界范围回归，验证相同世界坐标的海岸距离和构造影响不随精度变化。

验证：
- go test ./internal/phase1 -count=1：通过。
- go test ./... -count=1：通过。
- 4 Seed / 640×400 mcprofile audit：通过，Unknown=0，平均 confidence 0.691–0.764。
- consistency audit：731894205 / 10400 的宏观结果与 6.1a 一致；最终连续海岸与河道问题仍未解决。

未完成：
- climate 的 CoastalTempDecay / 海温与水汽传播距离。
- erosion slope 与 relief 等尺度。
- hydrology / geomorph 距离核。
- Phase 1 其它以 grid-frequency 表达的 warp/noise 仍待后续评估。

下一步：继续 6.1b climate 空间尺度迁移。


## 2026-09-30 — Phase 6.1b 第二切片：Climate 物理尺度迁移

基线：1439bcb，分支 agent/world-model-foundation。

本切片完成：
- 将旧 128 blocks/node 提升为 shared reference scale，只用于把历史 Cell 参数翻译成物理距离。
- Climate Input / Model 接入 GridGeometry；Phase 1 adapter 传递真实 Geometry。
- CoastalTempDecay 与 OceanDistance 改为 blocks。
- SST 平流/扩散、水汽平流/扩散改为按 128-block 物理步长解析。
- 默认 MoistureIterations 改为按世界 Span 推导，不再随相同世界的采样节点数变化。
- Orographic Lift 的高程梯度加入真实节点间距校正。
- 固定相同世界范围的 128 / 64 blocks-node 回归验证 OceanDistance、温度衰减、SST、水汽与降水对应世界坐标一致。

验证：
- climate 单元测试通过。
- go test ./... -count=1 通过。
- 16 Seed climate audit 与 12 Seed monsoon audit 通过。
- 4 Seed / 640×400 Profile audit 与第一切片默认结果一致。
- 2 Seed consistency audit 与第一切片默认结果一致。
- 6.2 的最终海岸/水体/河床问题没有被隐藏或提前标记完成。

未完成：
- Erosion slope / local relief 的物理尺度。
- Hydrology / Geomorph 的距离核与河谷扩散。
- Phase 1 非距离型 grid-frequency warp/noise 的进一步评估。

下一步：继续 6.1b Erosion。


## 2026-09-30 — Phase 6.1b 第三切片：Erosion 物理尺度迁移

基线：6a3ecf0，分支 agent/world-model-foundation。

本切片完成：
- Erosion Input / Model 接入 GridGeometry。
- Slope 中心差分加入真实节点间距，并保持旧 128-block reference slope 的 SlopeScale 语义。
- ReliefRadiusCells 改为 ReliefRadiusBlocks；默认值按物理 Span 在旧 reference grid 上解析。
- LocalRelief 使用固定 blocks 邻域，不再让提高网格精度缩小实际观察范围。
- 新增 128 / 64 blocks-node 固定物理范围测试，Slope、HillslopeErosion 和 LocalRelief 对应坐标一致。

验证：
- erosion 单元测试通过。
- go test ./... -count=1 通过。
- 8 Seed erosion audit 通过。
- 4 Seed Profile / 2 Seed Consistency 最终复验通过，默认数值与 6a3ecf0 一致；继续保留 6.2 未解决海岸/河床诊断。

未完成：
- Hydrology 的距离/坡降量纲。
- Geomorph / 河谷扩散物理尺度。
- Phase 1 非距离型 grid-frequency warp/noise 评估。

下一步：继续 6.1b Hydrology。


## 2026-09-30 — Phase 6.1b 第四切片：Hydrology 物理尺度迁移

基线：6b3b23d，分支 agent/world-model-foundation。

本切片完成：
- Hydrology Input / Model 接入 GridGeometry。
- Priority Flood FillEpsilon 按真实节点间距缩放。
- D8 Flow Direction / downstream slope 改为物理距离量纲，并保持旧 128-block reference slope 的阈值语义。
- LocalRunoff 按 Cell 物理面积缩放，避免加密网格后相同流域的累积流量按 Cell 数膨胀。
- MinLakeCells 改为 MinLakeAreaBlocks2，Lake 增加 AreaBlocks2。
- 新增 128 / 64 blocks-node 固定物理范围测试，覆盖 epsilon、坡降、径流面积和最小湖泊面积。

验证：
- hydrology 单元测试通过。
- go test ./... -count=1 通过。
- 8 Seed hydrology audit 通过。
- 4 Seed Profile / 2 Seed Consistency 最终复验通过，默认 128-block 数值与 6b3b23d 一致。
- 6.2 的最终连续海岸 / 河床逆坡诊断继续保留，没有被本切片隐藏。

未完成：
- D8 河网拓扑在不同采样精度下仍允许合理的离散差异。
- Geomorph 的坡降 / 上游记忆 / 扩散空间尺度。
- Phase 1 非距离型 grid-frequency warp/noise 评估。

下一步：继续 6.1b Geomorph。


## 2026-09-30 — Phase 6.1b 第五切片：Geomorph 物理尺度迁移

基线：1dae70c，分支 agent/world-model-foundation。

本切片完成：
- Geomorph Input / Model 接入 GridGeometry。
- UpstreamSteepMemory / UpstreamMountainMemory 每-hop 衰减按 nodeSpacing / 128 解析，保留默认 128-block 结果。
- 冲积扇旧 2..7 Cell 半径迁为 256..896 blocks，并将前向 / 横向距离衰减改为 reference world scale。
- 新增 128 / 64 blocks-node 固定物理范围测试，验证上游记忆、SlopeDrop 与冲积扇权重在对应世界坐标一致。
- Geometry 尺寸错误明确拒绝。

验证：
- geomorph 单元测试通过。
- go test ./... -count=1 通过。
- 4 Seed / 640×400 geomorph audit 通过。
- 4 Seed Profile / 2 Seed Consistency 最终复验通过，默认 128-block 数值与 1dae70c 一致。
- 6.2 最终连续海岸 / 河床逆坡问题继续保持 UNRESOLVED。

未完成：
- UpstreamSlope 即时一跳统计仍是诊断字段，当前不参与权重决策。
- Phase 1 grid-frequency warp/noise、Landform 空间核及其它模块还需要一次全仓残余尺度审计。
- 不要求不同宏观精度的 D8 拓扑逐 Cell 相同。

下一步：6.1b 全仓残余尺度审计，然后决定是否进入 6.2。


## 2026-09-30 — Phase 6.1b 第六切片：Landform 物理尺度迁移

基线：074e8d7，分支 agent/world-model-foundation。

残余尺度审计发现 Landform 仍以 grid min dimension 推导 MountainInfluence / ValleyRadius / ValleyDecay。Phase 4.2 的旧修复只是“随网格尺寸缩放 Cell 半径”的过渡方案，不是真正世界单位。

本切片完成：
- Landform Input / Model 接入 GridGeometry。
- MountainDistance 改为 blocks。
- MountainInfluenceCells / ValleyRadiusCells / ValleyDecayCells 改为 blocks。
- 默认值在旧 128 blocks/node reference world 上解释原有规则，再映射到当前 Geometry。
- Valley spread 按真实 block 半径和衰减计算。
- 新增固定相同物理 Span 的 128 / 64 blocks-node 回归，验证 MountainDistance、MountainInfluence、ValleyWeight 对应世界坐标一致。

验证：
- landform 单元测试通过。
- go test ./... -count=1 通过。
- 8 Seed landform audit 通过。
- 4 Seed Profile audit 通过，Unknown=0，平均 confidence 0.691–0.764。
- 2 Seed Consistency 默认 128-block 数值不变；最终连续海岸 / 河床逆坡问题继续保留。

未完成：
- Phase 1 grid-frequency warp/noise 评估。
- Terrain CellSizeBlocks 与 GridGeometry 关系。
- 其它残余空间参数全仓审计。

下一步：继续 6.1b 残余尺度审计。


## 2026-09-30 — Phase 6.1b 第七切片：Phase 1 reference-world 残余迁移

基线：92d40be，分支 agent/world-model-foundation。

残余尺度审计发现 Phase 1 仍有两类节点绑定：plate / continent 几何使用当前 Width / Height 坐标，以及 continent potential / crust thickness / final macro elevation 的固定节点频率。

本切片完成：
- Plate center、plate warp、continent seed center / radius 统一使用旧 128 blocks/node reference-world 坐标。
- 当前网格节点根据真实 spacing 映射到 reference-world 后参与 nearest plate / warp / continent potential。
- continent potential、CrustThickness、Macro Elevation 的固定节点频率迁为 reference-world 频率。
- Edge ocean buffer 的默认物理宽度不再随采样精度改变。
- 新增固定物理 Span 128 / 64 blocks-node 回归，验证 Plate center、PlateID、LandPotential、noise 与 edge buffer。

验证：
- phase1 单元测试通过。
- go test ./... -count=1 通过。
- 32 Seed phase1 audit 通过，edge-buffer-warnings=0。
- 4 Seed Profile / 2 Seed Consistency 默认 128-block 基线不变。

未完成：
- Terrain CellSizeBlocks / Origin 与上游 GridGeometry 的关系。
- 6.1b 最终全仓残余尺度审计。

下一步：迁移 Terrain Geometry 契约。


## 2026-09-30 — Phase 6.1b 第八切片：Terrain Geometry 与收口

基线：24d124b，分支 agent/world-model-foundation。

残余审计最后确认 Terrain 仍可单独指定 CellSizeBlocks / Origin，存在和上游 Geometry 脱节的风险。本切片完成正式 Geometry 绑定，并结束 6.1b。

本切片完成：
- Terrain Input / Model 接入 GridGeometry，正式 adapter 继承 Erosion Geometry。
- ScalarGrid 全部由 `NewScalarGridWithGeometry` 创建。
- legacy CellSizeBlocks / Origin 只在 Input 未提供 Geometry 时作为兼容入口；与显式 Geometry 冲突时直接报错。
- Render 支持非零 Origin。
- Terrain Lab / Audit / Consistency 的 cell-size 从 Phase 1 定义整个流水线 Geometry。
- Consistency 用 Geometry.NodePosition 采最终 Terrain。
- 新增非零 Origin 的 128 / 64 blocks-node 固定物理范围回归。
- 最终全仓残余尺度搜索完成，未发现新的固定节点生成尺度。

验证：
- terrain 单元测试通过。
- go test ./... -count=1 通过。
- 4 Seed Terrain audit 通过；|detail| 0.03314–0.03689，chunk jump 1.12e-7–1.48e-7，macro slope mismatch 2.26e-6–6.12e-6。
- 默认 2 Seed Consistency 与既有基线完全一致。
- cell-size=64 的非默认完整 Consistency 路径成功，跨模块 MacroMaskDisagreements=0。

Phase 6.1b 状态：完成。

仍明确未完成：最终连续海岸 / 湖面 / 河床逆坡属于 6.2，不能用 6.1b 的尺度稳定性代替。

下一步：Phase 6.2 — GenerateWorld / WorldModel 与最终水体一致性。


## 2026-09-30 — Phase 6.2 第一切片：统一 GenerateWorld / WorldModel

基线：0bd8cff，分支 agent/world-model-foundation。

本切片建立正式统一调度层，不修改现有地理算法输出。

完成：
- 新增 internal/generator Config / ResolvedConfig / WorldModel / GenerateWorld。
- Seed / GridGeometry / SeaLevelNormalized 提升为顶层共享契约。
- Phase 1 新增 GenerateChecked error API，legacy Generate 继续兼容。
- 阶段 SeaLevel、Terrain seed / legacy geometry 与顶层冲突时拒绝。
- 9 阶段固定进度回调；context 可在阶段边界取消。
- WorldModel 收集各阶段实际解析后的 Config，作为后续持久化输入。
- Consistency Audit 改用 GenerateWorld。

验证：
- generator + phase1 单测通过。
- go test ./... -count=1 通过。
- 与旧手工流水线关键字段/采样结果一致。
- SeaLevel=0.47 共享传播回归通过。
- Consistency 两个正式 Seed 默认诊断逐项保持 6.1b 基线。

仍未完成：最终连续海岸、水体、湖面、河床纵剖面与生态同步。

下一步：Phase 6.2 第二切片建立最终 Terrain / Water reconciliation 数据与约束。


## 2026-09-30 — Phase 6.2 第二切片：Terrain / Water reconciliation 约束

基线：395c1cd，分支 agent/world-model-foundation。

本切片没有直接改连续 Terrain，而是先把当前海岸/水体/河床冲突变成可执行约束数据。

完成：
- 新增 internal/reconcile，作为 GenerateWorld 第 10 阶段。
- 保存最终 Terrain 宏观节点采样、RawTerrainSurface 与 CoastTargetElevation。
- 海洋控制点被抬高时生成 carve target；非封闭陆地被压低时生成 raise target；合法封闭低地不填平。
- Ocean / Lake / River 水体类型统一标记；Ocean / Lake 有明确水面，Lake 生成 bed ceiling。
- 强河流生成独立 RiverBedTarget，按 FlowTo DAG 只向下 carve，处理多支流汇合。
- Consistency Audit 新增 constraint 诊断并对 constraint 自身失败设默认护栏；strict-final 仍检查真实连续 Terrain。

正式 Seed 结果：
- 731894205：raw surface mismatch 806；constraint 调整 665 节点后 0。raw upslope 72；carve 98 河床节点后 target upslope 0。LakeBedViolations=1。
- 10400：raw surface mismatch 608；constraint 调整 510 节点后 0。raw upslope 126；carve 162 河床节点后 target upslope 0。LakeBedViolations=7。

测试：reconcile/generator 单测与 go test ./... 全通过。

未完成：约束尚未应用到 Terrain.Sample；湖床/河床/海岸连续 overlay、最终生态同步仍待后续切片。


## 2026-09-30 — Phase 6.2 第三切片：连续海岸 correction 应用

基线：2ab028d，分支 agent/world-model-foundation。

第二切片只证明 coast / river 约束可解；本切片把 CoastTarget 真正挂到连续 Terrain broad surface。

完成：
- 新增 smoothstep bilinear correction sampler，命中控制点且 correction 在宏观 Cell 边界斜率趋零。
- 新增 terrain.WithHeightCorrection；raw Terrain 不变，WorldModel 增加 CoastTerrain。
- GenerateWorld 实际应用 CoastCorrection，并验证应用后的宏观海陆 mismatch。
- RiverBedTarget 改为基于 CoastTerrain 重新计算，避免 coast 修正后仍沿用旧 raw terrain 河床基线。
- Consistency 增加 applied coast node/error/Chunk/Macro 连续性护栏。

验证：
- 全量 go test ./... -count=1 通过。
- Seed 731894205：applied coast mismatch=0，node error=0，chunk jump=5.59045e-8，macro slope mismatch=6.55886e-6；coast-only river upslope=58。
- Seed 10400：applied coast mismatch=0，node error=0，chunk jump=6.29519e-8，macro slope mismatch=6.74817e-6；coast-only river upslope=124。
- raw 806/608 surface changes 与 72/126 river upslope 继续保留为校正前基线。

未完成：RiverBedTarget / LakeBedCeiling 的局部 bed overlay、连续 water surface/runtime 查询、最终 Ecology/Profile 同步。

下一步：Phase 6.2 第四切片，应用 carve-only river/lake bed overlay。


## 2026-09-30 — Phase 6.2 第三切片：连续 CoastTerrain 应用

基线：2ab028d，分支 agent/world-model-foundation。

第二切片只证明海岸 constraint 可解，本切片把它真正接入连续 Terrain。

完成：
- 新增 smoothstep-bilinear correction sampler：命中控制点、无 overshoot、Macro boundary correction slope 归零。
- Terrain 增加可选 HeightCorrection 与 WithHeightCorrection，不覆盖 raw Terrain。
- GenerateWorld 新增独立 CoastTerrain，并反向采样验证 applied surface / node error。
- 湖床 violation 与 RiverBedTarget 在 CoastTerrain 应用后重新计算，修正依赖顺序。
- Consistency 增加 CoastTerrain Chunk seam / Macro slope / applied surface 护栏。

正式 Seed：
- 731894205：Applied coast mismatch=0，node error=0，chunk jump=5.59045e-8，macro slope mismatch=6.55886e-6；river upslope 72→58，重算 bed target carve 110 节点后 target upslope=0。
- 10400：Applied coast mismatch=0，node error=0，chunk jump=6.29519e-8，macro slope mismatch=6.74817e-6；river upslope 126→124，重算 bed target carve 171 节点后 target upslope=0。

测试：sampling / terrain / reconcile / generator 专项回归与 go test ./... 全通过；strict-final 仍按预期失败。

未完成：RiverBedTarget / LakeBedCeiling 尚未应用为局部 bed overlay；最终 Runtime Terrain 和最终 water semantics 尚未形成。


## 2026-09-30 — Phase 6.2 第四切片：Local Waterbed 应用

基线：a2ca47a，分支 agent/world-model-foundation。

本切片完成：
- 新增 internal/waterbed，真实应用 RiverBedTarget / LakeBedCeiling。
- River carve-only，河宽使用 4..18 blocks 的物理尺度。
- River profile 以 block 步长采 broad terrain，并按河网 DAG 传播实际 outlet height，避免汇流 overlap 造成深坑后回升。
- Lake basin 局部 carve 到 bed ceiling，并保留明确 water surface datum。
- GenerateWorld / WorldModel 加入 Waterbed 阶段。
- Consistency Audit 改为实际连续中心线高密度采样，检查非节点 river upslope / lake bed / chunk seam。
- Coast broad correction 增加 homogeneous Cell interior refinement，修复仅节点命中但正常陆地 Cell 内仍跌破海平面的情况。
- Raw Terrain 历史 mismatch 保留，仅作回归对照。

真实中间失败：
- 731894205 初次 Waterbed：425 / 258152 steps upslope，max 2.12536e-4。
- 10400 初次 Waterbed：813 / 388804 steps upslope，max 9.66002e-5。
- 根因是汇流处相邻 corridor overlap，而非当前 segment profile 自身上坡。
- DAG actual-outlet propagation 修复后，两 Seed 均 0 upslope。
- Coast 非节点检查又发现 731894205 正常陆地 Cell 内 160 个采样点低于 sea，max deficit 0.0061282；interior refinement 后两 Seed 均 0。

最终验证：
- go test ./... -count=1 通过。
- 731894205：Applied coast mismatch=0，coast interior ocean/land violations=0/0，Waterbed upslope=0，Lake bed violations=0。
- 10400：Applied coast mismatch=0，coast interior ocean/land violations=0/0，Waterbed upslope=0，Lake bed violations=0。
- Chunk seam 仍为约 5.59e-8 / 6.30e-8；Macro slope mismatch 仍约 6.56e-6 / 6.75e-6。

未完成：
- River water depth / water surface。
- 统一 Runtime-facing Terrain + Water sampler。
- strict-final 仍必须等待 Runtime 水面语义完成。

下一步：Phase 6.2 第五切片，统一最终 Terrain/Water sampling。


## 2026-10-01 — Phase 6.2 第五切片：Runtime Water Surface / WorldSample

功能提交：11e93ee，分支 agent/world-model-foundation。

本切片完成最终 Ground/Water Runtime 最小闭环：
- 新增 internal/worldsample，统一 GroundHeight / WaterKind / WaterSurface / WaterDepth。
- Ocean 使用 shared SeaLevel + continuous ocean eligibility；EnclosedBelowSea 不自动灌海。
- Lake 使用 Hydrology Lake.SurfaceElevation。
- River 使用 center depth × RiverWeight，并支持 receiver backwater floor。
- Waterbed 沿 Hydrology FlowTo 传播 downstream Lake/Ocean datum，同时识别 river segment 与 square lake footprint 的空间相交。
- 修复 D8 对角河道穿过 lake raster corner 时 Euclidean 圆形判定漏掉 receiver 的问题。
- Consistency Audit 把 Runtime Ocean/Lake/River 纳入 strict-final。
- Raw Terrain 历史 mismatch 继续保留，但不再冒充 applied final world。

真实回归：
- Runtime River water-surface upslope：9 → 1 → 0。
- 两个正式 Seed 的 Runtime Ocean node / enclosed ocean leak / Lake node / River undefined / River dry / River water-surface upslope 全部为 0。
- Waterbed centerline upslope=0；LakeBedViolations=0。
- homogeneous coast interior ocean/land violations=0。
- go test ./... -count=1 全通过。
- geoworld-consistency-audit -strict-final 两个正式 Seed 通过。

未完成：
- Ecology / MCProfile 与最终 Runtime water semantics 的一致性验收。
- Mixed coast cell / model outer boundary / corners 的 final Runtime sampling audit。
- Phase 6.3 持久化 schema 与 save/reopen。

下一步：Phase 6.2 第六切片，收口生态/Profile 与最终 Runtime 语义。


## 2026-10-01 — Phase 6.2 第六切片：Runtime Profile 与阶段收口

功能提交：7f8722f。

完成：
- 新增 internal/runtimeprofile，不重跑宏观 Ecology；最终坐标按 WorldSample water semantics 选择同一 land/ocean 侧的 MC Profile。
- Ocean 强制 ocean family / TerrainOcean / VegetationNone。
- River/Lake 保留 terrestrial Ecology/Profile，不把局部水体误当 Ocean biome。
- GenerateWorld 正式流水线扩展到 13 stages，WorldModel 增加 RuntimeProfile。
- Consistency Audit 增加 macro node、mixed coast、完整外围 Runtime Profile 扫描。
- strict-final 纳入 Runtime Profile fallback / mismatch / Unknown 与 boundary non-ocean / mismatch。

正式结果：
- Seed 731894205：274809 RuntimeProfile samples，fallback=0，semantic mismatch=0，Unknown=0；mixed coast 10505 点全通过；boundary 8304 点全 Ocean 且 profile mismatch=0。
- Seed 10400：275984 RuntimeProfile samples，fallback=0，semantic mismatch=0，Unknown=0；mixed coast 11680 点全通过；boundary 8304 点全 Ocean 且 profile mismatch=0。
- go test ./... -count=1 全通过。
- 4 Seed MC Profile audit 保持旧基线，Unknown=0。

Phase 6.2 正式完成。下一步 Phase 6.3：最小持久化 schema、原子保存、版本校验、重开复现。

## 2026-10-01 — Phase 6.3 第一切片：Versioned World Model persistence

功能提交：0b830de。

完成：
- 新增 internal/worldmodelio，V1 = gzip+JSON envelope + SHA-256 payload checksum。
- manifest 保存 schemaVersion=1、generatorVersion=phase6.3-v1、Seed、Geometry、SeaLevel、encoding/checksum 元数据。
- payload 只保存 Runtime 重建需要的 ResolvedConfig / Surface / CoastTerrain grids / Waterbed river+lake geometry / minimal Ecology+MCProfile。
- 新增 waterbed.Restore，加载时重建 buckets，不重新跑 Hydrology/Reconciliation。
- Save 使用 same-directory completed temp + hard-link publish；目标存在即 ErrTargetExists，原完整文件不变。
- Load 明确拒绝 checksum mismatch、未知 schema、未知 generator version 和共享契约冲突。
- 新增 geoworld-model-audit。

验证：
- worldmodelio unit tests 与 go test ./... 全通过。
- 320×200：1,820,161 bytes，484 samples mismatch=0，Save≈0.51s，Load≈0.40s。
- 640×400 / Seed 731894205：7,631,145 bytes，payload=37,323,242 bytes，RiverSegments=908，LakeNodes=1322，484 samples mismatch=0，Save≈2.13s，Load≈1.55s。

下一步：safe replace、manifest inspect/validate、cross-language golden vectors。

## 2026-10-01 — GeoWorld Tester：优先提供可人工验证入口

功能提交：0a10067；相关持久化增强：8cd95b9。

用户反馈当前开发虽然底层审计进展较多，但缺少可直接操作和观察的测试入口，因此暂停 MOD / Java reader 优先级，先完成本地 Web Tester。

已完成：
- 新增 `cmd/geoworld-tester`，静态前端嵌入 Go 可执行程序，不依赖 Node。
- Seed / Width / Height / CellSize / PlateCount / OceanFraction 可直接重新生成真实 WorldModel。
- 生成进度读取 GenerateWorld 13 stages。
- 最终地形 / 水体 / Biome / Profile confidence 四个实时图层。
- 地图点击查询最终 Runtime Ground / Water / EcoRegion / Biome / Terrain / Surface / Vegetation / semantic flags。
- 模型保存 / 更新 / 已保存模型列表 / 重开。
- Windows 根目录新增 `open-geoworld-tester.cmd`，默认打开 `http://127.0.0.1:8788`。
- worldmodelio 同步完成安全 replace 与 manifest Inspect / Validate。

真实 E2E：
- 120×80 Seed 731894205：生成约 0.31s；Final layer PNG 成功。
- (0,0) Runtime：Savanna / Rolling Plain / Ground=0.5030537260440189，semantic fallback/mismatch=false。
- 保存 `_tester-e2e.geoworld` 成功；文件约 287 KB。
- 重开约 56ms；(0,0) 采样逐字段一致。
- go test ./... -count=1 全通过。

后续优先完善 Tester 图层/异常诊断；MOD/Loader 后置到人工世界验收之后。


## 2026-10-02 — Tester 多大陆形态控制与随机 Seed

功能提交：a4e3353；前置可用性修复：ba4aa3d。

完成：
- 修复原大陆势能相加导致多数 Seed 被焊成单一超级大陆的问题；大陆核改为独立主导并受分离度约束。
- Phase 1 / generator 正式增加 ContinentFragmentation 与 ContinentSeparation，默认 0.35 / 0.65。
- Tester 增加随机 Seed、大陆破碎度、大陆平均分离度，并明确“洋壳比例”不等于最终海水覆盖率。
- 当前世界面板增加主要陆块、全部陆块、最大陆块占比；新参数随模型保存/重开恢复。
- 旧模型缺少新字段时按默认 0.35 / 0.65 展示。
- Tester 静态资源使用 no-store，避免浏览器继续使用旧 JS/CSS。

算法验证：
- 旧算法抽样 12 Seed 时，10/12 的 Continental Crust 只有 1 个连通块。
- 新默认参数在 8 Seed 诊断中平均约 3.5 个主要最终陆块，最大陆块平均约 58.7%。
- 同一 Seed 1771841485 的 BSK A/B：破碎度 0.80 / 分离度 0.90 => 3 个主要陆块、5 个全部陆块、最大陆块 58.6%；破碎度 0.10 / 分离度 0.15 => 1 个陆块、最大陆块 100%。
- BSK 实测随机 Seed 按钮、滑条、生成、保存、重开；保存后把 UI 改到 1.00 / 1.00，再重开模型可恢复 0.10 / 0.15。
- go test ./... -count=1 全通过；git diff --check 通过。

下一步：继续 Tester-first，补河网 / 湖泊 / 海岸 / Raw-vs-Final 等可视化诊断，再做更广 Seed 的大陆形态基线。

## 2026-10-02 · NeoForge 26.3 地形/生态质量与创建页收口

- 创建页：洋壳比例、大陆破碎度、平均分离度改为 0.01 步进滑块；预设切换会同步滑块，手调后自动识别/切换自定义。
- Seed：GeoWorld 自定义页与 Minecraft 外层创建页继续共享 `WorldCreationUiState`。空 Seed 进入自定义页会立即生成并回写随机 long；随机按钮和手动修改都会同步外层并使旧 generated model 失效。
- 预览生命周期：关闭 `GeoWorldPreviewScreen` 返回同一参数页时，保留最新 config/generated model；每次 init 重建 `HeaderAndFooterLayout`，避免旧 layout/构造期 config 使“完成/查看世界地图”重新失效。
- 沙漠：正式 MC Profile 的 Hot Desert 固定使用 Desert Sand；Temperate Desert 高裸地也使用 Desert Sand，低裸地保留 Shrub Soil。Minecraft 26.3 映射同时把兼容旧模型的 `Desert Gravel` 映射为 sand/sandstone；海底 Ocean Gravel 保留。
- 群系硬边：RuntimeProfile 的离散生态/Profile 选择加入低频 deterministic domain warp；只弯曲分类坐标，不改变 Ground/Water 坐标，并在海岸处回退原始坐标寻找正确 land/ocean 语义侧。Go Runtime 与 Java Runtime 公式一致。
- 地形层次：plain/hill/foothill/mountain detail amplitude 分离，并增加平滑 orographic lift。当前 lift 为 hill=.012、foothill=.030、mountain=.100、plateau=.020 normalized；新单测要求平均高度 plain < hills < foothills < mountain，且 mountain-plain 至少 0.05 normalized。
- 山地生态垂直分带：在最终 Runtime Profile 层基于实际 GroundHeight、terrain variant 与气候 family 计算高度带；山腰可从热带/亚热带森林逐级向温带/寒带过渡，峰部进入 Alpine Rock / Alpine Snow。雪线随基础气候变化并加入低频噪声，避免完美等高线圆环。Go/Java 同步实现。
- 预览地图颜色现在优先识别 Alpine Snow/Ice 和 Alpine Rock，使创建页预览能看出雪顶/裸岩带。
- Engine 构建：Gradle `buildGeoWorldEngine` 已把 Go 源码声明为 inputs，Go 代码变化会强制重编 bundled engine；preview cache revision 升到 v3，避免旧 `.geoworld` cache 污染本轮验收。
- HMCL 自动部署：`java/neoforge-mod/build.cmd` 在构建成功后自动覆盖 `D:\games\hmcl\.minecraft\versions\26.3\mods\GeoWorld-NeoForge-26.3-0.1.0-dev.jar`，之后由用户手动启动 HMCL 26.3 做真实客户端验收。

验证：
- `go test ./...`：全绿。
- NeoForge `build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn`：BUILD SUCCESSFUL；bundled engine bridge 通过；固定 ocean-center Seed `-1273393314970095624` 的安全 Spawn 回归通过，当前候选为 Chunk (-12,-164)、Block (-184,101,-2616)、Biome=Subtropical Forest。
- cross-language Java GoldenVerifier：15/15，通过，tolerance=1e-9。
- 最新 JAR 已自动复制到 HMCL 26.3 mods；下一步是用户真实 UI/世界观感验收，而不是再以 build 通过冒充 UI 验收。
## 2026-10-03 · 群系边界侵蚀与游戏内模型预览

- RuntimeProfile 在已有低频 domain warp 基础上增加“近边界竞争”：当同一宏观 Cell 内两个或更多同侧生态候选权重接近时，使用与 Seed/候选节点绑定的两级短波 value-noise 对权重做小幅竞争扰动。扰动强度只在候选接近时生效，离边界越远越快速衰减，因此群系内部保持稳定，边界则形成几十方块尺度的指状侵入、斑块和不规则锯齿，而不是一条规则直线。
- 侵蚀选择保持整套 Profile 一起切换：BiomeFamily / TerrainVariant / SurfaceProfile / VegetationProfile 来自同一 SourceNode，避免把群系颜色柔化后却让地表/植被仍硬切成另一套语义。
- Ground / Water 坐标和海岸判定不参与这次扰动，海陆几何继续由正式 Runtime 决定；近海若 warped profile 越过语义侧，仍回退 unwarped cell 搜索正确 land/ocean 侧。
- Go Runtime 与 Java Runtime 同步实现完全相同的竞争参数与噪声种子；新增单测验证边界带内两个生态候选都能反向侵入，避免 winner-takes-all 直切退化。
- 单人游戏 PauseScreen 通过 NeoForge ScreenEvent.Init.Post 注入 `GeoWorld 预览` 按钮；仅当前集成服务器已激活 GeoWorld 模型时出现，不影响普通 Vanilla 世界。
- 游戏内预览直接读取 `GeoWorldWorldService.active(server).model()`，不是重新生成另一份预览。打开后显示当前存档绑定 `.geoworld` 的完整模型，并在玩家坐标仍位于模型范围内时绘制红色十字；底部同步显示玩家 X/Z，方便和游戏里真实地形、地图 MOD 对拍。
- GeoWorld Tester 同步新增 `地貌类型` 与 `地表方案` 两个图层，专门检查 Plain/Hills/Foothills/Mountain 分区、Desert Sand/legacy Desert Gravel、Alpine Rock/Snow 等质量问题。

验证：
- `go test ./... -count=1` 全绿。
- `node --check cmd/geoworld-tester/web/app.js` 通过；`go test ./cmd/geoworld-tester -count=1` 通过。
- NeoForge `build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn` 通过并自动部署 HMCL 26.3 JAR。
- Java/Go cross-language GoldenVerifier 15/15 通过，tolerance=1e-9。
- `javap` 已确认最终 client class 包含 `addPausePreviewButton(ScreenEvent.Init.Post)`，不是只改源码未编译。

验收注意：本轮 Runtime Profile 算法改变会影响尚未生成的新 Chunk。为了判断“群系侵蚀”观感，应优先新建 GeoWorld；旧世界已经落盘的 Chunk 不会被重写，而新 Chunk 会走新 Runtime，旧测试世界可能因此故意暴露新旧边界差异。

## 2026-10-03 · Seed 联动、雨林边缘与游戏内局部预览修正

用户在 HMCL 26.3 实测发现三类问题：GeoWorld 自定义页随机 Seed 与外层“世界生成器的种子”显示未联动；雨林树冠与相邻森林仍形成明显硬切；暂停菜单的全局 GeoWorld 预览在玩家雪山位置显示成绿色，疑似与实际世界不一致。

定位结果：
- Seed 的实际 `WorldCreationUiState` 早已同步，问题是 Vanilla `CreateWorldScreen.WorldTab` 的 Seed EditBox 只在构造时从 UiState 读取一次，之后只做 EditBox -> UiState 单向 responder，没有 UiState -> EditBox listener，因此外层显示陈旧。
- 雨林模型本身在实测区域已有 Tropical Rainforest -> Subtropical Forest -> Temperate Forest 的宽过渡；硬边主要发生在 Minecraft 映射层：Tropical Rainforest 直接映射 `minecraft:jungle`，Subtropical Forest 直接映射 `minecraft:birch_forest`，密集 Jungle 树冠因此像墙一样结束。
- 当前实测存档 `world.geoworld` Seed=4577469160985922023。直接加载持久化模型采样玩家截图坐标 X=1649.5/Z=939.5，结果为 Ground≈0.804736、BiomeFamily=Alpine、Terrain=Mountain、SurfaceProfile=Alpine Snow；与游戏里 Y≈222 的大面积雪山一致。旧预览不一致不是模型/Chunk 错位，而是 40832×25472 blocks 的全局图只有约 128×80 样点，每像素约 322 blocks，红十字虽然是精确玩家坐标，底下颜色却可能来自数百格外的样点。

本轮修正：
- GeoWorld 自定义 Seed 改动仍写入同一 `WorldCreationUiState`，并同步更新父级 Vanilla WorldTab 中现存的 Seed EditBox；随机 Seed、手动输入、生成预览、完成时都走同一 helper。
- `GeoWorldBiomeSource` 增加 `minecraft:sparse_jungle`，Minecraft 群系实现层对热带边缘做局部邻域判定：雨林内侧靠边区域改为 Sparse Jungle；Subtropical Forest 靠雨林一侧先经过 Sparse Jungle，再经过一小段普通 Forest，最后才进入 Birch Forest。Terrain/Water/GeoWorld Profile 本身不被改写。
- `GeoWorldPreviewScreen` 在有玩家焦点时默认使用局部视图：128×80、约 8 blocks/sample，即约 1016×632 blocks；保留“查看全局/查看当前位置”切换。局部/全局摘要都会明确显示 blocks/sample。
- 预览红十字按当前视图范围计算；当前位置附带精确模型地表 Y、BiomeFamily、SurfaceProfile，便于直接与 F3/地表外观对拍。
- 预览的陆地颜色改为复用 `GeoWorldBiomeSource.minecraftBiomeId(...)`，因此 Sparse Jungle / Forest / Birch Forest 过渡在预览和真实新生成 Chunk 中使用同一 Minecraft 群系实现逻辑；Alpine Snow 继续优先显示白色。

验证：
- 当前实测模型精确坐标 X=1649.5/Z=939.5 确认为 Alpine / Alpine Snow，证明存档模型和真实雪山一致。
- `gradlew.bat compileJava --no-daemon` 成功。
- `go test ./... -count=1` 全绿。
- zh_cn/en_us 语言 JSON 使用 Node UTF-8 `JSON.parse` 验证通过。
- `git diff --check` 通过。

验收注意：雨林 Minecraft biome 映射只影响新生成 Chunk。已经生成过的 Chunk 不会被重写；观察新过渡带时应新建 GeoWorld 或前往此前从未加载过的区域。

## 2026-10-03 · 河流入海 / 河床嵌入 / River biome / 高级预览

用户在 HMCL 26.3 的新存档 test11（Seed 5441493196688466877）实测发现：河流在海岸前断开、部分河水铺在地表上、河道仍显示为丛林群系，以及游戏内 GeoWorld 预览面积和交互能力不足。

定位：
- 当前存档旧模型的河口链路 #72 只到 `(2240,-192)->(2368,-192)`；而同一纬线约 X=2490 开始才进入正式 Ocean。原因是 `reconcile.buildRiverConstraints` 要求上下游节点都必须是陆地，最终 `land -> ocean/lake` 边被直接丢弃，因此河道天然少最后一个宏 Cell。
- 旧模型在 `(2240,320)` 的 river centerline 上 broad terrain=0.529628（Minecraft 约 Y=78），Waterbed final 仍为 0.529628（carve=0），但河面达到 0.532128（约 Y=80）。也就是 Runtime 确实声明“这里有河”，却没有保证河床低于周围地表，视觉上就会出现水铺在地面上的情况。
- Minecraft biome 映射此前只看 `BiomeFamily`，所以热带雨林中的 Runtime River 最终仍写成 `minecraft:jungle`。
- 预览已有局部视图，但仍是固定小图，缺少连续缩放和指针位置诊断。

本轮实现：
- Reconcile 保留河流源节点必须是陆地的约束，但允许接收节点为 lake/ocean，从而把最后一段河道真正连接到接收水体；新增 land->land->ocean 单测。
- Waterbed 新增 `MinRiverBedDepthNormalized`，默认 0.004；河流中心线每个采样点至少下切该深度，同时继续执行 carve-only 与单调下坡约束。旧模型的零值字段使用 JSON `omitempty` 保持 V1 文件 checksum/加载兼容。
- `downstreamReceiverSurface` 会先检查当前节点自身是否已经是 lake/ocean，保证直接终止在接收节点上的最后一段河道能拿到正确水面 datum。
- Minecraft `GeoWorldBiomeSource` 加入 River / Frozen River holder，Runtime `waterKind=River` 优先映射成 `minecraft:river`，寒带/冻原/高山河流映射 `minecraft:frozen_river`。
- GeoWorld cache key 从 v3 提升到 v4，避免相同 Seed/参数继续命中旧算法生成的模型。
- 游戏内预览重做为更大地图，保持世界纵横比并尽量占满屏幕；局部默认 1600×1000 blocks，最高 160×100 样点；滚轮以鼠标位置为锚点缩放，最小可缩到约 128×80 blocks；保留“查看全局/查看当前位置”，新增“重置视图”。
- 鼠标悬停地图会即时显示精确 X/Z、模型地表 Y、水体类型/水面 Y/深度、GeoWorld BiomeFamily/TerrainVariant/SurfaceProfile、Minecraft biome id 与 vegetation profile。

对用户 test11 同 Seed/同参数重新生成模型验证：
- 河流段数由旧模型 80 增加到 109；原河口链路现在多出 `(2368,-192)->(2496,-192)`，直接跨入 Ocean，旧模型 X≈2432 的 dry gap 已消失。
- `(2240,320)`：旧模型 broad≈Y78 / river bed≈Y78 / water≈Y80（零下切）；新模型 broad≈Y78 / river bed≈Y76 / water≈Y77，河道已经嵌入周围地表。
- `(2240,-192)` 同样从零下切变成 normalized 0.004 的有效河床下切。
- 旧 test11 的 `.geoworld` 仍可正常读取，证明新增 Waterbed 配置没有破坏现有存档格式。

验证：
- `go test ./... -count=1` 全绿。
- Golden fixture 已按新河口语义重建；Java/Go GoldenVerifier 15/15 通过。
- NeoForge `compileJava` 通过。
- `build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn` 通过，并已部署新的 HMCL 26.3 JAR。
- zh_cn/en_us 使用 UTF-8 JSON.parse 验证通过。
- `git diff --check` 通过。

验收注意：
- 河口链路和最小河床下切属于 World Model 生成结果，已有 test11 的 `.geoworld` 不会自动替换；要验证 1/2 最稳妥的是新建 GeoWorld。
- River biome 对尚未生成的新 Chunk 会生效；已经生成并写盘的旧 Chunk biome 不会被 MOD 自动重写。

## 2026-10-03 · 预览信息本地化

- GeoWorld 预览中的摘要、悬停信息、GeoWorld Profile 枚举不再直接显示英文内部值。
- 所有预览文本改为 Minecraft Component.translatable(...)，因此直接跟随当前游戏语言；当前提供 zh_cn / en_us。
- GeoWorld 内部值（WaterKind、BiomeFamily、TerrainVariant、SurfaceProfile、VegetationProfile）补齐本地化键；中文示例包括“热带雨林 / 丘陵 / 森林壤土 / 阔叶林 / 河流”等。
- Minecraft biome 不再显示 minecraft:birch_forest 这类原始 ID；优先使用 Minecraft 自带 biome.<namespace>.<path> 翻译，因此会自动跟随当前语言包。
- 全局/局部预览摘要中的 Seed/zoom/span/blocks/sample 已整体改为翻译资源；中文创建页中的 Seed、blocks/node、模型范围也同步中文化。
- 未知或未来新增值若暂时没有 GeoWorld 翻译键，会安全回退原始内部名称，不会显示缺失翻译 key。
- gradlew compileJava 与完整 build.cmd 均通过，最新 JAR 已自动部署到 HMCL 26.3。
## 2026-10-03 · 高级预览、地图传送、湖岸自然化与山峰强化

用户基于 HMCL 26.3 实测继续提出：悬浮信息窗位置不稳定、预览需要拖动与传送、Savanna 等非雨林群系仍有硬边、湖泊/地形出现宏网格方块感、湖岸有薄水层漫流、山体缺少山峰，同时希望预览能展示大陆/气候/季风等真实模拟层。

本轮实现：

- 预览 UI：移除跟随鼠标的浮动信息框，改成左侧固定信息栏 + 右侧地图；保留鼠标锚点滚轮缩放，新增按住左键拖动平移。
- 地图左键点击可选定世界坐标，黄色十字标记目标；新增“传送到选中位置”。传送只在单人集成服务器中启用，服务端先加载目标 Chunk，再用真实 WORLD_SURFACE 高度决定 Y，避免仅按预览高度把玩家传到地下/空中。
- 玩家自身位置继续用红色十字；PauseScreen 的 GeoWorld 预览按钮由右上角移动到原版暂停菜单主按钮列下方。
- 预览新增图层：生态群系 / 大陆与海洋 / 地形高度 / 年均温度 / 年降水 / 季风指数。
- 新 .geoworld 额外持久化真实 Climate 阶段的 AdjustedAnnualTempC / AnnualPrecipMM / MonsoonIndex / WindX / WindZ 作为解释性预览字段；Java Runtime 在任意坐标连续采样，不从最终 biome 反推气候。
- 旧模型没有 climatePreview 时继续兼容加载，气候图层明确显示旧模型未包含数据。Go 与 Java 均已用旧 t2/world.geoworld 验证兼容。
- 同 Seed 新模型在 (6217,-2435) 的气候采样：年均温约 27.50°C、年降水约 1695.75 mm、季风指数约 0.337。
- 通用群系边界：保留雨林的 Sparse Jungle / Forest 肩带，同时增加通用 ecotone。在约 56 blocks 邻域检测不同陆地生态，以约 24 blocks 的平滑 Seed 噪声让双方在边界互相侵入；Savanna、森林、草原等统一走同一机制。
- River 继续映射 River/Frozen River；Lake 因 Vanilla 没有独立 lake biome，暂映射为 River/Frozen River 淡水 biome，至少不会再把湖面标成 Savanna/Jungle/Desert。
- 湖泊与宏网格：Waterbed 湖泊 footprint 从 max(abs(dx),abs(dz)) 的方形距离改为 hypot(dx,dz) 径向 footprint；相邻 LakeNode 通过重叠组合，不再把宏网格方格直接投影进 Minecraft。
- 湖边增加最小显示水深 0.003 normalized。低于该深度只保留侵蚀后的岸坡，不铺一层极薄水膜，从而形成干湿岸带。
- 用户 t2 世界原先在 (6000,-2400) / (6217,-2200) 一类湖缘位置只有约 0.0001 normalized 水深；新逻辑这些位置转为干岸，湖心和深水区域保持 Lake。
- 山峰：在真实 MountainWeight 允许的区域加入约 900-block 波长的 sharpened ridge/peak 场，并把 mountain orographic lift 从 0.100 提高到 0.125。峰脊不会随机出现在平原/湖区，仍受 Geomorph 山地权重约束；Go Terrain 与 Java Runtime 同步。
- Engine cache key v4 -> v5，确保同 Seed/同参数新世界不会命中旧算法缓存。
- climatePreview 改为可选 pointer + omitempty，旧 .geoworld checksum 仍可原样验证；Golden fixture 已按新地形/湖岸/Climate payload 重建。

盆地现状：
- 当前水文链路能够形成封闭低地、湖泊和局部洼地，但尚没有独立 Basin 地貌分类/生成阶段。
- 后续 Basin 应基于闭合/半闭合流域、周边 relief、spill outlet 与侵蚀沉积结果识别，而不是按 biome 或随机噪声硬生成，继续遵守“先大陆架/构造与水文，再形成地貌与生态”的原则。

验证：
- go test ./... -count=1 全绿。
- Java/Go GoldenVerifier 15/15，tolerance=1e-9。
- NeoForge compileJava、build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn 全通过。
- t2 旧模型在 Go/Java 均可重新加载；旧模型 climate unavailable，新同 Seed 模型 climate 可采样。
- 新同 Seed模型 output/t2-v5-check.geoworld 的 Climate payload 正常；湖缘薄水位置按新岸带逻辑转为干岸。
- 最新 MOD 已由 build.cmd 自动部署至 HMCL 26.3。

## 2026-10-03 · t2 河床 / River biome / 冲积平原诊断与修正

用户在 t2（Seed 4061520997047253165）坐标约 X=2248 / Z=-5876 实测发现河水仍像铺在地表、地图群系显示平原而模型预览显示河流、河旁大块黑色地表，以及怀疑河道不连续。

精确模型诊断：
- 玩家截图坐标 (2248.691,-5875.576)：Runtime 原先 WaterKind=River，但 RiverWeight 只有 0.056，连续水深仅 0.000141 normalized，折算当前高度约 0.07 个方块。也就是说模型把几乎没有物理深度的河岸 fringe 仍标成 River；Minecraft 整块量化后可能突然出现一整格水，形成“水铺在地面上”的视觉。
- 河中心 (2241,-5894)：RiverWeight=0.973，水深约 0.002432 normalized ≈ 1.25 blocks，是真正河道。
- 该区域生态不是红树林。模型是 Temperate Grassland / Alluvial Plain / Alluvial Silt / Grassland。黑色来自 Minecraft surface mapping 把 Alluvial Silt 直接映成 minecraft:mud；这是表现层过度，不是模型判定成沼泽/红树林。
- 当前河网拓扑本身连续：#3 (2112,-6208)->#43->#71 (2240,-5952 -> 2240,-5824)->#93->#113，最终到接收水体；全模型未发现“无 receiver 的内部断头河”。截图里看起来断续主要来自可见水深/方块量化与地图显示，不是 Hydrology 链路中间缺 segment。
- 模型在河中心目标 Minecraft biome 明确应为 minecraft:river。第三方地图仍显示“平原”说明实际已生成 Chunk 的 biome 数据和模型目标存在不一致。

本轮修正：
- WorldSample 增加 MinRiverWaterDepthNormalized，默认 0.0015。低于该物理深度的河流 fringe 只保留河床/岸坡，不再暴露 River 水面，也不再把该点标成 River biome；Go/Java Runtime 同步，旧模型缺字段时使用相同默认值。
- 对当前 t2 旧模型重新采样：原 (2248.691,-5875.576) 已从 River 变为 dry bank；河中心 (2241,-5894) 仍保持 River，水深约 1.25 blocks。
- Minecraft surface mapping 将 Alluvial Silt 从 mud 改为 grass_block；Wet Mud / Peat 仍使用 mud。也就是普通冲积平原回到草地视觉，真正湿泥/泥炭区才黑。
- GeoWorldChunkGenerator.createBiomes 不再只依赖通用 super resolver 路径；在确保 Runtime model 已 attach 后，直接用 GeoWorldBiomeSource 填充 chunk biome，避免 chunk 捕获 fallback resolver 后把 River 存成周边 Plains/Forest。
- 预览中的“Minecraft 群系”改名为“模型目标群系”，避免把模型期望值误当成已落盘 Chunk 的实际 biome。
- Engine cache key v5 -> v6，确保相同 Seed/参数新世界不会继续复用旧水面/biome 语义缓存。

验证：
- go test ./... -count=1 全绿。
- Golden fixture 已重建，Java/Go GoldenVerifier 15/15。
- NeoForge compileJava、build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn 全通过。
- 最新 MOD 已自动部署到 HMCL 26.3。

## 2026-10-03 · 河流水面向上取整 + 原版群系覆盖扩展

- 河流从连续 GeoWorld 高度映射到 Minecraft 整格方块时，水面边界改为向上取整；已经被 Runtime 判定为有效 River 的位置至少保留 1 格水层。之前新增的最小可见河流深度仍保留，因此极浅 fringe 不会重新变成“水板”。
- t2 实测点 (2241,-5894)：River ground boundary=114，ceil water boundary=116，最终 2 格水；旧异常 fringe (2248.691,-5875.576) 仍为 dry bank。
- 抽出 Minecraft26_3BiomeRealizer，动态 MOD 与静态 save writer 共用同一套原版群系实现，不再各自映射。
- 群系映射从原先约 18 个基础群系扩展为按 GeoWorld 的水文、温度、降水、季风、地形类型、表层、植被和高度挑选 Vanilla 26.3 群系：海洋深浅/温度、海滩/石岸/雪滩、平原/向日葵平原、森林变体、针叶林变体、热带雨林/稀疏丛林/竹林、热带草原变体、恶地变体、山地/山峰/草甸/樱花林、沼泽/红树林、河流/冻河、蘑菇岛等。
- 地下 Y 维度开始使用洞穴群系：滴水石洞穴、繁茂洞穴、幽匿深暗之域、硫磺洞穴；不会把 Nether / End 群系硬塞进主世界。
- 对当前 t2/world.geoworld 做全图规则扫描：52 个主世界地表原版群系全部出现；4 个主世界洞穴群系全部出现。即 26.3 Biomes 中除 THE_VOID 以及 Nether/End 专属群系外，主世界可用群系均已进入 GeoWorld 实现路径。
- Go 全仓、Java Golden 15/15、Minecraft storage/save writer、NeoForge build、Engine bridge、Spawn verifier 全通过；最新 MOD 已自动部署 HMCL 26.3。

## 2026-10-03 · 标签化 Vanilla 群系匹配 + 预览选点/箭头修复 + Savanna 草面

用户确认前一版 HMCL JAR 仍显示十字标记且无法选出传送目标，并指出热带草原表面是泥土；同时提出 Vanilla 主世界群系与 GeoWorld 地区都应打语义标签，在相似候选中按 Seed 做稳定的区域级随机选择（例如橡树林/桦木林）。

本轮处理：

- 核对 HMCL 原 JAR：时间 21:00:17、SHA256 CB8E...，对应 commit d13835f。该版本确实仍在 GeoWorldPreviewScreen 中把玩家和目标都画成十字；之前称“已换成箭头”不准确。
- 预览玩家标记改为真正的红色方向箭头。箭头直接读取当前玩家 yaw，地图 +Z 向下，因此 0° 朝南会指向地图下方，东西北方向同步旋转。
- 传送目标改为小型黄色菱形，不再与玩家位置混淆。
- 地图输入重新实现为“点击/拖动阈值”模型：左键按下仅进入 pending，累计移动 >=3px 才开始平移；若释放前未达到拖动阈值，则在释放坐标创建传送目标并启用传送按钮。解决旧逻辑把点击与拖动绑定到同一次 press，导致选点交互不稳定的问题。
- 传送仍只允许游戏内单人世界预览；选择成功后目标按钮启用，服务端先加载目标 Chunk，再取 WORLD_SURFACE 实际高度后 teleport。
- Savanna 在当前 t4 模型中 581/581 个抽样点的 SurfaceProfile 都是 Dry Grass Soil。此前 ProfileMapping 将 Dry Grass Soil 映射为 coarse_dirt，因此“热带草原是泥土”来自 Minecraft 表现映射，而非生态模型错误。现改为 Dry Grass Soil -> grass_block；Shrub Soil 仍保留 coarse_dirt。
- 新增 Minecraft26_3BiomeCatalog：
  - Vanilla 群系拥有温度、湿度、生态、地貌、土壤/表层、季风、海岸等语义标签和权重；
  - GeoWorld 每个地区从真实 Runtime 的 BiomeFamily、TerrainVariant、SurfaceProfile、VegetationProfile、年均温、年降水、季风、是否近海生成同一套标签；
  - 先过滤不兼容候选，再按匹配分数选出“足够接近”的候选集合，最后使用 Seed + 360/150 blocks 平滑区域噪声做确定性随机选择；
  - 因此同一片温带森林可自然形成 forest / birch_forest / dappled_forest / flower_forest / old_growth_birch_forest 等连续斑块，而不是每个格子随机跳变，也不再“一种 GeoWorld family 永远只对应一个 Vanilla biome”。
- 海洋、河流、湖泊、海岸和洞穴仍保持强物理约束，不交给泛化随机池；例如 River 一定是 river/frozen_river，Ocean 仍按温度与深度分层。

验证：
- 当前 t4（Seed 6311471718325446117）96-block 全图抽样得到 48 种地表主世界群系；温带/亚热带森林实际出现 forest、birch_forest、dappled_forest、flower_forest、dark_forest、old_growth_birch_forest 等多个匹配变体。
- t4 的 Savanna SurfaceProfile 全部为 Dry Grass Soil，新 Minecraft 表面映射全部为 grass_block。
- go test ./... -count=1 全绿。
- NeoForge compileJava --rerun-tasks 通过。
- Minecraft storage/save writer 验证通过。
- Java GoldenVerifier 15/15。
- build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn 通过并自动部署 HMCL 26.3。

## 2026-10-03 · 预览鼠标交互根因修复

- 用户反馈最新版中地图仍无法点击、拖动、传送。
- 根因已确认：Minecraft 26.3 的 MouseButtonInfo 使用 1 表示主/左键；GeoWorldPreviewScreen 之前仍按传统 GLFW 值 0 判断左键。因此地图区域的 click/drag/release 分支实际上从未进入。原版 AbstractWidget.isValidClickButton() 同样明确判断 buttonInfo.button() == 1。
- 统一新增 PRIMARY_MOUSE_BUTTON = 1，并修复地图点击、拖动、释放三处判断。现在左键单击会生成黄色传送目标，按住左键移动 >=3px 会平移地图，松开结束拖动；传送按钮在目标选中后启用。
- 去掉固定左侧信息栏，地图重新占用整个可用宽度；位置信息改为鼠标悬停时使用 Minecraft 原生 tooltip 显示，并按屏幕宽度自动折行。
- 保留红色玩家朝向箭头、黄色目标菱形、滚轮缩放和底部图层/重置/传送操作。
- NeoForge compileJava --rerun-tasks 通过；build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn 通过并已部署 HMCL 26.3。

## 2026-10-03 · 山脉内部峰谷塑形

用户反馈：当前已经有山脉走向/抬升趋势，但整体更像持续抬升的大坡，没有明显山峰、山谷、鞍部。

根因核对（t5 / Seed=123456）：
- 截图附近 X=1798 Z=24 的 Runtime 为 Mountain / Alpine，旧高度约 Y216；
- 旧 X 向剖面从约 Y59 持续升到 Y270，宏观山带存在，但内部结构基本单调；
- Mountain 区中位高度约 Y248，90% 高位约 Y297，且若干区域已经逼近 Y320；
- 旧山谷默认 ValleyDepth=0.007、CanyonExtraDepth=0.006，折算到海平面以上只有约 3~4 格量级；截图山带的 ValleyWeight 甚至为 0，因此几乎没有可读的山谷切割。

本轮实现：
- 保留 tectonic / landform MountainWeight 作为“山脉在哪里”的宏观包络，不用噪声重新决定山脉位置。
- 山脉内部改为多尺度塑形：
  - 760 blocks broad ridged field：主山脊；
  - 360 blocks fine ridged field：支脉/鞍部；
  - 两级 ridge 交汇生成稀疏主峰；
  - 270 blocks 小尺度波动用于打破长坡平面感；
  - 低 crest 区生成 structural saddle / dry head valley。
- 山脉整体抬升从 0.125 降到 0.090，把高度预算从“整片抬高”转给峰谷结构。
- 对 base > 0.82 的高山带做只作用于 MountainWeight 的 upper-envelope compression，为真正峰顶预留 Minecraft Y 上限空间，避免大片贴 Y320。
- 已有水文 ValleyWeight/CanyonWeight 在 foothill/mountain 内增强切深：平原保持旧深度，高山谷地可获得明显 V 型落差；不会随机把平原挖成峡谷。
- Go terrain 与 Java GeoWorldV1 Runtime 完整镜像同一算法，并重新生成跨语言 golden vectors。

t5 实测变化：
- Mountain Y：p10≈181.7 / p50≈236.0 / p90≈287.6 / max≈313.8；
- Y>=319.5 的 Mountain 抽样由旧的顶格现象降为 0；
- 截图附近 X 剖面 32-block 采样已经出现明显局部峰/鞍，例如：
  - Y213.3 -> 203.7 -> 221.1 -> 214.1；
  - Y245.6 -> 235.9 -> 245+；
  不再只有单调抬升。
- 统一 Mountain fixture 2.4km 剖面 peak-to-saddle normalized relief=0.0875（约 45 blocks），24 个局部极值；
- 高位 base=0.985 fixture 不再产生大面积 ceiling saturation，采样范围约 0.9496~0.9950。
- 水文山谷 fixture ValleyCarve≈0.0317，最终垂直差约 0.0264（约十几格以上）。

验证：
- go test ./... -count=1 全绿；
- terrain 新增峰谷/山谷深度/高山 headroom 三组回归测试；
- Java compileJava --rerun-tasks 通过；
- Java GoldenVerifier 15/15；
- Minecraft storage/save writer 通过；
- verifyGeoWorldEngine / verifyGeoWorldSpawn 通过；
- 最新 MOD 已自动部署 HMCL 26.3。

## 2026-10-04 · 河网连续性、湖泊泄洪口、河面与山峰收口

用户针对 t7 / Seed=123456 实测反馈：
- 山脉已有趋势但缺少明确峰体；
- X≈9769 Z≈-474 附近湖泊出现高架水库/水墙；
- 河流缺少支脉，主脉/支脉存在断流；
- 河流水面仍有两侧有水、中间无水的情况，希望直接检查已加载 Chunk。

调查与修复：
- 核对 t7 的实际存档与 Region：截图位置属于 overworld Region r.19.-1.mca，确认问题来自真实 ChunkGenerator 结果，不是地图 MOD 显示误差。
- 河网拓扑闭合：RiverStrength 只决定河流从哪里开始；一旦开始，后续沿 FlowTo 持续追踪直到湖泊、海洋或模型出口，不再因为下游局部 strength/坡度不足而中断。
- Reconcile 默认 RiverStrengthThreshold 0.30 -> 0.20；Hydrology RiverQuantile 0.965 -> 0.95，保留更多合理支流。
- 新 Seed=123456 模型：RiverSegment 由旧约 368 增至 948；70 个终点，错误终点 0。
- 主脉深度随河宽增加：headwater/base depth 保持原值，宽河可达到约 3x 基础中心深度；河床 profile 同样随 width 加深。
- 修复重叠河段造成的断水：River 水面 datum 绑定“横向权重最高的实际所属河段”，而不是误取邻近更深分支；在 exact confluence tie 时取较高上游 bed，避免两条已连接中心线之间出现干格。
- 对 Seed=123456 的 948 条河流中心线按约 2 blocks 间隔全量采样：67398 个采样点，dry=0。
- 河流横断面改成同一水面 datum，靠河床/河岸起伏产生深度变化；946 个抽查横断面 nonFlat=0。
- 湖泊增加 detailed spill-rim 复核：宏观 hydrology 判定 lake 后，再沿实际细节地形外缘寻找低泄洪口；若低于原湖面则主动降低 lake datum，并同步压低 lake bed ceiling，避免高架水库墙。
- t7 截图附近原问题湖的旧 lake surface 约 Y124；新模型按真实 rim 修正后约 Y108。
- 山地增加 cellular localized summit：约 720 blocks 分布尺度的稀疏峰节点叠加在 ridge network 上，不再只生成长而圆的山脊；保持已有 saddle / hydrologic valley 结构。

验证：
- go test ./... -count=1 全绿；
- 新增 Reconcile downstream topology closure、Lake detailed spill-rim 等回归测试；
- golden vectors 重新生成；
- Java Runtime GoldenVerifier 15/15；
- Minecraft save writer/storage 验证通过；
- NeoForge compileJava --rerun-tasks 通过；
- build.cmd verifyGeoWorldEngine verifyGeoWorldSpawn 通过；
- 最新 MOD 已自动部署 HMCL 26.3。

## 2026-10-06 · 预览性能、湖泊连续性与悬河修复

针对 Seed=123456 / t8 实测截图：
- 世界预览打开及拖动明显卡顿；
- 大型湖泊面积异常且视觉上破碎；
- 河流水面被下游湖/海水位抬高后形成明显“地上悬河”。

根因与修复：
- Java Runtime 原先每次 sample 都线性扫描全部 RiverSegment / LakeNode；当前 Seed=123456 有约 948 条河段，预览 160x100 一次就会产生大量无效相交计算。GeoWorldV1 现按 128-block 级 feature bucket 建立河流/湖泊空间索引，只检查当前位置附近候选。
- 生态总览原先每个预览像素还会再次调用 Minecraft26_3BiomeRealizer 做邻域群系匹配。总览颜色现在直接使用 GeoWorld Runtime 的 biomeFamily / surfaceProfile；最终 Minecraft 群系仍在悬停/选中点时精确解析。
- 地图拖动不再每个 mouseDragged 事件同步全量重算；约 50 ms 节流，并在释放鼠标时保证最后一次精确重建。
- 160x100（16000 点）本机基准：旧逻辑约 2996 ms；空间索引后且生态总览不做逐像素最终 MC 群系解析后，首轮约 100 ms、JIT 热身后约 32-34 ms。

湖泊：
- Hydrology lake candidate 继续来源于真实 depression/fill/spill 结果，但同一显式湖组件要求 BasinID 一致且 spill/fill level 在约 0.003 normalized 容差内，避免不同水位的相邻洼地被简单 D8 连通成一个湖。
- Waterbed 湖泊 footprint 使用可覆盖 D8 对角邻居的 rounded-square/superellipse 支撑；outer radius 约 0.72 cell，避免合法同一湖内部出现棋盘式空洞。
- 详细地形最终决定真实岸线：若 detailed terrain 已高于 spill-controlled lake datum，则不再为了宏观湖候选强行向下挖成水面。
- detailed spill-rim 仍会压低湖面，防止高架水库/竖直水墙。
- 对 t8 截图对应的大湖区域按 16-block 网格复核：旧持久化模型约 7805 个 lake sample；新模型约 1778，减少约 77%；新结果仅 2 个连通水体，对应模型中的 2 个湖，不再由一个湖内部碎成大量孤立水块。

河流 / 悬河：
- ReceiverSurface 不再沿整条 downstream graph 反向传播到所有上游河段，只允许“直接进入湖泊/海洋”的最终河段拥有 receiver datum。
- Receiver backwater 仅在终端河段靠近河口的末段平滑生效，避免整条上游河被抬到湖面/海面高度。
- 河床 profile 额外参考河道两侧 detailed terrain 的局部低廊道；宏观 D8 直线跨过细节小脊时会向低地切入，不再把河架在邻近低地上。
- Seed=123456 dry-bank audit（河心水面高于明确干岸 1.5 blocks 以上）：旧 t8 = 275 侧、最大约 17.87 blocks；新模型 = 3 侧、最大约 2.39 blocks。

兼容与验证：
- GeoWorld engine cache key 升级 v6 -> v7，避免新世界误复用旧水文模型。
- go test ./... -count=1 全绿。
- Golden vectors 已刷新，Java GoldenVerifier 15/15。
- NeoForge compileJava、verifyGeoWorldEngine、verifyGeoWorldSpawn 全绿。
- 最新 MOD 已自动部署到 HMCL 26.3。
- 已存在的 t7/t8 会继续读取自己持久化的旧 world.geoworld 和旧 Chunk；必须新建世界验证本轮地形/水文变化。

## 2026-10-07 · 汇流水面连续性与河口回水收口

继续针对 Seed=123456 的剩余悬河风险做 Runtime 逐点审计后，确认剩余问题不是互不相干河流穿叉，而是真实汇流节点的水面 datum 切换：

- 原模型只保证河床 profile 单调下降，但 Runtime 水面为“河床 + 随河宽增长的中心水深”。支流汇入更宽主河时，即使河床不抬升，也可能因为主河更深而让水面突然上跳。
- Seed=123456 严格扫描原有 568 次 Runtime 水面上升中，561 次为 River -> River；其中绝大多数发生在汇流节点或刚离开汇流节点。另有 6 次 River -> Ocean、1 次 River -> Lake，来自接收水体已经接管但旧 backwater blend 尚未开始。
- Waterbed 现在与 WorldSample 共享同一 RiverCenterDepthNormalized，并在构建河网 DAG 时同时传播有效河床和有效水面；下游河变宽时主动降低 junction bed，保证 bed + depth 不向上跳。
- 多支流汇合时，在几何 footprint 开始重叠之前平滑收敛到共享水面；精确汇流中心的 datum tie 由 downstream segment 接管，Go 与 Java Runtime 使用同一规则。
- Receiver backwater 从旧的终段 68%-100% 改为约 28%-42% 完成过渡，使水面在实际进入湖泊/海洋 footprint 前已经衔接到接收水位；合法河口回水单独计数，不再与悬河异常混在一起。
- Hydrology LakeID 继续表示宏观 depression candidate；详细地形高于最终 lake datum 的候选节点允许保持干地。Consistency Audit 与 Tester 的湖床异常统计都同步采用该语义，避免为了“审计归零”重新把高地强挖成湖。

模型验证：
- Seed=123456：RuntimeRiverSurfaceUpslopeSteps 568 -> 0；RuntimeLakeNodeViolations 268 -> 0；WaterbedRiverUpslopeSteps=0；WaterbedLakeBedViolations=0；合法 Receiver backwater steps=50。
- 正式 Seed 731894205：Waterbed/Runtime 河湖异常均为 0；合法 Receiver backwater steps=82。
- 正式 Seed 10400：Waterbed/Runtime 河湖异常均为 0；合法 Receiver backwater steps=152。
- 当前 strict-final 命令仍会因既存 CoastTerrainMacroSlopeMismatch > 1e-4 门槛退出 1（731894205≈2.39e-4、10400≈4.50e-4、123456≈2.72e-4）。该门槛早于本轮水系改动存在，且与项目文档中“正式 Seed strict-final 应通过”的记录不一致；本轮不通过放宽阈值掩盖这个独立海岸问题。

回归与部署：
- go test ./... -count=1 全绿。
- Java/Go GoldenVerifier 15/15，通过 tolerance=1e-9。
- NeoForge compileJava、verifyGeoWorldEngine、verifyGeoWorldSpawn 全绿。
- Engine cache key v7 -> v8；verifier 已确认 v8 首次生成 cached=false。
- java/neoforge-mod/build.cmd BUILD SUCCESSFUL，并已自动部署最新 GeoWorld-NeoForge-26.3-0.1.0-dev.jar 到 HMCL 26.3。

## 2026-10-07 · GeoWorld 模型群系正确但 Chunk 落成 Plains 修复

用户在 Seed=123456 / t9 的高山区域发现：游戏地图 X≈1908 Z≈-428 显示 Plains，但 GeoWorld 预览 X≈1945 Z≈-382 明确为 Alpine / Mountain / Alpine Snow，模型目标群系为 Stony Peaks。

定位：
- 直接加载 t9/data/geoworld/world.geoworld，精确采样 X=1908 Z=-428：BiomeFamily=Alpine、TerrainVariant=Mountain、SurfaceProfile=Alpine Snow，SemanticFallback=false、SemanticMismatch=false；模型本身正确。
- 直接读取 t9 对应 Chunk (119,-27) 的 Anvil NBT，biome palette 只有 minecraft:plains，证明不是地图 MOD 显示错误，而是 Chunk 群系写入错误。
- 根因是 world_preset/geoworld.json 仍停留在早期 15 个 biome 的列表，而 Minecraft26_3BiomeRealizer 已可输出数十个 Vanilla 26.3 biome。当前高山点目标为 minecraft:stony_peaks，但旧 preset 没注册该 Holder；GeoWorldBiomeSource 的 getOrDefault 最终静默退回 minecraft:plains。

修复：
- GeoWorld world preset 的 overworld biome_source 扩充为 56 个当前 Realizer 实际可输出的 biome，包括 stony/frozen/jagged peaks、snowy slopes、grove、river、beach、深海变体、cave biomes 等。
- GeoWorldBiomeSource 不再对缺失目标 biome 直接无脑退回 Plains；对旧 level.dat 仍只保存旧 biome 列表的世界，按语义选择兼容 fallback。例如 stony_peaks/frozen_peaks/snowy_slopes 优先退回 jagged_peaks/snowy_plains。
- Minecraft26_3BiomeCatalog 暴露 candidateBiomeIds，Minecraft26_3BiomeRealizer 暴露 requiredBiomeIds。
- 新增 verifyGeoWorldBiomePreset，并接入 Gradle check。以后 Realizer 增加了新 biome 但 preset 未注册，正式 build 会直接失败。

验证：
- verifyGeoWorldBiomePreset：preset=56 / required=56，通过。
- Java/Go GoldenVerifier 15/15 通过。
- verifyGeoWorldEngine / verifyGeoWorldSpawn 通过。
- NeoForge build SUCCESSFUL，最新 JAR 已自动部署 HMCL 26.3。
- 已生成的 t9 Chunk biome palette 不会被 MOD 自动重写；新建 GeoWorld 可验证精确 biome，t9 尚未生成的新 Chunk 会使用兼容 fallback，不再因缺 Holder 直接退回 Plains。

## 2026-10-07 · 河/湖/海短岸带与稳定水边

用户确认群系落地修复后，继续优化河湖海岸：既要消除河边低一格空槽被 Minecraft 流体更新填成非水源水的问题，又不能把所有岸线做成统一高一格堤坝；同时需要保留甘蔗等原版滨水植物的自然生长条件。

实现：
- 新增 GeoWorldShoreline，作为 Minecraft 方块落地层的短距离 shoreline pass，不改宏观水文/河湖拓扑。
- buildTerrain、getBaseHeight、getBaseColumn 统一使用同一 Column plan，避免实际方块、结构高度查询和 NoiseColumn 对岸线高度认识不一致。
- 直接邻接源水的 dry column 若因整数高度量化落到水面以下，则补到稳定水边界；实现层硬限制最多补 2 格，避免未来异常模型被掩盖成人工高堤。当前 Seed=123456 fresh model 验证最大补高 2 格，湖/海最大 1 格。
- 低坡岸允许形成 1~2 格 littoral shelf；陡岸、高山/岩质岸保留 hard bank。美术性削坡最多 1 格。
- 材质带严格保持短距离：River 最多 2 格、Lake 最多 3 格、Ocean 最多 4 格。使用 coherent Seed noise 形成连续小斑块，不做逐格椒盐噪声。
- 岸材质按环境选择 sand / mud / dirt / coarse_dirt / gravel，并用少量 clay 点缀；湿地偏 mud，干旱区偏 sand/gravel，岩岸偏 gravel/coarse_dirt，海岸偏 sand。
- 原版 26.3 SugarCaneBlock 规则已核对：SUPPORTS_SUGAR_CANE 接受 substrate_overworld（dirt/mud/grass 等）和 sand，且要求同高度水平邻接支持甘蔗的水。普通河/湖直接水边因此优先保留这些基质，gravel/clay 仅作少量局部斑块。

验证：
- 新增 verifyGeoWorldShoreline，并接入 Gradle check；verifier 使用独立临时 engine cache 强制 fresh model，避免长期 v8 cache 污染回归判断。
- Seed=123456 同时覆盖 River / Lake / Ocean：immediate=615，stable=615，littoral shelves=714，material transitions=941。
- 适合植物生长的直接邻水岸 caneReady=566/579（约 97.8%）。
- max aesthetic lowering=1；max stability raising=2（River），Lake=1，Ocean=1；4+ 格人工堤坝=0。
- go test ./... -count=1 全绿。
- Java/Go GoldenVerifier 15/15。
- verifyGeoWorldBiomePreset、verifyGeoWorldShoreline、verifyGeoWorldEngine、verifyGeoWorldSpawn 全绿。
- NeoForge build SUCCESSFUL，最新 GeoWorld-NeoForge-26.3-0.1.0-dev.jar 已自动部署到 HMCL 26.3。

## 2026-10-07 · 岸线后填充、连续材质与稀疏山崖

用户对上一版短岸带实机验证后反馈：
- 岸边整体已经明显改善，但一格宽的阶梯/锯齿应向陆地方向补齐，不能改水体范围；
- 岸材质不能一格泥土一格沙子跳变，应形成连续的小段/小片；
- 世界大部分平缓符合预期，但山峰也过于平缓、基本可以直接走上去，希望少量真正陡峭峰体和自然悬崖。

岸线：
- River/Lake 低坡 littoral shelf 的第二排更积极向陆地补齐；仅修改 dry-bank 高度/表层，不扩张河湖海 water footprint。
- 岸材质 coherent noise 从原先约 18/7 blocks 的细碎尺度改为 46/21 blocks，主材质由更低频场决定，小尺度只弯曲边界。
- Seed=123456 fresh verifier：immediate shore 615/615 稳定；littoral shelves 714 -> 1132；material transitions=1167；适合植物环境的直接邻水岸 caneReady=579/579。
- 相邻直接水边材质仅 5/347 次发生切换（约 1.4%），避免“一格沙一格泥”的椒盐外观。

山体：
- 保留平原/丘陵现有低频平缓尺度，不修改 PlainAmplitude。
- 现有 broad/fine ridge 继续决定山脉内部结构；localized summit 收窄，提高峰体集中度。
- 新增更窄的 summitSpire，仅在 MountainWeight + ridge crest 内出现，不会在平原生成孤立尖刺。
- 新增稀疏 structural fault escarpment：由低频 fault field 形成连续但窄的陡坎，并使用独立 regional gate 控制稀疏度；不是给所有山坡统一加噪声。
- synthetic 6000x6000 mountain scan（16-block slope）：cliff-scale >=0.040 的坡面 51/282752，约 0.018%；最大 delta=0.053582。Plain 对照 cliff=0。
- Seed=123456 当前新模型实测：X=16760 Z=-3532 为 Mountain/Alpine，16 格内 delta=0.067456，折算约 34.7 Minecraft blocks；高峰 X=-12388 Z=5164，约 Y=318，未形成大片 Y=320 ceiling。
- 新增 TestMountainStructureIncludesSparseCliffs 与 TestPlainTerrainRemainsGentleAfterMountainCliffs，约束“确实有悬崖但必须稀疏、不能泄漏到平原”。
- Engine cache key v8 -> v9，确保新世界不复用旧的平缓地形缓存。

验证：
- go test ./... -count=1 全绿。
- Golden fixtures 已重建；Java/Go GoldenVerifier 15/15，tolerance=1e-9。
- verifyGeoWorldBiomePreset / verifyGeoWorldShoreline / verifyGeoWorldEngine / verifyGeoWorldSpawn 全通过。
- 最新 NeoForge JAR 已自动部署到 HMCL 26.3。

## 2026-10-07 · 可航行河网、河谷塑形、湿地浅水与湖泊排水闭环

用户实机发现河面跟随地形频繁产生 Minecraft 整数 Y 台阶，划船顿挫且无法向上跨 1 格水；部分河段过直；沼泽几乎只有泥地；大型内陆湖看起来既无入流也无出流。同时复核 Minecraft 26.3 主世界群系覆盖。

河流纵坡与游戏性：
- Waterbed 对半宽 >=8 blocks、单宏段低坡（raw normalized drop <=0.006）的河段加入 navigable surface snapping。
- 导航水位层级 navigationSurfaceStepNormalized=0.009，约对应 4~5 个 Minecraft 高度格；累计高差集中到更少的急流/跌水位置，而不是每几十格掉一级。
- total drop >0.014 的陡河不做导航平整，保留山溪、急流和瀑布语义，不把所有河削成运河。
- Receiver backwater 从“河口斜坡渐升”改为“接收水位 floor”：若湖/海水位高于自然河面，回水区保持接收水位，并只向上游传播到自然河面重新高于接收水位为止。
- 最终渲染几何若河道实际穿过湖泊 footprint，即使粗 Hydrology FlowTo 未标记入湖，也视为真实水力连接并使用湖面 backwater floor，避免河->湖突然出现向上水墙。
- 正式新增 cmd/geoworld-river-navigation-audit。Seed=123456 / 每 4 blocks 中心线采样 strict 结果：
  - low-gradient small 8-12: 4.5868 changes/km
  - low-gradient medium 12-16: 3.0073 changes/km
  - low-gradient large >=16: 1.4030 changes/km
  - downstream rises=0
  - dry centerline samples=0
  - steep/rapid reaches单独统计，不纳入可航行低坡阈值。

河道平面形状与河谷：
- RiverSegment 新增 MeanderOffset；低坡河段可生成轻微单弯曲流，约 20% 河段保持直线，避免所有河都机械蛇形。
- 汇流前 meander 自动收敛到 35%；confluence profile harmonization 将 meander support 算入提前统一水位的距离，修复两条弯曲支流在汇流口前过早重叠导致约 4~5 格水位错位。
- 河流采样改为真实曲线中心线 + Newton 最近点，不再以直线 chord 近似。
- 河道横断面与外侧 dry bank 统一成“河床最低 -> 水边 shoulder -> 12~24 blocks 内平滑恢复原地形”的连续河谷缓坡。
- GeoWorld Tester 的异常扫描和河流诊断图同步改为沿曲线中心线采样/绘制。

湖泊水文闭环：
- Reconcile 不再在河流进入湖后永久终止整条水路。
- 每个湖至少保留一条 strongest direct feeder；即使该支流略低于全局可见阈值，也向上游追最多 3 个宏节点以形成可读的入流，而不是湖边一小截蓝线。
- 对有 SpillOutlet 且 FlowTo 离开湖体的湖，从 SpillOutlet 强制继续 trace 到下游河网/海洋。
- 新增 TestLakeKeepsVisibleFeederAndSpillOutflow。
- Seed=123456 截图大湖（LakeID=2，约 1.884M blocks²）当前确认：
  - 29 条宏观汇水边
  - 5 条可见入流
  - strongest feeder 9152,-832 -> 9024,-832，strength=0.743
  - spill outlet 8000,-576 -> 7872,-576 已进入最终 RiverEdges，strength=0.743

沼泽浅水：
- GeoWorldShoreline 新增 wetland pool：Wetland / Wet Mud / Peat 语义中的低平 dry 区域可生成 1 格深浅水洼，池底 mud，保留泥地/草岛/浅水混合。
- 使用低频 coherent noise，不做逐格随机孔洞；避开已有河湖海并限制局部坡差。
- Seed=123456 verifier：center=1568,-3168，wetland pools=393/7059，ratio≈5.6%。

岸线：
- 直接邻水仍最多 solid raise 2 格；若仍恰差 1 格可改成一格 source-water fringe，而不是造 3 格人工堤。
- 最新 Seed=123456 shoreline verifier：immediate=574，stable=574，shallowFringes=10，shelves=1046，materials=1069，caneReady=528/528，4+ raise=0。
- 岸材质连续性仍保持：materialChanges=5/318。

Minecraft 26.3 群系：
- GeoWorldBiomePresetVerifier 现在直接读取 Minecraft 26.3 data/minecraft/tags/worldgen/biome/is_overworld.json 与 GeoWorld preset 比较。
- Vanilla Overworld=56，GeoWorld preset=56，missing=0，extra=0。
- 因此当前已注册全部 Minecraft 26.3 主世界群系；后续若有“看不到某群系”，应调查实际生成概率/可达条件，而非注册缺失。

版本与验证：
- Engine cache v9 -> v10，确保相同 Seed 新建世界不复用旧河网/水面缓存。
- go test ./... -count=1 全绿。
- geoworld-river-navigation-audit -strict PASS。
- Java/Go GoldenVerifier 15/15，tolerance=1e-9。
- verifyGeoWorldBiomePreset：56/56，missing=0，extra=0。
- verifyGeoWorldShoreline + wetland pool PASS。
- verifyGeoWorldEngine：v10 cached=false PASS。
- verifyGeoWorldSpawn PASS。
- NeoForge BUILD SUCCESSFUL；最新 GeoWorld-NeoForge-26.3-0.1.0-dev.jar 已自动部署到 HMCL 26.3。


## 2026-10-08 · 河流阶梯支撑、河岸去沼泽化与真实湖岸入流闭合

用户实机反馈：
- 主河两岸几乎必然出现沼泽群系，生态表现不合理。
- 沼泽浅水/河岸出现开放水源，形成大面积流动水并带来 fluid update 卡顿。
- 平缓地形的河流台阶仍出现多格悬落，应该是河床同步下降的一格阶梯。
- 即便是真正悬崖/峡谷，瀑布也应贴着实体崖壁，不允许自由悬空水帘。
- 大湖部分支流虽然 Hydrology/Reconcile 有边，最终 Runtime 仍未真正碰到湖水。

本轮修正：

### 1. 河岸不再自动湿地化
- Hydrology WetlandPotential 增加强河道 exclusion：高 RiverStrength 本身只表示河槽，不再直接把整条主河两岸提升成 wetland。
- Minecraft26_3BiomeRealizer 取消 “Wet Mud + Floodplain => Wetland” 与 “高降水 Floodplain => Wetland” 的自动升级。
- Wet Mud 仍作为冲积/湿润土壤存在，但不再等价于 Minecraft swamp biome。
- Seed=123456 GeoWorldShorelineVerifier：
  - riverbank cells=1568
  - swampCells=0
  - unsupported=0
  - regression biome=minecraft:sunflower_plains

### 2. 湿地浅水改成封闭小盆地
- 浅水池不再逐格随机生成 source water。
- 采用约 10x10 方块候选尺度，生成约 3x3 的一格深浅水盆地，并规划一格泥岸 rim。
- 同一池使用统一 water boundary；池底为 mud。
- 外圈若仅低一格可抬为泥岸封闭；更深逃逸口拒绝生成该池。
- 浅水地貌允许出现在 Wetland / Wet Mud / Peat 语义，但不会因此把周围 climate biome 改成 swamp。
- Seed=123456：wetland pools=254/6416，ratio≈4.0%。

### 3. 河流纵向台阶不再多格自由悬落
- Waterbed 最终增加 supported-cascade pass，对每条 RiverSegment 最终 Profile 再约束连续纵坡。
- 河流内部、junction、receiver mouth/outflow 都统一处理，不只看单段 profile。
- 普通/急流在 Minecraft 整数柱中相邻水面最多差 1 格；河床同步下降。
- 正式 cmd/geoworld-river-navigation-audit 新增：
  - maxAdjacentWaterStepBlocks
  - adjacentWaterStepViolations
  - lakeFeeders
  - disconnectedLakeFeeders
- Seed=123456 strict：
  - maxAdjacentWaterStepBlocks=1
  - adjacentWaterStepViolations=0
  - downstream rises=0
  - dry centerline samples=0
  - low-gradient changes/km:
    - small 8-12 = 1.7304
    - medium 12-16 = 1.2809
    - large >=16 = 0.2210
- 对“瀑布必须贴崖壁”的语义收口：只有连续暴露 2 格以上的竖直水面才属于真正 waterfall；当前 Seed=123456 已不存在这种自由悬落。普通一格阶梯不强行塞支撑块，避免把河道做成障碍物。

### 4. Receiver 回水同时塑岸
- 之前 receiver/backwater 只抬最终水面，Waterbed 横断面仍按自然河面塑岸，导致低 RiverWeight 河边出现 Y105 地面面对 Y109 回水面的悬空水墙。
- Go Waterbed 与 Java GeoWorldV1 同步改为：河槽/岸肩横断面按最终 receiver surface 计算。
- 最新 Shoreline verifier：
  - immediate=579 / stable=579
  - shallowFringes=0
  - shelves=1084
  - materials=1140
  - caneReady=543/543
  - maxRaising=1
  - 2-3 raise=0
  - 4+ raise=0

### 5. 大湖支流连接到“最终真实湖面”而不是 LakeID 抽象边界
- 确认根因：Hydrology LakeID 是宏观候选盆地；详细 Terrain 会让大量候选 cell 最终保持干地。
- 原先 feeder 到达 LakeID 就算“入湖”，会出现 9024,-832 已是 LakeID=2，但真实湖岸约在 10048 一带，中间超过 1 km 干地的情况。
- Reconcile 仍以 LakeID 作为宏观 handoff；Waterbed 在最终湖 footprint 建成后执行 connectFeedersToRenderedLake：
  - 从真正有 Runtime Lake 水的 macro node 建立目标集合。
  - 对未碰真实湖水的 feeder，在同 LakeID basin 内使用带高程惩罚的最小代价路径寻找真实湿节点。
  - connector 优先走低谷，不直切山脊。
  - connector 生成正式 RiverSegment，并继承 lake receiver surface / cascade / confluence 约束。
- 正式 navigation audit：
  - lakeFeeders=6
  - disconnectedLakeFeeders=0
- 截图中的 LakeID=2 五条主要 feeder 复核全部 hit=true：
  - 10944,-1216 -> 10816,-1216
  - 9152,-832 -> 9024,-832（原断开）
  - 8128,-704 -> 8000,-704（原断开）
  - 10560,192 -> 10432,192
  - 10048,576 -> 10176,448

### 6. 验证环境与版本
- Engine cache v10 -> v11，确保新世界不会复用旧河网/岸线缓存。
- GeoWorldShorelineVerifier 不再把 embedded engine 解压到 Windows %TEMP% 执行；改用 module build/geoworld-shoreline-verifier-cache，避免 CreateProcess error=5。
- Go Golden 已重建；go test ./... -count=1 全绿。
- Java Golden 15/15，tolerance=1e-9。
- NeoForge：
  - verifyGeoWorldBiomePreset: 56/56
  - verifyGeoWorldShoreline: PASS
  - verifyGeoWorldEngine: PASS, v11 cached=false
  - verifyGeoWorldSpawn: PASS
  - BUILD SUCCESSFUL
- 最新 GeoWorld-NeoForge-26.3-0.1.0-dev.jar 已自动部署到 HMCL 26.3。

## 2026-10-08 · 岸后狭长沟槽闭合与截图河湖拓扑核验

用户实机截图新增两个问题：
- 河/湖岸后第 2～4 格仍会出现沿岸延伸的单格深条形沟槽。
- 10720,-1312、9958,190 一带河道与 Lake 2 很近，需要按模型拓扑确认是否应当连接，不能仅凭视觉强连。

### 河湖连接结论
对 Seed=123456 逐层核对 Hydrology -> Reconcile -> Waterbed -> Runtime。

Lake 2 的正式 SpillOutlet 为：
- node 30622: 8000,-576
- FlowTo node 30621: 7872,-576
- 最终 Reconciliation.RiverEdges 中存在 30622 -> 30621

截图附近两个看似“湖泊出口”的原始 FlowTo：
- 10816,-1344 -> 10688,-1344 (28724 -> 28723)
- 10048,192 -> 9920,192 (32558 -> 32557)

两者都满足 raw Hydrology FlowTo，但：
- recEdge=false
- spillFrom=false

因此它们不是最终河网连接，也不是 Lake 2 的正式出湖口。保持截图位置河湖局部不连接是符合最终模型的；不得因为视觉距离近而强制开渠。

Lake 2 最终边界河网仍为：
- 入湖：10944,-1216 -> 10816,-1216
- 入湖：9152,-832 -> 9024,-832
- 入湖：8128,-704 -> 8000,-704
- 入湖：10560,192 -> 10432,192
- 入湖：10048,576 -> 10176,448
- 正式出湖：8000,-576 -> 7872,-576

geoworld-river-navigation-audit -strict 保持：
- lakeFeeders=6
- disconnectedLakeFeeders=0
- maxAdjacentWaterStepBlocks=1
- adjacentWaterStepViolations=0

### 岸后条形沟槽
将截图坐标 2023,-2985 与 10883,-1293 加入 Shoreline 回归窗口后，确认问题不是视觉错觉。

典型最终方块横断面：
- shore distance 2: Y89
- shore distance 3: Y88
- shore distance 4: Y89

即最终 Shoreline 规划形成连续的 89-88-89 单格深长槽。初次扫描共识别 narrowBackshoreGrooves=227。

根因：
- 原 Shoreline 每列独立规划。
- 距离 2 的岸行会被 shelf/stability 规则抬高，但距离 3 行只看到 Runtime 原始高度，无法知道相邻列最终会变成更高的岸台。
- 因此在单列阶段使用原始邻居无法可靠消除最终 Minecraft 方块中的狭长沟。

修复：
- Shoreline 改成两阶段规划。
- 第一阶段先生成包含 1 格 fringe 的 preliminary Column 网格。
- 第二阶段只针对最终 planned geometry 做窄沟闭合。
- 仅当 dry shore distance=2/3，且同一水体岸线法向两侧分别是 distance-1 / distance+1，并且最终高度都比中心高时，中心抬 1 格。
- 最大只抬 1 格，并受 shoreWaterBoundary + 1 限制。
- 不修改 Runtime / Hydrology / Terrain，不填宽谷、排水洼地或普通自然坡面。
- planChunk() 与单点 plan() 使用同一两阶段规则，避免 Heightmap / BaseColumn 与真实区块地形不一致。

最新 Seed=123456 Shoreline verifier：
- narrowBackshoreGrooves: 227 -> 0
- pointPlanMismatches=0
- immediate=1197 / stable=1197
- maxLowering=1
- maxRaising=2
- wetland pools=254/6416 (~4.0%)
- riverbank swampCells=0
- Biome preset=56/56

完整回归：
- go test ./... -count=1: PASS
- Java Golden: 15/15
- geoworld-river-navigation-audit -strict: PASS
- NeoForge Engine / Spawn / BiomePreset / Shoreline: PASS
- BUILD SUCCESSFUL，最新 dev JAR 已部署到 HMCL 26.3。

本轮只修改 Minecraft Shoreline 最终方块整形与验证，不改变 .geoworld 模型，因此 Engine cache 继续使用 v11。

## 2026-10-08 · 海岸围水短缓坡与岸后短坑收口

用户实机截图新增：
- 河岸/湖岸在长槽修复后仍残留少量局部单格短坑。
- 海岸低地未完全围住海水，且陆地侧没有向内做短距离平滑，出现平地直接面对高海面的水墙。

诊断（Seed=123456）：
- 河岸截图中心：2100,-3010。
- 海岸截图中心：10325,2673。
- 海岸 Runtime 陆地约 Y60~62，Ocean water boundary=Y64，因此原规则“最多抬 2 格”不足以同时围水和向内平滑。

修复：
- River / Lake 保持原 raise cap <=2。
- Ocean 使用独立 4 格 backshore profile：
  - distance 1~2: 至少抬到 sea water boundary
  - distance 3: 至少 waterBoundary-1
  - distance 4: 至少 waterBoundary-2
- Ocean 最大允许 raise=3；不是全局放宽。
- 新增 oceanBackshoreStepViolations：向陆地每前进 1 格，高度不得突然下降超过 1 格。

岸后短坑：
- 在原有 narrowBackshoreGroove 二阶段修复后增加 smallBackshorePit。
- 仅处理 shore distance 2~4、同一水体岸后、至少 3 个相邻干地最终高度更高的局部单格坑。
- 同一格通过 small-pit 规则最多抬 1 次。
- 由于长槽闭合会产生新的邻接关系，规划扩展为有限多阶段：base -> groove close -> pit close -> one cascade pit close。
- Chunk plan 与 point plan 都使用相同 halo/stage，确保 Heightmap/BaseColumn 一致。

最新 Shoreline verifier（包含 2100,-3010 与 10325,2673）：
- immediate=1717 / stable=1717
- narrowBackshoreGrooves=0
- smallBackshorePits=0
- oceanBackshoreStepViolations=0
- pointPlanMismatches=0
- maxRaising=3 (Ocean)
- maxByKind={River=2, Lake=1, Ocean=3}
- wetland pools=254/6416 (~4.0%)
- riverbank swampCells=0

完整回归：
- go test ./... -count=1: PASS
- Java Golden: 15/15
- geoworld-river-navigation-audit -strict: PASS
- maxAdjacentWaterStepBlocks=1, violations=0
- lakeFeeders=6, disconnectedLakeFeeders=0
- NeoForge Engine / Spawn / BiomePreset / Shoreline: PASS
- BUILD SUCCESSFUL，最新 dev JAR 已部署到 HMCL 26.3。

本轮只修改最终 Minecraft Shoreline 方块整形与验证，不改变 .geoworld 模型，因此 Engine cache 仍使用 v11。

## 2026-10-08 · 河流横断面水面量化修复

用户在 10824,2515 附近发现河缘出现高一格的孤立 source water，并伴随局部岸线视觉缺口。

诊断：
- Runtime 河流水面本身基本同高，问题发生在 Minecraft 方块化层。
- 原 GeoWorldShoreline.waterBoundary 对 River 使用 boundaryYCeil 后，又强制 max(mapped, groundBoundary+1)。
- 当浅河边缘地面量化到与模型水面同一 Y boundary 时，该规则会把边缘水面抬高 1 格，形成 河边64 | 河心63 | 河边64 的横断面。

修复：
- River waterBoundary 严格服从 Runtime waterSurfaceElevation，不再为保证水深抬高水面。
- 若河床量化后顶到水面，则将该列河床下压 1 格，保证至少一格水层。
- findNearestWater 对 River 不再因 water<=ground 而丢失邻接水体识别。

回归：
- 新增截图回归中心 10824,2515。
- riverWaterBoundaryModelMismatches=0。
- riverDryVoxelColumns=0。
- riverBedQuantizationLowers=103：这些列由“抬水面”改为“下压河床 1 格”。
- local riverbank material gaps=0。
- Shoreline / Biome / Engine / Spawn 全部通过。
- go test ./... -count=1 全绿。

本轮只修改最终 Minecraft voxel shoreline，不改变 .geoworld 模型，因此 cache 仍为 v11。

## 2026-10-09 · 禁止 GeoWorld 随机开放式 vanilla 水泉

用户在高山/悬崖区域发现孤立单格水源。区块更新后该水源会开始流动，可能形成很长的水帘并产生大量 fluid tick。

定位：
- 截图附近约 10429,-1125 的 GeoWorld Runtime 为 Water=None / WaterSurfaceDefined=false。
- 因此该水源不属于 River / Lake / Ocean / WetlandPool，而来自 vanilla biome decoration。
- Minecraft 26.3 的 minecraft:spring_water 使用 SpringFeature：要求周围 5 个侧/下方位置恰好 4 个有效岩石、1 个空气；放置 source water 后立即 scheduleTick(..., 0)。
- 这正好允许悬崖侧壁随机生成开放泉眼，并启动自由水流。

策略：
- GeoWorld 的开放地表水由 Runtime 水文模型负责；暂时禁止 vanilla 随机 spring_water。
- 仅绑定 GeoWorldChunkGenerator，不修改普通 Minecraft 世界。
- 通过 ChunkGenerator(biomeSource, generationSettingsGetter) 为 GeoWorld 单独提供过滤后的 BiomeGenerationSettings。
- 只删除注册键 minecraft:spring_water。
- 保留 spring_lava、树木、花草、矿物、结构和其他所有 biome decoration feature。
- 保持 Decoration step 数量与其他 feature 的原有顺序不变。
- 对过滤后的 BiomeGenerationSettings 使用 WeakHashMap 缓存，避免区块生成时反复重建。
- BiomeFilter.biome() 的二次归属检查会调用 generator.getBiomeGenerationSettings(biome)，因此同样看到过滤结果。

验证：
- 新增 verifyGeoWorldDecorationPolicy，并接入 check。
- synthetic feature-step 验证：suppressed=minecraft:spring_water。
- preserved=minecraft:spring_lava,minecraft:flower_default。
- steps=10，order=stable。
- go test ./... -count=1: PASS。
- Java Golden: 15/15。
- geoworld-river-navigation-audit -strict: PASS。
- totalDownstreamRises=0，adjacentWaterStepViolations=0，disconnectedLakeFeeders=0。
- NeoForge BiomePreset / DecorationPolicy / Shoreline / Engine / Spawn: PASS。
- BUILD SUCCESSFUL，最新 dev JAR 已部署到 HMCL 26.3。

边界：
- 这一步不新增 GeoWorld 自有泉眼。未来如需要山泉，应由地下水/降水/集水区模型明确生成，并接入溪流或受控贴壁瀑布。
- 本轮不修改 .geoworld 模型与 Engine cache。

## 2026-10-09 · 最终 Minecraft 水体所有权审计

新增 verifyGeoWorldWaterOwnership，直接扫描最终 GeoWorldShoreline.Column，而不是只看 Hydrology/Runtime。

扫描范围（Seed=123456）：
- 12 个历史问题/关键水文坐标周围各 9x9 chunks。
- 完整模型范围按约 2048 blocks 间隔补充稀疏扫描。
- 共 1183 chunks / 302848 columns。

正式指标：
- unownedSurfaceWaterSources=0
- unsafeExposedWaterSources=0
- freeFallWaterFaces=0
- runtimeWaterLost=0
- riverWaterBoundaryModelMismatches=0
- riverLocalWaterPeaks=0
- wetlandPoolLeaks=0

审计初次运行真实抓出：
- WetlandPool 泄口 4 个；示例 5594,-1224 -> 5593,-1224，pool Y110、外侧 dry Y109。
- RuntimeWaterLost=154；确认属于 Lake/Ocean 详细岸界零水深采样的语义误判。
- RiverLocalWaterPeak=1；坐标 2118,-3004。

修复：
- Java Shoreline 的 Lake/Ocean isWater 只在 voxel water boundary 严格高于 ground 时认作实际水柱；River 仍允许通过下压河床保证浅水层。
- WetlandPool 在最终 shoreline morphology 完成后增加 cardinal mud rim seal；附近 River shoreline 不再覆盖池边封口。
- Waterbed 在 receiver/backwater/cascade 全部后处理完成后新增 final confluence harmonization，使多支流尾段不会被后续处理再次拉开水面。
- 2118,-3004 实例的三条连接河段原先连续水面约 Y89.039 / Y88.854 / Y88.186，最终现在在重叠区统一为 Y88.186。
- 新增 Go 回归 TestFinalConfluencePassKeepsOverlappingTributarySurfacesEqual。
- Waterbed 模型发生变化，Engine cache v11 -> v12，并重建 Golden。

验证：
- go test ./... -count=1: PASS
- Java Golden: 15/15
- geoworld-river-navigation-audit -strict: PASS
- maxAdjacentWaterStepBlocks=1, adjacentWaterStepViolations=0
- lakeFeeders=6, disconnectedLakeFeeders=0
- BiomePreset=56/56
- DecorationPolicy: PASS
- Shoreline: PASS
- WaterOwnership: PASS (1183 chunks / 302848 columns, all safety counters zero)
- Engine v12: PASS, cached=false
- Spawn: PASS
- NeoForge BUILD SUCCESSFUL，最新 dev JAR 已部署到 HMCL 26.3。

## 2026-10-09 · 最终 Minecraft 水体所有权审计

目标：将近期反复出现的随机泉水、河岸高一格水源、湿地水洼泄口、悬空水帘等问题统一放到最终 voxel 层自动审计，而不是只依赖 Hydrology/Runtime 或实机肉眼检查。

新增 verifyGeoWorldWaterOwnership：
- 扫描已知回归坐标周围 9x9 chunks。
- 再按约 2048 blocks 间隔覆盖整个模型范围。
- 直接审计 GeoWorldShoreline.Column，即 Minecraft 最终落方块前的列计划。
- Seed=123456 本轮共扫描 1183 chunks / 302848 columns。

水体所有权只允许：
- Runtime River
- Runtime Lake
- Runtime Ocean
- GeoWorld WetlandPool

正式指标：
- UnownedSurfaceWaterSources = 0
- UnsafeExposedWaterSources = 0
- FreeFallWaterFaces = 0
- RuntimeWaterLost = 0
- RiverWaterBoundaryModelMismatches = 0
- RiverLocalWaterPeaks = 0
- WetlandPoolLeaks = 0

审计初次运行真实抓到：
- WetlandPoolLeaks=4，例如 5594,-1224 -> 5593,-1224，water Y110、外侧 dry ground Y109。
- RiverLocalWaterPeaks=1。
- RuntimeWaterLost=154。

进一步核对后：
- RuntimeWaterLost=154 主要来自 Lake/Ocean 语义边界：Runtime 仍带 WaterKind，但详细 terrain 在 voxel 化后已经顶到 water boundary，本质是最终干岸。
- Java Shoreline isWater 统一为最终 voxel 水列语义：River 始终保留水文归属；Lake/Ocean 只有 waterBoundary > groundBoundary 才属于真实水列。
- 这与此前 Go Waterbed 对宏观 Lake 候选和最终真实湖面的区分保持一致。
- 统一语义后，湿地 rim 规划能正确识别附近最终干岸，4 个泄口归零；河流局部峰值也归零。

最终 Seed=123456：
- chunks=1183
- columns=302848
- waterColumns=88348
- owners={WetlandPool=279, River=23068, Lake=11967, Ocean=53034}
- unownedSurfaceWaterSources=0
- unsafeExposedWaterSources=0
- freeFallWaterFaces=0
- runtimeWaterLost=0
- riverWaterBoundaryModelMismatches=0
- riverLocalWaterPeaks=0
- wetlandPoolLeaks=0

验证：
- verifyGeoWorldWaterOwnership 接入 Gradle check。
- NeoForge build/check: PASS。
- go test ./... -count=1: PASS。
- Java Golden: 15/15。
- geoworld-river-navigation-audit -strict: PASS。
- maxAdjacentWaterStepBlocks=1，adjacentWaterStepViolations=0。
- lakeFeeders=6，disconnectedLakeFeeders=0。
- 最新 dev JAR 已部署到 HMCL 26.3。

## 2026-10-09 · strict-final 审计语义收口

在最终水体所有权审计完成后继续处理长期遗留的 geoworld-consistency-audit -strict-final 失败。

### 1. Runtime 河流审计改为沿真实 meander 中心线
- 旧 auditRuntimeWater 使用 X0->X1 / Z0->Z1 直线采样。
- RiverSegment 已有 MeanderOffset，直线会大量离开真实河槽，正式种子因此出现 27万~29万 RuntimeRiverUndefinedSteps。
- 改为与 river-navigation audit 一致的 sin(pi*t) meander centerline。
- 两个正式种子 RuntimeRiverUndefinedSteps: -> 0。

### 2. Waterbed upslope 只审计本段纵向 profile
- 旧 Waterbed.Audit 使用 composite FinalHeight 判断逆坡。
- 汇流区 composite bed 是重叠河槽的最小值；较深支流会形成合法局部冲刷坑，离开重叠区后地面回到下游本段河床，看起来会“升高”。
- 最大旧案例中本段 own profile 实际仍在下降，且有 2~3 条河段重叠；最终 Runtime 水面同时保持不升、不断水。
- 正式 invariant 改为：每条 RiverSegment 自身 Profile 不得下游升高；汇流 composite scour recovery 不算 profile upslope。
- Runtime 继续独立保证 water surface non-rising / defined / non-dry。
- Tester anomaly report 同步使用本段 profile 语义。
- 正式种子 WaterbedRiverUpslopeSteps: 1325/1972 -> 0。

### 3. CoastTerrain slope 检查恢复原始职责
- 旧 CoastTerrainMacroSlopeMismatch 于 10da905 (apply continuous coast terrain correction) 引入，本意是防止 HeightCorrection 在宏格边界新增 slope kink。
- 后续山地加入 ridge / sharp summit / summit spire / fault scarp 后，原始 Terrain 本身允许真实尖锐坡度变化。
- 诊断证明最大点 Raw Terrain 与 CoastTerrain mismatch 完全相同，HeightCorrection kink=0：
  - Seed 731894205 @ 27776,32896: raw=2.09941154443e-4, coast same。
  - Seed 10400 @ 31872,14976: raw=2.67725739004e-4, coast same。
- 因此拆分指标：
  - RawTerrainMacroSlopeMismatch：仅报告自然地形自身尖锐度。
  - CoastTerrainMacroSlopeMismatch：只测 Coast HeightCorrection 相对于 Raw Terrain 额外引入的 slope mismatch。
- strict 阈值仍为 1e-4，没有放宽。

最终 strict-final：
- Seed 731894205:
  - RawTerrainMacroSlopeMismatch=2.0994115444317174e-4
  - CoastTerrainMacroSlopeMismatch=2.9773275003197597e-6
  - WaterbedRiverUpslopeSteps=0
  - RuntimeRiverUndefinedSteps=0
  - RuntimeRiverDrySteps=0
  - RuntimeRiverSurfaceUpslopeSteps=0
- Seed 10400:
  - RawTerrainMacroSlopeMismatch=2.6772573900446517e-4
  - CoastTerrainMacroSlopeMismatch=4.701832812870066e-6
  - WaterbedRiverUpslopeSteps=0
  - RuntimeRiverUndefinedSteps=0
  - RuntimeRiverDrySteps=0
  - RuntimeRiverSurfaceUpslopeSteps=0
  - RuntimeReceiverBackwaterSteps=1 (合法 receiver backwater)
- geoworld-consistency-audit -strict-final: exit 0。

完整回归：
- go test ./... -count=1: PASS。
- Java Golden: 15/15。
- geoworld-river-navigation-audit -strict: PASS。
- maxAdjacentWaterStepBlocks=1，adjacentWaterStepViolations=0。
- lakeFeeders=6，disconnectedLakeFeeders=0。
- NeoForge BiomePreset / DecorationPolicy / Shoreline / WaterOwnership / Engine / Spawn: PASS。
- GeoWorld WaterOwnership: unowned/unsafe/freeFall/lost/riverPeak/wetlandLeak 全 0。
- NeoForge BUILD SUCCESSFUL，最新 dev JAR 已部署到 HMCL 26.3。

本轮只修审计/诊断语义，不改变世界模型生成结果，因此 Engine cache 保持 v12。

## 2026-10-09 · 山体自然度量化与 Terrain Morphology Audit

水体与 strict-final 收口后，继续核查此前“山有了但峰感不足 / 不能为了峰感把世界整体变崎岖”的地形目标。

### 旧 geoworld-terrain-audit 已失效
- 旧守卫仍要求 maximum |RegionalDetail| <= 0.06；当前 ridge / localized summit / summit spire / fault scarp 体系实际可到约 0.21~0.28。
- 旧 macro slope mismatch 还要求整个自然 Terrain 接近 C1；这与当前明确允许稀疏尖峰和断崖的设计冲突。
- 因此不再用这两个旧指标作为失败条件，但继续报告 maximum |RegionalDetail|、valley carve、height range 和 chunk seam。

### 新 Morphology Audit
- 默认正式种子：123456, 731894205, 10400。
- 从 Landform.MountainWeight 中选最多 12 个强山地中心，中心间距 >= 1400 blocks。
- 每个中心扫描 1024x1024 blocks，16-block 采样步长。
- 直接按 Minecraft 26.3 高度映射换算为实际 Y，再统计：
  - p50 / p90 / p95 / p99 / max 坡度角；
  - >=35° steep、>=50° cliff、>=60° extreme cliff 比例；
  - 5x5 邻域局部峰与 >=6 blocks 的 strong peaks；
  - 1024-block 山地区域最大峰谷落差；
  - Y>=318 的世界顶饱和比例；
  - strongest high / low / cliff checkpoint 坐标。
- 全局陆地区域再按 dominant landform 分 plain / hill / foothill / mountain / plateau，检查山地细节不能泄漏到平原和台地。

### 当前三颗正式种子的实测结论
- 平原 p99 坡度：6.9°~7.8°；最大 14.8°~16.0°；>=35° 和 >=50° 均为 0。
- 丘陵 p99：9.2°~10.5°；山麓 p99：14.8°~15.5°；均无 >=50° cliff 泄漏。
- dominant mountain：
  - p90 约 14.2°~14.4°；
  - p99 29.5°~36.5°；
  - >=35° 约 0.411%~1.154%；
  - >=50° 约 0.048%~0.313%；
  - 最大坡度 60.4°~67.6°。
- 强山地 12-window 扫描：
  - Seed 123456: relief 245.8 blocks, strong peaks 139, cliff50 0.488%, top saturation 0%。
  - Seed 731894205: relief 183.0 blocks, strong peaks 131, cliff50 0.026%, top saturation 0%。
  - Seed 10400: relief 272.3 blocks, strong peaks 109, cliff50 0.400%, top saturation 0.1085%。

这说明当前模型已经形成目标结构：
- 大多数陆地仍平缓可走；
- 山地多数坡面不是悬崖；
- 真正高山有明显峰谷落差和大量局部峰；
- 50°以上断崖稀疏但确实存在；
- 高山没有大面积顶到 Y=320 形成平顶。

### 新 guardrails
- chunk epsilon jump <= 1e-5。
- strong mountain windows: p90 8°~25°，p99 20°~55°，max 50°~80°。
- strong mountain cliff50 <= 1.5%，steep35 <= 4%。
- max local relief >= 100 blocks。
- strong peaks >= 3 * selected mountain centers。
- Y>=318 saturation <= 1%。
- plain: p99 <= 10°，max <= 20°，>=35°/>=50° 必须为 0。
- hill: p99 <= 20°，max <=45°，不得出现 >=50° cliff。
- foothill: p99 <=25°，>=35° <=0.2%，>=50° <=0.1%，max <=65°；允许山脚过渡边缘存在极少量天然岩壁。
- plateau: p99 <=6°，max <=25°。

默认 geoworld-terrain-audit 已用三颗正式种子通过。此轮只升级审计，不改变 Terrain / WorldSample / Java Runtime，因此不需要 bump engine cache。
- 另外用旧随机序列 7000 / 13151 / 19302 做了非正式交叉验证；只有 Seed 7000 暴露山麓边缘 0.024% 的 >=50° 稀疏岩壁，最终 guardrail 调整为按比例限制而非绝对禁止，Seed 7000 复跑 exit 0。

## 2026-10-09 · 山峰陡峭度与稀疏悬崖收口

用户实机反馈：整体地形较平缓符合预期，但真正的山峰仍偏圆缓、很多可以直接走上去；希望高山有更明显峰面，并允许少量自然悬崖，同时不能把普通世界变成崎岖地形。

### 诊断结论
- 现有模型不是“完全没有悬崖”，而是不同 Seed 的山体陡峭度差异较大。
- 640x400 形式化基线中，Seed 731894205 最平：p99 slope=28.4deg，cliff>=50deg=0.026%，明显弱于 123456 / 10400。
- 游戏默认 320x200 居中世界复核后，三颗 Seed 都有真实陡峰，但仍保持大多数山坡可步行。

### 模型调整
只调整强 MountainWeight 区域内部形态，不抬高大范围山体基底，也不改平原/丘陵：
- localized summit: pow 3.8 -> 4.2，amplitude 0.115 -> 0.125。
- summit spire: pow 5.2 -> 5.8，crest gate 0.58..0.88 -> 0.56..0.86，amplitude 0.095 -> 0.110。
- sparse fault cliff gate 稍放宽：faultRegion 0.58..0.82 -> 0.56..0.80；crest 0.48..0.80 -> 0.46..0.78。
- cliff amplitude 0.085 -> 0.090。
- 高山顶部 headroom compression 仍保持 0.42；曾试验 0.50，但收益很小，已撤回，避免额外压低高峰。

Java Runtime Reader 同步上述参数，确保 Go / Java terrain sampling 完全一致。

### Terrain Morphology Audit
现有 geoworld-terrain-audit 已包含山体窗口、地形分类、峰谷/坡度/悬崖比例和 guardrail，本轮直接用它验收。

Seed 123456（640x400）：
- mountain window p50=4.0deg, p90=14.3deg, p99=44.7deg, max=67.6deg。
- steep>=35deg=1.838%，cliff>=50deg=0.482%，cliff>=60deg=0.047%。
- strong peaks=144，top>=318=0%。
- dominant mountain category p99=37.2deg，cliff50=0.335%。

Seed 731894205（640x400）：
- mountain window p50=3.9deg, p90=13.5deg, p99=29.5deg, max=66.2deg。
- steep>=35deg=0.424%，cliff>=50deg=0.046%，cliff>=60deg=0.010%。
- strong peaks=136，top>=318=0%。
- dominant mountain category p99=30.8deg，cliff50=0.075%。

Seed 10400（640x400）：
- mountain window p50=3.6deg, p90=15.8deg, p99=41.8deg, max=65.7deg。
- steep>=35deg=1.761%，cliff>=50deg=0.416%，cliff>=60deg=0.058%。
- strong peaks=112，top>=318=0.1538%。
- dominant mountain category p99=35.5deg，cliff50=0.243%。

分类结果保持空间隔离：
- plains p99 slope 约 6.9..7.8deg，35deg+ 与 50deg+ 均为 0。
- foothills p99 约 15.4..15.8deg，35deg+ 基本为 0。
- plateaus p99 < 1deg。
- cliff shaping 没有泄漏到普通地形。

游戏默认 320x200 居中世界补充核验：
- Seed 123456: p99=46.0deg, cliff50=0.673%。
- Seed 731894205: p99=32.7deg, cliff50=0.089%。
- Seed 10400: p99=46.1deg, cliff50=0.606%。

### 缓存与跨语言基准
- 世界模型发生变化，GeoWorldEngineBridge cache prefix v12 -> v13。
- runtime-v1.geoworld / runtime-v1.json 已重新生成。
- Java Runtime Reader terrain 常量同步。
- Java Golden: 15/15 PASS，tolerance=1e-9。

### 完整回归
- geoworld-terrain-audit: PASS（三颗 Seed）。
- geoworld-consistency-audit -strict-final: PASS（正式 Seed 731894205 / 10400）。
- go test ./... -count=1: PASS。
- geoworld-river-navigation-audit -strict: PASS。
- totalDownstreamRises=0，dryCenterlineSamples=0。
- maxAdjacentWaterStepBlocks=1，adjacentWaterStepViolations=0。
- lakeFeeders=6，disconnectedLakeFeeders=0。
- NeoForge BiomePreset / DecorationPolicy / Shoreline / WaterOwnership / Engine / Spawn: PASS。
- WaterOwnership: unowned / unsafe exposed / free-fall / runtime lost / river peak / wetland leak 全 0。
- NeoForge BUILD SUCCESSFUL，最新 dev JAR 已部署到 HMCL 26.3。

## 2026-10-11 · Seed 123456 高山视觉反馈与定向峰脊/裸岩改造

截图复核（约 X=1411,Z=8024 山脚和 X=1699,Z=8087 峰顶）：虽然有高差，但整体像圆滑大坡，峰顶过宽；雪线附近形成成片灰色裸岩横带。此次针对山体形态而非整体抬高。

- Terrain：将较大尺度的圆形 cellularPeak 改为 cellularRidgePeak，确定性方向、长短轴让峰脊延展且横向更陡；保留小尺度 summitSpire，并在次级山脊之间加入少量 ridgeIncision；只在原 MountainWeight 区域发生作用。
- RuntimeProfile：取消原先雪线以上 0.055 高度区间统一铺石的规则；改为局部地形坡度和缓变噪声共同决定裸岩，缓坡雪覆盖、陡坡暴露岩石。Go / Java Runtime 同步。
- EngineBridge 模型缓存前缀 v13 -> v14；实机必须新建世界或验证未生成区块。
- Terrain Audit 增加 -centered 可选项，以复现 MOD 使用的居中模型布局；640x400 正式基线及 guardrail 保持不变。
- 新增定向峰脊沿/横衰减与坡度裸岩规则测试。
- go test ./... -count=1 PASS；Java Golden 15/15 PASS。
- 正式 640x400 三 Seed Terrain Morphology PASS：Seed123456 山区 cliff50=0.493%、Mountain cliff50=0.335%，平原 steep35=0；731894205 与 10400 通过。
- strict-final（731894205 / 10400）PASS，最终 Runtime 水文/群系一致性无新增失败。
- 320x200 -centered Seed123456 作为实景补充审计，按原 guardrail **未通过**：Foothill p99=29.5°、steep35=0.573%（限制25°/0.2%），需进一步调查与改善，不放宽阈值。
- NeoForge build/check PASS：BiomePreset 56/56、DecorationPolicy、Shoreline、WaterOwnership 全绿；WaterOwnership 扫描 1183 chunks / 302848 columns，unowned/unsafe/freeFall/runtimeLost/riverPeak/wetlandLeak 均为 0。
- geoworld-river-navigation-audit -strict PASS：totalDownstreamRises=0、totalDryCenterlineSamples=0、maxAdjacentWaterStepBlocks=1、disconnectedLakeFeeders=0。
- 最新 dev JAR 已通过正式 build.cmd 部署到 HMCL 26.3，仅用于新世界视觉测试；客户端尚未进行新版视觉复测，不能宣布山体视觉验收通过。

## 2026-10-11 · 主世界全矿物与全群系生成保证

用户要求：所有主世界矿物都必须真正生成矿脉，主世界全部群系必须存在，不接受仅在注册列表中出现的假通过。

- 扫描现有 56 种主世界群系，原版 Seed 123456 完整模型实际 70,231 次采样只发现 36/56，旧 BiomePreset 56/56 只证明已注册。新增 Minecraft26_3BiomeCoverage：确定性搜索气温、雨量、地形、水域与高度合适的真实坐标，为 56 群系分配地理上优先合理的保底区域，并保留原始生态群系/水文/地形生成。
- 新增真正的世界三维群系覆盖审计；Seed 123456 与 Seed -4852173505788415804 各 80,264 次实际采样，均为 56/56、missing=0，且校验每个保底区域有连续实际 biomeId。
- 新建 GeoWorldMineralVeins：八类常规 Overworld 矿石（煤、铁、铜、金、青金石、红石、钻石、绿宝石），各有 STONE/DEEPSLATE 变体，物理 Y 高度、地质场与生态权重。世界空间独立确定性曲线脉、偶发大矿脉、保底超级网格矿脉；只替换石头、深板岩及地表以下 4 格的主岩体。
- 新建 GeoWorldAmethystGeodes：保底隐藏紫水晶晶洞（紫水晶/紫水晶母岩/方解石/平滑玄武岩），且原版 amethyst_geode 继续保留；地下生成不会覆盖任何 River/Lake/Ocean 水面与非石层。
- GeoWorldDecorationPolicy 仅移除目标 8 类 vanilla ore Feature，保留 spring_lava、ore_gravel、ore_tuff、amethyst_geode、结构、植被及其他 MOD 的矿物 Feature。
- 引擎 cache v14 -> v15；矿物和群系分布变化需要新建世界或验证未生成区块。
- verifyGeoWorldMinerals：Seed 123456 和 -4852173505788415804 均 PASS（8 类矿、16 普通/深板岩变种、紫水晶晶洞、有效地下石层、不同地质权重、跨区块顺序无关）。
- NeoForge build/check：56 群系真实覆盖、8 类矿石、原版装饰策略、Shoreline 与 WaterOwnership 均 PASS；所有水体所有权异常仍为 0。
- 注意：矿物地质判断仍来自地表形态/群系倾向，没有独立持久化地下地层模型。群系完整性保证实际生物群系 ID 与区域连续性，不代表每个群系的面积、地物或结构完全等价于 Minecraft 原版；后续仍需实机观察。
- 完整实现与验证方法见 docs/MINERALS_AND_BIOME_COVERAGE.md。

## 2026-10-11 · 所有主世界群系的最终地表/浅层材质统一审计

用户截图发现：世界预览已显示「温带荒漠，冲积土/灌丛土」，游戏是沙漠或恶地，但地面成片泥土或草块。根因是最终群系由 BiomeRealizer 决定，实际地表 voxel 却只按 GeoWorld 中间 SurfaceProfile 生成。

- 新增 GeoWorldSurfaceMaterials：所有 56 种主世界群系明确映射（52 种地表/水域、4 种地下），GeoWorldChunkGenerator 以最终 biomeId 决定表面和浅层，buildTerrain 与 getBaseColumn 规则一致。
- 沙漠：表面沙、浅层砂岩；恶地：平缓处红沙，侵蚀区/岩质露头陶瓦，地下50格以内维持 Y 连续的分层陶瓦。其余森林、雨林、针叶林、雪地、冰刺、蘑菇岛、苍白之园、红树林、沼泽、海滩、海床、风袭丘陵、高山等都有明确策略。
- 保留水文岸线和湿地池塘的 surfaceOverride、underOverride 最高优先级，避免材质替换破坏已通过的岸线/河流水源。
- 发现旧 56/56 群系保底不同中心碰撞问题：在 Seed -563659577384754347 上 minecraft:river 被 frozen_river 覆盖。修复为非重叠中心（间隔至少256方块）、距离最近优先选取，并为窄河流补充更细的候选采样。
- 修复 Minecraft26_3BiomeCatalog 中沙漠地貌可错误选为 windswept_gravelly_hills/windswept_hills 的问题。仍保留真实 GeoWorld 生态交界 56 格范围内的自然过渡。
- 新增 verifyGeoWorldSurfaceBiomes，纳入 Gradle check：严格验证全部56群系、52个地表群系保底中心周围共468列、936项表层/浅层材质检查、水文优先情况，以及荒漠内部生态群系一致性。
- 默认 Seed 123456：该专项全绿（56/56，468列、936材料检查，naturalDesert=21，desertConflicts=0）；NeoForge build.cmd 全绿并部署到 HMCL 26.3；WaterOwnership 核查302848列，全部水源异常仍为0。
- 额外两种子 Seed -563659577384754347（用户预览）与 -4852173505788415804 均通过 verifyGeoWorldSurfaceBiomes（各56/56、468列/936材料检查，生态沙漠内部冲突0）。在用户预览种子上 X382 Z-3089 为恶地+岩质陶瓦，X396 Z-3924 为疏林恶地+红沙，X-159 Z-6808 为沙漠+沙子/砂岩。
- 完整实机视觉复核尚未做，已生成区块不会自动重置地表。文档见 docs/BIOME_SURFACE_CONSISTENCY.md。

## 2026-10-11 · 区块生成性能第一轮 + 稀疏岩壁矿露头

- 实机截图出现大范围区块等待/渲染滞后。新增 benchmarkGeoWorldChunkPlanning（默认 Seed123456，固定48区块，3轮预热），初测模型规划：shoreline=3.059ms、biomes=2.190ms、minerals=0.159ms/区块；独立 JVM 复测波动较大（4.364/3.622/0.295ms），无法声称客户端已加速。
- 优化 Minecraft26_3BiomeCoverage：采用 256 方块空间桶选出附近稀有保底群系，避免查询时无条件扫描56个中心，保留半径和Y区间。
- 优化 GeoWorldChunkGenerator：超过普通浅层3格的宿主岩石走快速路径（恶地50格以内按Y保留陶瓦层），不再每块石头重复匹配群系材质；buildTerrain/getBaseColumn 同步。
- 优化 GeoWorldMineralVeins：完全离开目标区块的矿脉包围盒提前退出体素化循环，仍按原序消耗所有 PRNG 随机参数，不改变既有矿脉空间生成。
- 允许少量煤/铁/铜/绿宝石在真正的 Mountain/Foothills 裸露石面上自然出露，按 8×8 固定微区稀疏门控。禁止在草、雪、泥、岸线、海水等表面出矿。
- 新增可选真实客户端阶段耗时日志：JVM `-Dgeoworld.profileChunks=true`，每128区块汇报 model、shoreline、biomes、voxel、minerals、geodes 的平均耗时。不开启不计时。
- 用户确认不希望大量天然洞穴，本轮不主动增加洞穴、洞口或原版 Carver 密度。未来地下河—地下水位—地层渗透/溶蚀—稀疏洞穴路线图见 docs/UNDERGROUND_WATER_AND_SPARSE_CAVES.md。
- NeoForge build.cmd 通过（BiomeCoverage 56/56、矿石 8/16、Shoreline、SurfaceBiomes、WaterOwnership 等），已部署 HMCL 26.3。水体所有权 302848 列检查关键异常均 0；该结果不代表客户端区块延迟问题已经解决。
- 具体性能基准和客户端验收文档见 docs/CHUNK_PERFORMANCE.md。

## 2026-10-11 · 开发版区块性能诊断自动化

- 2026-10-11 HMCL v26.3 日志调查：压缩日志 5 份加最新日志均无 GeoWorld generated 记录；旧性能统计未开启。内置服务器累计发现 19 次 can't keep up 警告，最大一次 13732 ms（274 ticks），但无法仅凭该警告定位是何阶段。
- 改造 GeoWorldChunkPerformance：开发版的嵌入构建标记自动开启（无需用户修改 JVM 参数），正式构建默认关闭；系统属性 -Dgeoworld.profileChunks=true / false 可显式覆盖。
- 正式构建通过 build.gradle 的 generateGeoWorldBuildProfile 资源生成，build.cmd 固定 -Pgeoworld.devBuild=true。性能日志采用 Mod SLF4J 日志通道写入 Minecraft latest.log。
- 首次32区块汇报，之后仅在128区块节点检查且至少间隔30秒，避免刷屏。只在 GeoWorld 自定义生成器的区块统计 model、shoreline、biomes、voxel、minerals、geodes，其他生成器不计时。
- 新增 verifyGeoWorldChunkPerformanceMode 并纳入 check，自动验证构建嵌入标记与默认开关一致。`-Pgeoworld.devBuild=true` 通过 automaticDevMode=true；`-Pgeoworld.devBuild=false` 通过 automaticDevMode=false。
- Dev build.cmd 完整通过并部署至 HMCL v26.3；解包部署 JAR 确认 geoworld-build.properties=devBuild=true；其他生成内容验收仍通过。
- 随后重新进入 GeoWorld Seed -563659577384754347 存档 t25，最新日志实际记录 4736 区块：平均 biomes 18.960ms、voxel 7.539ms、shoreline 3.341ms、minerals 0.108ms、geodes 0.006ms；期间两次服务器 can't keep up（2029ms、17173ms）。群系解析成为当前最明确可优化项，不可据此认定全部 17s 卡顿均属 GeoWorld。

## 2026-10-11 · 复用群系模型采样及真正的高山岩壁露头

- Minecraft26_3BiomeRealizer 3D biome 计算去掉同坐标 RuntimeSample/ClimateSample 的重复读取；新增 biomeIdFromSample，ChunkGenerator.buildTerrain/getBaseColumn 直接复用 GeoWorldShoreline 已经算好的每列 RuntimeSample，保留原群系选择、洞穴候选与所有56群系生态区域。
- 24,576个三维取样指纹对比：Seed 123456 维持 `9f3de994862521fb`，Seed -563659577384754347 维持 `71bfd1e1968b4db6`，前后完全一致；纳入正式 Gradle check 中的 fingerprintGeoWorldBiomeDecisions。
- 模型侧48区块/3轮预热的群系阶段测得 0.103ms/区块。旧独立 JVM 基准约2.19ms，受工作负载/JIT和预热等影响不能据此认定客户端同比例提速，下一步必须读取新日志。
- 稀疏矿脉出露从单点候选改为山区表面高度锚定曲线矿脉，约每12个合格64x64源格一个；仅原版石头高山/山麓真实外露的顶部或侧面能生成铁、绿宝石、煤、铜矿；高山铁/绿宝石扩展到Y319，不铺在水/雪/草/沙或岸线。
- 新增 verifyGeoWorldSurfaceOutcrops：Seed -563659577384754347 高山 X1680..2700/Z-2950..-1650 测试计划点5495，Y>240点4926、近表岩石可选2757、顶部石面相交692，跨区块一致性PASS。测试参考坐标 2011,297,-2914（仅新世界/新生成区块）。
- 完整 NeoForge build.cmd 通过：56/56 群系、8类原版矿石及16种变体、矿脉露头、表层材质、Shoreline 与 WaterOwnership（302848列，异常0），已部署HMCL 26.3。未改动引擎cache版本、原版方块/生物/实体注册或天然洞穴密度。
- 详细见 docs/BIOME_PERFORMANCE_AND_ROCK_OUTCROPS.md。

## 2026-10-11 · t26 岩面矿脉规模与客户端耗时复核

- 用户在 X2856 Y220 Z-3299 实机发现铁矿出露细、小、少。实际日志确认 GeoWorld Seed -563659577384754347，存档 t26，累计3072区块：biomes=48.505ms、voxel=13.687ms、shoreline=7.052ms、minerals=0.776ms、geodes=0.007ms（每区块平均）。两次 Can't keep up 6146/2623ms。比先前 t25 的 biomes=18.960ms更高，但不同位置及机器负载不能直接推定真实性能回退，群系阶段仍是优先性能排查点。
- GeoWorldMineralVeins 仅扩大真实山区/山麓石质岩壁的矿化出露：源格概率约1/12→1/8、源点扰动±12格、长度30~52（少数50~60）、半径2.5~3.6（少数3.6~4.4）、岩壁深度放宽到表层9格；不允许水体/土/雪/草/沙露出，不改其他地下矿石和全部8种原版矿物覆盖。
- 对扩大的表层矿脉用2.5格取样步长，减少体素候选重复计算；地下矿脉保持原1.5格。新增去重露头和最大连续面积测试及密度上下限；验证种子同区域裸露顶部去重候选414→959，最大连续露头64→202，跨区块PASS。
- build.cmd 完整回归通过：群系56/56、矿石8种和16变种、地表一致性、岸线、WaterOwnership 302848列/异常0。已部署 HMCL 26.3；旧世界已生成区块不会自动重新分布矿石。下一步需要同运行环境实机再次看 GeoWorld generated 耗时及矿体外观。
