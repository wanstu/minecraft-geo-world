# Phase 1 — 板块与大陆骨架

## 目标

验证 GeoWorld 最重要的第一条因果链：

```text
Seed
  ↓
板块中心
  ↓
板块划分
  ↓
板块类型 + 运动方向
  ↓
板块边界
  ↓
汇聚 / 张裂 / 转换
  ↓
大陆骨架 / 海盆 / 主要山系倾向
```

本阶段只追求“宏观结构可解释”，不追求最终 Minecraft 地貌细节。

## 当前实验模型

第一版实验使用二维有限平面。

- 网格：默认 320 × 200 个宏观 Cell
- 板块：默认 18
- 每个板块有：
  - ID
  - seed point
  - velocity vector
- Cell 通过最近板块中心归属板块
- 大陆 / 海洋地壳由独立的平滑“大陆势场”生成，并按目标海洋面积阈值切分
- 大陆势场在地图外围施加海洋偏置，给风暴洋 / 冰冻洋保留缓冲
- 相邻 Cell 属于不同板块时形成边界
- 根据两侧板块相对速度在边界法线方向的投影判断：
  - convergent
  - divergent
  - transform

这只是 Phase 1 Lab，不代表最终算法一定使用 Voronoi。

## 当前输出

`geoworld-lab` 输出：

- `plates.png`：板块 ID 分区
- `crust.png`：独立大陆 / 海洋地壳场
- `boundaries.png`：三类板块边界
- `tectonics.png`：地壳与板块边界叠加，用于检查大陆是否跨板块
- `features.png`：构造地貌分类，包括大陆碰撞山系、火山弧、海沟、洋中脊、大陆裂谷等
- `elevation.png`：基于地壳类型和构造地貌的粗略高程倾向

## 下一步算法改进

1. 让板块形状避免“标准 Voronoi 多边形感”。
2. 引入板块年龄与地壳厚度。
3. 区分大陆-大陆、大陆-海洋、海洋-海洋汇聚。
4. 对汇聚边界建立更合理的山系 / 海沟 / 岛弧模型。
5. 让大陆地壳可以跨多个相邻板块，而不是“一个板块等于一块大陆”。
6. 加入侵蚀前的基础高度场。
7. 明确主地理区与外围海洋缓冲区。

## 重要原则

板块和大陆不是同一个概念。

最终设计中：

- 一个大陆可以跨多个板块。
- 一个板块可以同时包含海洋和大陆地壳。
- 板块边界决定主要构造地貌，但不会直接等价于海岸线。

当前 Lab 已把 crust field 从 plate field 中拆开。下一步需要继续细化：大陆地壳年龄 / 厚度、板块边界两侧地壳组合，以及俯冲方向。


## 当前构造地貌规则

### 汇聚边界

- continental + continental → Collision Orogen
- continental 一侧面对 oceanic → Volcanic Arc
- oceanic 一侧面对 continental → Trench
- oceanic + oceanic → Oceanic Subduction（后续根据地壳年龄决定俯冲侧）

### 张裂边界

- oceanic + oceanic → Mid-Ocean Ridge
- 任一侧为 continental → Continental Rift

### 转换边界

- Transform Fault

这些规则仍是宏观近似，但已经让“板块边界类型”真正开始影响地貌类型，而不是只负责画一条边界线。


## Phase 1.2 完成记录

已实现：

### 海洋地壳年龄

- 海洋-海洋张裂边界作为年轻地壳源。
- 使用多源传播建立海洋地壳年龄场。
- 洋中脊附近年龄接近 0。
- 远离洋中脊年龄逐渐增大。
- 不连通小海盆使用稳定回退值，避免年龄场失效。

### 海洋-海洋俯冲

当两个海洋板块汇聚时：

- 更老的海洋地壳优先成为俯冲侧并形成 Trench。
- 更年轻的上覆板块侧形成 Island Arc。
- 年龄非常接近时使用稳定的 Plate ID 打破平局，保证同 Seed 确定性。

### 构造影响带

边界 Feature 只作为“源”。

影响通过距离衰减扩展到周围区域：

- Collision Orogen：宽强抬升
- Volcanic Arc：中等抬升
- Island Arc：海洋侧抬升
- Trench：窄强下沉
- Mid-Ocean Ridge：海洋侧弱抬升
- Continental Rift：大陆侧下沉
- Transform Fault：弱影响

当前高程已经改为：

```text
基础大陆 / 海盆高度
+ 海洋地壳年龄导致的海盆冷却下沉
+ 构造影响带
+ 少量局部噪声
= Phase 1 粗略高程
```

### 新增测试

`internal/phase1/generator_test.go`

验证：

- 确定性
- 海洋面积
- 海洋地壳年龄存在有效范围
- 构造影响范围大于边界源范围

### 新增图层

- `ocean-age.png`
- `subduction.png`
- `tectonic-influence.png`


## Phase 1.3 完成记录

第一版板块边界自然化使用低频平滑 Domain Warp。

同时保留：

- `RawPlateID`：标准最近中心 Voronoi。
- `PlateID`：经过坐标扭曲后的实际板块归属。

新增：

- `plates-raw.png`：原始 Voronoi 对照。
- `plates.png`：实际自然化板块。

算法原则：

- 扭曲必须连续。
- 不能逐 Cell 加随机 plate id。
- 不能为了“自然”制造大量孤立板块碎片。
- 扭曲只改变板块边界形状，不改变 Seed 可复现性。

自动测试要求：

- naturalized plate map 必须相对 raw map 有明显变化。
- 改变比例不能过大，避免整个板块拓扑被 warp 重写。


## Phase 1.4 完成记录

### 大陆地壳厚度

大陆海岸附近地壳较薄，向大陆内部逐渐增厚。

当前为归一化“有效地壳厚度”，主要用于控制：

- 大陆基础高程
- 碰撞造山强度
- 大陆裂谷响应

### 板块运动强度

不再只有 Boundary Type。

新增：

- `BoundaryStrength`
- `TectonicIntensity`

Boundary Strength 由两块板块的相对速度分解得到：

- 汇聚 / 张裂主要看边界法线方向速度。
- 转换断层主要看切向相对速度。

Tectonic Intensity 再结合：

- Boundary Strength
- Continental Crust Thickness
- Ocean Crust Age

决定实际构造作用强弱。

### 构造带动态宽度 / 幅度

`featureBand` 已从固定值改为：

```text
Feature Type
+ Tectonic Intensity
→ Band Radius
+ Elevation Amplitude
```

因此相同类型的碰撞不再生成完全一样的山系或海沟。

### 新增图层

- `crust-thickness.png`
- `tectonic-intensity.png`

### 新增测试

- Continental crust thickness 必须存在明显变化范围。
- Tectonic intensity 必须在 0..1 内并存在明显强弱差异。


## Phase 1.5 完成记录

### 海岸距离场

新增 `CoastDistance`：

- 大陆 Cell：正值。
- 海洋 Cell：负值。
- 海岸两侧 Cell：0。
- Phase 1.5 初版距离单位为宏观 Grid Cell；Phase 6.1b 已迁移为 Minecraft blocks，旧 1 Cell 按 128 blocks 作为兼容参考尺度。

这成为大陆内部程度、陆架、陆坡、海盆等地貌的统一空间依据。

### 基础高程与构造高程分层

高程不再直接从“大陆 / 海洋 + 构造”一步得到，而拆成：

```text
Crust + Coast Distance + Crust Thickness + Ocean Age
        ↓
Base Elevation
        +
Tectonic Influence
        +
Small Local Roughness
        ↓
Final Macro Elevation
```

这样以后气候、水文可以明确区分“长期地形骨架”和“构造异常”。

### 海洋剖面

当前宏观海洋剖面：

- Phase 1.5 初版：0～6 Cell / 6～18 Cell / >18 Cell。
- Phase 6.1b 当前物理尺度：0～768 blocks 为 Continental Shelf；768～2304 blocks 为 Continental Slope；>2304 blocks 为 Ocean Basin。
- 这些 blocks 阈值来自旧 128 blocks/node 基线，默认输出保持一致；提高宏观精度时不再缩短实际地理作用距离。
- 深海盆地继续叠加海洋地壳年龄导致的冷却下沉。

### 宏观地形分区

新增 `MacroTerrain` 与 `TerrainZone`。

当前可识别：

- ContinentalInterior
- ContinentalCoast
- ContinentalShelf
- ContinentalSlope
- OceanBasin
- MountainBelt
- Plateau
- TrenchZone
- IslandArcZone
- MidOceanRidgeZone
- RiftValleyZone

### 高原修正

初版仅凭“内陆 + 厚地壳”会造成大量高原，测试 Seed 一度达到全图 15.34%。

已经修正为：

```text
厚大陆地壳
+ 远离海岸
+ 明显广域构造抬升
→ Plateau
```

修正后当前测试 Seed 高原占比约 0.43%。

### 自动测试

新增验证：

- CoastDistance 大陆 / 海洋符号正确。
- 同时存在大陆海岸与海洋海岸 Cell。
- 海洋基础高程满足：Shelf > Slope > Basin。
- 主要宏观地形区实际存在。
- 新字段仍保持同 Seed 确定性。

### 新增图层

- `coast-distance.png`
- `base-elevation.png`
- `terrain-zones.png`


## Phase 1.6 验收记录

### 多 Seed 不变量

新增 `TestMultipleSeedsPhase1Invariants`，当前覆盖：

- 1
- 2
- 42
- 2026
- 123456789
- 731894205
- -99
- -987654321

验证：

- 海洋覆盖率不漂移。
- 最终高程无 NaN / Inf。
- 最终高程保持 0..1。
- 模拟区域四条最外边均为海洋。

### 性能基线

新增：

- `BenchmarkGenerate640x400`
- `BenchmarkGenerate1024x640`

2026-09-29 当前开发机基线：

```text
640x400    ≈ 147 ms   ≈ 66 MB alloc
1024x640   ≈ 375 ms   ≈ 168 MB alloc
```

该数据只用于回归比较。

当前 Phase 1 Lab 保留了大量调试数组：

- Plate IDs
- Crust
- Coast Distance
- Crust Thickness
- Ocean Age
- Boundary / Feature
- Tectonic Intensity / Influence
- Base Elevation
- Terrain Zone
- Final Elevation

因此这不是最终 Minecraft Runtime 的目标内存占用。


### 多 Seed 静态视觉验收

新增：

`prototype/acceptance.html`

当前样本：

- Seed 1
- Seed 42
- Seed 2026
- Seed 731894205

每个样本保留四张核心图：

- crust.png
- tectonic-influence.png
- terrain-zones.png
- elevation.png

当前 480 × 300 验收样本的最终高程顶格比例：

- Seed 1：0.424%
- Seed 42：0.048%
- Seed 2026：0.927%
- Seed 731894205：0%

新增自动护栏：

- Elevation >= 0.999 的 Cell 不允许超过全图 3%。
- TectonicIntensity >= 0.999 的构造源 Cell 不允许超过 20%。

这些阈值目前只是防止算法失控，不代表最终地貌调参目标。


### Phase 1 最终收口

新增 `cmd/geoworld-audit` 批量验收。

64 Seed / 320 × 200 宏观网格结果：

```text
land edge buffer cells   min 8.000   avg 17.438   max 40.000
mountain %               2.097       5.761        8.930
plateau %                0.000       0.747        1.697
trench %                 1.202       2.333        3.891
ridge %                  2.830       6.675       10.984
rift %                   1.134       4.091        7.587
elevation saturation %   0.000       0.223        0.869
tectonic saturation %    0.000       2.062        8.855
```

审计期间发现一个 Seed 的大陆会触碰模拟区边界。

修复方式：

- 新增 `EdgeOceanBufferFraction`。
- 默认 4% 的最短边尺寸作为硬海洋缓冲区。
- 大陆候选只允许出现在缓冲区内侧。
- 仍按全图 Cell 数精确选取大陆数量，所以目标海洋覆盖率保持不变。
- 软海洋偏置继续保留，用于避免大陆沿 guard band 形成矩形切边。

Phase 1 至此正式完成。


## Phase 6.1b 追加：Reference-world 板块与宏观噪声

Phase 1.6 收口时，板块 warp、大陆种子和若干低频扰动仍以当前宏观节点坐标为输入。它们在默认 128 blocks/node 下表现正常，但相同物理 Span 改为 64 blocks/node 时会缩短实际波长并改变板块几何。

Phase 6.1b 已进一步迁移：

- Plate Center 不再以当前 Width / Height 节点坐标定义，而是在旧 128 blocks/node reference grid 中定义。
- 当前节点通过 `nodeSpacing / 128` 映射到 reference-world 坐标，再执行 nearest plate 与 plate warp。
- `warpedPlatePoint` 的 width / height / amplitude 使用物理 Span 对应的 reference grid 尺寸。
- continent seed center / radius、edge bias 与大陆潜力低频扰动使用同一 reference-world 坐标。
- CrustThickness 低频扰动与最终 Macro Elevation local roughness 的固定节点频率改成 reference-world 频率。
- Edge ocean buffer 默认宽度先按 reference grid 解释旧规则，再换算成当前网格 Cell 数。

默认 128 blocks/node 时 reference 坐标与旧节点坐标完全相同，因此旧正式基线不变。

新增固定物理 Span 128 / 64 blocks-node 回归：

- Plate Center 一致。
- RawPlateID / warped PlateID 在对应世界位置一致。
- LandPotential 在对应世界位置一致。
- continent / crust thickness / macro elevation 三类低频噪声一致。
- Edge ocean buffer 的物理宽度一致。

当前 `go run ./cmd/geoworld-audit` 32 Seed / 320×200 通过；默认 128-block Profile / Consistency 复验保持此前基线。
