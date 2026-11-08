# GeoWorld 真实地貌强化 — R1 验收框架

更新时间：2026-11-08

状态：**第一切片已实现。**

对应主计划：[REALISTIC_GEOMORPHOLOGY.md](REALISTIC_GEOMORPHOLOGY.md)  
前置审计：[REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md](REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md)

## 1. 定位

R1 只增加只读测量，不修改世界生成算法。

新增：

- `internal/geomorphaudit/audit.go`
- `internal/geomorphaudit/audit_test.go`
- `cmd/geoworld-realism-audit/main.go`

命令：

```bash
go run ./cmd/geoworld-realism-audit -seed 123456
```

本工具**没有 strict 模式**。

所有输出统一标记：

```text
evidenceClass = investigation_signal
```

表示它只能触发后续调查，不能单独证明 Minecraft 世界存在生成缺陷。

## 2. 为什么不能复用 river-navigation-audit 的 strict 结论

DADM 历史已经证明：

- `9313fc5` 曾因为 receiver 单调性统计提升规则到 v1.1.17。
- 随后 `ea2aa9f` 明确回退。
- 当前正式 WaterOwnership / Shoreline 验收没有证明对应 Minecraft 缺陷。

因此 R1 不再设置“某个统计必须为 0，否则生成失败”的总开关。

正确流程：

```text
R1 signal
→ 定位固定 Seed / 坐标
→ 对照最终 Runtime + SpecialFeatures
→ 对照 Minecraft voxel / WaterOwnership / Shoreline
→ 只有最终结果确实错误才进入 R2
```

## 3. 河流整河纵剖面

R1 从没有上游 Waterbed segment 的 headwater 开始，沿 Waterbed DAG 追踪到最终 sink。

采样使用：

- authoritative `RiverCenterPoint()`
- 最终 `RuntimeProfile`
- `SpecialFeatures.Apply()`

因此不会只测 SpecialFeatures 之前的中间结果。

记录：

- source / end node。
- segment 数。
- 实际 centerline 长度。
- 总落差。
- 上 / 中 / 下游 drop per km。
- dry sample。
- downstream surface rise。
- downstream-steeper route。

### receiver 分类与去重

整河路径会共享下游 trunk，因此同一个物理 rise 可能被多个 headwater route 重复经过。R1 同时记录：

- `traversed*RiseSteps`：路径遍历次数。
- `unique*RiseSteps`：按 `segment + t` 去重后的物理位置数。
- `internalRiseAboveQuarterBlock` / `internalRiseAboveOneBlock`：按 Minecraft 方块尺度分级。
- `maxInternalRiseBlocks` / `maxReceiverRiseBlocks`。

receiver-related 判定读取：

- Waterbed 的 `RiverReceiverSurfaceDefined`。
- 最终 WaterKind 是否进入 Lake / Ocean。

这避免把合法 backwater / receiver handoff、共享 trunk 重复计数与真正内部河段问题混在一起。

## 4. 汇流

对每个至少两个 incoming segment 的 junction：

- 检查 incoming → downstream 最终水面。
- 检查 persisted downstream half-width 是否满足现有 Waterbed 汇流宽度契约。

宽度契约直接读取 downstream segment 的 `HalfWidth`。

**不在 junction 重叠中心使用 NavigationSampleAt 判定下游宽度。**

原因：多个 segment 在交汇点 bank distance 都可能为 0，NavigationSampleAt 可能合法返回 incoming segment，从而产生假“下游缩窄”。

## 5. 河口

对 OceanMouth：

- 比较 t=0.80 → t=0.98 的最终水面。
- 读取实际 Navigation half-width。
- 记录 approach rise。
- 记录 mouth narrowing signal。

这些仍然只是 signal。

如果 river→ocean 出现统计 rise，但最终 Shoreline / WaterOwnership 不存在水墙、断流、不可导航台阶，则不能据此改生成算法。

## 6. 地貌过渡

R1 在 macro 相邻节点的 landform 类型发生变化时，沿边界法线对最终 Ground 做小尺度采样。

分类来自 Landform 连续权重中的主导类型：

- plain
- hill
- foothill
- mountain
- plateau
- valley

记录：

- 边界样本数。
- slope p50 / p90 / p99 / max。
- 局部二阶高度变化（curvature）p90 / max。

目的不是直接规定“山地→平原不能超过多少度”，而是先建立 GeoWorld 自己的统计分布，后续再和真实 DEM 对照。

## 7. DADM 快速集成基线

```text
Seed = 123456
Grid = 96 × 60
Cell = 128 blocks
Rules = geoworld-rules-v1.1.16
```

结果：

```text
river segments                   = 103
source→sink routes               = 23
traversed rise steps             = 19
unique rise locations            = 10
unique internal rises            = 0
unique receiver-related rises    = 10
internal rises >= 0.25 block     = 0
internal rises >= 1 block        = 0

confluences                      = 6
confluence surface rises         = 0
confluence width regressions     = 0
ocean mouths                     = 14
mouth approach rises             = 0
mouth narrowing signals          = 0
terrain transition samples       = 926
```

因此小网格仍然**不支持恢复已回退的 receiver v1.1.17 方案**。

## 8. 标准 320 × 200 基线

标准尺寸 Seed `123456`：

```text
river segments                   = 955
source→sink routes               = 104
traversed rise steps             = 113
unique rise locations            = 84
unique internal rises            = 41
unique receiver-related rises    = 43
internal rises >= 0.25 block     = 33
internal rises >= 1 block        = 16
max internal rise                = 1.712 blocks

confluences                      = 33
confluence surface rises         = 0
confluence width regressions     = 0
ocean mouths                     = 64
mouth approach rises             = 0
mouth narrowing signals          = 0
terrain transition samples       = 3293
```

定位明细显示，大幅 internal rise 集中在：

- `basin_cherry`
- `basin_snow`

例如：

```text
segment 82  basin_cherry  rise ≈ 1.612 blocks
segment 512 basin_snow    rise ≈ 1.191 blocks
```

而普通汇流与河口没有对应异常。

### 盆地定向复核

现有 `TestReportedBasinRiverBelowBanks` 已证明 v1.1.16 的盆地河流不会高过最近干岸。本轮增加纵向诊断后确认：

```text
SpecialFeatures 前：
  max downstream rise = 0

SpecialFeatures 后：
  max downstream rise ≈ 0.861 block / 8 blocks
```

也就是说：

> 原 Waterbed 河面保持下游单调；盆地 radial depression overlay 把穿盆地河流压成“先下降、再爬出盆地”的纵剖面。

这已经把 R2 对象缩小到**特殊盆地与河流的组合语义**，不是普通 Waterbed、receiver、汇流或河口。

同时验证了两个错误修法：

1. **只保留原河面、继续挖低盆地两岸**：478/478 个检查点都会变成河水高过干岸，最大高差约 `0.0393 normalized`，不可采用。
2. **保留原河面，同时保护 Waterbed 已塑造的近河岸 corridor**：当前固定样例的 478 个近岸检查点均可保持水位不高过岸。这个结果说明河流走廊需要和盆地地形一起处理，但仍需避免形成一条狭窄的“高架河谷”。

## 9. 第一批地貌过渡基线

标准 320 × 200 Seed `123456`：

```text
foothill ↔ hill:
  n=744, p90≈5.58°, p99≈10.79°

foothill ↔ mountain:
  n=1381, p90≈14.81°, p99≈28.27°

hill ↔ plain:
  n=902, p90≈3.89°, p99≈6.75°

mountain ↔ plain:
  n=3, p90≈18.00°, p99≈18.50°

hill ↔ valley:
  n=58, p90≈14.96°, curvature p90≈9.20 blocks

plain ↔ valley:
  n=58, p90≈12.48°, curvature p90≈6.54 blocks
```

R5 优先调查 valley 边界的高 curvature；当前仍只是形态候选，不修改算法。

## 10. R1 结论与 R2 入口

R1 第一阶段完成后，目前有一个足够明确的 R2 对象：

**穿越特殊盆地的河流纵剖面。**

R2 约束：

- 不恢复历史 receiver v1.1.17。
- 不修改普通 Waterbed 河网。
- 旧 `geoworld-rules-v1.1.16` 保持锁定。
- 新规则必须同时保证：
  - 河面沿下游不因盆地 radial shape 重新爬坡；
  - 河流不高过两岸；
  - 盆地仍然保持“边缘较陡、内部较平”的地貌身份；
  - 河流通过盆地时表现为合理的出口谷/河流走廊，而不是狭窄高架水槽。

河谷宽度、山口冲积扇和更多真实 DEM 对照继续作为 R1/R4/R6 后续指标。
