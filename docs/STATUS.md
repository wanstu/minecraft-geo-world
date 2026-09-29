# GeoWorld 项目状态

> 本文件是项目进度的单一入口。每完成一个阶段性功能，应同步更新本文件。

更新时间：2026-09-29

## 当前阶段

**Phase 1 — 板块与大陆骨架**

当前子阶段：**Phase 1.6 — Phase 1 验收（当前）**

## 当前稳定基线

最近提交：

- `3f85f67` — show complete phase1 lab maps
- `8e672b1` — naturalize plate boundaries with smooth domain warp
- `4cee4af` — model ocean crust age and tectonic influence bands
- `4f31402` — record geoworld status decisions and roadmap
- `f3f0ba0` — classify tectonic landform interactions

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

- 大陆轮廓合理。
- 山系与构造有因果关系。
- 海沟、岛弧、洋中脊位置合理。
- 大陆不贴主世界外围。
- 同 Seed / 配置完全可复现。

## 下一阶段

Phase 1 验收后进入：

**Phase 2 — 气候骨架**

纬度 → 基础温度 → 大尺度气压 / 盛行风 → 海温 → 洋流 → 降水。
