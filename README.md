# GeoWorld

GeoWorld 是一个面向 Minecraft Java Edition 的“地理规律驱动世界生成”项目。

目标不是用随机噪声拼接出看起来像大陆的地形，而是先建立可解释的世界模型：

```text
Seed
  ↓
板块划分与运动
  ↓
大陆 / 海盆 / 山脉 / 裂谷 / 火山带
  ↓
长期风场 + 洋流
  ↓
季风影响 + 温度 / 降水
  ↓
河流 / 湖泊 / 土壤湿度
  ↓
生态区 / Minecraft Biome
  ↓
Chunk 地形与方块
```

## 当前产品形态

GeoWorld 不限定为单一 Mod 或单一离线工具。

- **GeoWorld Core**：世界模型与算法规范。
- **GeoWorld Generator**：游戏外预生成、检查、可视化和导出。
- **GeoWorld Runtime**：Minecraft 游戏内按需生成 Chunk。
- **Loader Adapter**：NeoForge / Fabric / Forge 等运行时适配。

## V1 世界拓扑

Minecraft 坐标仍保持理论无限。

GeoWorld 只在中央有限区域生成完整地理世界：

- 东 / 西：逐步进入外洋，最终成为 **Infinite Storm Ocean**。
- 北 / 南：逐步进入极区，最终成为 **Infinite Frozen Ocean**。
- 四角：风暴洋与冰冻洋自然叠加。
- 主地理区之外不再生成新的大陆。

这样既保留 Minecraft 无限世界的技术模型，又让板块、洋流、季风和生态拥有有限且可控的模拟尺度。

## 当前阶段

当前 Core 已完成板块、气候、水文、侵蚀、连续地形、最终水体、生态 / Minecraft Profile 与 Runtime 一致性闭环；Phase 6.2 已完成。Phase 6.3 正在做版本化模型保存/重开与测试工具。MOD / Loader 接入暂后置，先通过交互测试器把世界生成、保存和视觉验收做好。

原型入口：

- `prototype/index.html`
- `prototype/generator.html`
- `prototype/runtime.html`


## 项目文档

- `docs/STATUS.md`：当前进度、稳定基线、正在进行和下一步。
- `docs/PROGRESS.md`：完整历史进度与既往验收记录。
- `docs/PHASE6.md`：最新评审数据、调整后的步骤与验收标准。
- `docs/DECISIONS.md`：关键设计决定及原因。
- `docs/ROADMAP.md`：V1 分阶段路线。
- `docs/PHASE1.md`：板块 / 大陆 / 宏观高程实验记录。
- `docs/PHASE2.md`：气候骨架实验记录。
- `docs/PHASE3.md`：季风与水文实验记录。
- `docs/PHASE4.md`：侵蚀、Relief 与地貌细化实验记录。
- `docs/PHASE5.md`：长期生态区与 Minecraft 映射实验记录。
- `docs/ARCHITECTURE.md`：整体架构。
- `docs/SAMPLING.md`：连续世界坐标采样与 Chunk 无缝原则。
- `docs/PRODUCT.md`：产品边界。


## 交互测试器（推荐）

Windows 直接双击：

`open-geoworld-tester.cmd`

或在仓库根目录运行：

```text
go run ./cmd/geoworld-tester
```

默认打开 `http://127.0.0.1:8788`。当前可直接测试：Seed / 网格 / 板块 / 海洋比例重新生成、最终地形 / 水体 / 群系 / Profile 图层、地图点击 Runtime 采样、模型保存/更新与重开。

默认 `320×200` 适合快速测试；需要正式基线时可改 `640×400`。

## 查看静态原型

原型不需要启动 Web 服务，也不依赖 Node。

Windows 可直接双击：

`open-prototype.cmd`

或直接打开：

- `prototype/index.html`
- `prototype/generator.html`
- `prototype/runtime.html`
- `prototype/plates.html`
- `prototype/acceptance.html`：Phase 1 多 Seed 静态验收
- `prototype/climate.html`：Phase 2 气候骨架实验
- `prototype/monsoon.html`：Phase 3 季风 / 季节性实验
- `prototype/hydrology.html`：Phase 3 流域 / 河网 / 湖泊实验
- `prototype/erosion.html`：Phase 4 侵蚀 / Relief 实验
- `prototype/landform.html`：Phase 4 丘陵 / 山麓 / 高原连续权重实验
- `prototype/geomorph.html`：河漫滩 / 冲积平原 / 冲积扇 / 峡谷实验
- `prototype/terrain.html`：最终连续地形 / Chunk 无缝实验
- `prototype/ecology.html`：长期生态区 / 植被 / 湿地 / 高山生态实验
- `prototype/mcprofile.html`：Minecraft 群系家族 / 地形形态 / 地表 / 植被 Profile

所有 CSS 与生成图片均使用相对路径，可通过 `file://` 直接加载。

## 当前算法验收

```text
go test ./... -count=1
go run ./cmd/geoworld-mcprofile-audit -count 4
go run ./cmd/geoworld-consistency-audit
```

一致性审计已区分 Raw Terrain 历史基线与最终 Runtime；当前 `go run ./cmd/geoworld-consistency-audit -strict-final` 应通过两个正式 Seed 的 Ground / Water / Profile 验收。`prototype/generated` 是早期实验静态快照；当前可视化优先使用交互测试器。
