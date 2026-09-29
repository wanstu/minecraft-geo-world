# GeoWorld 项目状态

> 本文件是项目进度的单一入口。每完成一个阶段性功能，应同步更新本文件。

更新时间：2026-09-29

## 当前阶段

**Phase 1 — 板块与大陆骨架**

当前子阶段：**Phase 1.4 — 地壳与构造参数深化（准备开始）**

## 当前稳定基线

最近提交：

- `4cee4af` — model ocean crust age and tectonic influence bands
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

状态：**完成第一版**

实现：

- 保留原始 Voronoi 作为可解释拓扑骨架。
- 在查询板块归属前对坐标做低频平滑 Domain Warp。
- 实际板块边界出现弯曲与不规则变化。
- 不使用逐 Cell 随机抖动，避免盐胡椒碎片。
- 保留 `plates-raw.png` 与 `plates.png` 对照。
- 自动测试限制 warp 必须产生明显变化，但不能重写过多 Cell。

## 当前实验参数

默认 Lab：

- Seed：731894205
- Grid：640 × 400
- Plate Count：18
- Ocean Area：68%
- 当前测试 Seed 的 Plate Warp 变化比例：6.21%
- Ocean Age：0.000 ～ 0.908
- Tectonic Influence：覆盖约 16.41% Cell

当前输出：

- `plates-raw.png`
- `plates.png`
- `crust.png`
- `boundaries.png`
- `tectonics.png`
- `features.png`
- `ocean-age.png`
- `subduction.png`
- `tectonic-influence.png`
- `elevation.png`

## 下一步：Phase 1.4 — 地壳与构造参数深化

目标：

1. 大陆地壳厚度场。
2. 板块相对运动速度进入“构造强度”。
3. 碰撞越强，山系 / 海沟 / 裂谷影响越明显。
4. 海洋地壳年龄参与俯冲强度。
5. 构造影响带宽度不再完全是固定常量。
6. 给 Generator 增加 crust thickness / tectonic intensity 图层。

## Phase 1.5 — 基础高程场

计划：

- 大陆内部基础高程
- 大陆架
- 大陆坡
- 海盆
- 高原
- 山系
- 海沟
- 岛弧
- 洋中脊
- 裂谷

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
