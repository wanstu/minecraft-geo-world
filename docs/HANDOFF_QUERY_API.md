# 交接：GeoWorld Query API 契约化与门面落地

交接日期：2026-11-06
仓库：`D:\projects\minecraft-geo-world`，分支 `master`
基线提交：`386aace`（Fix runtime mineral discovery and dimension-aware WebUI）
ADM 环境：`env_59c743bd89627fe7`（可选；允许直接操作仓库目录）

---

## 1. 背景与目标

GeoWorld 是**世界地基**：它只负责产出确定、可复现的自然地理事实，不提供任何玩法。
后续的玩法模组（季节、聚落、航运、矿业等）都会通过 Query API 读取这些事实。

目前的问题是：`GeoWorldQueryService`（`geoworld-query-v1`，86 个字段）已经写进了代码，但**没有登记、没有契约、没有完整验收，也没有稳定入口**。
本轮已经用文档把这几件事补齐。**下一步是按文档实现。**

本轮只写了文档，没有动代码，也没有运行任何构建或测试。

---

## 2. 必读文档（按顺序）

| 顺序 | 文件 | 用途 |
|---|---|---|
| 1 | `docs/QUERY_API_ACCEPTANCE.md` | **执行主文档**：验收项 QUERY-01~10、门面切片 S1~S5、不做清单、发布检查表 |
| 2 | `docs/FACT_REGISTRY.md` | 86 个字段的权威契约：类型、单位、缺失条件、附录 A 取值域、附录 B 依赖 |
| 3 | `docs/DECISIONS.md` D-048 / D-049 / D-050 | 封控规则、版本升级规则、门面暴露方式 |
| 4 | `docs/FOUNDATION_API.md` | 背景、分层模型、倾向性原则（只需理解，不需要执行） |
| 5 | `docs/PRODUCT.md`「永不提供」一节 | 硬边界 |

文档和代码不一致时：**先核对代码中的事实**；如果确实是文档写错，修正文档，并在 `STATUS.md` 中注明。不要为了迁就文档而去改字段语义。

---

## 3. 本轮已产出的文件（未提交）

```text
M  docs/ARCHITECTURE.md   新增 §7 对外事实查询层
M  docs/DECISIONS.md      追加 D-048 / D-049 / D-050（CRLF，追加时保持一致）
M  docs/PRODUCT.md        新增「永不提供」硬边界
M  docs/STATUS.md         记录本批文档改动（标注仅文档、未验证）
?? docs/FOUNDATION_API.md
?? docs/FACT_REGISTRY.md
?? docs/QUERY_API_ACCEPTANCE.md
?? docs/HANDOFF_QUERY_API.md（本文件）
```

建议接手后先把以上文件提交为一个独立的 docs 提交，再开始写实现，这样文档基线和实现提交能分开。

---

## 4. 执行顺序

严格按以下顺序进行，**每个切片单独提交**：

1. **QUERY-01 ~ QUERY-09**：写进 `GeoWorldModelVerifier`，不依赖门面，可以先做。
   - 验收通过后，把 `FACT_REGISTRY.md` 中对应字段的验收列回填为 `V`，并在附录 C 注明对应的 QUERY 编号。
2. **S1 调查**：查清楚动态世界运行时 `GeoWorldV1` 由谁持有，以及它的创建点和失效点，结论写进 D-050 的「遗留」小节。**先完成 S1 再开始写门面。**
3. **S2 门面**：在 `geoworld.runtime.api` 包中实现 `GeoWorldFacade` / `WorldFacts`。只暴露 S2 中列出的方法，不暴露 `GeoWorldV1`。
4. **S3 接入**：bind / unbind 写在版本族共享层，不要在 9 个适配器里各写一份。
5. **S4 验收**：完成 QUERY-10 和生命周期检查，并加入各适配器的 `check` 任务。
6. **S5 解封**：确认发布检查表全部完成后，在 D-048 下追加封控解除记录，并给 `runtime-reader/README.md` 新增「Query API」一节。

---

## 5. 关键代码位置

| 内容 | 路径 |
|---|---|
| 查询实现、`NaturalField` 枚举（约第 56–147 行）、字段分派（约第 585–968 行） | `java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldQueryService.java` |
| 黑盒验收入口（`VERIFIED_IDS`） | `java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldModelVerifier.java` |
| 验收登记表 | `java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldAcceptanceRegistry.java` |
| 模型、取值域常量（`ECO_REGION_NAMES`、`TERRAIN_NAMES` 等） | `java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldV1.java` |
| 派生评分 | 同目录下 `GeoWorld{Navigation,Settlement,Civilization,RegionalAdvantage,TransportGeography,Geology}*.java` |
| 矿业分类（Go 侧） | `internal/mining/model.go`，`miningClass()` |
| S1 候选的模型持有者 | `java/minecraft-common/.../engine/GeoWorldEngineBridge.java`、`runtime/GeoWorldBinding.java`、`java/minecraft-v26-common` 与 `minecraft-v1_21_1-common` 下的 WorldService |
| Golden 模型 | `internal/goldenvectors/testdata/runtime-*.geoworld`（6 个） |
| 现有 Query 验收任务 | 各适配器 `build.gradle` 中的 `verifyGeoWorldModelQuery` |

---

## 6. 验证命令

```text
go test ./...
pwsh -NoProfile -File java/runtime-reader/verify-golden.ps1 -All
java/adapters/<target>/gradlew.bat -p java/adapters/<target> check
```

- 至少在 `1.21.1-neoforge` 和 `26.3-fabric` 两个目标上运行 `check`。26.3 NeoForge 模块是 `java/neoforge-mod`。
- 实机检查（S3）：在两个目标上手动走一遍「进入世界 → 返回标题 → 进入另一个世界」，记录 `currentWorld()` 的变化。
- 汇报结果时区分「已写入、已验证、已提交、已推送」。没有实际运行的，不要写成已通过。

---

## 7. 硬约束（违反即视为回退）

1. **不新增、不删除、不改名任何 `NaturalField`，也不改变已有字段的单位、取值域或缺失条件。** 需要变更时，必须先按 D-049 判断是否属于 breaking 变更。
2. **不改变世界生成结果。** 门面和验收都是只读的，不能动 `generationRulesVersion`、golden 文件或缓存版本号。
3. **查询不能产生副作用**：不能触发区块生成，不能写盘，不能阻塞世界线程（MODEL-03 / MODEL-04 必须继续通过）。
4. **门面只提供主世界、逻辑服务端的事实**；未就绪时返回 `empty`，不抛异常。
5. **本轮不做**：区域查询原语、`LocalRealityProvider` 的具体实现、季节风等新字段、命名风格统一。
6. **GeoWorld 不提供玩法**（见 `PRODUCT.md`）：不新增群系、物品、生物、结构或视觉表现。
7. 修改 `DECISIONS.md` 时保持 CRLF 换行；其他 docs 文件保持各自原有的换行风格。
8. Git：不 force push，不 amend 已有提交，只提交相关文件。推送前要获得用户确认。

---

## 8. 已知风险与提示

- **字段数**：目前是 86 个，不是早期草稿里写的「100+」。现有 Verifier 只引用了其中 23 个字段。
- **取值域有两种命名风格**：附录 A.1–A.4 是带空格的标题式写法（例如 `Alluvial Plain`），A.6–A.11 是蛇形小写写法。这是历史遗留，**不要统一**。
- **QUERY-01 要读取 md 文件**：Verifier 需要找到 `docs/FACT_REGISTRY.md` 的路径，可以通过参数传入，也可以基于仓库根目录推导。CI 中的工作目录是仓库根目录。
- **QUERY-05 的成本**：查询 `REGIONAL_REGION_*` 会触发跨节点区域聚合，必须使用 `BLOCK_16`，并控制采样数量。
- **S1 是本轮最大的不确定点**：目前还没有公开路径能拿到运行时的 `GeoWorldV1`。如果调查发现必须调整 `GeoWorldEngineBridge` 的结构，先在 D-050 中写清楚方案，再实现。
- **多世界与集成服务器**：单人游戏中返回标题界面后再进入另一个世界，最容易残留旧的绑定。S4 的生命周期检查正是为这种情况设计的。
- 仓库里 `.tmp/`、`build/`、`geoworld-cache/` 下有大量本地产物，都已被 `.gitignore` 忽略，不要提交，也不要删除。

---

## 9. 完成定义

- `QUERY_API_ACCEPTANCE.md` 第 5 节的发布检查表全部勾选；
- D-048 下已有封控解除记录（含日期和提交）；
- `FACT_REGISTRY.md` 的验收列中，所有被 QUERY 覆盖的字段都已回填；
- `STATUS.md` 中有本轮实现记录，并且区分了已验证和未验证的部分。
