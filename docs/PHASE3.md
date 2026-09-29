# Phase 3 — 季风与水文

## 目标

Phase 2 已经给出长期平均气候：

- 年平均温度
- 盛行风
- 洋流
- 年平均水汽
- 年降水

Phase 3 进一步回答：

> 降水是在一年中均匀出现，还是存在明显雨季 / 旱季？

并在此基础上进入流域、河流、湖泊与湿地。

GeoWorld 仍然不实现游戏内实时四季。

## Phase 3.1 — 暖季 / 冷季代表态与季风指数

状态：**完成第一版**

### 代表态，而不是动态季节

内部字段目前命名为：

- SummerTempC
- WinterTempC
- SummerWind
- WinterWind

语义上实际代表：

- **当地暖季**
- **当地冷季**

因此南半球不会被强行当作和北半球同月份进入“夏季”。

未来如果暴露公共 API，可改用更中性的：

- WarmSeason
- ColdSeason

### 季节温差

海洋季节温差较小：

```text
Ocean Seasonality
= base
+ latitude term
```

陆地季节温差更大，并随大陆性增强：

```text
Land Seasonality
= latitude term
× continentality
```

大陆性由离最近海洋的距离近似。

当前默认：

- LandSeasonalityBaseC = 3.5
- LandSeasonalityLatitudeC = 15
- OceanSeasonalityBaseC = 1.5
- OceanSeasonalityLatitudeC = 5

### 季风指数

第一版季风指数来源：

```text
Land–Sea Seasonal Thermal Contrast
+
Low / Mid Latitude Weight
+
Ocean Accessibility
→ Monsoon Index
```

限制：

- 海洋 MonsoonIndex = 0。
- 赤道附近逐渐减弱。
- 高纬度逐渐减弱。
- 离海过远时海洋可达性下降。

当前 Seed 731894205：

```text
Seasonal amplitude  1.52 .. 20.90°C
Monsoon index        avg 0.516 / max 0.865
Positive cells       60637
```

平均值仅统计 MonsoonIndex > 0 的 Cell。

### 暖季 / 冷季季风风场

季风异常不是替换全球盛行风，而是叠加：

```text
Prevailing Wind
+
Monsoon Anomaly
→ Seasonal Wind
```

利用 OceanDistance 的梯度获得“从海向大陆内部”的方向。

暖季：

```text
land hotter
→ lower pressure tendency
→ onshore anomaly
```

冷季：

```text
land colder
→ higher pressure tendency
→ offshore anomaly
```

### 自动测试

验证：

- 同纬度陆地季节温差 > 海洋。
- 暖季温度 > 冷季温度。
- 季风主要出现在低中纬陆地。
- 高纬季风指数受抑制。
- 海洋 MonsoonIndex = 0。
- 暖季季风向陆。
- 冷季季风离陆。

### 当前可视化

`prototype/monsoon.html`

图层：

- `seasonal-amplitude.png`
- `summer-temperature.png`
- `winter-temperature.png`
- `monsoon-index.png`
- `summer-monsoon-wind.png`
- `winter-monsoon-wind.png`

## 下一步 — Phase 3.2

分别使用暖季与冷季风场计算两套水汽输送：

```text
Warm-season Wind → Warm-season Moisture → Warm-season Precipitation
Cold-season Wind → Cold-season Moisture → Cold-season Precipitation
```

然后得到：

- WarmSeasonPrecipitation
- ColdSeasonPrecipitation
- RainfallSeasonality
- MonsoonRainRatio

这些仍是长期气候数据，不改变 Minecraft Biome 本身。
