# GeoWorld 作为「世界地基」的 API 指导

更新时间：2026-11-07
状态：**指导文档，不含实现**。本文不新增/修改任何代码、群系、物品、方块、生物或玩法。

2026-11-07 实现进度：字段契约、QUERY-01～10、只读门面和共享服务端生命周期均已落地；本文后面的缺口分析保留为设计背景，当前验证与剩余人工步骤以 `QUERY_API_ACCEPTANCE.md`、`HANDOFF_QUERY_API.md` 为准。D-048 对外封控尚未解除。

---

## 0. 本文的定位与边界

### 0.1 GeoWorld 的职责

GeoWorld 是**世界地基**。它回答的问题是：

> 这个世界的自然事实是什么？

它不回答：

> 玩家在这个世界里做什么？

因此 GeoWorld **只做四件事**：

1. **确定性地产生自然地理事实**（地形、水体、气候、生态、地质、资源、宜居度、交通）；
2. **把这些事实以稳定、可查询、可复现的形式暴露出去**；
3. **保证这些事实在不同 Minecraft 版本 / Loader / 时间下语义不变**；
4. **保持干净**：不引入任何玩法内容，删除模组后原版机制正常。

### 0.2 GeoWorld 明确不做的事

| 不做 | 原因 |
|---|---|
| 新增 / 删除群系 | 群系是原版或内容模组的资产，GeoWorld 只做映射 |
| 新增物品、方块、生物、实体 | 属于玩法模组 |
| 新增结构、地牢、战利品 | 属于玩法模组 |
| 实现季节、天气、机制循环 | 属于玩法模组 |
| 渲染草色 / 植被颜色等视觉表现 | 属于表现层模组 |
| 强制任何玩法节奏 | 属于设计者 |

### 0.3 「倾向性」的准确含义

本文通篇使用的「倾向性」指的是：

> GeoWorld 在**不实现玩法**的前提下，让某些地理事实**更容易被发现、更容易被正确解读、更容易被组合**。

倾向性是**关于信息的**，不是关于内容。例如：

- ✅ 「矿区应该有一个稳定的、跨版本相同的标识，让玩法模组可以据此放置矿区聚落」
- ❌ 「矿区里应该自动出现矿工村落」

前者是事实暴露，后者是玩法实现。本文只讨论前者。

---

## 1. 现状评估：地基已经建成，但门还没开

### 1.1 已有的资产（实测核对）

`java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldQueryService.java` 已经实现了
一个设计非常正确的查询门面：

- `API_VERSION = "geoworld-query-v1"`，已带版本号；
- `NaturalField` 枚举 **86 个字段**，覆盖：
  - 地形：`ELEVATION`、`SLOPE`、`ASPECT_DEGREES`、`TERRAIN_VARIANT`、`SURFACE_PROFILE`
  - 气候：`ANNUAL_MEAN_TEMP_C`、`ANNUAL_PRECIP_MM`、`MONSOON_INDEX`、`WIND_X`、`WIND_Z`
  - 水文：`WATERSHED_ID`、`FLOW_ACCUMULATION`、`RIVER_ID`、`RIVER_ORDER`、
    `DISTANCE_TO_RIVER`、`DISTANCE_TO_COAST`、`WATERWAY_NAVIGABILITY`、`PORT_SUITABILITY`
  - 地质资源：`GEOLOGICAL_REGION`、`ORE_PROVINCE`、`MINING_DISTRICT_CLASS`、
    `DOMINANT_MINERAL_RESOURCE`、`MINERAL_RESOURCE_PROFILE`
  - 宜居度：`SETTLEMENT_SUITABILITY`、`AGRICULTURE_SUITABILITY`、`FLOOD_RISK`、`FRESH_WATER_ACCESS`
  - 聚落/交通：`CIVILIZATION_NODE_CLASS`、`REGIONAL_ADVANTAGE_CLASS`、
    `NATURAL_CORRIDOR_CLASS`、`TRANSPORT_CHOKEPOINT_STRENGTH`
- `Resolution` 三档（1/4/16 方块），兼顾精度与开销；
- **`NaturalResult` 带 `unavailableFields`**，允许旧模型字段不可用而不失败——这是正确的向后兼容姿态；
- **`LocalRealityProvider` / `DynamicStateProvider` 双层抽象**，且注释明确写了
  「Dynamic state must not be inferred from natural-model values」「Querying Local Reality
  must never（强制生成）」——这个边界划分是整份代码里最有价值的决定；
- `WeakHashMap` 共享实例 + 缓存 + `cacheRevision()`，防止重复计算。

**结论：地基的「数据面」已经相当完整。** 86 个字段的覆盖度，已经足够支撑若干玩法模组。

### 1.2 关键的缺口（按严重程度）

#### 缺口 A：这个 API 没有任何契约文档，也没有兼容性政策

- 仓里搜不到任何文档描述 `geoworld-query-v1` 的字段语义、单位、取值范围、缺失条件；
- `API_VERSION` 存在但**没有说明什么情况下必须升级**；
- `NaturalField` 是枚举，**加一个字段就是 ABI 变更**，但没有任何规则约束；
- 没有「字段废弃」流程。

> 风险：未来你开发第二个模组时，会面对「我不敢改 GeoWorld，怕弄坏我自己的模组」的僵局。
> **这是当前最高优先级的问题**，因为它会直接阻碍你后续所有模组化开发。

#### 缺口 B：不是所有适配器都暴露了同一个稳定门面

- `GeoWorldQueryService` 位于 `java/runtime-reader`，靠 **Gradle 打包把源码/class 塞进最终 JAR**；
- `runtime-reader` 的 README 只描述了「读取 .geoworld + 采样」，**完全没有提到 QueryService**；
- 各适配器（9 个目标）没有统一的「暴露 API 的入口」约定；
- `ARCHITECTURE.md` 里 `minecraft-common` 的接口清单**不包含 QueryService**。

> 风险：第三方模组不知道「我该 import 什么、从哪个实例拿 QueryService」。
> 现在只有你自己知道这个类的存在。

#### 缺口 C：没有「事实 → 语义标识」的稳定域

- 字段返回 `String`（如 `DOMINANT_MINERAL_RESOURCE`、`MINING_DISTRICT_CLASS`、`TERRAIN_VARIANT`），
  **但没有一处文档列出合法取值**；
- 这些字符串直接来自 Go 侧的 `normalizedProfile` 与 `mining.Evaluate`；
- 未来 Go 侧改了分类名，Java 侧字符串静默变化，玩法模组的 `if ("iron".equals(...))` 静默失效。

> 风险：**静默失败**。这比崩溃更糟，因为玩法模组不会知道自己的逻辑已经失效。

#### 缺口 D：没有「查询成本」与「批量查询」承诺

- 单点查询有缓存，但**没有批量/区域查询**；
- 玩法模组做「找最近的矿区」「扫描周围的可航行河段」时，会在循环里单点查询；
- 没有 `ComputationStats` 的公开语义（`executionHistory.bin` 里能看到这个内部类，但没暴露）。

> 风险：玩法模组会成为性能问题的第一责任人，而问题实际来自 GeoWorld 的查询形状。

#### 缺口 E：性能与稳定性门槛没有对着 API 定义

`docs/CHUNK_PERFORMANCE.md` 记录 shoreline ≈ 3–4ms/区块、biomes ≈ 2–4ms/区块，
但**没有上限断言**。你未来的玩法模组会在这个预算之上叠加，地基的抖动会被放大。

---

## 2. 设计原则（建议写入 DECISIONS）

以下原则建议作为 `D-0xx` 追加进 `docs/DECISIONS.md`。它们约束的是**未来所有**对 GeoWorld 的改动。

### P-1：事实与表现分离

GeoWorld 只说「这里降水 800mm」，不说「这里的草应该更绿」。
后者的拥有者是表现层模组。

### P-2：宁可缺失，不可瞎猜

字段不可用时必须返回 `unavailable`，**禁止**用启发式填补成看似合理的值。
现有 `NaturalResult.unavailableFields` 已体现这一原则，应上升为正式契约。

### P-3：只增量演进，不原地改语义

已发布的 `NaturalField` 语义**永远不变**。语义需要改变时，新增字段（如 `ANNUAL_PRECIP_MM_V2`），
旧字段标记废弃但保留一段时间。**绝不**修改已有字段的单位或含义。

### P-4：字符串域必须穷举且稳定

所有返回分类字符串的字段，必须有一份**权威取值表**（写在本文 `docs/FACT_REGISTRY.md` 中，
见第三节）。新增取值属于**次要版本变更**，删除或改名属于**主版本变更**。

### P-5：查询不得产生副作用

查询不得触发区块生成、不得加载模型、不得写缓存到磁盘。
`LocalRealityProvider` 的现有注释已经写对了，应扩展到**所有**字段。

### P-6：模型版本必须能被调用方看到

任何查询结果都应能追溯「这个值来自哪个 `generationRulesVersion` / `worldProfileVersion`」。
`NaturalResult` 已带这三个版本号，应保证**在所有适配器上都填充**。

### P-7：地基的代价必须是可预算的

GeoWorld 应对外公布自己的性能预算，玩法模组在此预算之上做设计。
预算变化必须显式记录在 `docs/CHUNK_PERFORMANCE.md`。

---

## 3. 建议的 API 分层（仅描述，不实现）

### 3.1 三层数据，责任清晰

```text
┌─────────────────────────────────────────────────────────────┐
│ Layer 1 · Natural Facts（自然事实）                          │
│ 拥有者：GeoWorld Core / Go 模型                              │
│ 性质：确定性、与种子绑定、永不随时间变化                       │
│ 例子：海拔、气候、河网、矿区、宜居度、走廊                     │
│ 现状：已基本完成（NaturalField 86）                         │
├─────────────────────────────────────────────────────────────┤
│ Layer 2 · Local Reality（本地现实）                          │
│ 拥有者：Minecraft 适配器（由玩法模组注册 Provider）            │
│ 性质：描述已加载区块的实际情况，可能被玩家改变                  │
│ 例子：这里实际是草方块还是被玩家挖掉了、水位实际高度            │
│ 现状：接口已定义（LocalRealityProvider），但无实现、无文档      │
├─────────────────────────────────────────────────────────────┤
│ Layer 3 · Dynamic State（动态状态）                          │
│ 拥有者：玩法模组（季节、天气、事件、经济…）                    │
│ 性质：随时间变化，GeoWorld 完全不感知                         │
│ 例子：现在是不是雨季、这块地今年种没种过                       │
│ 现状：接口已定义（DynamicStateProvider），正确保持不实现       │
└─────────────────────────────────────────────────────────────┘
```

**这个分层是 GeoWorld 最重要的资产之一。** 它天然回答了你担心的「玩法不该混进地基」问题：
**玩法只活在 Layer 3，GeoWorld 只提供 Layer 1 和 Layer 2 的读口。**

### 3.2 建议补的第 2.5 层：Derived Facts（派生事实）

在 Layer 1 与 Layer 3 之间，有一类事实**不是原始模型的一部分**，但**可以被多个玩法模组复用**：

| 派生事实 | 由 Layer 1 输入 | 典型消费者 |
|---|---|---|
| 「附近 2km 内有几个矿区」 | `ORE_PROVINCE`、`MINING_DISTRICT_CLASS` | 聚落生成、贸易 |
| 「从这个点顺流到海需要几段急流」 | `RIVER_ORDER`、`WATERWAY_NAVIGABILITY` | 航运玩法 |
| 「这片区域在生态上属于哪个生态区」 | `ECO_REGION`、`WATERSHED_ID` | 生物分布 |
| 「这个位置是否是天然隘口」 | `NATURAL_CORRIDOR_CLASS`、`SLOPE` | 交通、关卡设计 |

**建议**：这类计算**不由 GeoWorld 提供实现**，而是由 GeoWorld 提供**查询原语组合的能力**（见 4.3 区域查询），
让每个玩法模组自己决定要不要做。这样：
- GeoWorld 不承担玩法语义；
- 玩法模组不用重复实现空间索引；
- 两边都能独立演进。

> 若你未来发现 3 个以上模组都在重复实现「最近矿区查询」，再考虑把这一层提升为 Layer 1.5。
> **在此之前不要预先抽象**，这正好符合 GeoWorld 现有的「先收口约束，再扩展」纪律。

---

## 4. 建议补齐的 API 能力（按优先级，仅描述）

### 4.1 【最高】契约文档：`docs/FACT_REGISTRY.md`

**这是未来 6 个月内最值得做的一件事，且完全不需要改代码。**

一份权威表，对**每一个** `NaturalField` 描述：

| 列 | 含义 |
|---|---|
| 字段名 | 例如 `DOMINANT_MINERAL_RESOURCE` |
| 单位 | 例如 `mm/year`、`blocks`、`0..1`、`null` |
| 定义 | 一句话说清测的是什么，**不是怎么算的** |
| 合法取值 | 枚举全部，或数值区间 |
| 缺失条件 | 什么情况下返回 `unavailable` |
| 稳定性 | `stable` / `beta` / `deprecated` |
| 引入版本 | 从哪个 `generationRulesVersion` 起有效 |
| 依赖 | 是否依赖 `Resolution`、是否为区域量 |

**为什么这比写代码更重要：**

1. 你写第二个模组时，不用重新读 `GeoWorldQueryService.java` 的 1000 行；
2. 第三方模组会因此**愿意**依赖 GeoWorld——没有契约的 API 没人敢用；
3. 它让「静默失败」（缺口 C）从「靠运气」变成「可检查」；
4. 它可以直接从 `NaturalField` 枚举 + Go 侧常量**半自动生成**，维护成本低。

**验收标准（不实现代码也能的人工验证）**：
- 每个 `NaturalField` 都在表里有对应行；
- 每个字符串型字段的合法取值都能在 Go 常量里找到一一对应；
- 三条随机抽查：取一个真实 `.geoworld`，用 QueryService 查 10 个点，确认所有字符串返回值都在表中。

### 4.2 【高】列举式 API：`listFields()` / 支持的解析度 / 版本

玩法模组需要一个**探针**，能在启动时问 GeoWorld：

- 你现在支持哪些字段？（`NaturalField.values()` 已有，但需要**语义可用性**：哪些是 `beta`、哪些在旧模型里缺失）
- 这个世界的模型是哪一版？（`NaturalResult` 已带，但需要**世界级**的一次性查询）
- 当前 `API_VERSION` 是多少？兼容区间是什么？

**这一步让玩法模组可以在加载时就判断「我要的字段这个世界有没有」**，而不是在玩家跑到某个坐标时崩溃。

### 4.3 【高】区域查询原语（不实现语义）

现状只有单点 `QueryRequest(worldX, worldZ, resolution, fields)`。

建议增加**原语级**的区域查询：

```text
queryRegion(bounds, resolution, fields) → 网格结果
queryNearest(bounds, field, predicate) → 最近满足条件的点
queryScan(bounds, field, aggregate) → 聚合（min/max/mean/count）
```

注意关键设计约束：

- **这些是几何原语，不是业务查询**。`queryNearest` 不内置「找矿区」语义，
  只接受 `predicate`；
- 必须**强制带 `Resolution`**，防止调用方在 1 方块精度上扫 10km；
- 必须**带结果上限和预算反馈**（见 4.4）；
- 必须**可中断 / 可异步**（世界加载线程不允许长阻塞）。

这样玩法模组能高效实现「找最近矿区」，而 GeoWorld 完全不涉及「矿区里该有什么」。

### 4.4 【中】查询成本契约

建议 `GeoWorldQueryService` 对外公布：

- 每类查询在给定 `Resolution` 下的**典型耗时**（进 `CHUNK_PERFORMANCE.md`）；
- **每次查询的推荐预算**（例如「玩法模组每 tick 的查询总量不超过 X」）；
- 暴露**累计统计**（已有内部 `ComputationStats`，建议整理为公开只读对象）。

理由：你未来的玩法模组是**在 GeoWorld 之上叠加**的。如果地基不给出预算，
玩法模组只能靠猜，最终所有人一起掉帧，而没人知道是谁的锅。

### 4.5 【中】字段废弃与迁移流程

写进 `DECISIONS.md`：

```text
stable    → deprecated → removed（至少跨一个大版本）
新增字段   → 次版本号增加，旧版读者忽略未知字段
语义改变   → 新增字段，旧字段永不改变含义
```

**并与 `generationRulesVersion` 联动**：`NaturalResult` 已经返回这个版本号，
建议在 `FACT_REGISTRY.md` 里记录「每个字段从哪个规则版本起有效」，
让玩法模组能在旧世界（旧规则版本）里优雅降级。

### 4.6 【低但重要】本地现实的显式化

`LocalRealityProvider` 现在只有接口，没有文档说明：

- 谁负责实现它？（答案是：玩法模组，不是 GeoWorld）
- 它返回 `Map<String, Object>`，**键名是否有约定**？
- 缓存 250ms TTL 的设计意图？

**建议只写文档，不动代码。** 明确「GeoWorld 提供一个槽位，玩法模组填」，
这本身就是对边界的一次公开声明。

---

## 5. 倾向性建议：GeoWorld 可以怎样「为玩法让路」，但不实现玩法

以下都是**表达层 / 暴露层**的倾向性，不涉及任何内容新增。

### 5.1 让自然事实「可指认」

**问题**：一个 `ORE_PROVINCE=iron` 的数值，玩家无法感知，玩法模组也难以「指向」它。

**倾向性做法（不实现玩法）**：

- 保证每个自然事实都有一个**稳定的、跨世界可复现的标识**（如 `RIVER_ID`、`WATERSHED_ID` 已经存在）；
- 对**区域级**事实提供**区域标识 + 质心 + 面积**（`REGIONAL_REGION_CENTROID_X`、
  `REGIONAL_REGION_AREA_BLOCKS2` 已经有雏形）；
- 这样玩法模组可以给区域起名字、放地标，而 GeoWorld 仍然只提供坐标和 ID。

> 这不是新增内容，而是**保证已有事实具备「可被引用」的性质**。

### 5.2 让自然事实「有层次」

玩法模组需要区分「这是背景」还是「这是有意义的点」。建议 GeoWorld 在字段层面提供**显著性**：

- 已有：`REGIONAL_SPECIALIZATION_STRENGTH`、`TRANSPORT_CHOKEPOINT_STRENGTH`、`SETTLEMENT_SUITABILITY`
- 缺失：**「这个点的某项事实在该区域里有多突出」** 的统一口径

这样玩法模组可以问：「这里是**普通**河流，还是**这里唯一**的大河出口？」——
这是个**信息层次问题**，不是玩法问题。

### 5.3 让气候数据「自成一组」

`monsoonStrength`、`precipitationSeasonality`、`windSummer/Winter` 是 GeoWorld
**独有而其他地形模组无法提供**的数据（见 `PRODUCT.md` §3 和 `DECISIONS.md` D-005）。

倾向性建议：把这几个字段**成组暴露**并标注为 `seasonal-climate-v1` 组，
让未来的季节模组可以**一次性声明依赖**，而不是逐个字段试。

- 这只是**API 分组与文档**，不是实现季节；
- 但它让「季节模组」这个产品方向变得可行。

### 5.4 让「外围世界」具备可判断性

`DECISIONS.md` D-004 定义了风暴洋 / 冰冻洋。目前这是一个**地形结果**。

倾向性建议：把「我是否在世界外围」变成一个**可查询的事实**（例如 `REGION_KIND` =
`core` / `transition` / `storm-ocean` / `frozen-ocean` / `polar`），而不是让玩法模组
自己去比对坐标范围。

- GeoWorld 仍然只报告事实；
- 玩法模组可以自行决定「外围要不要刷怪、要不要给特殊资源」——
  **这是它的玩法，不是你的**。

### 5.5 明确「不提供」的清单

建议在 `PRODUCT.md` 里维护一个**明确的否定清单**，并且这是一条**硬边界**：

```text
GeoWorld 永不提供：
- 任何生物、物品、方块、结构
- 任何战利品表、交易表、掉落表
- 任何季节/天气的运行时实现
- 任何视觉表现（颜色、着色器、渲染层）
- 任何玩法节奏、难度、进度
```

把「不做什么」写得和「做什么」一样清楚，是地基类项目的核心纪律。
你已经有了 `PRODUCT.md`，但否定清单目前是分散在各 `PROGRESS` 条目里的（「不引入任何非原版方块…」），
**建议收拢成一节**。

---

## 6. 建议的实施顺序（全部在上文边界内）

| 阶段 | 内容 | 是否需要改代码 | 收益 |
|---|---|---|---|
| ① | 写 `docs/FACT_REGISTRY.md`，穷举 86 字段 | 否 | 立即可用；解锁后续开发 |
| ① | 把 P-1 ~ P-7 写入 `DECISIONS.md` | 否 | 冻结边界，防止后续漂移 |
| ① | 在 `PRODUCT.md` 增加「永不提供的清单」 | 否 | 边界公开化 |
| ② | 写 `listFields()` 语义与版本兼容区间的**设计稿**（不实现） | 否 | 让玩法模组能做启动期探测 |
| ② | 明确「适配器如何暴露 QueryService」的约定 | 否（文档） | 第三方可用 |
| ③ | 区域查询原语的设计稿（bounds + resolution + 预算） | 否 | 为玩法模组预留 |
| ③ | 查询成本写进 `CHUNK_PERFORMANCE.md` | 否 | 预算可见 |
| ④ | 上述设计落地实现（到那时再单独开计划） | 是 | — |

**注意**：①②③ 全部是文档工作，**一行代码都不用改**，但它们是「未来所有玩法模组能否成立」的前提。
在你写第一个玩法模组之前完成 ①，收益最大。

---

## 7. 给未来的玩法模组作者的一段话（可作为对外说明草稿）

```
GeoWorld 提供的是「世界是怎么样的」，不是「你该怎么做」。

如果你要做一个季节模组：
    读 GeoWorld 的 ANNUAL_PRECIP_MM、MONSOON_INDEX、WIND_X/Z，
    自己决定什么季节该下雨。GeoWorld 不提供天气。

如果你要做一个定居点模组：
    读 GeoWorld 的 SETTLEMENT_SUITABILITY、FRESH_WATER_ACCESS、FLOOD_RISK，
    自己决定把村落放在哪、给什么交易。GeoWorld 不提供村落。

如果你要做一个矿工玩法：
    读 ORE_PROVINCE、MINING_DISTRICT_CLASS、DOMINANT_MINERAL_RESOURCE，
    自己决定给什么矿、给什么奖励。GeoWorld 不提供矿工。

GeoWorld 永远不会替你做这些决定。
但它会保证：同一个 Seed，你每次读到的都是同一份事实。
```

---

## 8. 与现有文档的关系

| 文档 | 关系 |
|---|---|
| `PRODUCT.md` | 本文是其「暂不负责」一节的**接口化延伸**；第 5.5 节建议回填到 PRODUCT |
| `DECISIONS.md` | 本文第 2 节建议追加为新的 D-0xx 条目 |
| `ARCHITECTURE.md` | 本文第 3 节的三层数据模型应回填到架构图；并补入 `GeoWorldQueryService` |
| `PHASE6.md` | 本文不改变 Phase 6 的任何验收项，只补充「模块间接口」这一未覆盖的维度 |
| `STATUS.md` | 完成第 6 节 ① 后，应在 STATUS 中记录「地基 API 契约已建立」 |

---

## 9. 决策结果（原未决问题）

| 问题 | 结论 | 位置 |
|---|---|---|
| 1. API_VERSION 升级规则 | 以行为契约为准；新增字段不算 breaking | D-049 |
| 2. QueryService 暴露方式 | 单一门面 `geoworld.runtime.api` + 适配器自注册 | D-050 |
| 3. 区域查询精度下限 | 单次 ≤65,536 点，分辨率分级限范围，超限拒绝 | QUERY_API_ACCEPTANCE §1.1 |
| 4. FACT_REGISTRY 维护纪律 | 接受；由 QUERY-01 自动校验 | D-048 规则 3 / QUERY-01 |
| 5. Layer 1.5 | 暂缓；≥2 个玩法模组重复实现后再立项 | QUERY_API_ACCEPTANCE §1.2 |

原问题记录：

1. **`API_VERSION` 的升级规则**：什么算 breaking？字段新增算不算？
2. **QueryService 的暴露方式**：是 `minecraft-common` 直接 re-export，还是要求玩法模组显式依赖 `geoworld.runtime`？
3. **区域查询原语的精度下限**：是否允许 1 方块精度扫大范围？如果不允许，用什么机制拒绝？
4. **`FACT_REGISTRY.md` 的维护成本**：是否接受「改 Go 分类名就必须同步改文档」的纪律？
5. **是否预留 Layer 1.5（Derived Facts）**：现在决定还是等 3 个模组都重复实现了再决定？

以上问题已全部给出结论，后续执行见 `docs/QUERY_API_ACCEPTANCE.md`。
