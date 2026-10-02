# GeoWorld V1 Roadmap

2026-09-29 调整：Phase 1–5 为算法第一版；跨阶段一致性和游戏验收单独列入 Phase 6。详见 [PHASE6.md](PHASE6.md)。

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

状态：**完成第一版**

### Phase 4.1 — 侵蚀驱动力与河谷切割
状态：**完成第一版**

### Phase 4.2 — 丘陵 / 山麓 / 平原连续权重
状态：**完成第一版**

### Phase 4.3 — Regional Terrain 合成与连续采样
状态：**完成第一版**

### Phase 4.4 — 沉积型地貌、河谷横断面与 Phase 4 验收
状态：**完成第一版**

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

状态：**算法第一版完成；已提交基线 4a5f685**

### Phase 5.1 — 生态区分类
状态：**完成第一版**

输入：

- 年平均温度
- 暖季 / 冷季温度
- 年降水
- 降水季节性
- 季风指数
- 湿润度 / 干旱度
- 海拔
- Relief
- 地貌权重
- 水文 / 湿地 / 河漫滩倾向

输出：

- Climate Type
- Eco Region
- Vegetation Profile
- Surface Moisture / Soil Tendency

原则：

- 生态区表示长期稳定生态。
- 不用瞬时天气决定 Biome。
- 不直接用一两个阈值把世界切成大片矩形区域。
- 先得到连续生态倾向，再映射到有限的 Eco Region 类别。

### Phase 5.2 — Minecraft Biome / Surface Mapping
状态：**完成第一版**

输出：

- Climate Type
- Eco Region
- Minecraft Biome
- Surface Material
- Vegetation Profile
- Terrain Profile

Biome 表示长期生态，不直接代表瞬时天气。

## Phase 6 — 统一模型、最小 Generator 与提前游戏验证

状态：**当前：6.4 首次 Minecraft Preview 已生成并通过官方 codec；下一步客户端真实进入世界验收**

| 子阶段 | 交付 | 状态 |
|---|---|---|
| 6.0 | Phase 5.2 提交、评审与进度保全 | 完成：4a5f685 |
| 6.1a | 世界范围/原点/节点尺度/海平面，地壳与实际宏观陆海分离 | 完成 |
| 6.1b | 构造/气候/坡度/水文空间参数迁移到统一尺度 | 完成 |
| 6.2 | GenerateWorld / WorldModel，以及最终海岸、河床、水面、生态一致性 | 完成 |
| 6.3 | 真实参数生成、Tester、版本化保存/重开、golden 与 Java reader | 完成（migrate 后置） |
| 6.4 | Go 模型 -> Java 采样 -> Minecraft 26.3 Vanilla Preview save -> 客户端验收 | 进行中：官方 codec 289/289 通过，待客户端验收 |
| 6.5 | 扩展 surface/vegetation、边界策略、UI 与后续预生成能力 | 首次游戏验收之后 |

最近里程碑：GeoWorld V1 已可跨语言读取，并能生成 Minecraft Java 26.3 Vanilla Preview save；Mojang 官方 codec 已全量解析 289/289 Chunk，下一步是客户端真实进入世界验收。
详细验收和依赖见 PHASE6.md；不能只把现有模块放进一个结构体就宣称完整模型验收通过。

## Phase 7 — Runtime MVP

状态：未开始；最小接入验证已前移到 6.4。

- 创建世界选择 GeoWorld，世界级配置随存档固定。
- 明确游戏内创建宏观模型的方案；外部模型导入只是早期验证路径。
- 完善按需 Chunk、地表/洞穴/结构/装饰兼容。
- 完善 Infinite Storm Ocean / Infinite Frozen Ocean 与四角过渡。
- GeoWorld API 与性能/缓存。
- 首个 Loader 稳定后再扩展其它 Loader。

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
