# GeoWorld 真实地貌强化计划

更新时间：2026-11-08

## 1. 目标

本阶段不增加玩法、方块、物品，也不重写 GeoWorld 的世界体系。目标是在现有大陆、构造、气候、水文、侵蚀、地貌和 Minecraft Runtime 基础上，提高最终世界的自然性与因果一致性。

核心目标：

1. 完善侵蚀与沉积，让山坡、峡谷、河谷、山麓与冲积区更像长期演化结果。
2. 加强水文与最终地形一致性，重点检查河流纵剖面、支流汇合、湖泊出口和河口。
3. 优化地貌过渡，减少山地、丘陵、平原、盆地与海岸之间的程序生成痕迹。
4. 建立真实地貌对照验收，用 DEM、河网和典型地貌形态约束结果，而不是只看截图。

本阶段称为 **Realistic Geomorphology / 真实地貌强化**。

## 2. DADM 开发约束

本阶段只在 dadm 开发。当前基线：

- 开发机：dadm。
- 基线提交：`788dc3f`。
- 工作分支：`adm/realistic-geomorphology`。
- 当前正式规则：`geoworld-rules-v1.1.18`；`v1.1.16` 保持锁定，历史 receiver `v1.1.17` 已回退且不复活。

必须遵守现有 DADM 交接原则：

- 先定位可复现失败样例。
- 再补定向自动回归。
- 最小范围修改算法。
- 最后跑 Go / Java / Loader 验证。
- 任何改变生成结果的改动必须单独评估 rules version 与旧存档兼容。
- 不修改历史 golden 来“让测试通过”。
- 不把额外 strict audit 计数直接当成 Minecraft 缺陷。

### 2.1 证据规则

Phase 8 中所有 audit 结果分成两类：

**调查信号：**

- 纵剖面局部上升。
- 汇流水面差。
- 河口 receiver handoff。
- 地貌梯度突变。
- 统计分布偏离。

它们只能说明“这里值得检查”。

**正式缺陷：**

至少需要满足其一：

1. 最终 Runtime / Minecraft voxel 形成违反既有契约的可重复错误；
2. 现有 WaterOwnership / Shoreline / Model / Loader 验收失败；
3. 固定 Seed 能稳定复现明显错误地貌；
4. Go 与 Java 最终采样语义不一致。

没有正式缺陷证据，不改生成算法。

## 3. 因果链原则

优先顺序：

```text
构造与宏观高程
    ↓
气候与径流
    ↓
水文拓扑
    ↓
侵蚀 / 搬运 / 沉积
    ↓
区域地貌
    ↓
连续 Terrain
    ↓
局部河床 / 湖盆 / 河口
    ↓
Runtime
    ↓
Minecraft voxel
```

禁止：

- 在 ShorelinePlanner 里补宏观地貌。
- 用统一 Blur 解决所有地貌过渡。
- 用额外噪声冒充侵蚀。
- 为修单一截图破坏世界尺度规则。
- 未经证据直接重开已回退的 river receiver v1.1.17 方案。

## 4. 工作流 A：侵蚀与沉积

现有 Erosion / Geomorph 已经有：

- Slope
- FluvialPower
- ValleyIncision
- HillslopeErosion
- ErodedElevation
- SedimentSupply
- FloodplainWeight
- AlluvialPlainWeight
- AlluvialFanWeight
- DepositionPotential

因此不新造平行系统。

重点升级：

- 上游产沙来源。
- 搬运能力。
- 坡降变化与沉积关系。
- 山口出口沉积。
- 湖泊入口沉积。
- 河口沉积。
- 河谷受限程度和谷宽变化。

## 5. 工作流 B：水文与最终地形一致性

### 河流纵剖面

检查：

- source → mouth 总体下降。
- 上、中、下游坡降趋势。
- 局部逆坡是否只是合法 backwater / receiver 语义。
- 最终 Minecraft 水面是否产生真实台阶。
- 河床是否高于两侧地形形成地上河。

### 支流汇合

检查：

- incoming / downstream 河床和水面连续。
- 下游河宽不无故缩窄。
- 汇流处不干涸。
- 汇流后的谷宽和局部沉积是否合理。

### 湖泊

区分：

- 有出口湖。
- 封闭湖 / 内流盆地。

当前 Hydrology 是否真正支持内流盆地必须先审计，不预设已经实现。

### 河口

检查：

- 主河真实进入 Ocean。
- mouth approach 不突然缩窄。
- river → estuary / ocean handoff 无不合理水墙。
- delta 必须有上游 sediment、低坡降和海岸条件背景。

## 6. 工作流 C：地貌过渡

不采用统一模糊。

分别处理：

- 山地 → 山麓 → 丘陵 → 平原。
- 山口 → 冲积扇 → 冲积平原。
- 盆地边缘 → 盆底。
- 河谷 → 河漫滩。
- 陆地 → 海岸 → 海底。

重点统计：

- Relief 梯度。
- 坡度梯度。
- 局部高度二阶变化。
- 规则直线边界。
- 固定宽度重复过渡带。

## 7. 工作流 D：真实地貌对照

第一批参考类型：

- 年轻山系。
- 老年侵蚀山地。
- V 型河谷 / 峡谷。
- 冲积平原。
- 山前冲积扇。
- 河口 / 三角洲。
- 开放盆地。
- 内流盆地。

真实资料只用于形态和统计参考，不复制具体地点。

## 8. 验收层级

### Level 1 — 拓扑与数学

- FlowTo 无环。
- 河网连通。
- 湖口合法。
- 河口连通。
- 河床约束有效。

### Level 2 — Continuous Runtime

- 最终 ground / water / special features 组合后仍符合水文语义。
- 无可复现地上河。
- 无可复现断河、水墙、错误河口收窄。

### Level 3 — Minecraft voxel

- ShorelinePlanner 后实际方块地形正确。
- 跨 Chunk 连续。
- 水面量化无异常跳变。
- 岸线与河床没有由整数映射制造的新缺陷。

### Level 4 — 地貌统计与真实参考

- 河流纵坡。
- 谷宽。
- 坡度分布。
- Relief。
- 冲积区尺度。
- 盆地平坦度。
- 海岸梯度。

## 9. 实施顺序

### R0 — 现状审计

状态：**完成。**

真实调用链、已有功能、重复职责与当前缺口见
[REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md](REALISTIC_GEOMORPHOLOGY_R0_AUDIT.md)。

### R1 — 验收框架

状态：**第一阶段完成并持续作为 Phase 8 验收基础。**

增加只读统计和定位工具。详细结果见
[REALISTIC_GEOMORPHOLOGY_R1_ACCEPTANCE.md](REALISTIC_GEOMORPHOLOGY_R1_ACCEPTANCE.md)。

**R1 的结果默认是调查信号，不是自动生成缺陷结论。**

### R2 — 河流纵剖面与汇流

状态：**完成两项有最终证据的问题修复。**

仅针对 R1 + 正式最终验收共同证明的问题修改算法：

1. v1.1.18：特殊盆地持久化 drainage corridor，解决盆地河流爬坡而不制造悬高河。
2. v1.1.19：成熟冲积低地河 longitudinal grading，改善 BaseElevation / Terrain broad downstream steepening。

文档：

- [REALISTIC_GEOMORPHOLOGY_R2_BASIN_DRAINAGE.md](REALISTIC_GEOMORPHOLOGY_R2_BASIN_DRAINAGE.md)
- [REALISTIC_GEOMORPHOLOGY_R2_LONGITUDINAL_GRADE.md](REALISTIC_GEOMORPHOLOGY_R2_LONGITUDINAL_GRADE.md)

没有恢复已回退的 receiver geometry v6；Waterbed receiver low-bank 语义仍保持 v5。

### R3 — 湖泊、内流盆地与河口

先确认当前 drainage-complete 语义，再决定是否新增 endorheic basin。

### R4 — 侵蚀 / 搬运 / 沉积

强化已有 Erosion / Geomorph 因果关系。

### R5 — 地貌语义过渡

逐类优化山地、平原、盆地、海岸过渡。

### R6 — 真实地貌统计对照

建立固定参考区间。

### R7 — Minecraft 实机验收

固定 Seed 走图和 WebUI 对照后完成阶段。

## 10. 完成标准

必须同时满足：

1. 现有正式验收不退化。
2. 新的 Phase 8 验收通过。
3. 固定 Seed 不出现新的宏观拓扑漂移。
4. Go / Java Runtime 保持一致。
5. 旧世界锁定规则不被新算法偷偷覆盖。
6. 性能没有不可接受回退。
7. Minecraft 实机地貌确实比原基线更自然，而不是只让 audit 数字更漂亮。

最终目标：

> 大多数普通地貌合理、连续、可解释；特殊地貌只在合适的地理背景中出现。
