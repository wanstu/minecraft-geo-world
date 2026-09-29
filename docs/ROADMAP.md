# GeoWorld V1 Roadmap

## Phase 0 — 原型与模型定义

当前阶段。

目标：

- 确定产品边界
- 确定 Generator / Runtime 双形态
- 确定 V1 世界拓扑
- 页面原型
- 定义核心数据结构

交付：

- docs
- HTML Prototype

## Phase 1 — 大陆与板块

只验证宏观地理。

输入：

- Seed
- 地理区域尺寸
- 板块数量
- 海陆比例

输出图层：

- Plate ID
- Plate Type
- Plate Velocity
- Plate Boundary
- Elevation
- Ocean Basin

验收重点：

- 大陆不会只是随机噪声斑块
- 主要山系与板块边界有因果关系
- 主大陆与外围海洋留出合理缓冲

## Phase 2 — 气候骨架

输入：

- 纬度
- 海拔
- 海陆分布
- 盛行风

输出：

- Temperature
- Pressure / Wind
- Ocean Current
- Precipitation

先追求大尺度正确，不追求真实气象 CFD。

## Phase 3 — 季风与水文

加入：

- 夏 / 冬代表态
- 陆海热力差
- 季风指数
- 地形雨
- 雨影
- 流域
- 河流
- 湖泊

## Phase 4 — 生态与 Minecraft 映射

输出：

- Climate Type
- Eco Region
- Minecraft Biome
- Surface Material
- Vegetation Profile

## Phase 5 — Generator MVP

能力：

- 新建世界模型
- 参数配置
- 分阶段生成
- 图层查看
- 导出世界模型
- 导出 PNG / Heightmap
- 基础 Minecraft 预生成

## Phase 6 — Runtime MVP

先选择一个 Loader 作为首个 Adapter。

目标：

- 创建世界时可选 GeoWorld
- 加载同一世界模型
- 按 Chunk 生成
- Infinite Storm Ocean
- Infinite Frozen Ocean
- 世界数据持久化
- GeoWorld API

## 暂不进入 V1

- 动态四季
- 区域实时天气
- 球形坐标
- 实时全球板块演化
- 大规模文明模拟
