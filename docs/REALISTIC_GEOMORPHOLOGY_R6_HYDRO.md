# GeoWorld 真实地貌强化 — R6.2c 水文专题提取验收

日期：2026-11-10  
工作树：`adm/realistic-geomorphology`  
基线：`e68119d`  
状态：**R6.2c 四类适配与真实数据专项验收通过；尚未开始 R6.3 / 未修改世界生成算法。**

## 1. 已实现

1. **Registry bbox**：从 `docs/realistic_geomorphology_reference_registry.json` 按 `regionId` 读取 W/S/E/N、校验 EPSG:4326，在地理栅格的**原始像素中心**实施 bbox 裁剪，拒绝未知区域与无重叠。
2. **JRC Global Surface Water v1.5**：occurrence（0–100%）和 seasonality（0–12 月），255 为 NoData。分别计算有效采样数、出现水体数、季节/常年分类比例、百分位。occurrence ≥90% 是显式注明的长年水体**代理定义**，不是官方永久水体矢量。
3. **HydroSHEDS core v1.1**：flow direction 的 ESRI D8 八方向频数（1/2/4/8/16/32/64/128，0 标记 outlet 或 sink）与 flow accumulation 上游累计像素数百分位。ACC 的 `4294967295` 为 NoData。**禁止将累计像素数直接解释为平方公里**。
4. **HydroRIVERS v1**：逐条读取 SHP + DBF，按参考区逐段裁剪折线，计算落在边界内的河网长度、河网密度、Strahler 级别分布、**仅包含区内节点**的汇流数量。不会把整条外伸河流的长度算到参考区内。
5. **GeoTIFF 兼容修复**：`third_party/gocog` 固定原依赖版本的本地小补丁，修复大 HydroSHEDS TIFF 延迟加载的 GeoKey、PixelScale、TiePoint，新增 ZSTD (compression 50000) 解码并纠正 LZW MSB/LSB fallback。离线专用，不进入 Minecraft 游戏运行时。

本阶段不依赖 GDAL、Python GIS 包；除用于开发验收的原始数据外，所有处理代码为 Go。

## 2. CLI

统一入口仍为：

```bash
go run ./cmd/geoworld-realdata-extract \
  --layer jrc-occurrence \
  --input .tmp/realdata-r62c/occurrence.tif \
  --region REF-04-zambezi-barotse-floodplain \
  --version 1.5 \
  --sample-stride 64 \
  --output .tmp/realdata-r62c/jrc-occurrence-smoke.json
```

替换 `--layer` 和 `--input` 分别可得到：

| Layer | 输入 |
| --- | --- |
| `jrc-occurrence` | JRC occurrence.tif |
| `jrc-seasonality` | JRC seasonality.tif |
| `hydrosheds-direction` | HydroSHEDS hyd_af_dir_15s.tif |
| `hydrosheds-accumulation` | HydroSHEDS hyd_af_acc_15s.tif |
| `hydrorivers` | HydroRIVERS_v10_af.shp，要求同目录 .dbf |

HydroRIVERS 同样使用 `--region`，但不使用栅格 `--sample-stride`。层名独立指定专题语义，避免把流量、方向或者水体月份误当作 elevation/slope。

### 注意

- `--region` 对上述所有水文层启用 registry bbox 裁剪；**现有 elevation 模式未改变**，仍需先用提供方区域裁切功能制作匹配的 DEM 子集。
- `--sample-stride` 作用于全局原始像素坐标，分块窗口不会导致重复/遗漏。
- TIFF 解码器校验所需的输出宽高，发现 overview 降采样会拒绝而不是悄悄采纳。
- 每层只有一份输入影像或 SHP；多源合并需后续独立的明确合并规范。

## 3. 真实数据烟测 — REF-04

参考区：REF-04-zambezi-barotse-floodplain，bbox `[22, -17.5, 24.8, -14]`。

| 数据 | 抽样步长 | 实际结果 |
| --- | ---: | --- |
| JRC occurrence | 64 | 38,325 个有效像素；1,501 有水体，active fraction = 0.039165 |
| JRC seasonality | 32 | 153,300 个有效像素；722 有水体，active fraction = 0.004710 |
| HydroSHEDS DIR | 16 | 2,226 有效像素；均为合法 D8 编码 |
| HydroSHEDS ACC | 32 | 567 有效像素；p50=2、p90=59、p99≈11,751.32 个源像素 |
| HydroRIVERS | 全部河段 | 遍历 1,527,551 条记录，5,058 条相交；裁剪后河网长 26,530.315 km |

HydroRIVERS 参考区地理面积约 `116,603.137 km²`，河网密度约 `0.227527 km/km²`，区内下游节点汇合统计 `2,483`。本工具基于线段的球面近似距离，而非 GIS 软件投影后平面长度，输出标记了该定义。

本次 occurrence 和 seasonality 并非同一语义，步长亦不同，**不得把 3.92% 与 0.47% 当成两版同一指标对比**。也不能未经 R6.3 区间建模就称这些值为 GeoWorld 的正确目标值。

输出证据保存在工作树的 `.tmp/realdata-r62c/*-smoke.json`（原始数据与 smoke 不提交 Git）。所有数据范围、源分辨率、裁剪窗口和有效样本数量都写入 JSON，方便重算。

## 4. 质量边界

- JRC 零值不代表完全观测；需要按产品的观测/掩膜定义谨慎解释。occurrence 采样比例不是洪水实际宽度测量。
- HydroSHEDS v1.1 的流向 0 不区分海洋出口和内流区洼地，后续应结合 MSK；不能直接认定 0 是海口。
- HydroSHEDS ACC 为汇流累计**栅格单元数**，跨纬度推算绝对面积需要地理单元面积换算。
- HydroRIVERS 的局部 confluence 是 bbox 内连接数量下界，边界外上游连接未计入。
- 真实水文网络的纵剖面、河床坡降、支流汇合宽度、河道/洪泛区活动走廊的匹配仍需 R6.3/R6.4 结合 DEM 和 GeoWorld 样本实现。
- GEBCO TID 是独立的海底数据质量分类，不能混进当前水文层。后续进行海岸/水深参考时单独适配。

## 5. 验收

- `go test ./internal/realgeom ./cmd/geoworld-realdata-extract` 已通过。
- 单测涵盖 registry bbox、像素中心裁剪、直方图分位数、NoData 配置、D8 层识别、SHP/DBF 构造样例的精确线段裁剪。
- 四类专题（JRC 两项、HydroSHEDS 两项）及 HydroRIVERS 已各跑真实数据 smoke 并产出 JSON。
- `go test -p 1 -run '^$' ./...` 已通过（全部 Go 包编译检查；未执行测试用例）。
- `go vet ./internal/realgeom ./cmd/geoworld-realdata-extract` 已通过。
- 整仓 `go test ./...` 首次执行连接未返回结果，**整仓运行测试不计作通过**；编译级验收已单独完成。

## 6. 下一阶段

**R6.3：建立每个真实 archetype 的 reference envelope。** 优先固定 REF-04 的参考值和质量标记，随后其他 6 个参考区逐个按数据可用性补齐，避免先手填目标或者凭单张卫星图调参。R6.3 不应自动修改 GeoWorld 世界生成规则。
