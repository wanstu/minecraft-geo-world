# GeoWorld 真实地貌强化 — R6.3a REF-04 空间参考区间

日期：2026-11-10  
状态：**R6.3a REF-04 真实水文参考区间候选完成；R6.3 整体尚未完成。**  
阶段限制：仅离线真实数据统计，**不改 GeoWorld 世界生成算法，不调参，不把绝对公里数等同 Minecraft 方块数。**

## 1. 为什么使用空间参考区间

R6.2c 的百分位主要针对单个像素或全参考区的一次汇总，这不是具有空间变异性的 *reference envelope*。本阶段把 REF-04 按经纬度切分为 **8 列 × 10 行 = 80 个固定空间块**，每个块从原始数据分别抽取水体比例 / 河网密度，再对 80 个块的观测值计算 p10 / p50 / p90 / p99。

* 参考区：`REF-04-zambezi-barotse-floodplain`
* 锁定 bbox：`[22, -17.5, 24.8, -14]`（W/S/E/N）
* 每块约 `0.35° × 0.35°`（角度相同，**面积 km² 不完全相同**）
* JRC 两种栅格均按原始像素坐标每 32 个取 1 个；HydroSHEDS 两种栅格每 8 个取 1 个
* HydroRIVERS 读取完整 SHP/DBF，仅裁剪所需河段并保持**每块线段长度的总和与参考区线段总长守恒**

这是 **单个地区内部的空间变异性样本**，80 个相邻格子**并非 80 个独立地貌实例**，也不是全球真实地貌的总体分布。数据密集 ≠ 独立验证完成。

## 2. 已生成的真实数据

所有表格中的数值来自提交的 JSON，而不是手填期望值：

`docs/reference_envelopes/REF-04-zambezi-barotse-floodplain/`

| 数据层 | 指标（跨空间块） | p10 | p50 | p90 | 身份 |
| --- | --- | ---: | ---: | ---: | --- |
| JRC occurrence | `jrc_occurrence_positive_fraction` | 0 | 0.007490 | 0.110985 | 候选可比的比例代理 |
| JRC occurrence | `jrc_occurrence_ge90_fraction` | 0 | 0 | 0.003783 | 长期高频水体代理，不是官方 permanent 类别 |
| JRC seasonality | `jrc_2024_water_any_month_fraction` | 0 | 0.000517 | 0.018605 | **2024 单年度** |
| JRC seasonality | `jrc_2024_water_1to11_months_fraction` | 0 | 0.000258 | 0.005177 | **2024 单年度** |
| JRC seasonality | `jrc_2024_water_12_months_fraction` | 0 | 0 | 0.010702 | **2024 单年度** |
| HydroSHEDS DIR | `hydrosheds_d8_direction_entropy` | 0.905107 | 0.943977 | 0.972420 | 诊断，不作模型目标 |
| HydroSHEDS ACC | `hydrosheds_accumulation_p90_cells` | 15.09 | 30.15 | 120.32 | 单位为上游**像素数**，不作模型目标 |
| HydroRIVERS | `hydrorivers_vector_density_km_per_km2` | 0.197218 | 0.228433 | 0.255835 | km/km²，**区域诊断**，尚未尺度规范化 |

注意：两个水体系列使用不同时间窗口，**不可直接比较其大小并解释为季节变化或年度退化**。

## 3. 数据定义和版本约束

### JRC Global Surface Water v1.5

- Occurrence：1984–2024 年间检测到水体的观测频率，以百分比编码。每块 `value > 0` 的有效像素比例是**长期至少观测过水体的比例代理**；`value >= 90` 是自定义高频水体代理。
- Seasonality：当前 v1.5 的 2024 年单年度月份数（0–12），不能当作 1984–2024 多年平均，也不能直接对应多年 JRC occurrence。
- NoData：255；计入各块 `noDataSamples`，但不进入有效像素比例的分母。零值是有效类别而不是 NoData。
- 原始数据只用于科研 / 离线验收，不放入游戏运行时，不随 Git 传播。

参考：<https://data.fao.org/catalog/iso/f1b1d45a-e360-44bd-95a9-d628c468a7f6>（Occurrence）；<https://data.fao.org/catalog/iso/237c6087-2a11-41b6-af64-b3f79df1df8e>（Seasonality）。

### HydroSHEDS core v1.1

- DIR：D8 流向分布归一化熵只是对栅格流向多样性的观测，单块 0/8 方位编码缺失不应自动解释为河口。
- ACC：`hyd_af_acc_15s.tif` 是上游**累计栅格单元数**，不等于面积 km²。跨纬度比较要考虑栅格面积变化。此层目前只输出诊断结果。
- 参考：<https://data.hydrosheds.org/file/technical-documentation/HydroSHEDS_TechDoc_v1_4.pdf>。

### HydroRIVERS v1

- SHP 对线段按格子 bbox 精确裁剪；片段距离采用球面近似并作长度守恒。
- DBF 记录用于与主河网结构同步读取；整区上游跨边界的联系不能从单格片段恢复。
- 输出河网密度 = 落入块内的河网线段长度 / 该格经纬度边界对应的球面面积。
- 当前不将这一密度数值直接映射到 GeoWorld 的世界尺度。

## 4. 来源 / 可重复性

每个 JSON 都包含：

- `regionId`, `grid` 和 `cells[].bounds`
- `source.datasetId`, `source.datasetVersion`, `source.observationPeriod`
- `source.fileName` 和 `source.sha256`（HydroRIVERS 另外包含 DBF 的 `sidecarSha256`）
- raster 主影像尺寸 / 原始 EPSG:4326 bounds / 裁剪窗口
- `validSamples`, `noDataSamples`，以及每块实际有效样本数
- 每项指标的 `sampleBlocks`, `excludedBlocks`, p10/p50/p90/p99/min/max
- `qualityFlags`、单位、是否可进入模型比对

复算脚本：

```bash
bash scripts/reference_envelope_r63_ref04.sh
```

依赖：必须先合法取得并放置相应源数据至 `.tmp/realdata-r62c/`。脚本读取现场文件，不会自动下载或运行修改世界的程序。Go 可用 `GO_BIN` 指定。

**缺失事项：** 当前原始数据的下载 URL、下载日期/对应生产元数据没有全部登记在固定源清单；SHA256 能证明文件复现，但**不能单独证明源平台和产品年度**。在补齐溯源前全部候选区间保持 `candidate_not_calibrated` 或 `diagnostic_only_not_for_geoworld_comparison`。

## 5. 采样与格网敏感性检查

以下来自真实文件的额外运行，结果只保存在 `.tmp/realdata-r62c`：

| 抽样 | 水体出现代理 p50 | p90 |
| --- | ---: | ---: |
| JRC occurrence, 8×10, stride=32（基线） | 0.007490 | 0.110985 |
| JRC occurrence, 8×10, stride=16 | 0.00824 | 0.11976 |
| JRC occurrence, 4×5, stride=32 | 0.01201 | 0.11870 |
| JRC 2024 seasonality, 8×10, stride=32（基线） | 0.000517 | 0.018605 |
| JRC 2024 seasonality, 8×10, stride=16 | 0.00040 | 0.01856 |

结论：采样步长改变时 p90 相对接近，但 p50 在部分近零指标上波动明显；格网从 8×10 改为 4×5 也会改变分布。**不能把这些候选 p10/p90 直接升格为硬阈值。**后续需要固定相同的地貌分析尺度、必要时按实际河道走廊掩膜重抽样。

## 6. 不足和后续门槛

R6.3a 只覆盖 REF-04 已有的水文和水体数据。以下尚未完成：

- REF-04 真实高程 DEM / slope / local relief 的空间区间；
- 区分主河道和洪泛活动走廊的掩膜：当前 bbox 宽度远大于局部河道宽度，比例值会被大片干燥区域稀释；
- floodplain-width / channel-width、远岸起伏 / corridor 宽度、洪泛区非对称等真正形态学指标；
- 参考区另外六个 `REF-01/02/03/05/06/07` 的真实候选 envelope；
- 时空分辨率灵敏度和独立来源的交叉验证；
- GeoWorld 四个固定 Seed 对照（属于 R6.4），没有开始。

**R6.3a 结果**：可复算的 REF-04 真实水文空间参考候选输出 **PASS**；完整真实地貌 reference envelope **NOT YET**。  
**R6.4 准入**：只有候选指标经过来源、代表性、尺度、安全边界确认后才能按对应 archetype 对照四个固定 Seed。无稳定偏差不得调整生成算法。

## 7. 本阶段验收命令

- `go test -count=1 ./internal/realgeom ./cmd/geoworld-reference-envelope ./cmd/geoworld-realdata-extract`：R6.3 专项 Go 测试通过，包含矢量格网长度守恒和未知来源版本拒绝。
- `go vet ./internal/realgeom ./cmd/geoworld-reference-envelope`：通过。
- `go test -p 1 -run '^$' ./...`：整个 Go 仓库编译检查通过（此命令不运行测试用例）。
- `bash scripts/reference_envelope_r63_ref04.sh`：五个真实源数据均已成功提取并写出 80 个格子结果。
- `git diff --check`：无空白格式错误。
- 整仓实际执行所有测试的 `go test -p 1 ./...` 没有收到完整结果，**不宣称整仓运行测试通过**。

