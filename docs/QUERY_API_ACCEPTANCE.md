# GeoWorld Query API 验收与门面落地指导

更新时间：2026-11-06
状态：**指导文档**。本文只规定做什么、按什么顺序做、怎样算完成，不包含实现代码。
上游文档：`DECISIONS.md` 中的 D-048 / D-049 / D-050、`FACT_REGISTRY.md`、`FOUNDATION_API.md`

---

## 0. 结论

`geoworld-query-v1` 要对外宣称可用，必须同时满足下面三项：

1. 第 2 节列出的 **QUERY-01 ~ QUERY-10** 全部进入 Verifier 并通过；
2. 第 3 节的 **门面切片 S1 ~ S5** 全部完成；
3. 第 5 节的发布检查表全部勾选。

三项都满足后，在 `DECISIONS.md` 的 D-048 下追加一行「封控解除：日期 / 提交」。在此之前，Query API 只供 GeoWorld 内部使用。

---

## 1. 两个遗留问题的决定

这两项是 `FOUNDATION_API.md` 第 9 节里第 3、5 个问题，本文直接给出结论。

### 1.1 区域查询的精度下限（第 3 题）

区域查询原语（`queryRegion / queryNearest / queryScan`）尚未实现。现在先把约束定下来，实现时必须遵守：

| 规则 | 值 |
|---|---|
| 单次调用的最大采样点数 | 65,536（相当于 256×256） |
| `BLOCK_1` 允许的最大范围 | 256 × 256 block |
| `BLOCK_4` 允许的最大范围 | 1,024 × 1,024 block |
| `BLOCK_16` 允许的最大范围 | 4,096 × 4,096 block |
| 超出上限时的处理 | 请求非法，直接拒绝（`IllegalArgumentException`），**不做静默降采样** |
| `REGIONAL_REGION_*` 字段 | 只能用 `BLOCK_16` 查询 |

理由：静默降采样会让同一个请求在不同实现里返回不同结果，违反 D-049 第 2 节「单点与区域结果一致」的要求。明确拒绝，调用方立刻就知道。

### 1.2 Layer 1.5 派生事实（第 5 题）

**暂缓，不预先抽象。** 触发条件是：至少 2 个实际存在的玩法模组，重复实现了同一种派生查询，例如「最近矿区」。触发后用新的 D-0xx 决定立项。在那之前，派生计算一律由玩法模组自己完成。

---

## 2. 验收项（QUERY-01 ~ QUERY-10）

### 2.1 通用要求

- 验收入口沿用 `GeoWorldModelVerifier`，新编号加进 `VERIFIED_IDS`，同时在 `GeoWorldAcceptanceRegistry` 中登记；
- 数据源是 `internal/goldenvectors/testdata/` 下全部已提交模型（目前 6 个），与 `verify-golden.ps1 -All` 保持一致；
- 采样集固定为两部分：
  - **网格点**：每个模型在模型范围内取 32×32 的均匀网格；
  - **特殊点**：复用 golden 用例里的特殊坐标，包括四角、四边、负坐标 Chunk 边界、海岸、河口回水、湖心、模型外远洋/极地/外岛；
- 每个点用 `QueryRequest.all(x, z, BLOCK_4)` 请求全部 86 个字段；QUERY-05 例外，用 `BLOCK_16`；
- 失败信息必须包含：模型文件名、坐标、字段名、实际值。

### 2.2 验收项

| ID | 检查内容 | 通过标准 |
|---|---|---|
| **QUERY-01** 字段登记同步 | `NaturalField.values()` 与 `FACT_REGISTRY.md` 的字段集合是否一致 | 数量相同（目前 86），名字逐一对应。实现方式：Verifier 读取 md 文件，用正则提取反引号里的大写字段名做比较 |
| **QUERY-02** 字符串取值域 | 附录 A 中所有字符串字段的返回值 | 每个值都在附录 A 的集合内；`GEOLOGICAL_REGION` 匹配 `^(oceanic-crust\|orogenic-belt\|uplifted-block\|exposed-crust\|sedimentary-basin\|continental-interior):[0-9a-f]+$`；`ORE_PROVINCE` 匹配 `^ore-[0-7]-[0-9a-f]+$`；`DOMINANT_MINERAL_RESOURCE` 是 `none` 或当前矿物字典里的一项 |
| **QUERY-03** 缺失语义 | `values` 与 `unavailableFields` 的关系 | 两者没有交集；两者的并集等于请求的字段集；`values` 里没有 `null`；所有 double 都是有限值（不是 NaN/Inf） |
| **QUERY-04** 数值范围 | 数值字段的取值范围 | 所有标注为 0..1 的字段 ∈ [0,1]；所有距离字段 ≥ 0；`ASPECT_DEGREES` ∈ [0,360)；`ANNUAL_PRECIP_MM` ≥ 0；`ANNUAL_MEAN_TEMP_C` ∈ [-60,60]；`MONSOON_INDEX` ∈ [0,1]；`RIVER_ORDER` ≥ 0；`REGIONAL_HIGH_ADVANTAGE_COUNT` ∈ [0,5] |
| **QUERY-05** 区域一致性 | 同一个 `REGIONAL_REGION_KEY` 下的各点 | `CELL_COUNT`、`AREA_BLOCKS2`、`CENTROID_X`、`CENTROID_Z` 完全相等；每个区域至少找到 2 个采样点才参与判定 |
| **QUERY-06** 量化一致性 | 同一个 `Resolution` 格内的不同偏移 | 返回的 `sampleX/sampleZ` 等于量化后的坐标；同一格内的偏移请求返回完全相同的 `NaturalResult` |
| **QUERY-07** 版本字段 | 结果中的版本信息 | `apiVersion`、`worldProfileVersion`、`generationRulesVersion`、`worldModelVersion` 都不为空，并且与模型 manifest 一致 |
| **QUERY-08** 模型外行为 | 模型外的特殊点 | 符合 `FACT_REGISTRY.md` 第 5 节：河流/湖泊/海岸距离类字段缺失；海洋点上 `DISTANCE_TO_OCEAN` = 0；`WATER_KIND` = `Ocean` |
| **QUERY-09** 负向字段方向 | 抽查洪泛区 | 在 `TERRAIN_VARIANT ∈ {Floodplain, Alluvial Plain}` 且靠近河流的点上，`FLOOD_RISK` 的均值要**高于**全体陆地点的均值；只做方向性断言，不断言具体数值 |
| **QUERY-10** 门面等价 | 门面查询与直接查询 | 对同一模型、同一请求，通过 `GeoWorldFacade` 得到的结果与直接调用 `GeoWorldQueryService` 的结果 `equals` 相同。这一项依赖第 3 节的 S4 |

现有的 MODEL-01 ~ MODEL-04（确定性、重开一致、不强制加载、Local Reality 在未加载时停止）继续保留，不重复建项。

### 2.3 验收状态的回填规则

QUERY-02 ~ QUERY-04 一旦通过，`FACT_REGISTRY.md` 中被覆盖字段的验收列统一改为 `V`，并在附录 C 注明对应的 QUERY 编号。

---

## 3. 门面落地切片（D-050）

每个切片单独提交，单独验收，不合并。

### S1：确认模型持有者与生命周期（先调查，再写设计结论）

需要回答以下问题，结果写进 D-050 的「遗留」小节：

1. 动态世界运行时，权威的 `GeoWorldV1` 实例由哪个类持有？候选是 `GeoWorldEngineBridge`、`GeoWorldBinding`，以及 v26 / v1_21_1 的 WorldService；
2. 这个实例什么时候创建、什么时候失效？分别对应服务器启动、世界加载、世界关闭、返回标题界面；
3. 单人游戏（集成服务器）和专用服务器，生命周期入口是否不同？

完成标准：每个版本族（v26-common、v1_21_1-common、v1_20_1）各给出一个确定的 bind 点和 unbind 点。

### S2：门面契约（只定接口语义，实现细节自由）

固定以下语义：

- 包名：`geoworld.runtime.api`，这是唯一承诺稳定的包；
- `apiVersion()` / `minSupportedApiVersion()`：目前两者都是 `geoworld-query-v1`；
- `currentWorld()` → `Optional<WorldFacts>`：
  - **只对应主世界**。这与 2026-11-06 WebUI「地图只对应主世界」的维度语义一致；下界、末地和其他维度不提供 GeoWorld 事实；
  - 未绑定、正在加载或已经关闭时返回 `empty`，不抛异常、不阻塞；
  - 可以在任意线程调用，返回的视图不可变；
- `WorldFacts` 只暴露 `queryNatural(QueryRequest)`、`supports(NaturalField)` 和三个版本号，**不暴露** `GeoWorldV1` 本体；
- `supports(field)` 的判定方法：在模型中心点做一次探测查询，字段不在 `unavailableFields` 里，或者在旧模型上本来就属于条件可用字段，都记为支持。判定结果在本世界生命周期内缓存；
- 只提供**逻辑服务端**事实。客户端侧的模组在 v1 中不能使用门面，这一条写入对外说明。

### S3：适配器接入

- bind / unbind 放在各**版本族共享层**里，即 `minecraft-v26-common`、`minecraft-v1_21_1-common` 和 1.20.1 对应层，**不要**在 9 个适配器里各写一份。这遵守 ARCHITECTURE 的「薄桥接」规则；
- 一个世界关闭后，必须保证 `currentWorld()` 返回 `empty`，或者已经切换到新世界，不能残留旧世界。

完成标准：至少用 1.21.1 NeoForge 和 26.3 Fabric 两个目标，人工走一遍「进入世界 → 返回标题 → 进入另一个世界」。

### S4：验收接入

- 新增 QUERY-10；
- 新增一个生命周期黑盒检查：模拟 bind A → unbind → bind B，确认 `currentWorld()` 依次为 A、empty、B；
- 加入各适配器的 `check` 任务，与 `verifyGeoWorldModelQuery` 并列执行。

### S5：解除封控与文档同步

- 确认第 5 节检查表全部完成；
- 在 D-048 下追加封控解除记录；
- `FOUNDATION_API.md` 第 7 节的对外说明草稿定稿，放进 `java/runtime-reader/README.md` 的新小节「Query API」。

---

## 4. 明确不做（本轮）

- 不实现区域查询原语。第 1.1 节只定约束，实际实现等第一个玩法模组提出需求时再做；
- 不实现 `LocalRealityProvider` 的任何具体 Provider；
- 不新增 `NaturalField`。季节风字段的需求另行立项，按 D-049 新增；
- 不改动任何已有字段的取值或命名风格，包括附录 A 中两种命名风格并存的情况。

---

## 5. 发布检查表（每次涉及 Query 的发布都要过一遍）

- [ ] `NaturalField` 数量与 `FACT_REGISTRY.md` 一致（QUERY-01 自动检查）
- [ ] QUERY-01 ~ QUERY-10、MODEL-01 ~ MODEL-04 全部通过
- [ ] 如果字段的语义、单位、取值域或缺失条件有变化：已按 D-049 第 2 节判断是否属于 breaking 变更，属于的话 `API_VERSION` 已提升，并在 D-049 下追加了变更记录
- [ ] `FACT_REGISTRY.md` 的更新时间与本次发布一致
- [ ] `STATUS.md` 记录了本次 Query 相关改动，并且区分了「已验证」和「未验证」
