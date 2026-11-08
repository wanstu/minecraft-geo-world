# GeoWorld 真实地貌强化 — R2 盆地河流出口谷

更新时间：2026-11-08

状态：**第一项生成算法修复完成并通过自动验收。**

前置：
- [REALISTIC_GEOMORPHOLOGY.md](REALISTIC_GEOMORPHOLOGY.md)
- [REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md](REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md)
- [REALISTIC_GEOMORPHOLOGY_R1_ACCEPTANCE.md](REALISTIC_GEOMORPHOLOGY_R1_ACCEPTANCE.md)

## 1. R2 为什么从盆地河流开始

R1 标准 320 × 200、Seed `123456` 把普通 Waterbed、汇流、河口和特殊盆地拆开后发现：

```text
v1.1.16:
unique internal river rises       = 41
internal rises >= 0.25 block      = 33
internal rises >= 1 block         = 16
max internal rise                 ≈ 1.712 blocks

confluence surface rises          = 0
confluence width regressions      = 0
mouth approach rises              = 0
mouth narrowing signals           = 0
```

大幅内部逆升集中在 `basin_cherry` / `basin_snow`。

同一河流在 SpecialFeatures 应用前保持下游单调，应用盆地 radial depression 后才出现先降后升。

因此 R2 第一项正式缺陷是：

> 穿过特殊盆地的河流被盆地径向挖深逻辑重新塑形，导致它在离开盆地时沿下游方向爬坡。

这与历史 receiver v1.1.17 问题不同。

## 2. 为什么不能直接保留原河面

诊断验证：

```text
保持原 Waterbed 河面
+ 继续按 v1.1.16 挖低盆地干地
→ 478 / 478 个检查点河水高过最近干岸
→ max excess ≈ 0.039319 normalized
```

所以“河面不动、地形照旧挖”会重新制造高架河。

修复必须同时处理：

- 河流水面纵向单调。
- 河床与水深。
- 河岸。
- 盆地出口地形。

## 3. 新规则：geoworld-rules-v1.1.18

本修复提升当前规则到：

```text
geoworld-rules-v1.1.18
```

注意：

- 历史 `geoworld-rules-v1.1.17` 曾用于 receiver 实验并已被回退。
- 本阶段不恢复 v1.1.17。
- 为避免把两个完全不同的语义混在同一版本号里，本次直接使用 v1.1.18。
- 锁定的 v1.1.16 世界保持原行为。

Waterbed：

```text
RiverGeometryVersion = 5
```

没有重新启用被回退的 geometry v6。

## 4. BasinDrainageReach

v1.1.18 为 BasinCherry / BasinSnow 持久化实际穿越盆地的河流走廊。

每个 reach 保存：

- x0 / z0
- x1 / z1
- surface0
- surface1
- halfWidth

这些数据来自已经完成的 Waterbed / WorldSample 河流，而不是盆地自己重新猜河流。

规划过程：

1. 枚举正式 Waterbed river segment。
2. 按实际 meander centerline 采样。
3. 只保留落入盆地影响范围的连续 river reach。
4. 使用 Runtime 已确定的河流水面。
5. 下游 surface 不允许高于上游 surface。
6. 将结果写入 SpecialFeature.Drainage。

因此盆地获得的是“已有水文事实”，不是第二套河网。

## 5. Runtime 语义

### 河槽

当 v1.1.18 盆地存在 Drainage 且当前位置本来就是正式 River：

- 保留 Waterbed 已确定的 GroundHeight。
- 保留 WaterSurfaceElevation。
- 保留 WaterDepth。
- 不再用盆地径向函数重新压低/抬高河流 datum。

### 干地河谷

对 drainage corridor 附近的干地：

- 仍参与盆地挖深。
- 但地面必须高于对应河流水面。
- 从河岸向外逐渐升高。
- 再平滑回普通 basin target。

结果是：

```text
盆地底
  ↓
河流低点
  ↓
河岸 / 洪泛走廊
  ↓
出口谷
  ↓
自然地形
```

而不是：

```text
盆地底 → 高架水槽
```

也不是：

```text
盆地底 → 河流爬盆地壁
```

## 6. Go / Java 一致性

Drainage 持久化在 World Model 的 SpecialFeature 中。

Go Runtime 和 Java Runtime Reader 使用同一套：

- drainage 最近 reach 采样。
- surface 插值。
- corridor radius。
- bank rise。
- basin target blend。

Java reader 会逐项解析并校验 persisted drainage reach；旧模型没有 `drainage` 字段时读取为空列表。

因此：

- v1.1.16 旧模型保持历史逻辑。
- v1.1.18 新模型重载后仍保留出口谷。
- 不依赖重新生成才能恢复正确语义。

## 7. 固定 Seed 生成结果

标准尺寸：

```text
Seed = 123456
Grid = 320 × 200
Cell = 128 blocks
```

### v1.1.16 R1 基线

```text
unique internal rises             = 41
internal rises >= 0.25 block      = 33
internal rises >= 1 block         = 16
max internal rise                 ≈ 1.712
```

### v1.1.18

```text
unique internal rises             = 2
internal rises >= 0.25 block      = 0
internal rises >= 1 block         = 0
max internal rise                 ≈ 0.0291

unique receiver rises             = 43
confluence surface rises          = 0
confluence width regressions      = 0
mouth approach rises              = 0
mouth narrowing signals           = 0
```

剩余两个 internal signal：

```text
≈ 0.0232 block
≈ 0.0291 block
```

远低于 Minecraft 一个方块，且不在 special basin 中，不作为生成缺陷。

## 8. 盆地定向 Go 回归

`TestCurrentBasinRiverDrainageIsContainedAndMonotonic`：

```text
drainedBasins                     = 3
drainageReaches                   = 174
max reaches per basin             = 141

cross-section checks              = 573
river above dry bank              = 0
max bank excess                   = 0

longitudinal basin samples        = 448
rises > 0.25 block                = 0
max downstream rise               = 0
```

同时保留了 v1.1.16 的历史单元测试，并新增：

`TestV1118BasinDrainageKeepsRiverDatumAndShapesOutletValley`

用于验证：

- v1.1.18 不修改 authoritative river datum。
- v1.1.18 不修改 authoritative river bed。
- drainage corridor 的河岸高于水面。
- corridor 外仍是正常盆地挖深。
- v1.1.16 不会偷偷采用 v1.1.18 语义。

## 9. Minecraft voxel 验收

### 1.21.1 NeoForge

```text
verifyGeoWorldBasinRivers:
  checks=573
  rawRaised=0
  voxelRaised=0
  dryChannel=0

verifyGeoWorldShoreline:
  PASS
  riverWaterBoundaryModelMismatches=0
  riverDryVoxelColumns=0
  pointPlanMismatches=0
  adaptiveBankStepViolations=0

verifyGeoWorldWaterOwnership:
  chunks=1183
  columns=302848
  waterColumns=157104
  unownedSurfaceWaterSources=0
  unsafeExposedWaterSources=0
  freeFallWaterFaces=0
  runtimeWaterLost=0
  riverWaterBoundaryModelMismatches=0
  riverLocalWaterPeaks=0
  wetlandPoolLeaks=0
```

Gradle：

```text
BUILD SUCCESSFUL
```

### 26.3 Fabric

JDK 25：

```text
compileJava
BUILD SUCCESSFUL

verifyGeoWorldBasinRivers:
  checks=573
  rawRaised=0
  voxelRaised=0
  dryChannel=0

BUILD SUCCESSFUL
```

## 10. Go 全仓

```text
go test -count=1 ./...
PASS
```

包括：

- generator
- geomorphaudit
- hydrology
- erosion
- geomorph
- terrain
- waterbed
- worldsample
- runtimeworld
- worldmodelio
- specialfeatures
- geoworld-tester

全部通过。

## 11. R2 第一项结论

盆地河流问题已经从：

> “盆地像一个径向碗，河流跟着碗面一起先下后上”

改成：

> “盆地保留既有下泄河流，并围绕真实河流形成出口谷/河流走廊”。

这符合 Phase 8 的原则：先尊重水文事实，再让局部地貌适配水文，而不是由特殊地形覆盖层重新定义河流。

下一步不继续改河流 receiver。

R2 后续进入：

1. 河流上/中/下游坡降分布的真实形态评估。
2. 河谷宽度随流量、坡降和约束地形变化。
3. 湖泊出口与 lake approach 的整河级验收。
4. 然后进入 R3：内流盆地 / 终止湖正式语义。
