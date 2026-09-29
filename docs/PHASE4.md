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

## Phase 4.3 — Regional Terrain 合成与连续采样

状态：**完成第一版**

模块：

- `internal/terrain`
- `cmd/geoworld-terrain-lab`
- `cmd/geoworld-terrain-audit`
- `prototype/terrain.html`

### 最终区域高度函数

Regional Terrain 不以 Chunk 为生成单位。

统一入口：

```text
H(worldX, worldZ)
```

第一版结构：

```text
CubicBSpline(ErodedElevation)
+
Σ LandformWeightᵢ × RegionalDetailᵢ
-
ValleyCarve
→ RegionalTerrainHeight
```

其中：

- worldX / worldZ 是绝对世界坐标。
- Chunk 坐标不进入数学公式。
- 相同世界坐标无论从哪个 Chunk 请求都得到相同结果。

### Cubic B-Spline

Phase 1 的 `ScalarGrid` 原本只有 Bilinear。

Phase 4.3 新增：

- `SampleCubicBSpline(worldX, worldZ)`

原因：

- Bilinear 保证 C0 连续，即高度不断裂。
- 但一阶坡度会在宏观 Cell 边界改变。
- Regional Terrain 需要更平滑的坡面。

Uniform Cubic B-Spline 具有：

- 网格内部 C2 连续。
- 基函数非负。
- 基函数总和为 1。
- 结果保持在邻域控制点凸包内。
- 不产生普通 interpolating cubic 容易出现的 overshoot 尖峰。

宏观节点因此被解释为：

> 地形控制点，而不是最终曲线必须精确穿过的点。

### Regional Detail

噪声只允许负责区域细节。

第一版各地貌振幅关系：

```text
Mountain
>
Foothill
>
Hill
>
Plain

Plateau < Hill
```

因此：

- 山地可以有更强区域起伏。
- 丘陵有中等起伏。
- 平原只有轻微波动。
- 高原则保持高海拔但较平缓。

所有噪声只读取：

- Seed
- worldX
- worldZ

不读取 Chunk 坐标。

### 噪声连续性

Regional Detail 使用：

- 世界坐标 Value Noise。
- Quintic Fade。
- 多 octave FBM。

Quintic Fade 在晶格边界提供平滑导数。

因此噪声自身不会在：

- Minecraft Chunk 边界
- Macro Cell 边界

产生跳变。

### 河谷形状

ValleyWeight 不再只作用在一条河线。

Phase 4.2 已把 ValleyCore 向两侧连续扩散。

Phase 4.3：

```text
ValleyWeight
→ ValleyCarve
+ local detail damping
```

效果：

- 谷底被进一步轻微压低。
- 河谷内部高频粗糙度降低。
- 不会把河流处理成“一条直线切槽”。

### 平滑高度限幅

初版曾使用：

```text
clamp(base + detail, 0, 1)
```

多 Seed 审计发现少量高山被 Regional Detail 推到 1.0，可能造成硬截平。

已改成 bounded additive：

- 接近 1.0 时，正向细节按剩余 headroom 平滑压缩。
- 接近 0.0 时，负向细节按剩余 footroom 平滑压缩。
- 中间高度区仍近似正常加法。

因此不会为了防越界而整体降低山地细节振幅。

### 当前测试 Seed 731894205

正式网格：

```text
Macro Grid       640 × 400
Macro Cell       128 blocks
World Extent     ≈ 81,792 × 51,072 blocks
```

结果：

```text
Height              0.14571 .. 0.91951
Regional Detail    -0.03224 .. +0.03088
Max Valley Carve    0.00454
```

连续性：

```text
Max Chunk Boundary epsilon jump
= 0.000000102249

Max Macro Boundary slope mismatch
= 0.000005356980
```

### 4 Seed 正式尺寸审计

640 × 400 / 128 blocks per macro cell：

```text
Maximum |Regional Detail|
0.03314 .. 0.03689

Maximum Valley Carve
0.00441 .. 0.00558

Chunk Boundary epsilon jump
1.13e-7 .. 1.48e-7

Macro Boundary slope mismatch
2.26e-6 .. 6.13e-6
```

审计护栏：

- `|RegionalDetail| <= 0.06`
- `Chunk epsilon jump <= 1e-5`
- `Macro slope mismatch <= 1e-4`

超过护栏时 `geoworld-terrain-audit` 直接失败。

### 自动测试

验证：

- B-Spline 常量场保持不变。
- B-Spline 不超出控制值范围。
- B-Spline 宏观边界一阶导连续。
- Minecraft Chunk 边界无接缝。
- 同坐标结果完全确定。
- 不同 Seed 会改变区域细节。
- Mountain detail 振幅 > Hill > Plain。
- Plateau detail 比 Hill 平缓。
- Valley Overlay 会压低地形并削弱粗糙度。
- 连续采样后的基础地貌权重仍保持归一化。
- 最终高度无 NaN / Inf。
- bounded add 不新制造硬 0 / 1 截断。

### 当前可视化

`prototype/terrain.html`

图层：

- `regional-detail.png`
- `valley-carve.png`
- `regional-height.png`

同时对照：

- Phase 4.1 `eroded-elevation.png`
- Phase 4.2 `dominant-landform.png`

## Phase 4.4 — 河谷横断面与沉积型地貌

状态：**完成第一版**

新增模块：

- `internal/geomorph`
- `cmd/geoworld-geomorph-lab`
- `cmd/geoworld-geomorph-audit`
- `prototype/geomorph.html`

新增连续字段：

- FloodplainWeight
- AlluvialPlainWeight
- AlluvialFanCore
- AlluvialFanWeight
- CanyonWeight
- ValleyFloorWidth
- DepositionPotential
- SedimentSupply
- SlopeDrop
- UpstreamSteepMemory
- UpstreamMountainMemory

### 河漫滩

河漫滩主要由：

```text
大河 / 高汇流量
+ ValleyWeight
+ 低坡度
+ 低 Relief
+ 湿润 / 常流倾向
→ FloodplainWeight
```

湖泊区域会抑制河漫滩判定，避免整片湖面被当成河谷平地。

### 冲积平原

冲积平原要求更大尺度的低坡、低 Relief 平地：

```text
Rolling Plain
+ Floodplain
+ Wetland
+ 高汇流量
+ 极低坡度
+ 低 Relief
→ AlluvialPlainWeight
```

它不是“河边一格地形”，而是大河长期作用下的区域低地倾向。

### 冲积扇

冲积扇不能只看“山脚”或者随机画扇形。

第一版采用：

```text
上游陡峭河段
+ 上游山地记忆
+ 明显坡度突然变缓
+ 有持续河流
+ 上游侵蚀物源
→ AlluvialFanCore
→ 沿下游方向向两侧展开
→ AlluvialFanWeight
```

为了识别“刚从山里出来”的河流，沿 FlowTo 向下游传播：

- UpstreamSteepMemory
- UpstreamMountainMemory

并逐步衰减。

因此山口下游即使已经连续经过几格缓坡，也不会立刻忘掉上游山区背景。

冲积扇展开方向来自真实下游方向：

- 前方范围逐渐扩宽。
- 横向按距离衰减。
- 只在低坡、较低 Relief 的可沉积地形保留。
- 湖区显著抑制。

### 峡谷

峡谷倾向来自：

```text
强河流
+ 高汇流量
+ 高 Relief
+ 较陡坡度
+ 已存在 ValleyIncision
→ CanyonWeight
```

它不是独立随机生成的沟。

只有已经存在真实河网和河谷切割条件的地方，才可能继续形成峡谷。

### 谷底宽度

ValleyFloorWidth 主要由：

- ValleyWeight
- 汇流量
- FloodplainWeight
- 低坡度

共同增宽。

CanyonWeight 会显著压窄谷底。

因此：

```text
平缓大河 → 宽谷底
陡峭峡谷 → 窄谷底
```

### 沉积潜力

DepositionPotential 来自：

- FloodplainWeight
- AlluvialPlainWeight
- AlluvialFanWeight
- SedimentSupply

它只表示“适合沉积多少”的连续倾向，不代表已经模拟真实沉积厚度或地质年代。

### 接入最终地形

Phase 4.4 字段已经接入 `internal/terrain`。

最终 Regional Terrain 现在包括：

```text
Macro / Eroded Elevation
+ Landform-controlled Regional Detail
+ small Deposition Lift
- Valley Carve
- Canyon Extra Carve
→ H(worldX, worldZ)
```

具体效果：

- Floodplain / Alluvial Plain：抑制局部粗糙度，使低地更平缓。
- ValleyFloorWidth：宽谷底进一步降低细节振幅。
- Alluvial Fan：轻微堆积抬升，并适度变平缓。
- Canyon：在已有 Valley Carve 基础上额外下切。
- DepositionPotential：只产生很小的填平 / 堆积，不重写宏观高程。

新调试图：

- `prototype/generated/terrain/deposition-lift.png`
- `prototype/generated/terrain/canyon-carve.png`

### 正式尺寸多 Seed 审计

640 × 400，4 个固定验收 Seed：

```text
Floodplain >= 0.35
0.175% .. 0.311% land

Alluvial Plain >= 0.30
1.141% .. 1.923% land

Alluvial Fan >= 0.20
0.000% .. 0.062% land

Canyon >= 0.30
0.000% .. 0.013% land

Wide Valley Floor >= 0.45
0.146% .. 0.280% land

Deposition >= 0.20
0.098% .. 0.150% land
```

最大权重：

```text
Floodplain       0.780 .. 0.824
Alluvial Plain   0.892 .. 0.918
Alluvial Fan     0.187 .. 0.359
Canyon           0.286 .. 0.514
Valley Floor     0.727 .. 0.784
Deposition       0.302 .. 0.318
```

重要结论：

> 不要求每个 Seed 都必须出现强冲积扇或强峡谷。

这些属于特殊局部地貌。

允许某些世界：

- 没有明显大峡谷。
- 没有强山前冲积扇。

但固定多 Seed 验收批次中，相关机制不能整体消失。

### 自动护栏

Geomorph Audit 会直接失败，如果：

- 河漫滩大面积泛滥。
- 冲积平原覆盖异常过大。
- 宽谷底或沉积区异常铺满大陆。
- 固定多 Seed 批次中冲积扇机制彻底消失。
- 固定多 Seed 批次中峡谷机制彻底消失。
- 任一标准化字段超出 0..1。

### Phase 4 最终连续性复验

接入 Phase 4.4 后，正式 640 × 400 Terrain Audit 仍保持：

```text
Chunk Boundary epsilon jump
≈ 1.3e-7 .. 1.5e-7

Macro Boundary slope mismatch
≈ 2.3e-6 .. 4.4e-6
```

说明新增沉积 / 峡谷字段没有破坏：

- Minecraft Chunk 连续性。
- Macro Grid 边界平滑性。
- 世界坐标确定性。

## Phase 4 结论

Phase 4 第一版正式完成。

当前因果链已经形成：

```text
Tectonics
+ Climate
+ Hydrology
→ Erosion
→ Relief
→ Plain / Hill / Foothill / Mountain / Plateau
→ Valley / Floodplain / Alluvial Plain / Fan / Canyon
→ Continuous Regional Terrain
```

丘陵、河谷、峡谷和冲积地貌都不由独立随机噪声决定。

下一阶段进入：

**Phase 5 — 生态区与 Minecraft 映射**。
