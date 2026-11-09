# GeoWorld 真实地貌强化 — R2 成熟冲积河纵剖面

更新时间：2026-11-09

状态：**v1.1.19 实现与最终验收完成。**

前置：

- [REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md](REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md)
- [REALISTIC_GEOMORPHOLOGY_R1_ACCEPTANCE.md](REALISTIC_GEOMORPHOLOGY_R1_ACCEPTANCE.md)
- [REALISTIC_GEOMORPHOLOGY_R2_BASIN_DRAINAGE.md](REALISTIC_GEOMORPHOLOGY_R2_BASIN_DRAINAGE.md)

## 1. 问题不是 Waterbed 首先制造的

R1 在 Seed `123456`、320 × 200 标准世界上，对每个 outlet 选取最长 free-flow 主干，分别测量：

- River surface
- Terrain
- CoastTerrain
- BaseElevation

结果显示，大量“下游明显比上游更陡”的长河，在 Waterbed 之前已经存在。

代表例：

```text
source 29701 → outlet 29415

River:
  upper = 10.55 block/km
  lower = 25.00 block/km

Terrain:
  upper = 9.86
  lower = 20.81

BaseElevation:
  upper = 7.04
  lower = 20.68
```

因此：

> Waterbed 不是这类 broad downstream steepening 的第一责任层。

如果只修改 Waterbed 水面，会掩盖真实 Terrain / ErodedElevation 纵剖面问题。

## 2. 修复位置

本轮把纵剖面 grading 放在：

```text
Erosion.ErodedElevation
    ↓
Geomorph
    ├─ existing sediment / floodplain / alluvial facts
    └─ LongitudinalGradeCorrection
    ↓
Terrain.BaseElevation
    ↓
Terrain / Reconcile / Waterbed
```

新增：

- `geomorph.Config.LongitudinalGradeVersion`
- `geomorph.Model.LongitudinalGradeCorrection`

Terrain adapter 在构造 BaseElevation 时应用该 correction。

旧规则版本 correction 为全 0。

## 3. 只处理成熟冲积低地河

不是所有河流都应该拥有平缓的下游纵剖面。

山地河、峡谷河、构造控制河和局部受限河谷可以自然出现较大坡降。

因此算法先使用 FlowTo / FlowAccumulation 选择主干，再只允许满足下列条件的连续 lowland reach 进入 grading：

- 有明确 RiverStrength / accumulation。
- RollingPlain / broad Valley 背景足够。
- Mountain / Foothill / Canyon / ReliefIndex 不高。
- TectonicSupport 不强。
- 连续 reach 至少 5 个 macro node。
- 物理长度至少 512 blocks。
- 原始下游三分之一显著比上游三分之一陡。
- reach-level 冲积过程证据足够。

reach-level 冲积证据：

```text
0.50 * AlluvialPlainWeight
+ 0.30 * FloodplainWeight
+ 0.20 * DepositionPotential
```

要求：

```text
upper process >= 0.14
lower process >= 0.22
lower - upper >= 0.04
```

最后一条很重要：

> 下游必须比上游更明显地表现出冲积 / 沉积过程。

仅仅“河在平原里”不足以触发 grading。

## 4. 目标纵剖面

第一轮实验用直线连接 reach 两端，能够降低部分下游陡化，但过于人工。

第二轮改为轻度凹形 graded profile：

```text
target =
  endElevation
  + (startElevation - endElevation)
    * (1 - fraction)^1.22
```

它保持：

- reach 起点不变。
- reach 终点不变。
- 上游允许较陡。
- 下游逐渐趋缓。
- 不把整条河压成直线坡面。

曾测试过基于 FlowAccumulation 的 `S ∝ A^-θ` 目标，但当前模型上总体效果不如固定轻度凹形目标，因此没有采用。

## 5. 侵蚀与沉积不是同一个强度

如果目标高于原高程：

- 属于沉积 / 填充。
- correction 受 DepositionPotential、SedimentSupply、AlluvialPlainWeight 约束。

如果目标低于原高程：

- 属于河流侵蚀 / 削低 convex high point。
- correction 由 FluvialPower 增强。

单节点 correction 最大限制：

```text
±0.024 normalized
```

同时保持 minimum land freeboard。

## 6. 标准 Seed A/B

基线：

```text
Seed = 123456
Grid = 320 × 200
Cell = 128 blocks
baseline rules = geoworld-rules-v1.1.18
candidate = longitudinal grade v1
```

R1 原始代表流域：

```text
long representative drainages = 27
broad downstream-steepening drainages = 22
```

候选算法：

```text
broad route signals:
  33 → 29

broad representative drainages:
  22 → 20

unique internal local rises:
  2 → 2

internal rises >= 0.25 block:
  0 → 0

internal rises >= 1 block:
  0 → 0

confluence surface rises:
  0 → 0

confluence width regressions:
  0 → 0

mouth approach rises:
  0 → 0

mouth narrowing:
  0 → 0
```

### 6.1 Base / Terrain 责任层

对原来的 22 个 broad downstream-steepening 代表流域：

```text
BaseElevation gap:
  mean 20.01 → 17.91 block/km
  improved = 9
  worsened = 0

Terrain gap:
  mean 20.51 → 18.41 block/km
  improved = 9
  worsened = 0
```

这里的 gap 是：

```text
lower-third drop/km - upper-third drop/km
```

因此当前 candidate 在它真正负责的 BaseElevation / Terrain 层没有观察到代表流域回退。

### 6.2 成熟冲积候选

几个具有明确冲积背景的代表流域：

```text
outlet 29415:
  gap 14.45 → 6.52

outlet 29095:
  gap 17.91 → 5.82

outlet 31654:
  gap 16.77 → 8.75

outlet 32613:
  gap 14.97 → 10.89

outlet 10010:
  gap 33.35 → 24.39
```

其中 `29415` 和 `29095` 已经退出原 broad-steepening threshold。

## 7. 为什么不要求 River surface 指标全部下降

Waterbed 是 Terrain 后的独立河槽 / 水面实现层。

一个代表流域 `32933` 在 candidate 下：

```text
BaseElevation:
  gap 11.93 → 10.86

Terrain:
  gap 13.09 → 12.02
```

两层都改善。

但最终 Waterbed surface 的统计 gap：

```text
15.65 → 17.49
```

这是因为 Waterbed 对改变后的 broad terrain 重新求解局部河槽 / 水面，导致 upper-third surface 统计发生响应。

同时该流域没有：

- >0.25 block 局部逆升。
- 汇流异常。
- 河口异常。
- Terrain 纵剖面回退。

因此不能用单一 Waterbed surface aggregate 否决 Geomorph/Terrain 层的因果改善。

## 8. 版本策略

正式候选：

```text
geoworld-rules-v1.1.19
Geomorph.LongitudinalGradeVersion = 1
```

锁定：

```text
geoworld-rules-v1.1.18
Geomorph.LongitudinalGradeVersion = 0
```

v1.1.19 继承：

- v1.1.18 basin drainage corridor。
- RiverGeometryVersion = 5。
- LakeGeometryVersion = 1。
- 既有 shoreline / mining / biome / outer-world 语义。

明确不做：

- 不恢复被回退的 v1.1.17 receiver geometry v6。
- 不改 Waterbed receiver low-bank 规则。
- 不修改旧 v1.1.18 World Model 的地貌。

## 9. 多固定 Seed A/B

除 Seed `123456` 外，继续使用标准 `320 × 200` 世界，对锁定的
`geoworld-rules-v1.1.18` 与当前 `v1.1.19` 做同条件对照。

### Seed -4852173505788415804

```text
broad source-route signals:
  25 → 24

broad representative drainages:
  17 → 16

unique internal rises:
  1 → 1

internal rises >= 0.25 block:
  0 → 0

internal rises >= 1 block:
  0 → 0

confluence / mouth regressions:
  0
```

### Seed -563659577384754347

```text
broad source-route signals:
  10 → 7

broad representative drainages:
  6 → 5

unique internal rises:
  2 → 2

existing max internal rise:
  0.3801 → 0.3801 block

internal rises >= 0.25 block:
  1 → 1

internal rises >= 1 block:
  0 → 0

confluence / mouth regressions:
  0
```

### Seed 4034658472718554561

```text
broad source-route signals:
  14 → 13

broad representative drainages:
  6 → 5

unique internal rises:
  3 → 4

internal rises >= 0.25 block:
  2 → 2

internal rises >= 1 block:
  0 → 0

max internal rise:
  0.76784 → 0.76784 block
```

新增的那个内部 rise 只有约 `0.0108 block`，远低于 Minecraft 可见尺度。

该 Seed 原始汇流统计出现：

```text
confluence surface rises:
  0 → 1
```

进一步定位后，异常点位于湖泊 receiver/backwater 控制的下游河段，因此 R1
汇流审计增加了与整河审计相同的分类：

- `ConfluenceInternalSurfaceRises`
- `ConfluenceReceiverSurfaceRises`

分类后 candidate 为：

```text
internal confluence rises = 0
receiver-related rises    = 1
```

所以不存在普通汇流被 grading 抬坏的证据。

### 多 Seed 结论

四个标准 Seed 中：

- 代表流域 broad downstream steepening 均下降。
- 没有新增 >=0.25 block 的内部局部逆升。
- 没有新增 >=1 block 的内部局部逆升。
- 没有内部汇流水面回退。
- 没有新增河口 rise / narrowing。
- 没有发现 grading 对山地 / 峡谷河的系统性误触发。

因此 v1.1.19 的收益不是 Seed `123456` 特例。

## 10. 最终门禁

### Go

执行：

```text
go test -count=1 ./...
```

结果：**全仓通过。**

包括：

- generator
- geomorph
- geomorphaudit
- hydrology
- landform
- terrain
- reconcile
- runtimeprofile
- runtimeworld
- specialfeatures
- waterbed
- worldmodelio
- geoworld-tester

### Minecraft 1.21.1 NeoForge

JDK 21，执行：

```text
compileJava
verifyGeoWorldBasinRivers
verifyGeoWorldWaterOwnership
verifyGeoWorldShoreline
```

结果：**BUILD SUCCESSFUL。**

BasinRiver：

```text
checks      = 573
rawRaised   = 0
voxelRaised = 0
dryChannel  = 0
```

WaterOwnership：

```text
unownedSurfaceWaterSources       = 0
unsafeExposedWaterSources        = 0
freeFallWaterFaces               = 0
runtimeWaterLost                 = 0
riverWaterBoundaryModelMismatches= 0
riverLocalWaterPeaks             = 0
wetlandPoolLeaks                 = 0
```

Shoreline 最终扫描：

```text
narrowBackshoreGrooves           = 0
smallBackshorePits               = 0
oceanBackshoreStepViolations     = 0
riverWaterBoundaryModelMismatches= 0
riverDryVoxelColumns             = 0
pointPlanMismatches              = 0
adaptiveBankStepViolations       = 0
```

### Minecraft 26.3 Fabric

JDK 25，执行：

```text
compileJava
verifyGeoWorldBasinRivers
verifyGeoWorldShoreline
```

结果：**BUILD SUCCESSFUL。**

BasinRiver 同样：

```text
checks      = 573
rawRaised   = 0
voxelRaised = 0
dryChannel  = 0
```

26.3 Shoreline 全套扫描通过，关键异常计数保持 0。

构建只有现有 deprecated API / Gradle 10 compatibility warning，没有本次改动造成的编译或验收错误。

## 11. 结论与下一步

v1.1.19 只解决已经有因果证据的**成熟冲积低地河 broad longitudinal profile**：

- 不强行把所有 downstream-steepening 信号清零。
- 不拉平山地河、峡谷河或构造控制河。
- 不恢复已回退的 receiver geometry v6。
- 不修改 Waterbed receiver low-bank 规则。
- 不修改锁定的 v1.1.18 World Model。

R2 完成后进入 R4：

- 上游产沙来源。
- TransportCapacity。
- 下游 accommodation。
- 山口 / 湖泊入口 / 河口沉积。

后续仍遵循同一规则：先建立因果和验收证据，再修改生成算法。
