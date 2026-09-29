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

## Phase 1 — 大陆、板块与宏观高程

状态：**完成**

### Phase 1.1 — 最小构造链路
状态：**完成**

### Phase 1.2 — 地壳年龄与构造影响带
状态：**完成**

### Phase 1.3 — 板块边界自然化
状态：**完成第一版**

### Phase 1.4 — 地壳与构造参数深化
状态：**完成第一版**

### Phase 1.5 — 基础高程场
状态：**完成第一版**

### Phase 1.6 — Phase 1 验收
状态：**完成**

验收重点：

- 大陆不是随机噪声斑块。
- 大陆和板块不等价。
- 山系、海沟、裂谷有明确构造成因。
- 主大陆与外围海洋留有缓冲。
- 同 Seed / 同配置完全可复现。
- 多 Seed 结果具有差异，但不出现系统性异常。
- 宏观高程可以作为后续气候模型稳定输入。

## Phase 2 — 气候骨架

状态：**完成第一版**

### Phase 2.1 — 纬度与基础温度
状态：**完成第一版**

### Phase 2.2 — 理想化气压与盛行风
状态：**完成第一版**

### Phase 2.3 — 海表温度与表层洋流
状态：**完成第一版**

### Phase 2.4 — 海洋对陆地温度修正
状态：**完成第一版**

### Phase 2.5 — 长期降水骨架
状态：**完成第一版**

输入：

- 纬度
- 宏观海拔
- 海陆分布
- 海洋距离
- 大尺度日照差异

主要输出：

- Temperature
- Pressure
- Prevailing Wind
- Sea Surface Temperature
- Ocean Current
- Precipitation

目标是大尺度合理，不做完整气象 CFD。

## Phase 3 — 季风与水文

状态：**完成第一版**

### Phase 3.1 — 暖季 / 冷季代表态与季风指数
状态：**完成第一版**

### Phase 3.2 — 季节性降水
状态：**完成第一版**

### Phase 3.3 — 流域与河网
状态：**完成第一版**

### Phase 3.4 — 河网等级、湖泊连通与验收
状态：**完成第一版**

加入：

- 夏 / 冬代表态
- 陆海热力差
- 季风指数
- 地形雨
- 雨影
- 流域
- 河流
- 湖泊
- 湿地倾向

输出稳定的长期水文骨架。

## Phase 4 — 侵蚀、Relief 与地貌细化

状态：**进行中**

### Phase 4.1 — 侵蚀驱动力与河谷切割
状态：**完成第一版**

### Phase 4.2 — 丘陵 / 山麓 / 平原连续权重
状态：**当前**

这一阶段才正式形成：

- 丘陵
- 山麓
- 河谷
- 峡谷
- 起伏平原
- 台地
- 冲积平原
- 山前冲积扇

核心字段：

- Elevation：整体海拔
- Relief：区域起伏强度
- Erosion：侵蚀结果

丘陵不由独立随机噪声决定，而由：

```text
构造背景
+ 距主山系距离
+ 河流切割
+ 降水
+ 地形年龄
+ 岩性 / 抗侵蚀近似
→ Relief
→ 丘陵 / 山麓 / 河谷等
```

## Phase 5 — 生态与 Minecraft 地貌映射

状态：未开始

输出：

- Climate Type
- Eco Region
- Minecraft Biome
- Surface Material
- Vegetation Profile
- Terrain Profile

Biome 表示长期生态，不直接代表瞬时天气。

## Phase 6 — Generator MVP

状态：未开始

能力：

- 新建世界模型
- 参数配置
- 分阶段生成
- 图层查看
- 多 Seed 对比
- 导出 World Model
- 导出 PNG / Heightmap
- Minecraft 世界预生成

## Phase 7 — Runtime MVP

状态：未开始

先选择一个 Loader 作为首个 Adapter，但架构不绑定 Loader。

目标：

- 创建世界时可选 GeoWorld
- 保存世界级 GeoWorld 配置
- 加载同一 World Model
- 按世界坐标连续采样地形
- 按 Chunk 放置方块
- Infinite Storm Ocean
- Infinite Frozen Ocean
- GeoWorld API

## 最终地形连续性原则

最终 Minecraft 地形不以 Chunk 为生成数学边界。

```text
height = H(worldX, worldZ)
```

Chunk 只负责查询该函数覆盖的 16×16 世界坐标。

地形建议分为：

1. Macro Terrain：大陆、山系、海盆。
2. Regional Terrain：丘陵、山麓、支脉、河谷。
3. Local Terrain：小尺度粗糙度、岩壁、沟壑。

不同地貌之间使用连续权重混合，不允许硬切换。

## 暂不进入 V1

- 动态四季
- 区域实时天气
- 球形坐标
- 实时全球板块演化
- 大规模文明模拟
