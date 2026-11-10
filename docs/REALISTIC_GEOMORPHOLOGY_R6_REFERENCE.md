# GeoWorld 真实地貌强化 — R6 真实参考体系

更新时间：2026-11-10

状态：**R6.1 和 R6.2 已完成；R6.3a REF-04 水文参考空间区间候选已生成，其他区域及完整地貌指标尚未完成。仍未修改生成算法。**

## 1. 目标

R6 不再只问“GeoWorld 自己是否连续”，而开始问：

> GeoWorld 的地貌统计结构，是否落在真实世界同类地貌的合理范围内？

R6 不是复刻某一张卫星图，也不是让所有世界趋向同一种地貌。

固定流程：

1. 选定真实 archetype；
2. 锁定公开数据源与版本；
3. 从真实数据提取统计区间；
4. 用同一指标测 GeoWorld 固定 Seed；
5. 只有稳定、可解释的偏差才进入算法修复。

## 2. 数据源锁定

### 2.1 陆地高程

Primary:

- NASADEM Merged DEM Global 1 arc-second V001

Cross-check:

- MERIT DEM v1.0.3，3 arc-second

用途：

- slope；
- local relief；
- hypsometry；
- valley / basin / fan profile。

不在仓库中重新分发原始 DEM。

### 2.2 河网 / 水文

Primary:

- HydroSHEDS core v1.1
- HydroRIVERS v1

Cross-check:

- MERIT Hydro v1.0.1

用途：

- flow direction；
- upstream area；
- river-network topology；
- longitudinal gradient；
- drainage density。

### 2.3 水体 / 洪泛活动范围

- JRC Global Surface Water v1.5
- occurrence aggregate period: 1984–2024
- seasonality (v1.5): 2024 单年度；不得与 occurrence 的多年度统计混为同一时间范围

用途：

- permanent / seasonal water；
- braided / multi-channel active corridor；
- floodplain water occurrence；
- lake / delta / estuary extent。

### 2.4 海底

- GEBCO_2026 Grid
- 15 arc-second

用途：

- nearshore depth；
- continental shelf；
- continental slope；
- deep-ocean transition。

GEBCO TID 需要保留，用于判断统计值是直接测量还是预测 / 插值主导。

### 2.5 卫星图

NASA Earth Observatory、Landsat / Sentinel 等只作为：

- archetype 选择；
- 形态解释；
- 人工 sanity check。

**卫星图颜色和纹理不作为数值目标。**

## 3. 固定真实参考区

坐标仅定义 R6 的统计窗口，不代表行政边界或精确地貌边界。

### REF-01 Yarlung Zangbo / Tibetan braided valley

目的：

- 高原 / 山地河谷；
- 高沉积负荷；
- braided active valley floor；
- 受限河谷与宽谷的切换。

主要指标：

- longitudinal gradient ratio；
- valley-width / channel-width ratio；
- local relief；
- active-water corridor width；
- downstream channel-width trend。

### REF-02 Tarim–Kunlun basin margin

目的：

- 大型盆地；
- 山前冲洪积扇；
- basin rim → fan → basin floor。

主要指标：

- basin rim/floor relief ratio；
- basin floor slope；
- fan radial slope decay；
- fan width/length ratio；
- fan overlap / bajada tendency。

### REF-03 Southern Mongolia alluvial-fan field

目的：

- 狭窄山谷出山后 competence loss；
- braided fan channels；
- fan lateral spread。

主要指标：

- apex / toe slope ratio；
- radial relief decay；
- channel spreading angle；
- active fan footprint ratio。

### REF-04 Zambezi / Barotse-type floodplain

目的：

- 宽阔自然洪泛平原；
- seasonal active corridor；
- 主河道与 floodplain 尺度关系。

主要指标：

- floodplain-width / channel-width ratio；
- far-relief / corridor-width ratio；
- seasonal-water extent ratio；
- floodplain asymmetry。

### REF-05 Mississippi deltaic lowland

目的：

- 低坡度沉积平原；
- distributary / wetland / delta；
- 陆地—浅海连续沉积。

主要指标：

- lowland slope distribution；
- distributary density；
- water / land patch scale；
- shoreline complexity；
- nearshore depth gradient。

### REF-06 Cape Cod barrier coast

目的：

- 砂质障壁海岸；
- spit / barrier / back-barrier；
- 缓坡 shoreface。

主要指标：

- shore slope；
- barrier width / length ratio；
- back-barrier water ratio；
- nearshore / shelf slope ratio。

### REF-07 Banks Peninsula / volcanic steep coast

目的：

- 陡峭基岩 / 火山海岸；
- 海岸凹湾与高 relief 直接接海。

主要指标：

- inland relief；
- shore slope p90/p99；
- coastline curvature；
- nearshore depth gradient。

## 4. 禁止做全局绝对尺度映射

R6 不规定：

    1 Minecraft block = N metres

原因：

- GeoWorld 的世界尺度本来就是压缩的；
- Minecraft 垂直 / 水平视觉尺度也不是现实 1:1；
- 直接对绝对公里宽度会迫使所有地貌变成超大尺度。

优先比较 dimensionless / scale-safe 指标。

## 5. 核心对照指标

### 5.1 通用地形

- slope p50 / p90 / p99；
- local relief / window size；
- hypsometric integral；
- curvature distribution；
- relief anisotropy。

### 5.2 河流

- upper / middle / lower drop-per-distance ratio；
- channel-width growth ratio；
- valley-width / channel-width ratio；
- floodplain-width / channel-width ratio；
- drainage density；
- confluence width ratio；
- mouth approach gradient。

### 5.3 盆地 / 冲积扇

- rim-to-floor relief / basin radius；
- wall slope / floor slope ratio；
- floor slope p90；
- fan apex / toe slope ratio；
- fan width / length ratio；
- fan deposition footprint / channel width。

### 5.4 海岸 / 海底

- shore slope distribution；
- shelf slope / continental-slope ratio；
- continental-slope / deep-ocean slope ratio；
- nearshore / shelf / slope-foot depth ratios；
- coastline roughness at multiple scales。

## 6. 真实参考区间如何产生

每个 reference region 不存一个“正确值”，而存：

- sample count；
- p10；
- p50；
- p90；
- p99（必要时）；
- 数据源 / 版本；
- extraction resolution；
- quality flags。

正式比较使用 reference envelope。

例如：

    real slope p10–p90
    geoworld slope p10–p90

而不是：

    real slope = 8.35°
    geoworld 必须 = 8.35°

## 7. GeoWorld 对照规则

每个真实 archetype 只和对应的 GeoWorld archetype 比较。

禁止：

- 拿高山峡谷要求所有河流都陡；
- 拿 Zambezi 洪泛平原要求所有下游都宽；
- 拿 Cape Cod 要求所有海岸都有障壁岛；
- 拿 Banks Peninsula 要求所有海岸都成悬崖；
- 把不同真实 archetype 平均成一个“全球最佳参数”。

只有以下情况才进入生成算法修改：

1. 4 个固定 Seed 同类地貌稳定偏出 reference envelope；
2. 偏差方向一致；
3. 不是比例尺映射造成；
4. 不是不同 archetype 混淆造成；
5. 最终 Runtime / Minecraft 几何也能观察到。

## 8. R6 分阶段

### R6.1 Reference registry

状态：**本文件 + JSON registry 完成。**

### R6.2 Real-data extractor

建立离线工具：

    real DEM / hydro / water / bathymetry
    → normalized reference metrics JSON

工具不能成为世界生成运行时依赖。

### R6.3 Reference envelopes

固定各 archetype 的真实 p10/p50/p90 区间。**R6.3a 已完成 REF-04 的水文空间参考候选和质量标记，其他参考区、真实坡度/起伏、实际洪泛区形态仍待补齐。**

详见 [R6.3a REF-04 真实空间参考区间](REALISTIC_GEOMORPHOLOGY_R6_ENVELOPES.md)。

### R6.4 GeoWorld comparison

4 个固定 Seed 对应 archetype 与 reference envelope 对比。

### R6.5 Defect decision

只对稳定偏差开算法修改。

## 9. R6.1 验收

R6.1 完成条件：

- 数据源版本已固定；
- reference archetype 已固定；
- 指标已定义；
- 禁止全局米/方块映射；
- 禁止未算真实统计就手填 target；
- 不修改 generation rules。

结论：**R6.1 PASS。**

下一步：R6.2 real-data extractor。
