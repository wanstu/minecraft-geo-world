# GeoWorld Query API 当前交接

更新时间：2026-11-07。仓库：`D:\projects\minecraft-geo-world`，分支 `master`；ADM：`env_59c743bd89627fe7`。

## 1. 现在的结论

Query API 代码与自动验收已经落地，**仍未对外解封**。接下来直接补两个 Loader 的实机世界切换记录，然后才执行 S5。不要重做已通过的 QUERY-01～10，也不要把自动生命周期检查当成游戏客户端实测。

已单独提交的切片：

| 提交 | 内容 |
|---|---|
| `cd2e9fc` | 最初 8 份文档基线 |
| `47f1c03` | QUERY-01～09 与字段契约修正、回填 |
| `3d9472a` | S1 模型持有者及生命周期调查 |
| `d694678` | S2 只读门面和内部绑定 |
| `8e74df5` | S3 共享世界服务接入 |
| 当前 S4 提交（见 git log） | QUERY-10、生命周期、9 个目标共同 Gradle 检查、验证记录及验收器修正 |

本轮没有推送、打 tag 或发布。推送仍需用户明确指示。

## 2. 先读什么

1. `docs/QUERY_API_ACCEPTANCE.md`：当前执行状态、完整检查表与实机步骤。
2. `docs/FACT_REGISTRY.md`：86 个字段逐行契约，全部已回填自动验收 V。
3. `docs/DECISIONS.md` D-048 / D-049 / D-050，尤其 D-050 的 S1 结论。

字段/文档不一致时核对代码事实；不得改生成结果来迁就文档。

## 3. 实现位置与确定的生命周期

- 稳定读取入口：`java/runtime-reader/src/main/java/geoworld/runtime/api/GeoWorldFacade.java`；视图 `WorldFacts.java`。
- v1 请求/结果继续使用现有 `GeoWorldQueryService` 的四个嵌套值类型（NaturalField / Resolution / QueryRequest / NaturalResult）；不要迁移、复制或重命名字段。
- 内部绑定：`geoworld.runtime.GeoWorldFactsBinding`。弱 owner 标识，原子读取快照；`currentWorld()` 不采样、不等待、不访问区块或磁盘。supports 的 7 个模型级可选字段仅在 bind 时做中心探测，避免区域聚合。
- 动态模型由 `GeoWorldV26GeneratorRuntime.model` 持有；引擎 GeneratedModel 只是返回值。`GeoWorldV26WorldService.activateDynamicWorld` 在加载/附加/登记完成后 bind；activate 开始、失败与 deactivate 都移除旧绑定。1.20.1、1.21.1、26.x 都委托这一个服务，不在 9 个适配器复制。
- 普通世界和静态 QA Preview 不发布 WorldFacts；下界/末地不提供另一套模型，currentWorld 始终指主世界。
- QUERY-01～10 在 `GeoWorldQueryContractVerifier` 中，由 `GeoWorldModelVerifier` 执行。生命周期入口为 `GeoWorldFacadeLifecycleVerifier`。
- 全部 9 个目标应用 `java/query-api-verification.gradle`；证据在各自的 `build/query-api-evidence.json`。原始 177 项表不改，QUERY 用 Java 补充表及 `queryPassedIds`。

## 4. 已实际验证

- Go 全仓测试通过：`go test -p 2 ./...`，仅限制并行度。
- Java/Go golden：6 个已提交模型，108 个样点，容差 1e-9，通过。
- Query + 旧模型验收：34/34；每模型 32×32 网格 + 18 特殊点，共 6,252 个主采样点；86 字段均实际出现。
- 生命周期：A→empty→B、迟到 stop A 保留 B、同 owner 重新绑定、旧视图稳定、结果不可变、4 个线程读取与切换，通过。
- 完整 `check`：1.21.1 NeoForge 使用 JDK 21；26.3 Fabric 使用 JDK 25，均通过。系统默认 JDK 27 不适合 NeoForge 的 Gradle 8.10.2，不要据此误判代码失败。
- 其余 7 个目标只接入了共同检查，未在本轮本地重建。

验收器曾有两处旧问题，已在原始 `386aace` 快照核对：地表邻点不能沿用中心 Y；26.3 指纹常量需锁定到本轮之前就存在的实际基线。两 Seed 的旧基线/候选指纹一致，详情在验收主文档。没有改世界生成或已提交 golden 文件。

旧 golden 不含 Floodplain / Alluvial Plain 节点。QUERY-09 明确使用临时持久化副本的两个受控地形标签检查负向洪水风险，原文件必须逐字节不变；这不代表旧 golden 覆盖了真实生成的洪泛平原。

## 5. 唯一剩余验收与解封步骤

两个目标分别用本轮验收候选包操作，不要删除或改写用户存档：

1. 进入动态 GeoWorld 世界 A，日志应为 `GeoWorld Query bound Overworld facts: currentWorld=true, seed=A...`。
2. 返回标题，日志应为 `Released GeoWorld model and Query facts ... currentWorld=false`。
3. 进入不同 Seed 的动态世界 B，日志再次为 true，Seed 对应 B。
4. 在验收主文档记录目标、时间、A/B Seed 和对应日志，勾选两项人工检查。
5. 全部检查齐全后才在 D-048 **另行**写封控解除日期/提交，并将 `FOUNDATION_API.md` §7 的对外说明定稿到 `java/runtime-reader/README.md` 的 Query API 小节；同步 STATUS 后单独提交。

2026-11-07 09:22 已在确认 Minecraft 未运行后，为 1.21.1-NeoForge 与 26.3-Fabric 安装本轮验收包；SHA-256 与下表一致。两个实机步骤仍待人工进行，不能声称已经验证。只提供逻辑服务端主世界自然事实，不做区域查询、不实现玩法或 LocalReality Provider。保持字段、单位、缺失条件、生成结果、generationRulesVersion、缓存号及 golden 文件不变；DECISIONS 保持 CRLF；只提交相关文件，不 force push、不 amend。


### 验收候选包（2026-11-07 09:22 已安装）

目录：`D:\projects\minecraft-geo-world\output\query-api-acceptance`。JAR 已检查包含门面、视图、内部绑定和当前生命周期诊断；旧包已备份到 `D:/games/hmcl/geoworld-backups/query-api-20261107-092237/`，每个目标 mods 目录只有一个启用的 GeoWorld JAR。

| 文件 | SHA-256 |
|---|---|
| `GeoWorld-0.1.1-dev-query-api-mc1.21.1-neoforge.jar` | `a7d217ca33fa977e4a2f493758c6f4e0ced86b8ffd941348a4c0f629eedce63c` |
| `GeoWorld-0.1.1-dev-query-api-mc26.3-fabric.jar` | `ce7dcb2576cb99f0bd2cc7944b806075d3926f0dc273ddb45e29c7b6b85dc109` |

验证日志：`.tmp/query-api-validation-20261107/`（包括双版本完整检查、Go/golden 和改动前后指纹观测）；只读核验可直接读取这些文件。
