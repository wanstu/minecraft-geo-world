# Phase 4 — 侵蚀、Relief 与地貌细化

## 目标

Phase 1～3 已经提供：

- 构造高程
- 气候
- 降水
- 河网
- 湖泊
- 流域

Phase 4 开始把这些“原因”转化为真正的区域地貌：

```text
Tectonics
+ Climate
+ Hydrology
→ Erosion
→ Relief
→ Valleys / Foothills / Hills / Plains
→ Continuous Terrain
```

噪声仍然只允许用于最后的小尺度细节，不负责决定丘陵或山谷是否存在。

## Phase 4.1 — 侵蚀驱动力与河谷切割

状态：**完成第一版**

模块：

- `internal/erosion`
- `cmd/geoworld-erosion-lab`
- `cmd/geoworld-erosion-audit`
- `prototype/erosion.html`

### 输入

来自已有世界模型：

- Elevation
- CrustThickness
- TectonicIntensity
- TectonicInfluence
- AnnualPrecipitation
- FlowAccumulation
- RiverStrength
- LakeID

### 坡度

第一版使用局部中心差分近似宏观坡度。

坡度只是侵蚀驱动力之一，不单独决定山地 / 丘陵。

### 湿润度

以长期年降水相对于参考降水量归一化：

```text
Wetness = precipitation / reference precipitation
```

它影响：

- 坡面风化 / 削低
- 河流侵蚀功率

### Tectonic Support

当前用：

```text
max(TectonicIntensity, |TectonicInfluence|)
```

近似持续构造抬升支撑。

构造越活跃：

- 地形更不容易被长期削平。

### TerrainMaturity

当前是：

```text
TerrainMaturityProxy = 1 - TectonicSupport
```

它只是“稳定 / 成熟地形倾向”，不是实际地质年龄。

禁止把它解释成：

- 几千万年
- 几亿年
- 真实岩石年龄

未来如果加入显式地质历史模型，可替换该代理。

### Erosion Resistance

第一版抗侵蚀近似来自：

- Continental Crust Thickness
- Tectonic Support

它不是岩性模拟。

未来加入地质层 / RockType 后应由真实岩性替代。

### Fluvial Power

河流侵蚀功率来自：

```text
RiverStrength
× FlowAccumulation
× Slope
× Wetness
```

因此：

- 大河比小溪更容易切谷。
- 湿润区流量更高。
- 高坡度段更容易下切。

### Valley Incision

河谷切割只在 RiverStrength > 0 的河网附近显著发生。

LakeID 区域会抑制河谷切割：

```text
lake
→ accommodation space
→ 不允许把整片湖盆继续切成峡谷
```

### Hillslope Erosion

坡面侵蚀来自：

- Slope
- Wetness
- TerrainMaturity
- ErosionResistance
- TectonicSupport

这是广泛存在但幅度较小的背景削低。

### 地形只能降低，不能被“侵蚀抬高”

多 Seed 回归曾发现：

- 少数原始陆地 Cell 本来低于 `seaLevel + erosion floor`
- 如果直接套 `max(floor, original-total)`
- 会错误抬高地形

已修正为：

```text
if original <= erosionFloor:
    keep original
else:
    erode downward, never below floor
```

因此 Phase 4.1：

- 不抬高任何陆地。
- 不修改海洋。
- 暂不做跨海平面的河口切割。

### Relief

新增：

- LocalRelief
- ReliefIndex

LocalRelief 是一定邻域内：

```text
max elevation - min elevation
```

它和 Elevation 明确分离。

例如：

- 高海拔 + 低 Relief → 高原倾向。
- 中海拔 + 中 Relief → 丘陵倾向。
- 高海拔 + 高 Relief → 山地倾向。

### 当前 640 × 400 Seed 731894205

```text
Average erosion        0.00074
Maximum erosion        0.01962
Valley affected        1.53% land
Max valley incision    0.01386
Hillslope affected     75.66% land
Max hillslope erosion  0.01545
Average local relief   0.05254
Max local relief       0.37843
High relief            17.39% land
```

坡面侵蚀“affected”只是指大于 0.0001。

实际平均侵蚀幅度很小，因此不是大面积地形被显著削低。

### 8 Seed 审计

240 × 150：

```text
Average erosion                 0.00142 .. 0.00214
Maximum erosion                 0.01171 .. 0.01822
Valley incision land %          0.59 .. 2.36
Strong erosion >= 0.015         0 .. 0.035%
```

Relief 占比不作为死护栏，因为它依赖 Relief 观察窗口尺度。

### 自动测试

验证：

- 平坦无河地形不会凭空侵蚀。
- 强河流比弱河流切谷更深。
- 湿润坡面比干旱坡面侵蚀更强。
- 构造支撑能够抵抗坡面削低。
- 湖区抑制河谷切割。
- 海洋保持不变。
- 侵蚀不能抬高地形。
- 原始低海岸 Cell 不被 floor 错误抬高。
- 多 Seed 平均侵蚀 / 最大侵蚀 / 河谷占比存在回归护栏。

### 当前可视化

`prototype/erosion.html`

图层：

- `slope.png`
- `wetness.png`
- `tectonic-support.png`
- `terrain-maturity.png`
- `fluvial-power.png`
- `valley-incision.png`
- `hillslope-erosion.png`
- `total-erosion.png`
- `eroded-elevation.png`
- `relief.png`

## Phase 4.2 — 丘陵、山麓与连续地貌权重

状态：**完成第一版**

模块：

- `internal/landform`
- `cmd/geoworld-landform-lab`
- `cmd/geoworld-landform-audit`
- `prototype/landform.html`

### 基础地貌不是硬分类

每个陆地 Cell 同时输出：

- RollingPlainWeight
- HillWeight
- FoothillWeight
- MountainWeight
- PlateauWeight

并保证：

```text
Plain
+ Hill
+ Foothill
+ Mountain
+ Plateau
= 1
```

Dominant Landform 只用于调试画图。

Runtime / 后续 Terrain Composer 不允许直接按 dominant 分类硬切地形函数。

### Valley 是 Overlay

新增：

- ValleyCore
- ValleyWeight

河谷不参与基础五类归一化。

原因：

> 河谷可以穿过平原、丘陵、山麓、山地和高原。

第一版先从：

- ValleyIncision
- FluvialPower

建立 ValleyCore，再向河道两侧连续衰减形成有宽度的 ValleyWeight。

### 丘陵

丘陵不是：

```text
noise > threshold
```

第一版 HillWeight 主要来自：

```text
中等 Relief
+ TerrainMaturity
- MountainInfluence
- PlateauSignal
```

因此丘陵倾向于出现在：

- 经过较长期侵蚀的稳定陆地。
- 起伏明显但又不是强山地的地区。
- 山地外围以外的成熟地形。

### 山麓

山麓来自：

```text
Mountain Core
→ Distance / Influence
→ FoothillWeight
```

因此山麓是平原 / 丘陵与山地之间的连续过渡，不是独立随机生成的斑块。

### 高原

高原和山地明确区分：

```text
High / Mid-High Elevation
+ Thick Continental Crust
+ Low Relief
→ Plateau

High Elevation
+ High Relief
→ Mountain
```

高原信号会主动削弱普通 PlainWeight 的竞争。

这解决了一个实际问题：

> 低 Relief 同时是“平原”和“高原”的共同特征，如果不加入海拔 / 地壳厚度竞争，高原会永远被平原压住。

### 宏观网格尺度一致性

Phase 4.2 验收时发现：

同一个 Seed：

```text
640 × 400 → Mountain dominant 14.96%
240 × 150 → Mountain dominant 33.24%
```

原因不是 Seed，而是：

- ReliefRadius
- MountainInfluence
- ValleySpread

最初使用固定 Cell 数。

低分辨率时，同样 4～10 Cell 会覆盖更大的世界比例。

已改为根据宏观网格短边解析默认空间尺度。

修复后同 Seed：

```text
640 × 400 → Mountain dominant 14.96%
240 × 150 → Mountain dominant 12.42%
```

因此：

> 宏观网格尺寸主要表示采样精度，不应隐式改变区域地貌的物理尺度。

显式 Config 仍可覆盖默认尺度，方便实验。

### 当前 640 × 400 Seed 731894205

```text
Rolling Plain dominant   65.59%
Hill dominant            16.06%
Foothill dominant         2.78%
Mountain dominant        14.96%
Plateau dominant          0.61%
Mixed transition         11.45%
Strong valley overlay     0.22%
```

Mixed transition 表示：

- 第一权重与第二权重差值 < 0.20。

它不是异常，而是连续地貌混合真正存在的证据。

### 4 Seed 正式尺寸审计

640 × 400：

```text
Plain dominant        62.80% .. 65.64%
Hill dominant          9.63% .. 11.73%
Foothill dominant      3.64% .. 4.30%
Mountain dominant     18.20% .. 21.13%
Plateau dominant       0.05% .. 3.58%
Mixed transition      11.53% .. 16.45%
Strong valley overlay  0.64% .. 0.84%
```

测试 Seed 731894205 的丘陵比例更高（16.06%），但仍在合理范围，说明不同世界可以有不同成熟地貌结构。

### 自动测试

验证：

- 基础五类权重严格归一化。
- 低 Relief 成熟地形偏平原。
- 中 Relief 成熟地形偏丘陵。
- 高海拔 + 低 Relief + 厚地壳偏高原而不是山。
- 高 Relief 偏山地。
- 山地外围产生 FoothillWeight。
- ValleyWeight 是 Overlay，不破坏基础权重归一化。
- 海洋不会获得陆地地貌权重。
- Relief 小幅变化不会引发权重突变。
- 默认空间核随宏观网格尺寸变化。
- 同 Seed 在 240×150 / 640×400 下 Mountain 占比不得再次发生数量级漂移。

### 当前可视化

`prototype/landform.html`

图层：

- `mountain-core.png`
- `mountain-influence.png`
- `rolling-plain-weight.png`
- `hill-weight.png`
- `foothill-weight.png`
- `mountain-weight.png`
- `plateau-weight.png`
- `valley-weight.png`
- `dominant-landform.png`

## 下一步 — Phase 4.3

把宏观高程 + 连续地貌权重真正合成为可连续采样的 Regional Terrain：

```text
ErodedElevation
+ Landform Weights
+ Regional Detail Fields
+ Valley Shape
→ RegionalTerrainHeight(x,z)
```

重点验收：

- 宏观 Cell 边界无台阶。
- Minecraft 16×16 Chunk 边界无接缝。
- 平原 → 丘陵 → 山麓 → 山地连续过渡。
- 河谷不出现“切一条直槽”的人工感。
- 局部细节可以使用噪声，但不能改变宏观地貌归属。
