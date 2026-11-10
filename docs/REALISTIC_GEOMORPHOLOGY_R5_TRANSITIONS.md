# GeoWorld 真实地貌强化 — R5 地貌语义过渡

更新时间：2026-11-10

状态：**R5.1、R5.2 均完成并通过；未修改生成算法。**

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

## 7. R5.2：Basin rim → basin floor

第二切片只回答：盆地 rim → wall → floor 的最终 Runtime 几何，是否同时满足“盆底较平、盆壁明显变陡、外缘平滑回接”，并且不会把真实 drainage outlet valley 误判成盆壁缺陷。

每个 BasinCherry / BasinSnow 固定采 16 个方向：

    floor: r=0.15 → 0.45
    wall:  r=0.58 → 0.88
    outer: r=0.88 → 1.04

记录 floor / wall / outer slope、rim→floor relief、wall/floor slope ratio，以及 inverted rim / floor steeper than wall 信号。

### drainage outlet 单独分类

v1.1.18 起盆地会持久化真实 Drainage corridor。只要 wall / rim / outer 采样进入 drainage corridor，就归为 drainageAffected，而不是普通盆壁。出口方向本来就应该切开 rim，不能按普通盆壁失败处理。

## 8. 固定 Seed 基线

统一：320×200，geoworld-rules-v1.1.20，16 profiles / basin。

4 个固定 Seed 共 12 个盆地、192 条径向剖面。

Seed 123456：

    ordinary profiles          = 35
    drainage-affected profiles = 13
    ordinary inverted rim      = 0
    drainage inverted rim      = 1
    floor steeper than wall    = 0
    floor slope p90            = 1.03°
    wall slope p90             = 8.67°
    outer blend slope p90      = 1.43°
    rim→floor relief p90       = 28.10 blocks

唯一一条 inverted rim profile 位于 drainage outlet corridor，属于预期出口谷。

Seed -4852173505788415804：

    ordinary profiles          = 48
    drainage-affected profiles = 0
    ordinary inverted rim      = 0
    floor steeper than wall    = 0
    floor p90 / wall p90       = 0.05° / 8.25°
    outer p90                  = 1.31°
    rim→floor relief p90       = 27.39 blocks

Seed -563659577384754347：

    ordinary profiles          = 46
    drainage-affected profiles = 2
    ordinary inverted rim      = 0
    floor steeper than wall    = 0
    floor p90 / wall p90       = 0.08° / 8.26°
    outer p90                  = 3.00°
    rim→floor relief p90       = 29.83 blocks

Seed 4034658472718554561：

    ordinary profiles          = 48
    drainage-affected profiles = 0
    ordinary inverted rim      = 0
    floor steeper than wall    = 0
    floor p90 / wall p90       = 0.07° / 8.95°
    outer p90                  = 1.23°
    rim→floor relief p90       = 28.30 blocks

## 9. R5.2 结论

普通盆壁在 4 个固定 Seed 中 ordinary inverted rim = 0，floor steeper than wall = 0。

整体形态稳定表现为：

    very flat basin floor
    → clearly steeper wall
    → gentle outer blend

这与 GeoWorld 对特殊盆地的既定目标一致：边缘较陡，中间较平，同时给真实河流保留出口谷。

没有证据支持继续加深盆地、抬高 rim 或扩大 wall steepness。

## 10. R5.2 验收结果

结论：**PASS / 不需要算法修复。**

本切片只增加 investigation audit，不修改 SpecialFeatures / Terrain，不提升 generation rules version。

下一独立切片：R5.3 River valley → floodplain。

## 11. R5.3：River valley → floodplain

### 11.1 真实地貌参考

本切片把卫星图 / 航片对照正式纳入验收。参考重点不是复制影像纹理，而是抽取过程约束：

- NASA Earth Observatory《Braided River in Tibet Redraws Its Channels》：宽阔高原河谷中的辫状河会在活动谷底内持续重组。
- NASA Earth Observatory 的深切曲流案例：曲流本身不等于宽阔洪泛平原，河道可以被峡谷锁定。
- USGS Powder River 长期航片研究：河道迁移是洪泛平原更新的重要机制；河宽、迁移率和形态会随洪水过程变化。
- USGS Powder River 历史变化研究：局部基岩约束和谷坡会让同一条河重新进入受限河段。

因此禁止使用“所有河流越往下游洪泛平原必须单调变宽”作为验收标准。

### 11.2 审计方法

最终方案不比较彼此无关的河流，而比较**同一 representative drainage** 的纵向演化。

对每条代表性 drainage：

    长度 >= 1800 blocks
    至少 4 个 river segments
    upper cross-section = route 25%
    lower cross-section = route 70%

排除：

- receiver/backwater；
- ocean/lake mouth；
- lake outlet；
- SpecialFeature 覆盖位置。

横断面使用最终 Runtime GroundHeight，并以：

    river water surface + 6 blocks

作为 active low-relief floodplain corridor 的高程上限；从河岸向两侧最多探测 256 blocks。

记录：

- channel half-width；
- FloodplainWeight；
- AlluvialPlainWeight；
- ValleyFloorWidth；
- plain / upland context；
- active corridor half-width；
- 距河岸 160 blocks 的 far relief。

成熟 floodplain investigation signal 定义为：

    lower FloodplainWeight >= 0.35
    and lower FloodplainWeight > upper + 0.10

注意：这仍然只是 investigation signal，不自动等于缺陷。

### 11.3 为什么废弃第一版分类

第一版曾按单个河段的 `confined / transitional / unconfined` 标签直接横向比较不同河流。

固定 Seed 很快证明这种方法会混淆河流尺度：平原上的小河会因为 ValleyFloorWidth 小被误归为 confined，但最终横断面实际非常宽。

因此该思路被废弃，没有据此修改生成器。

最终只比较同一 drainage 的上 / 下游，并结合最终横断面和周围地貌背景解释。

## 12. R5.3 固定 Seed 结果

统一：

    Grid = 320 × 200
    Rules = geoworld-rules-v1.1.20

### Seed 123456

    representative drainages = 68
    long drainages           = 30
    mature floodplain        = 2
    channel widening         = 30
    mature narrowing signals = 0

两条成熟 floodplain 的 expansion ratio 分别约 0.84 和 1.10；均保留宽阔 active corridor，没有出现河道贴坡。

### Seed -4852173505788415804

    representative drainages = 84
    long drainages           = 25
    mature floodplain        = 4
    channel widening         = 25
    mature narrowing signals = 3

3 个 narrowing signal 不能直接判缺陷：

- 其中 2 条 lower section 明显重新进入丘陵 / 山麓约束，属于允许的再次收窄；
- 另 1 条保持纯平原 / 冲积背景，进行了额外连续横断面诊断。

该纯平原 drainage：

    source = 34452
    outlet = 42796

在 route 0.55–0.80 连续窗口内：

    channel half-width      ≈ 35–42 blocks
    FloodplainWeight        ≈ 0.37–0.48
    plain context           ≈ 0.96–1.00
    upland context          ≈ 0.00–0.02
    active corridor half-width ≈ 82–102 blocks

也就是说虽然它比 0.25 处原本就很宽的区域性平原更窄，但成熟河段仍然保留约 2 倍以上河道半宽的低平活动走廊，且不是单个切面的偶然结果。

因此“相对上游平原变窄”不能作为缺陷；上游区域性平原宽度和洪泛平原宽度不是同一个量。

### Seed -563659577384754347

    representative drainages = 85
    long drainages           = 33
    mature floodplain        = 0
    channel widening         = 28
    mature narrowing signals = 0

### Seed 4034658472718554561

    representative drainages = 84
    long drainages           = 17
    mature floodplain        = 0
    channel widening         = 17
    mature narrowing signals = 0

## 13. R5.3 结论

当前固定 Seed 没有确认出“FloodplainWeight 已成熟，但最终 Runtime 河谷仍被错误夹成窄槽”的稳定缺陷。

实际观察符合真实地貌原则：

- 河道沿 drainage 普遍增宽；
- 只有一部分 drainage 真正进入成熟洪泛阶段；
- 成熟洪泛阶段存在与河宽相称的低平活动走廊；
- 同一河流重新进入丘陵 / 基岩约束时允许再次收窄；
- 区域性大平原不能被错误当作洪泛平原基准宽度。

因此本阶段不修改 Geomorph / Terrain / Waterbed。

## 14. R5.3 验收结果

结论：**PASS / 保留 investigation signals，不需要算法修复。**

本切片：

- 只增加只读 geomorph audit；
- 不修改世界生成；
- 不提升 generation rules version；
- 不增加“强制下游扩宽”规则。

下一独立切片：R5.4 Land → coast → seafloor。
