# GeoWorld V1 Roadmap

## Phase 0 — 原型与模型定义

状态：**完成**

已完成：

- 产品边界
- Generator / Runtime 双形态
- V1 世界拓扑
- Runtime 世界配置生命周期
- 页面原型
- 初始核心数据结构

## Phase 1 — 大陆与板块

状态：**进行中**

### Phase 1.1 — 最小构造链路

状态：**完成**

- Seed → 板块
- 板块运动向量
- 汇聚 / 张裂 / 转换边界
- 大陆 / 海洋地壳独立于板块
- 构造地貌初步分类
- 粗略高程
- PNG 可视化 Lab

### Phase 1.2 — 地壳年龄与构造影响带

状态：**完成**

- 海洋地壳年龄
- 海洋-海洋俯冲方向
- 海沟 / 岛弧两侧差异
- 有宽度的造山 / 裂谷 / 洋中脊影响带

### Phase 1.3 — 板块边界自然化

状态：**完成第一版**

- 去除明显 Voronoi 多边形感
- 更自然的边界形状

### Phase 1.4 — 地壳与构造参数深化

状态：**完成第一版**

- 大陆地壳厚度
- 板块相对运动强度
- 海洋地壳年龄
- Tectonic Intensity
- 动态构造带宽度 / 幅度

### Phase 1.5 — 基础高程场

状态：**当前**

输出：

- Continental Interior
- Continental Shelf
- Continental Slope
- Ocean Basin
- Collision Mountain Belt
- Plateau
- Trench
- Island Arc
- Mid-Ocean Ridge
- Rift Valley

### Phase 1 验收

- 大陆不是随机噪声斑块。
- 大陆和板块不等价。
- 主要山脉、海沟、裂谷有明确构造成因。
- 主大陆与外围海洋留有缓冲。
- 同 Seed / 同配置完全可复现。

## Phase 2 — 气候骨架

状态：未开始

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

状态：未开始

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

状态：未开始

输出：

- Climate Type
- Eco Region
- Minecraft Biome
- Surface Material
- Vegetation Profile

## Phase 5 — Generator MVP

状态：未开始

能力：

- 新建世界模型
- 参数配置
- 分阶段生成
- 图层查看
- 导出世界模型
- 导出 PNG / Heightmap
- Minecraft 世界预生成

## Phase 6 — Runtime MVP

状态：未开始

先选择一个 Loader 作为首个 Adapter，但架构不绑定 Loader。

目标：

- 创建世界时可选 GeoWorld
- 保存世界级 GeoWorld 配置
- 加载同一 World Model
- 按 Chunk 生成
- Infinite Storm Ocean
- Infinite Frozen Ocean
- GeoWorld API

## 暂不进入 V1

- 动态四季
- 区域实时天气
- 球形坐标
- 实时全球板块演化
- 大规模文明模拟
