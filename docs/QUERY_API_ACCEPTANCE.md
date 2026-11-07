# GeoWorld Query API 验收与门面落地指导

更新时间：2026-11-07
状态：**QUERY-01～10 与 S1～S5 全部完成，2026-11-07 已解除 D-048 封控**。本文件同时保存执行规则、自动验证和双目标实机记录。
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

- 验收入口沿用 `GeoWorldModelVerifier`，新编号加进 `VERIFIED_IDS`，同时在 `GeoWorldAcceptanceRegistry.queryEntries()` 补充表中登记（保留既有 177 项原始表；证据文件用 `queryPassedIds` 单列）；
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
| **QUERY-02** 字符串取值域 | 附录 A 中所有字符串字段的返回值 | 每个值都在附录 A 的集合内；`GEOLOGICAL_REGION` 匹配 `^(oceanic-crust\|orogenic-belt\|uplifted-block\|exposed-crust\|sedimentary-basin\|continental-interior):[0-9a-f]+$`；`ORE_PROVINCE` 匹配 `^ore-[0-7]-[0-9a-f]+$`；`DOMINANT_MINERAL_RESOURCE` 是 `none` 或当前 Query 矿物家族（`GeoWorldMineralAffinity.MINERALS`）里的一项 |
| **QUERY-03** 缺失语义 | `values` 与 `unavailableFields` 的关系 | 两者没有交集；两者的并集等于请求的字段集；`values` 里没有 `null`；所有 double 都是有限值（不是 NaN/Inf） |
| **QUERY-04** 数值范围 | 数值字段的取值范围 | 所有标注为 0..1 的字段 ∈ [0,1]；所有距离字段 ≥ 0；`ASPECT_DEGREES` ∈ [0,360)；`ANNUAL_PRECIP_MM` ≥ 0；`ANNUAL_MEAN_TEMP_C` ∈ [-60,60]；`MONSOON_INDEX` ∈ [0,1]；`RIVER_ORDER` ≥ 0；`REGIONAL_HIGH_ADVANTAGE_COUNT` ∈ [0,5] |
| **QUERY-05** 区域一致性 | 同一个 `REGIONAL_REGION_KEY` 下的各点 | `CELL_COUNT`、`AREA_BLOCKS2`、`CENTROID_X`、`CENTROID_Z` 完全相等；每个区域至少找到 2 个采样点才参与判定 |
| **QUERY-06** 量化一致性 | 同一个 `Resolution` 格内的不同偏移 | 返回的 `sampleX/sampleZ` 等于量化后的坐标；同一格内的偏移请求返回完全相同的 `NaturalResult` |
| **QUERY-07** 版本字段 | 结果中的版本信息 | `apiVersion`、`worldProfileVersion`、`generationRulesVersion`、`worldModelVersion` 都不为空，并且与模型 manifest 一致 |
| **QUERY-08** 模型外行为 | 模型外的特殊点 | 符合 `FACT_REGISTRY.md` 第 5 节：河流/湖泊/海岸距离类字段缺失；海洋点上 `DISTANCE_TO_OCEAN` = 0；外围稀疏海岛可为陆地，此时该距离也缺失，不强制把海岛当成海洋 |
| **QUERY-09** 负向字段方向 | 抽查洪泛区 | 在 `TERRAIN_VARIANT ∈ {Floodplain, Alluvial Plain}` 且靠近河流的点上，`FLOOD_RISK` 的均值要**高于**全体陆地点的均值；只做方向性断言，不断言具体数值 |
| **QUERY-10** 门面等价 | 门面查询与直接查询 | 对同一模型、同一请求，通过 `GeoWorldFacade` 得到的结果与直接调用 `GeoWorldQueryService` 的结果 `equals` 相同。这一项依赖第 3 节的 S4 |

QUERY-09 的样本补充：已提交的 6 个旧 golden 没有 `Floodplain / Alluvial Plain` 节点。若固定网格与特殊点没有符合条件的陆地点，验收器会从同一模型选择两个不同的近河干陆节点，仅在临时测试副本中分别指定这两个地形标签，再经持久化加载与真实 QueryService 检查洪水风险方向；原文件的字节必须不变。输出明确区分真实洪泛样点与受控输入，这不代表世界生成器已产生或验证洪泛平原。

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

- [x] `NaturalField` 数量与 `FACT_REGISTRY.md` 一致（QUERY-01 自动检查）
- [x] QUERY-01 ~ QUERY-10、MODEL-01 ~ MODEL-04 全部通过
- [x] 如果字段的语义、单位、取值域或缺失条件有变化：已按 D-049 第 2 节判断是否属于 breaking 变更，属于的话 `API_VERSION` 已提升，并在 D-049 下追加了变更记录
- [x] `FACT_REGISTRY.md` 的更新时间与本次发布一致
- [x] `STATUS.md` 记录了本次 Query 相关改动，并且区分了「已验证」和「未验证」

- [x] 1.21.1 NeoForge 完整 `check`（JDK 21）
- [x] 26.3 Fabric 完整 `check`（JDK 25）
- [x] 1.21.1 NeoForge 实机：进入 A → 返回标题 → 进入 B，记录 `currentWorld` 为 true → false → true 与 A/B Seed
- [x] 26.3 Fabric 实机：同上

## 6. 2026-11-07 实现与自动验收记录

| 切片 | 当前结论 |
|---|---|
| QUERY-01～09 | `47f1c03`，6 个模型、每模型 1,042 点；86 字段全部出现 |
| S1 | `3d9472a`，模型持有者与三个版本族的共同生命周期已核实，详见 D-050 |
| S2 | `d694678`，`GeoWorldFacade / WorldFacts` 已实现，绑定与模型本体留在内部包 |
| S3 | `8e74df5`，共享 `GeoWorldV26WorldService` 在动态主世界激活后 bind，关闭/失败/重新激活时 unbind；后续补充日志打印实际 `currentWorld` 状态。两个目标实机均通过，见第 7 节 |
| S4 | `e9098ad`，QUERY-10、生命周期黑盒检查、所有 9 个发布目标共用的 Gradle 检查已接入并通过双目标完整检查 |
| S5 | 2026-11-07，双目标实机通过，第 5 节全部完成；D-048 已写入解封记录，README Query API 说明定稿 |

自动结果：QUERY-01～10 与现有 24 项验收合计 34/34，在 6 个模型上通过；生命周期覆盖 A→empty→B、迟到关闭、同 owner 重新绑定、保留旧视图、不可变结果、4 个线程读取与并发切换。Go 全仓测试与 6 个模型共 108 个 Java/Go golden 样点通过。最初 Gradle 运行误用了系统默认 JDK 27；实际验证固定为上表的 JDK 21 / 25。

完整检查中修正了两项既有验收基线问题，**没有修改生成算法或既有 golden 文件**：

1. 群系连续性检查在邻点沿用中心 Y，坡地上会落入地下。原始 `386aace` 也复现 `stony_shore → taiga` 断言；现让地表群系的邻点跟随当地地表高度，地下群系仍保留固定代表高度，连续性断言本身保留。
2. 26.3 指纹常量仍引用旧版结果。已分别编译未改动的 `386aace` 源码快照和 Query 候选，在相同两 Seed、各 24,576 个样点上独立重算，确认二者一致后，将测试常量锁定到本轮开始前就已存在的结果：

| Seed | 原始 `386aace` | Query 候选 |
|---|---|---|
| `123456` | `7cad8c0a851e21bc` | `7cad8c0a851e21bc` |
| `-563659577384754347` | `1d630f1f82c93fb8` | `1d630f1f82c93fb8` |

验收证据输出在两个目标各自的 `build/query-api-evidence.json`；原有 177 项 Go 验收表未改，新增 QUERY 项登记在 Java 补充表并写入证据的 `queryPassedIds`。

## 7. 2026-11-07 双目标实机与 S5 解封记录

用户使用已安装的验收候选包完成「进入旧世界 A → 返回标题 → 创建并进入新世界 B」，随后读取两个 HMCL 实例的 `latest.log`。下列时间均为 Asia/Shanghai；两个目标都还记录了退出 B 后恢复 false。

| 目标 | A 绑定 true：时间 / Seed / rules | 退出 A：false | B 绑定 true：时间 / Seed / rules | 退出 B：false |
|---|---|---|---|---|
| 1.21.1 NeoForge | 09:31:40.790 / `-4250254937638407070` / `geoworld-rules-v1.1.16` | 09:31:57.704 | 09:32:23.154 / `987654321` / `geoworld-rules-v1.1.16` | 09:33:08.964 |
| 26.3 Fabric | 09:34:40 / `123456` / `geoworld-rules-v1.1.14` | 09:35:11 | 09:35:32 / `987654321` / `geoworld-rules-v1.1.16` | 09:36:16 |

两个世界的 `profile` 都是 `geoworld-profile-v1.1`，`model` 都是 `geoworld-model-v1`。26.3 的旧世界保留自己的 rules v1.1.14，新世界使用 v1.1.16；没有升级或重生成旧存档。已检查生命周期附近的 GeoWorld 错误，未发现相关失败。

- NeoForge 原始日志：`D:/games/hmcl/.minecraft/versions/1.21.1-NeoForge/logs/latest.log`。
- Fabric 原始日志：`D:/games/hmcl/.minecraft/versions/26.3-Fabric/logs/latest.log`。
- [已提交的实机证据](../acceptance/evidence/query-api-world-switch-20261107.json) 保存原始生命周期行、日志路径/编码/SHA-256、A/B Seed 和安装包 SHA-256，避免 latest.log 被下次启动覆盖后丢失本次结论。包名与安装备份见 `HANDOFF_QUERY_API.md`。
- 实机覆盖集成服务器的两个要求目标；其余 7 个适配器未在本轮重建，未另做专用服务器实测。

第 5 节全部勾选后完成 S5：D-048 记录 2026-11-07 解封及验收实现提交 `e9098ad7970de132b8c3d4767584ef600b67c36c`；[runtime-reader README](../java/runtime-reader/README.md#query-api) 的对外说明定稿，字段表、架构、状态和交接同步。此结论是 Query API v1 的接口验收；本轮没有推送、打 tag 或发布模组。

S5 文档验证：发布检查表 9/9、字段表 86/86 V、DECISIONS 的 CRLF、文档相对链接、两份日志与安装包哈希全部通过；README Java 示例已使用 JDK 21、`--release 21` 和本轮 NeoForge 验收 JAR 实际编译通过。

## 8. 完整发布矩阵补充（2026-11-07）

`v0.1.2` 标签流水线 [42328096662](https://github.com/wanstu/minecraft-geo-world/actions/runs/42328096662) 的 Go / 全量 golden 和 5 个适配器通过，26.1.2 / 26.2 的两个 Loader 均在旧群系指纹常量处失败，Release 被阻止。保留该失败标签，后续修正使用 `v0.1.3`。

已核对临时源码快照与 `386aace` 的生成相关源码一致（比较时只归一化换行）；使用此前独立编译的原始快照与候选类，在与各版本相同的 MinecraftV26Profile 能力参数下复算。每个版本每个 Seed 24,576 点，结果如下；[结构化证据](../acceptance/evidence/release-biome-fingerprints-20261107.json) 已保存。

| Minecraft | Seed | 原始 386aace 与候选共同 FNV64 |
|---|---|---|
| 26.1.2 | `123456` | `599b1ee71dc2825c` |
| 26.1.2 | `-563659577384754347` | `f3ebcfafda40fdd8` |
| 26.2 | `123456` | `6eff9f92530b16e6` |
| 26.2 | `-563659577384754347` | `df976bd29eaf4429` |

只更新两个版本的指纹测试常量及说明，不跳过检查、不改变生成算法、既有 golden、字段或规则版本。其余 5 个目标的成功也包括 1.20.1 Fabric 的 Java 17 实际构建。
