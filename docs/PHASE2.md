# Phase 2 — 气候骨架

## 目标

在 Phase 1 的宏观高程之上建立长期气候背景。

当前因果链：

```text
Latitude
+ Macro Elevation
        ↓
Annual Mean Temperature
        ↓
Idealized Pressure Belts
        ↓
Prevailing Wind
        ↓
Sea Surface Temperature
        ↓
Wind-driven Ocean Current
        ↓
Regional Temperature Correction
        ↓
Precipitation
```

GeoWorld 在这一阶段仍然只生成长期气候，不模拟某一天的天气。

## Phase 2.1 — 纬度与基础温度

状态：**完成第一版**

### 纬度

V1 有限地理区使用显式纬度。

默认：

```text
North edge = +82°
Equator    =   0°
South edge = -82°
```

不使用 ±90°，避免在平面模型里强行引入数学极点。

外围 Infinite Frozen Ocean 可视为继续向极区延伸。

### 纬度基础温度

默认：

- EquatorTempC = 28°C
- PolarTempC = -18°C

纬度越高，基础年均温越低。

南北半球在同纬度绝对值下保持对称。

### 海拔降温

仅陆地应用：

- LapseRate = 6.5°C / km
- MaxLandElevation = 6000 m

当前宏观高程 `0.5` 作为海平面。

```text
land elevation
→ effective meters above sea level
→ lapse-rate cooling
```

海洋格子始终按海平面空气温度处理。

海沟 / 深海盆地的海底深度不会错误地让上方空气降温。

### 当前测试 Seed

Seed：731894205

```text
Latitude           ±82°
Annual temperature -37.84°C .. 27.95°C
Equator baseline    28°C
Polar baseline      -18°C
Lapse rate           6.5°C/km
```

### 自动测试

验证：

- 纬度从北到南连续。
- 南北同纬度温度对称。
- 同纬度高地比低地冷。
- 海底深度不会参与空气垂直递减。
- 高纬海平面区域比赤道冷。

## Phase 2.2 — 理想化气压与盛行风

状态：**完成第一版**

### 气压带

当前使用平滑压力带近似：

- 0°：Equatorial Low
- ~30°：Subtropical High
- ~60°：Subpolar Low
- 极区：Polar High

用连续 Gaussian band 叠加，不使用硬纬线分段。

### 盛行风

当前三套理想化环流：

#### 0° ～ 30°

Trade Winds / 信风：

- 东风成分
- 向赤道成分

#### 30° ～ 60°

Westerlies / 西风带：

- 西 → 东
- 向极成分

#### 60° ～ 极区

Polar Easterlies / 极地东风：

- 东 → 西
- 向赤道成分

30° 和 60° 附近使用 Smoothstep 混合，避免风向突然跳变。

赤道附近南北风分量逐渐衰减为 0，避免在 0° 产生方向断层。

### 当前测试 Seed

```text
Pressure Index  0.223 .. 0.792
Wind Magnitude  0.048 .. 0.956
```

### 自动测试

验证：

- 赤道低压存在。
- 副热带高压存在。
- 副极地低压存在。
- 极区高压存在。
- 南北半球风场镜像对称。
- 信风 / 西风带 / 极地东风方向正确。
- 30° / 60° 风带过渡连续。

## 当前可视化

`prototype/climate.html`

当前输出：

- `latitude.png`
- `temperature-latitude.png`
- `land-elevation.png`
- `temperature-annual.png`
- `pressure.png`
- `wind.png`

## Phase 2.3 — 海表温度与表层洋流

状态：**完成第一版**

### SST 初始场

海洋表层温度以纬度基础温度为起点，并限制在：

- 最低约 -1.8°C（近似海水冻结点）
- 最高 30°C

陆地没有 SST。

### 风驱表层洋流

第一版：

```text
Prevailing Wind
        +
Coriolis Deflection
        +
Coast Blocking
        ↓
Surface Ocean Current
```

默认 Coriolis 表层偏转角约 25°：

- 北半球向运动方向右侧偏。
- 南半球向运动方向左侧偏。
- 赤道附近逐渐弱化。

海岸附近会移除指向陆地的洋流分量，保留沿岸切向分量。

### SST 热量输送

使用有限次稳定迭代：

- 上游平流
- 邻域扩散
- 向纬度背景 SST 弱松弛

不是 Navier-Stokes，也不是完整海洋环流模式。

当前目的是让暖水 / 冷水可以沿表层流场搬运，同时保证数值稳定和确定性。

### 当前测试 Seed

```text
SST              -1.80°C .. 29.14°C
Current Magnitude 0.001 .. 0.593
```

### 自动测试

验证：

- 陆地 Current = 0。
- 南北半球 Coriolis 偏转镜像。
- 洋流近岸不继续指向陆地。
- SST 始终在允许范围内。
- 洋流热量输送确实使 SST 偏离纯纬度背景。

## Phase 2.4 — 洋流对陆地温度修正

状态：**完成第一版**

不是把陆地温度直接向 SST 混合，而是先计算：

```text
SST
-
same-latitude baseline SST
=
Ocean Thermal Anomaly
```

再将这个异常向最近陆地传播。

影响：

- 暖流异常 → 沿岸增温。
- 寒流异常 → 沿岸降温。
- 影响随离海距离指数衰减。
- 原本的海拔降温结构保留。

默认：

- CoastalCurrentInfluence = 0.75
- CoastalTempDecayCells = 18

当前测试 Seed：

```text
Ocean temperature correction  -0.391°C .. +0.289°C
```

### 自动测试

验证：

- 暖流影响随离海距离递减。
- 寒流产生负温度修正。
- 洋流修正不会抹掉原有高地 / 低地温差。
- 修正后温度无 NaN / Inf。

## 下一步 — Phase 2.5

长期降水骨架：

- 海洋蒸发水汽源。
- 盛行风输送水汽。
- 低压区上升增雨。
- 山地迎风坡地形雨。
- 背风侧雨影。
- 初步年降水量。
