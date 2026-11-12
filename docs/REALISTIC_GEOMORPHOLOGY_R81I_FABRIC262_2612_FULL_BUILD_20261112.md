# GeoWorld R8.1i — v1.1.26 26.2 / 26.1.2 Fabric 完整构建验收

日期：2026-11-12（dadm / Debian Linux）  
源提交：`4a64827ca302d31285f1e84075d71bddc1a7bb77`  
分支：`adm/geoworld-r81f-spawn-height-20261112`  
生成规则：`geoworld-rules-v1.1.26`

**结论：本阶段新增两个 v1.1.26 的 Fabric 完整构建 PASS。连同此前完成的 1.21.1 NeoForge、1.21.1 Fabric、26.3 Fabric，本地九目标真实完整构建现在通过 5/9；正式 Windows CI 与真实游戏客户端相机验收仍未完成，RC 保持 HOLD。**

## 1. 实际执行结果

| 游戏版本 | Loader | 本机 JDK | 完整构建结果 | 原生 JAR |
|---|---|---|---|---|
| **26.2** | **Fabric** | 25 | **BUILD SUCCESSFUL in 2m 18s**；15 项任务（14 执行、1 UP-TO-DATE） | 元数据 / 6 平台引擎 / v1.1.26 字节码均通过 |
| **26.1.2** | **Fabric** | 25 | **BUILD SUCCESSFUL in 2m 1s**；15 项任务（14 执行、1 UP-TO-DATE） | 元数据 / 6 平台引擎 / v1.1.26 字节码均通过 |

执行完整 `clean check build`，并非仅调用 `compileJava` 或离线预检。为避免影响 dadm 上的其他开发任务，Gradle 使用 `--no-daemon --max-workers=1`、JVM 堆上限约 1150 MB；JDK 25。

**26.2 Fabric 离线首试的真实情况：** Gradle Wrapper 首次下载 9.5.1 成功，但离线缺失 `Fabric Loom 1.17-SNAPSHOT`，配置阶段报插件不可用。之后联网解析成功，实际插件版本为 `Fabric Loom 1.17.21`，完整 `clean check build` 通过。不能把首次插件缺失记成源码失败。

26.1.2 Fabric 直接联网启动同一组构建任务，也使用 Loom 1.17.21 并最终通过。

两项构建均完成：
- `compileJava`、6 个原生引擎、`verifyGeoWorldAdapter`；
- 24,576 点生物群系决策指纹 PASS；
- 七个历史模型的 Query 验证 **34/34** 项 PASS；
- Query 生命周期/读写隔离与并发快照测试 PASS；
- 最终 `check`、`assemble`、`build` PASS。

## 2. JAR 字节检查

| 模块 | 文件字节数 | SHA256 |
|---|---:|---|
| 26.2 Fabric | 9,164,484 | `742f7ddd6e4aa16c131c5218bd3ddf3c025259cb763e90b576ba480c65c3a118` |
| 26.1.2 Fabric | 9,164,618 | `3c2a4b85cfa41fd5006de1748815717deb31a1a7463ac744f71da63f0ab2a840` |

独立解包检查确认两个 `fabric.mod.json` 都声明 `id=geoworld`，六套 Windows/Linux/macOS x64/ARM64 原生引擎全都存在，未缺资源项。读取两份 JAR 的实际 `GeoWorldEngineBridge.class` 字节码，检测到 `geoworld-rules-v1.1.26` 且不含旧的 `v1.1.25` 握手常量。

**注意：** 源分支模块默认 `mod_version=0.1.4`，本地 JAR 名称都是 `geoworld-0.1.4.jar`；它们是新规则开发构建，**不是正式 v0.1.4，也不是允许分发的 v0.1.5 RC 文件**。本轮只提交验收数据，不把 JAR 纳入 Git。

## 3. v1.1.26 当前九目标状态

| Minecraft | Loader | v1.1.26 本地完整 `clean check build` |
|---|---|---|
| **26.3** | **Fabric** | **PASS**（R8.1h） |
| **26.2** | **Fabric** | **PASS**（本轮） |
| **26.1.2** | **Fabric** | **PASS**（本轮） |
| **1.21.1** | **Fabric** | **PASS**（R8.1h） |
| **1.21.1** | **NeoForge** | **PASS**（R8.1g） |
| 26.3 | NeoForge | RESOURCE_BLOCKED：本机首次反编译消耗过多内存/Swap，已取消 |
| 26.2 | NeoForge | NOT_FULLY_CHECKED |
| 26.1.2 | NeoForge | NOT_FULLY_CHECKED |
| 1.20.1 | Fabric | NOT_FULLY_CHECKED：dadm 缺 JDK17 |

**5/9 PASS、1/9 机器资源阻塞、3/9 未完整检查。** 以上均是 Linux 上的本地构建结果，不是 GitHub Actions Windows 正式九目标 CI 结果，也不代表所有版本都通过游戏实际加载与视觉验收。

## 4. 固化证据

- `docs/acceptance/r81i_two_fabric_full_builds.json`：两个新版本的真实构建结果、可核验日志 SHA256、JAR SHA256、编译规则标记。
- `docs/acceptance/r81i_fabric262_v126_build_evidence.txt` 和 `r81i_fabric2612_v126_build_evidence.txt`：执行任务、关键 PASS 行和原始日志哈希。
- `docs/acceptance/r81i_fabric262_v126_jar_inspection.json` 和 `r81i_fabric2612_v126_jar_inspection.json`：完整 JAR 检查。
- `docs/acceptance/r81i_current_rules_local_matrix.json`：本轮更新后的**当前**九目标状态，发布标志固定为 `false`。
- `scripts/test_r81i_fabric_full_build_evidence.py`：回归约束不得宣称官方 CI 已通过或发布资格已通过。

本轮没有改世界模型、水文、地貌、出生点或历史存档，**不合并、不推送、不打 tag、不发布 JAR**。

## 5. 下一阶段的可执行边界

优先处理 **26.2 / 26.1.2 NeoForge** 的依赖与构建资源，再考虑 26.3 NeoForge。对高内存首次反编译应避免在 8 GB dadm 上重复无上限尝试；更适合在具备资源限制的构建机或正式 Windows CI 验证。1.20.1 Fabric 需先解决 JDK17 工具链。

**尚未完成：** 九目标官方 CI、客户端画面及整版真实地貌观感验证。RC 继续 HOLD。


## R8.1j 后续状态（2026-11-12）

本文中 **5/9** 是 R8.1i 阶段快照。R8.1j 已新增通过 **26.2 NeoForge、26.1.2 NeoForge**，因此当前规则 v1.1.26 的**最新本地完整构建进度为 7/9**。两项都使用独立 cgroup 内存限额，真正执行 `clean check build` 并通过，包含零水文异常和 JAR 完整性检查。

详细报告 `docs/REALISTIC_GEOMORPHOLOGY_R81J_NEOFORGE_FULL_BUILD_20261112.md`，最新九目标状态 `docs/acceptance/r81j_current_rules_local_matrix.json`。26.3 NeoForge 及 1.20.1 Fabric 尚未验收，正式 Windows CI、客户端画面验收与发布仍 HOLD。
