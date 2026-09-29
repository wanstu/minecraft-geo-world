# GeoWorld 项目状态

> 本文件是项目进度的单一入口。每完成一个阶段性功能，应同步更新本文件。

更新时间：2026-09-29

## 当前阶段

**Phase 1 — 板块与大陆骨架**

当前子阶段：**Phase 1.3 — 板块边界自然化（准备开始）**

## 当前稳定基线

最近提交：

- `4f31402` — record geoworld status decisions and roadmap
- `f3f0ba0` — classify tectonic landform interactions
- `fc6acf4` — decouple crust field from tectonic plates
- `f863d3e` — bootstrap geoworld planning and phase1 plate lab

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

### Phase 1.1 — 最小构造链路

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
- 已有构造地貌：
  - Collision Orogen
  - Volcanic Arc
  - Trench
  - Island Arc
  - Mid-Ocean Ridge
  - Continental Rift
  - Transform Fault

### Phase 1.2 — 地壳年龄与构造影响带

状态：**完成**

已完成：

- 海洋地壳年龄场。
- 海洋-海洋汇聚的俯冲侧判定。
- 较老海洋地壳优先俯冲。
- 较年轻上覆海洋板块形成岛弧。
- 大陆-海洋汇聚区分海沟侧与大陆火山弧侧。
- 构造边界从“一格线”扩展为有宽度的影响带。
- 山系、火山弧、海沟、洋中脊、裂谷使用不同宽度和影响幅度。
- 海洋地壳随年龄增加发生基础海盆加深。
- 加入自动测试：
  - Seed 可复现
  - 海洋面积目标
  - 海洋年龄范围
  - 构造影响带确实扩展到边界之外

当前输出图层：

- `plates.png`
- `crust.png`
- `boundaries.png`
- `tectonics.png`
- `features.png`
- `ocean-age.png`
- `subduction.png`
- `tectonic-influence.png`
- `elevation.png`

## 当前实验参数

默认 Lab：

- Seed：731894205
- Grid：640 × 400
- Plate Count：18
- Ocean Area：68%

## 下一步：Phase 1.3 — 板块边界自然化

目标：

1. 降低标准 Voronoi 的直线 / 多边形感。
2. 板块仍保持稳定、连续、可确定性复现。
3. 边界自然化不能破坏板块拓扑关系。
4. 构造地貌继续沿自然化后的边界生成。
5. 比较多种方法：
   - 扰动距离场
   - domain warp
   - plate growth / region growth
   - 混合方法

验收：

- 视觉上不再明显像几何 Voronoi。
- 不产生大量孤岛 Cell 或锯齿碎片。
- 相同 Seed 完全一致。
- 性能仍能支撑 1024² 甚至更高宏观网格。

## Phase 1 后续

### Phase 1.4 — 地壳与构造参数深化

- 大陆地壳厚度
- 海洋地壳年龄进一步校准
- 构造强度与相对板块速度绑定
- 海沟 / 山系宽度与碰撞强度绑定

### Phase 1.5 — 基础高程场

- 大陆内部
- 大陆架
- 海盆
- 高原
- 山系
- 海沟
- 岛弧
- 洋中脊
- 裂谷

### Phase 1.6 — Phase 1 验收

- 大陆轮廓合理。
- 山系与构造有因果关系。
- 海沟、岛弧、洋中脊位置合理。
- 大陆不贴主世界外围。
- 同 Seed / 配置完全可复现。

## 下一阶段

Phase 1 验收后进入：

**Phase 2 — 气候骨架**

纬度 → 基础温度 → 大尺度气压 / 盛行风 → 海温 → 洋流 → 降水。
