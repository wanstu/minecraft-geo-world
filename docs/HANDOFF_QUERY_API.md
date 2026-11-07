# GeoWorld Query API 当前交接

更新时间：2026-11-07。仓库：`D:\projects\minecraft-geo-world`，分支 `master`；ADM：`env_59c743bd89627fe7`。

## 1. 现在的结论

**Query API v1 验收完成，S1～S5 全部完成，D-048 已解除封控。** 用户已操作两个要求目标完成实机世界切换，日志证据已保存；无需再重复这两项人工步骤。当前没有约定新的功能开发范围。

已单独提交的切片：

| 提交 | 内容 |
|---|---|
| `cd2e9fc` | 最初 8 份文档基线 |
| `47f1c03` | QUERY-01～09 与字段契约修正、回填 |
| `3d9472a` | S1 模型持有者及生命周期调查 |
| `d694678` | S2 只读门面和内部绑定 |
| `8e74df5` | S3 共享世界服务接入 |
| `e9098ad7970de132b8c3d4767584ef600b67c36c` | S4：QUERY-10、生命周期、9 个目标共同 Gradle 检查、验证记录及验收器修正 |
| `3ed888bc` | 安装验收候选包与实机操作交接 |
| S5 文档提交（见 git log） | 双目标实机证据、README 对外说明、D-048 解封及最终状态同步 |

Query 验收阶段未推送或发布；随后用户已授权正式发布。完整矩阵发现的 26.1.2 / 26.2 旧指纹基线已独立复算并修正，发布进度以 STATUS 与 QUERY_API_ACCEPTANCE §8 为准，最终标签使用 v0.1.3。

## 2. 先读什么

1. [Query API 调用说明](../java/runtime-reader/README.md#query-api)：可用范围、兼容性、缺失语义与 Java 21 示例。
2. `docs/QUERY_API_ACCEPTANCE.md`：完整检查表、自动结果和 §7 实机时间 / Seed。
3. `docs/FACT_REGISTRY.md`：86 个字段逐行契约，全部已回填自动验收 V。
4. `docs/DECISIONS.md` D-048 / D-049 / D-050，尤其 D-048 解封记录与 D-050 生命周期结论。

字段/文档不一致时核对代码事实；不得改生成结果来迁就文档。

## 3. 实现位置与确定的生命周期

- 稳定读取入口：`java/runtime-reader/src/main/java/geoworld/runtime/api/GeoWorldFacade.java`；视图 `WorldFacts.java`。
- v1 请求/结果继续使用现有 `GeoWorldQueryService` 的四个嵌套值类型（NaturalField / Resolution / QueryRequest / NaturalResult）；不要迁移、复制或重命名字段。
- 内部绑定：`geoworld.runtime.GeoWorldFactsBinding`。弱 owner 标识，原子读取快照；`currentWorld()` 不采样、不等待、不访问区块或磁盘。supports 的 7 个模型级可选字段仅在 bind 时做中心探测，避免区域聚合。
- 动态模型由 `GeoWorldV26GeneratorRuntime.model` 持有；引擎 GeneratedModel 只是返回值。`GeoWorldV26WorldService.activateDynamicWorld` 在加载/附加/登记完成后 bind；activate 开始、失败与 deactivate 都移除旧绑定。1.20.1、1.21.1、26.x 都委托这一个服务，不在 9 个适配器复制。
- 普通世界和静态 QA Preview 不发布 WorldFacts；下界/末地不提供另一套模型，currentWorld 始终指主世界。只提供逻辑服务端事实。
- QUERY-01～10 在 `GeoWorldQueryContractVerifier` 中，由 `GeoWorldModelVerifier` 执行。生命周期入口为 `GeoWorldFacadeLifecycleVerifier`。
- 全部 9 个目标应用 `java/query-api-verification.gradle`；证据在各自的 `build/query-api-evidence.json`。原始 177 项表不改，QUERY 用 Java 补充表及 `queryPassedIds`。

## 4. 已实际验证

- Go 全仓测试：`go test -p 2 ./...`，仅限制并行度，通过。
- Java/Go golden：6 个已提交模型，108 个样点，容差 1e-9，通过。
- Query + 旧模型验收：34/34；每模型 32×32 网格 + 18 特殊点，共 6,252 个主采样点；86 字段均实际出现。
- 生命周期：A→empty→B、迟到 stop A 保留 B、同 owner 重新绑定、旧视图稳定、结果不可变、4 个线程读取与切换，通过。
- 完整 `check`：1.21.1 NeoForge 使用 JDK 21；26.3 Fabric 使用 JDK 25，均通过。系统默认 JDK 27 不适合 NeoForge 的 Gradle 8.10.2。
- 双目标实机：用户完成进入 A → 标题 → 进入 B，日志均为 true → false → true，退出 B 后也恢复 false。证据见下一节。
- 其余 7 个目标只接入了共同检查，未在本轮本地重建；没有另做专用服务器实测。

验收器曾有两处旧问题，已在原始 `386aace` 快照核对：地表邻点不能沿用中心 Y；26.3 指纹常量需锁定到本轮之前就存在的实际基线。两 Seed 的旧基线/候选指纹一致，详情在验收主文档。没有改世界生成或已提交 golden 文件。

旧 golden 不含 Floodplain / Alluvial Plain 节点。QUERY-09 明确使用临时持久化副本的两个受控地形标签检查负向洪水风险，原文件必须逐字节不变；这不代表旧 golden 覆盖了真实生成的洪泛平原。

## 5. 实机证据与已安装包

时间为 2026-11-07，Asia/Shanghai：

| 目标 | A Seed / 绑定时间 | 返回标题后 false | B Seed / 绑定时间 | 最后退出 B |
|---|---|---|---|---|
| 1.21.1 NeoForge | `-4250254937638407070` / 09:31:40.790 | 09:31:57.704 | `987654321` / 09:32:23.154 | 09:33:08.964 false |
| 26.3 Fabric | `123456` / 09:34:40 | 09:35:11 | `987654321` / 09:35:32 | 09:36:16 false |

26.3 的旧世界 rules 保留 `geoworld-rules-v1.1.14`，新世界为 `geoworld-rules-v1.1.16`。两个世界没有共享旧绑定；未升级或重生成用户旧存档。原始生命周期行、日志哈希/编码和安装包哈希已保存到 [实机证据](../acceptance/evidence/query-api-world-switch-20261107.json)。D-048 解封所指的验收实现提交为 `e9098ad7970de132b8c3d4767584ef600b67c36c`，S5 单独保存文档与证据。

验收包目录：`D:\projects\minecraft-geo-world\output\query-api-acceptance`。2026-11-07 09:22 已安装到两个 HMCL 实例；实机后再次核对 SHA-256 一致。旧包备份在 `D:/games/hmcl/geoworld-backups/query-api-20261107-092237/`。JAR 已检查包含门面、视图、内部绑定和生命周期诊断。

| 文件 | SHA-256 |
|---|---|
| `GeoWorld-0.1.1-dev-query-api-mc1.21.1-neoforge.jar` | `a7d217ca33fa977e4a2f493758c6f4e0ced86b8ffd941348a4c0f629eedce63c` |
| `GeoWorld-0.1.1-dev-query-api-mc26.3-fabric.jar` | `ce7dcb2576cb99f0bd2cc7944b806075d3926f0dc273ddb45e29c7b6b85dc109` |

自动验证日志：`.tmp/query-api-validation-20261107/`，包括双版本完整检查、Go/golden 和改动前后指纹观测；只读核验可直接读取这些文件。

后续继续保持：字段、单位、缺失条件、生成结果、generationRulesVersion、缓存号及 golden 文件不变；不擅自加入区域查询、玩法或 LocalReality Provider。DECISIONS 保持 CRLF；只提交相关文件，不 force push、不 amend。
