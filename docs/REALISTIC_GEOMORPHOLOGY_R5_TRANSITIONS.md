# GeoWorld 真实地貌强化 — R5 地貌语义过渡

更新时间：2026-11-09

状态：**R5.1 山地→平原过渡审计完成；未修改生成算法。**

前置阶段：

- R4 routed sediment / Terrain feedback 已在 `geoworld-rules-v1.1.20` 完成并验收。
- R5 不继续扩 sediment 细节，而是转向地貌之间的语义与几何过渡。

## 1. R5 范围

R5 按独立切片推进：

1. 山地 → 山麓 / 丘陵 → 平原；
2. 盆地边缘 → 盆底；
3. 河谷 → 河漫滩；
4. 陆地 → 海岸 → 海底。

每个切片先审计，再决定是否需要改生成器。

禁止：

- 因为 landform label 相邻就直接判定为断层；
- 用统一 blur 解决所有过渡；
- 为了让类别序列“看起来完整”强插 foothill / hill；
- 没有最终 Runtime 几何证据就提升 rules version。

## 2. R5.1 问题

第一切片只回答：

> 当 macro 节点出现 `mountain ↔ plain` 直接相邻时，最终连续地形是否真的发生不自然的跳变？

原有 R1 transition audit 已经记录：

- transition class；
- 最终 Runtime slope；
- curvature。

R5.1 新增：

- `directMountainPlainEdges`
- `mountainPlainLowBufferEdges`
- `mountainPlainBufferStrengthP10/P50/P90`
- `mountainPlainLowBufferEdgeFraction`

其中 midpoint buffer strength 定义为：

```text
max(FoothillWeight, HillWeight)
```

`< 0.10` 只记为 investigation signal，不是 defect。

## 3. 为什么必须同时看分类和几何

GeoWorld 的 landform 是连续权重场。

dominant class：

```text
mountain / foothill / hill / plain / plateau / valley
```

只是“哪个权重最大”的离散标签。

因此：

```text
mountain node
→ plain node
```

并不自动意味着：

```text
陡峭山壁
→ 突然平地
```

正式判断仍看最终：

- Ground slope；
- local curvature；
- Runtime / voxel 连续性。

## 4. 固定 Seed 基线

统一：

```text
Grid = 320 × 200
Rules = geoworld-rules-v1.1.20
```

### Seed 123456

```text
transition samples        = 3293
mountain↔plain edges      = 3
mid buffer < 0.10         = 3

mountain↔plain:
  slope p50 = 15.79°
  slope p90 = 18.00°
  slope p99 = 18.50°
  curvature p90 = 1.29 blocks
  max curvature = 1.48 blocks
```

### Seed -4852173505788415804

```text
transition samples        = 1896
mountain↔plain edges      = 0
```

### Seed -563659577384754347

```text
transition samples        = 3818
mountain↔plain edges      = 1
mid buffer < 0.10         = 1

slope = 8.25°
curvature = 0.10 blocks
```

### Seed 4034658472718554561

```text
transition samples        = 4028
mountain↔plain edges      = 2
mid buffer < 0.10         = 2

mountain↔plain:
  slope p50 = 14.09°
  slope p90 = 17.70°
  slope p99 = 18.51°
  curvature p90 = 1.01 blocks
  max curvature = 1.12 blocks
```

## 5. 阶段结论

4 个固定 Seed 合计只有：

```text
6 条 mountain↔plain direct edge
```

这些位置的 midpoint 确实没有明显 foothill / hill 权重缓冲。

但最终连续地形：

- 坡度约 8°–18°；
- curvature 很低；
- 没有证据显示为程序化断崖或高度突跳。

因此：

> **“语义分类跳级”当前不是“最终几何断层”。**

R5.1 不修改 Landform / Terrain。

尤其不应该为了让 label 顺序变成：

```text
mountain → foothill → hill → plain
```

而强行修改已经连续的最终地形。

## 6. R5.1 验收结果

结论：**PASS / 不需要算法修复。**

本切片只新增 investigation audit，不提升 rules version。

继续 R5 时，下一个独立切片是：

```text
R5.2 Basin rim → basin floor
```

R5.2 必须重新建立自己的固定 Seed / profile 证据，不能把 R5.1 的结论外推到盆地。
