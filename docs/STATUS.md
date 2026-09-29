# GeoWorld 项目状态

> 本文件是项目进度的单一入口。每完成一个阶段性功能，应同步更新本文件。

更新时间：2026-09-29

## 当前阶段

**Phase 1 — 板块与大陆骨架**

Phase 0 的产品边界、架构与页面原型已经完成，当前重点是建立可解释的构造地貌模型。

## 当前稳定基线

最近提交：

- `f863d3e` — bootstrap geoworld planning and phase1 plate lab
- `fc6acf4` — decouple crust field from tectonic plates
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
- 世界生成参数只在创建新世界时生效，并写入世界存档；已有世界只读，避免新区块割裂。

### Phase 1 算法

- Seed 确定性生成板块。
- 每个板块具有中心和运动速度向量。
- 板块边界分类：
  - convergent
  - divergent
  - transform
- 大陆 / 海洋地壳已从板块归属中拆离。
- 一个大陆可以跨多个板块。
- 一个板块可以同时拥有大陆和海洋地壳。
- 目标海洋面积按实际 Cell 面积控制；当前测试为 68.00%。
- 已有构造地貌分类：
  - Collision Orogen
  - Volcanic Arc
  - Trench
  - Oceanic Subduction
  - Mid-Ocean Ridge
  - Continental Rift
  - Transform Fault
- 已输出真实算法 PNG：
  - plates
  - crust
  - boundaries
  - tectonics
  - features
  - elevation

### 原型

- Generator 总览页
- Runtime / 新世界设置页
- 已有世界只读模式
- Phase 1 板块构造实验页

## 正在进行

### Phase 1.2 — 地壳年龄、俯冲方向、构造影响带

目标：

1. 给海洋地壳建立年龄场。
2. 洋中脊附近年龄最年轻，远离洋中脊逐渐变老。
3. 海洋-海洋汇聚时，让较老、较冷、密度更高的一侧优先俯冲。
4. 区分海沟侧与岛弧侧。
5. 把一格宽的构造边界扩展成有实际宽度的构造影响带。
6. 高程由“构造带影响”决定，而不是只修改边界 Cell。

预期新增输出：

- `ocean-age.png`
- `subduction.png` 或等价俯冲方向图
- `tectonic-influence.png`
- 更新后的 `elevation.png`

## Phase 1 后续顺序

### Phase 1.3 — 板块形状与边界自然化

- 降低标准 Voronoi 直线 / 多边形感。
- 引入扰动边界或更合理的 plate growth 方法。
- 保持确定性。

### Phase 1.4 — 地壳厚度与构造强度

- 大陆地壳厚度。
- 海洋地壳年龄。
- 碰撞强度。
- 抬升幅度。
- 俯冲带宽度。

### Phase 1.5 — 基础地形

- 构造高程场。
- 海盆深度。
- 大陆架。
- 山系宽度。
- 高原。
- 裂谷。
- 火山弧。

### Phase 1.6 — 侵蚀前验收

进入侵蚀和气候前，先确认：

- 大陆轮廓合理。
- 山系与板块构造有因果关系。
- 海沟、岛弧、洋中脊位置合理。
- 大陆不会贴到主世界外围。
- 同一 Seed 完全可复现。

## 下一阶段

Phase 1 验收通过后进入：

**Phase 2 — 气候骨架**

纬度 → 基础温度 → 大尺度气压 / 盛行风 → 海洋温度 → 洋流 → 降水。
