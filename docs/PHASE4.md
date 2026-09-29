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

## 下一步 — Phase 4.2

从连续字段推导区域地貌权重：

- RollingPlainWeight
- HillWeight
- FoothillWeight
- MountainWeight
- PlateauWeight
- ValleyWeight

要求：

- 不使用硬分类墙。
- 权重连续混合。
- 丘陵由中等 Relief + 地形成熟度 + 山系距离 / 河谷切割共同形成。
- 山麓应自然连接平原和山地。
- 高原必须是高海拔 + 低 / 中 Relief，而不是“高就是山”。
