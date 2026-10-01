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
