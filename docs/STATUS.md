# GeoWorld 项目状态

> 本文件是项目进度的单一入口。每完成一个阶段性功能，应同步更新本文件。

更新时间：2026-09-29

## 当前阶段

**Phase 3 — 季风与水文**

当前子阶段：**Phase 3.1 — 夏 / 冬代表态与季风指数（准备开始）**

## 当前稳定基线

最近提交：

- `ec4149e` — add phase1 multi-seed acceptance baselines
- `d8a369c` — build macro coastal and elevation terrain
- `b53169a` — derive tectonic intensity from crust and plate motion
- `3f85f67` — show complete phase1 lab maps
- `8e672b1` — naturalize plate boundaries with smooth domain warp

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
