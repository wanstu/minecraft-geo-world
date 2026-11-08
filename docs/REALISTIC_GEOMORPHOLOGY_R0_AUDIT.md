# GeoWorld 真实地貌强化 — R0 现状审计

更新时间：2026-11-08

状态：**DADM 第一轮完成。**

基线：`788dc3f`，当前规则 `geoworld-rules-v1.1.16`。

## 1. 当前真实调用链

```text
Phase1 macro elevation
  ↓
Climate
  ↓
Hydrology
  ├─ input = Phase1 Elevation
  ├─ priority-flood depression filling
  ├─ FlowTo / Basin / Accumulation
  └─ Lakes / SpillOutlet
  ↓
Erosion
  ├─ input elevation = Phase1 Elevation
  ├─ FlowAccumulation / RiverStrength
  └─ ErodedElevation
  ↓
Landform
  ↓
Geomorph
  ├─ SedimentSupply
  ├─ Floodplain / AlluvialPlain / AlluvialFan
  └─ DepositionPotential
  ↓
Terrain
  ├─ base = ErodedElevation
  └─ regional landform shaping
  ↓
Reconcile
  ├─ CoastCorrection
  └─ river/lake constraints
  ↓
CoastTerrain
  ↓
RebuildRiverConstraintsFromTerrain
  ↓
Waterbed
  ├─ river profile
  ├─ confluence / lake / mouth geometry
  └─ local bed shaping
  ↓
WorldSample
  ├─ Ground
  ├─ River/Lake/Ocean ownership
  └─ water surface
  ↓
RuntimeProfile
  ↓
SpecialFeatures overlay
  ↓
Java Runtime
  ↓
GeoWorldShorelinePlanner
  ↓
Minecraft voxel columns
```

结论：最终 Minecraft 地形不能用单一 `Terrain.FinalHeight` 代表。

## 2. Hydrology 的实际输入

`hydrology.InputFromWorld()` 明确使用：

```text
world.Elevation
```

即 Phase1 原始宏观高程，不使用 `ErodedElevation`。

当前顺序是：

```text
先定流域与河网
→ 再用河网驱动侵蚀
```

这个顺序本身合理，不应为了“真实侵蚀”简单改成循环重算河网。

## 3. Erosion 已经真实进入最终 Terrain

`terrain.InputFromModels()` 使用：

```text
BaseElevation = erosionModel.ErodedElevation
```

同时 Terrain 已使用：

- FloodplainWeight
- AlluvialPlainWeight
- AlluvialFanWeight
- CanyonWeight
- ValleyFloorWidth
- DepositionPotential

因此 Phase 8 不应新建第二套 erosion / deposition 系统。

## 4. 当前 Hydrology 是 drainage-complete

Priority Flood 以所有 Ocean cell 为固定出口：

```text
ocean → seed flood queue
inland depression → raise to minimum drain level
no lower neighbor → FlowTo fallback to flood parent
```

这意味着当前正式模型倾向于让陆地最终拥有排水路径。

`identifyLakes()` 也会从 FlowTo 中选择 `SpillOutlet`。

### 结构缺口

当前没有独立、正式的：

- Endorheic Basin
- Terminal Lake
- Salt Lake / evaporation-balanced closed lake

语义。

所以 R3 不能假定“封闭湖已经完整支持”，需要单独设计。

## 5. Waterbed 与 WorldSample 的职责

Waterbed 已经负责大量局部水系几何：

- 河槽纵剖面。
- 宽度。
- 汇流。
- lake inlet/outlet。
- receiver/backwater。
- mouth。
- delta 等局部几何。

WorldSample 再根据：

- RiverReceiverSurface
- OceanEligibility
- lake datum
- final ground

决定最终水面与 WaterKind。

因此：

> Waterbed 自身 profile 正确，不等于最终 Runtime / Minecraft 一定正确。

但反过来也成立：

> 单个 audit 在 receiver handoff 上看到数学“上升”，不等于 Minecraft 一定存在缺陷。

必须结合最终 ownership / voxel 证据。

## 6. SpecialFeatures 位于基础 Runtime 之后

`runtimeworld.Model.Sample()` 先得到有限世界/外海 sample，再执行：

```text
applySpecialFeatures()
```

所以特殊岛屿、盆地、天坑等属于最终地貌链的一部分。

DADM 当前已经补过：

- 海洋特殊岛水环。
- 湖中岛水环。
- Save → Load → Runtime 对拍。

这些验收必须保留。

## 7. ShorelinePlanner 是最终方块表示层

Minecraft 不是简单把 normalized height 取整。

ShorelinePlanner 还会处理：

- 河岸平滑。
- 高岸封闭。
- gentle bank。
- strict river containment。
- 水岸邻域。
- 整数格缝隙。

因此 Phase 8 新算法不能把宏观地貌责任继续塞进 ShorelinePlanner。

它只能处理连续模型到 voxel 的离散化问题。

## 8. 已回退的 receiver v1.1.17 方案

DADM 历史中存在：

```text
9313fc5 fix: keep river receiver transitions monotonic
ea2aa9f Revert "fix: keep river receiver transitions monotonic"
```

该方案曾：

- 提升规则到 v1.1.17。
- 提升 RiverGeometryVersion 到 6。
- 放松 receiver/backwater 低岸限制。

但 master 已明确回退。

DADM 交接同时记录：

> geoworld-river-navigation-audit 的额外严格统计不能单独作为当前 Minecraft 缺陷证据；正式 WaterOwnership / Shoreline 等验收已覆盖既有河道修复，不得据此重复开启历史问题。

### Phase 8 规则

R1 可以重新统计 receiver / longitudinal profile。

但除非出现新的最终 Runtime / voxel 可复现证据，否则：

**不得直接恢复 9313fc5，也不得仅为了让 strict audit 变成 0 而升级规则。**

## 9. R0 结论

已经存在、不要重复开发：

- Erosion。
- Sediment / Deposition 第一版。
- Floodplain / Alluvial Plain / Alluvial Fan。
- River profile。
- Confluence。
- Lake inlet/outlet。
- Backwater。
- Mouth / Delta 局部几何。
- Shoreline voxel repair。

真正要做：

1. 把侵蚀/沉积从启发式权重升级成更明确的成因链。
2. 建立整河尺度的地貌纵剖面统计。
3. 建立汇流处“水 + 谷宽 + 沉积”联合验收。
4. 正式设计内流盆地/终止湖语义。
5. 把 river mouth / estuary / delta 与 sediment、gradient、coast 背景绑定。
6. 增加山地→平原、盆地→盆底、海岸等语义过渡验收。
7. 最终验收必须覆盖 Continuous Runtime + Minecraft voxel，而不是只测内部模型。

## 10. 下一步 R1

R1 只增加只读测量：

- 整河 source → mouth route。
- 上/中/下游坡降。
- 汇流前后宽度与水面。
- 湖口与河口 approach。
- 谷宽。
- 地貌边界 gradient。
- 最终 voxel 对照采样。

R1 默认输出“调查信号”。

只有当信号能落到正式最终地形失败样例时，才进入 R2 修改算法。
