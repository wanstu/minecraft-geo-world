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

## Phase 3.2 — 季节性降水

状态：**完成第一版**

### 共用同一套水汽物理

Phase 2 的年平均降水求解器已抽成通用函数。

三套气候状态现在使用同一条链：

```text
Temperature
+ Wind
+ Ocean Surface Temperature
+ Pressure
+ Orographic Lift
→ Moisture Transport
→ Precipitation
```

分别输入：

- 年平均温度 / 盛行风
- 暖季温度 / 暖季风
- 冷季温度 / 冷季风

因此季节降水不是简单对年降水乘系数。

### 半年降水

暖季 / 冷季代表态各按约半年折算：

```text
WarmSeasonPrecipMM = warm potential × 0.5
ColdSeasonPrecipMM = cold potential × 0.5

SeasonalAnnualPrecipMM
= WarmSeasonPrecipMM
+ ColdSeasonPrecipMM
```

### 温度对水汽的季节耦合

完整的暖 / 冷季温差仍保存在气候字段中。

但降水水汽能力只吸收部分季节温度信号：

- SeasonalMoistureTempCoupling = 0.35
- SeasonalSeaSurfaceTempCoupling = 0.35

这样避免普通非季风区因为冷季温度较低就被错误算成近乎无雨。

### 长期背景水汽

季节水汽不是每半年从 0 重置。

默认：

- SeasonalBackgroundMoistureBlend = 0.20

即季节求解后保留一部分 Phase 2 年平均水汽背景。

作用：

- 防止强冬季离陆风把整个大陆水汽数值清空。
- 仍允许强季风区形成明显雨季 / 旱季。
- 保持季节年降水与 Phase 2 年降水处于同量级。

### 输出字段

新增：

- WarmSeasonPrecipMM
- ColdSeasonPrecipMM
- SeasonalAnnualPrecipMM
- RainfallSeasonality
- MonsoonRainRatio

其中：

```text
RainfallSeasonality
= |warm - cold| / (warm + cold)
```

而 MonsoonRainRatio 只在：

- 陆地
- 暖季降水 > 冷季降水

时，根据暖季降水盈余与 MonsoonIndex 计算。

### 12 Seed 审计

240 × 150 宏观网格：

```text
Seasonal annual / Phase2 annual   0.930 .. 1.164
Rainfall seasonality avg          0.288 .. 0.677
Monsoon rain ratio avg            0.156 .. 0.343
Weak monsoon warm/cold ratio      1.265 .. 1.924
Strong monsoon warm/cold ratio    1.683 .. 28.185
Strong monsoon land area          11.2% .. 23.6%
```

原始版本曾出现强季风区暖 / 冷季比上百到上万倍。

加入长期背景水汽后，数值塌缩已经消失。

### 自动护栏

多 Seed 自动测试要求：

- SeasonalAnnual / Phase2 Annual 保持 0.60 ～ 1.60。
- 平均 RainfallSeasonality 保持 0.10 ～ 0.85。
- RainfallSeasonality = 1 的陆地不得超过 10%。
- 弱季风暖 / 冷季平均比不得超过 3。
- 强季风暖 / 冷季平均比不得超过 50。

### 当前可视化

`prototype/monsoon.html`

新增：

- `warm-season-precip.png`
- `cold-season-precip.png`
- `rainfall-seasonality.png`
- `monsoon-rain-ratio.png`

## 下一步 — Phase 3.3

流域与河网：

- 对陆地高程做水文填洼 / 出口处理。
- 计算 Flow Direction。
- 计算 Flow Accumulation。
- 结合年降水与季节性估计 Runoff。
- 提取主河流 / 支流。
- 识别内流盆地、湖泊与湿地倾向。

这些仍属于宏观世界模型，不直接在这一阶段生成 Minecraft 方块河道。
