# GeoWorld 真实地貌强化 — R6.2 Real-data Extractor

更新时间：2026-11-10

状态：**R6.2a 通用 DEM / bathymetry 提取器、R6.2b GeoTIFF elevation/bathymetry 适配均完成；R6.2c hydro / surface-water 专题适配待做。**

## 1. 边界

R6.2 是离线开发工具，不进入世界生成运行时。

目标：

    real-world raster
    → normalized metrics JSON
    → R6.3 reference envelope

禁止：

- 让 Minecraft Runtime 依赖 GIS 数据；
- 在 R6.2 内手填“正确目标值”；
- 自动修改 GeoWorld 生成参数；
- 把现实米制绝对尺度强制映射成 Minecraft blocks。

## 2. 当前命令

    go run ./cmd/geoworld-realdata-extract

当前支持：

### NASADEM HGT

- 原始 .hgt
- NASADEM 官方 zip 内的 .hgt
- big-endian signed Int16
- tile 文件名自动解析西南角经纬度
- -32768 作为 NoData

示例：

    go run ./cmd/geoworld-realdata-extract \
      --input NASADEM_HGT_n29e090.zip \
      --input NASADEM_HGT_n29e091.zip \
      --dataset NASADEM_HGT \
      --version 001 \
      --region REF-01-yarlung-zangbo-braided-valley \
      --sample-stride 4 \
      --relief-window-km 5 \
      --output out/ref01_nasadem.json

### Esri ASCII Grid

用于：

- GEBCO user-defined-area ASCII；
- 其他已转换为 geographic lon/lat ASCII raster 的 DEM / bathymetry。

支持 header：

    ncols
    nrows
    xllcorner / xllcenter
    yllcorner / yllcenter
    cellsize
    NODATA_value

示例：

    go run ./cmd/geoworld-realdata-extract \
      --input gebco_ref06.asc \
      --dataset GEBCO_2026 \
      --version 2026 \
      --region REF-06-cape-cod-barrier-coast \
      --format esri-ascii \
      --sample-stride 1 \
      --relief-window-km 2 \
      --output out/ref06_gebco.json

### GeoTIFF elevation / bathymetry

R6.2b 增加纯 Go GeoTIFF 读取：

- 不依赖 GDAL；
- 不依赖 Python / rasterio；
- 支持 strip / tile；
- 支持 None / LZW / Deflate / JPEG；
- 支持 8/16/32-bit integer 与 Float32/Float64；
- 当前只接受单波段 EPSG:4326；
- GeoTIFF NoData 通过 --nodata 显式传入，不猜产品默认值。

大影像不会整张展开到内存。

工具按像素行窗口分块读取；如果源文件带 overview，会把 stripe 高度限制到主影像 1% 以下，并校验返回 width / height，防止读取库偷偷切换 overview。

示例：

    go run ./cmd/geoworld-realdata-extract       --input hydrosheds_dem_tile.tif       --dataset HydroSHEDS_core       --version 1.1       --region REF-04-zambezi-barotse-floodplain       --format geotiff       --nodata <product-specific-value>       --sample-stride 4       --geotiff-stripe-rows 512       --output out/ref04_hydrosheds_dem.json

GEBCO user-defined-area GeoTIFF 也可以直接使用同一路径。

## 3. 输出 schema

核心输出：

    schemaVersion
    regionId
    datasetId
    datasetVersion
    inputFormat
    inputs[]
    validElevationCells
    validSlopeCells
    reliefWindowKm
    elevationMeters
    slopeDegrees
    localReliefMeters
    hypsometricIntegral

percentile 字段统一：

    p10
    p50
    p90
    p99
    min
    max

## 4. 指标定义

### Elevation

直接从有效 raster cell 提取。

默认 sample-stride=4，避免高分辨率 DEM 为 percentile 重复占用过多内存。

### Slope

使用中心差分：

    dz/dx
    dz/dy

经纬度栅格的水平距离按所在纬度换算到 metre，再输出 degree。

因此 slope 可直接和 GeoWorld 的 slope degree 做跨尺度比较。

### Local relief

不是逐像素滑窗，而是固定公里尺度的 block relief：

    max elevation - min elevation

默认窗口：

    5 km

这样：

- 计算量线性；
- 对 NASADEM 多 tile 可控；
- 不依赖 Minecraft block 映射。

### Hypsometric integral

定义：

    (mean elevation - min elevation)
    /
    (max elevation - min elevation)

用于比较同类地貌整体高程分布形状。

## 5. 多 tile

--input 可以重复。

同一命令中的多个 tile：

- elevation / slope / relief percentile 合并统计；
- 每个 tile 保留自己的 bounds / dimensions / valid sample count。

当前不会自动裁剪到 registry bbox。

因此当前建议下载本身已经覆盖参考区的 tile，或先使用数据源自己的 area subset 功能。

registry bbox crop 留到 R6.2c，和 HydroSHEDS / JRC 专题适配一起完成。

## 6. 当前未支持

R6.2a / R6.2b **不宣称完整 R6.2 已完成**。

已经支持：

- NASADEM HGT / HGT zip；
- Esri ASCII elevation / bathymetry；
- 单波段 EPSG:4326 GeoTIFF elevation / bathymetry；
- HydroSHEDS v1.1 DEM 类 GeoTIFF 的格式读取路径；
- GEBCO elevation/bathymetry GeoTIFF 的格式读取路径。

尚未支持：

- JRC Global Surface Water occurrence / seasonality 等专题语义；
- HydroSHEDS flow direction / accumulation 专题语义；
- HydroRIVERS vector network；
- GEBCO TID 质量分类；
- registry bbox 自动裁剪；
- river longitudinal metrics；
- active-water corridor metrics；
- coastline vector metrics。

这些属于 R6.2c。

## 7. HydroSHEDS 版本说明

截至 2026-11：

- HydroSHEDS v2 已开始正式发布；
- 当前 v2 首批核心数据覆盖美洲；
- 大部分全球 core product 仍以 v1.1 为全球一致版本。

因此 R6 registry 继续以：

    HydroSHEDS core v1.1

作为全球主口径。

美洲参考区可以在后续 R6.2c 中额外加入 v2 cross-check，但不能把 v2 美洲结果和 v1.1 亚洲/非洲直接混成一个全球 envelope。

## 8. R6.2a 验收

已经验证：

- HGT raw parse；
- HGT zip parse；
- ESRI ASCII parse；
- NoData；
- slope；
- local relief；
- hypsometric integral；
- CLI JSON end-to-end smoke。

结论：**R6.2a PASS。**

## 9. R6.2b 验收

GeoTIFF elevation / bathymetry 适配已验证：

- pure-Go reader；
- EPSG:4326 bounds；
- signed Int16 / Int32；
- UInt8 / UInt16 / UInt32；
- Float32 / Float64；
- pixel-window chunking；
- overview 防误切换；
- explicit NoData；
- full-resolution sample count 无重叠 / 无遗漏。

真实 GeoTIFF smoke：

    dimensions = 1437 × 1188
    expected pixels = 1,707,156
    extracted valid samples = 1,707,156

独立 GeoTIFF 解析器交叉检查：

    min ≈ -67.681
    max ≈ 2131.559

本工具：

    min = -67.680824
    max = 2131.558838

一致。

结论：**R6.2b PASS。**

下一步：

    R6.2c hydro / surface-water / registry-crop adapters
