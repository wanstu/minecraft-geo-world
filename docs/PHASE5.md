# GeoWorld Phase 5 — 生态与 Minecraft 映射

> Phase 5 不把“气候值”直接等同于 Minecraft Biome。
>
> 先建立长期生态，再在下一阶段映射到 Minecraft 群系、表面材料和植被。

## Phase 5.1 — 长期生态区分类

状态：**完成第一版**

实现：

- `internal/ecology`
- `cmd/geoworld-ecology-lab`
- `cmd/geoworld-ecology-audit`
- `prototype/ecology.html`

## 输入

生态模型读取已经完成的长期世界字段：

### 气候

- 年平均温度
- 暖季温度
- 冷季温度
- 年降水
- 暖季 / 冷季降水
- 降水季节性
- 季风指数
- 海表温度

### 地形

- 海拔
- Local Relief
- 平原 / 丘陵 / 山麓 / 山地 / 高原权重

### 水文与沉积地貌

- 湿地潜力
- 常年河 / 季节河倾向
- 河漫滩
- 冲积平原
- 谷底宽度

## 连续生态字段

Phase 5.1 不先做硬分类，而是先输出连续生态倾向：

- MoistureIndex
- AridityIndex
- WarmSeasonRainShare
- ColdSeasonRainShare
- ThermalSeasonality
- WetlandEcology
- AlpineEcology
- TreeCoverPotential
- GrassCoverPotential
- ShrubCoverPotential
- BareGroundPotential
- SnowCoverPotential
- SoilMoisture
- VegetationDensity

这些字段保持在 `0..1`。

### 水分不是只看降水量

同样的年降水，在炎热地区和寒冷地区不能解释成相同湿润度。

第一版使用一个简化的温度相关需水量代理：

```text
precipitation
+
temperature-dependent water demand
→ Moisture / Aridity
```

它不是完整的物理 PET 模型，但比“年降水直接分湿润 / 干旱”更合理。

## Eco Region

第一版有限类别：

### 海洋

- Warm Ocean
- Temperate Ocean
- Cold Ocean
- Frozen Ocean

### 寒冷 / 高山

- Ice Sheet
- Tundra
- Alpine

### 森林

- Boreal Forest
- Temperate Forest
- Subtropical Forest
- Tropical Monsoon Forest
- Tropical Rainforest

### 草原 / 灌丛 / 干旱

- Temperate Grassland
- Temperate Desert
- Mediterranean Shrubland
- Hot Desert
- Savanna

### 特殊水文生态

- Wetland

## 分类原则

### 1. 长期生态，不是实时天气

一次下雨、一次降雪或某一天的温度不会改变 Eco Region。

### 2. 分类前先连续响应

例如：

```text
Temperature
+ Moisture
+ Seasonality
+ Terrain
+ Hydrology
→ continuous ecology scores
→ dominant Eco Region
```

因此阈值附近不会把所有连续生态字段一起跳变。

### 3. 高山生态可以覆盖纬度规律

温暖纬度中的高海拔强山地仍可以形成：

- Alpine
- Snow / bare-ground tendency

而不是因为纬度低就强行成为森林。

### 4. 湿地是水文覆盖层

强湿地、河漫滩、宽谷底和常年河补水可以让一个气候上适合森林 / 草原的位置成为 Wetland。

## Region Confidence

Eco Region 仍然是有限类别，因此保存：

`RegionConfidence`

它不是概率，而是：

- 最佳类别本身有多强。
- 与第二名相差多少。

用途：

- 找分类边界。
- 找模型薄弱区。
- 后续 Minecraft Biome 映射时允许低置信度区域做更柔和的混合。

## Phase 5.1 期间发现并修复的气候标尺问题

生态分类第一次接入真实世界时发现：

```text
land average annual temperature ≈ -4.7°C
Ice Sheet ≈ 41% of land
```

但：

```text
|latitude| >= 60° 的陆地只有约 12%
```

说明问题不在生态分类，而在上游温度标尺。

最终定位到两个问题。

### A. 归一化高程不能线性当成米制海拔

旧逻辑：

```text
normalized land height
→ linear 0..6000m
```

会让普通大陆内部也被解释为很高的高原。

改为：

```text
normalizedHeight ^ ElevationCurvePower
× MaxLandElevationM
```

默认：

`ElevationCurvePower = 1.70`

效果：

- 海平面仍为 0m。
- 顶端仍可达到 6000m。
- 低 / 中海拔明显压低。
- 真正高山仍保留高海拔能力。

### B. 纬度温度曲线中纬度过冷

旧曲线使海平面附近：

```text
30° ≈ 8°C
45° ≈ -2°C
```

明显过冷。

改为参数化：

`LatitudeTempCurvePower = 3.50`

保留：

- 赤道约 28°C。
- 模拟极区边缘约 -18°C。

但中纬度下降更缓，进入高纬后再明显降温。

修正后测试 Seed 731894205：

```text
land average annual temperature ≈ 12.8°C
land < 0°C ≈ 17.8%
land > 20°C ≈ 38.1%
Ice Sheet ≈ 5.9%
Tundra ≈ 5.2%
```

## 正式 640 × 400 多 Seed 生态审计

4 Seed：

```text
Cold ecology       5% .. 17%
Forest ecology    27% .. 37%
Dry ecology       15% .. 27%
Grass / shrub     27% .. 40%
Tropical ecology  39% .. 46%
Low confidence   1.4% .. 2.1%
```

当前护栏：

- Cold ecology 不得占超过 40% 陆地。
- Dry ecology 不得占超过 50% 陆地。
- 固定验收 Seed 中森林不能整体消失。
- 草原 / 灌丛不能整体消失。
- 低置信度区域不得超过 10%。
- 平均分类置信度不得低于 0.45。
- Tree / BareGround 全局平均不得接近饱和。

这些护栏不是要求世界都像地球，而是防止模型退化成：

- 全世界冰原。
- 全世界荒漠。
- 没有森林。
- 没有草原。
- 大面积无法明确分类。

## 原型

`prototype/ecology.html`

展示：

- 长期生态区
- 水分 / 干旱
- 植被密度
- 树木 / 草地 / 灌丛
- 裸地
- 积雪
- 土壤湿润
- 湿地生态
- 高山生态

## 下一步 — Phase 5.2

将 Eco Region 与连续生态字段映射为 Minecraft World Generation Profile：

```text
Eco Region
+ Terrain
+ Soil Moisture
+ Snow
+ Floodplain / Wetland
+ local detail
→ Minecraft Biome
→ Surface Material
→ Vegetation Profile
```

重要原则：

> 一个 Eco Region 不等于一个 Minecraft Biome。

例如：

```text
Temperate Forest
+ rolling plain
→ forest / birch-like lowland profile

Temperate Forest
+ hills
→ wooded hills profile

Temperate Forest
+ floodplain
→ riparian / wet forest profile
```

Phase 5.2 仍然不负责实时天气或动态四季。
