# GeoWorld v0.1.5 RC — 九目标本地构建阶段验收（2026-11-12）

分支：`adm/geoworld-r81f-spawn-height-20261112`  
起点提交：`6591da7`（包含已验收 R8.1e 地貌规则及 R8.1f 出生点高度修复）  
当前生成规则：`geoworld-rules-v1.1.25`  
**结果：本地九目标配置预检 9/9，通过完整 `clean check build` 的目标 2/9，仍不具备发布资格。**

## 本轮发现并修复的真实 CI 阻塞

对 **26.3 Fabric** 运行 `clean check build` 时，构建在 `verifyGeoWorldSpecialLandscapes` 失败：

```text
java.lang.AssertionError: depression center is not flat:
kind=sinkhole center=0.6140801788850474 inner=0.6299313323249379
```

**已查证原因：** R8.1e 在 **v1.1.25** 正式让天坑从中心到坑缘连续抬升，属于预期改变；但 Java 验收程序还在对所有天坑执行 `center` 与半径 30% 内坡高度差必须不超过 0.015 的**旧规则**断言。被测新规则中心约 0.61408、内坡约 0.62993，差值约 0.01585。也就是说，测试误把连续坡面当作错误。

只做如下范围有限的修复：

- `java/minecraft-common/src/main/java/geoworld/minecraft/common/verification/GeoWorldSpecialLandscapeVerifier.java`：仅对 `kind=sinkhole` 且持久化规则为 `geoworld-rules-v1.1.25` 的世界，改为严格验证中心→30% 内坡→55% 中坡→92% 外坡**逐级抬升**，并对各段高差设置上下限，防止新平底或新陡壁。
- **v1.1.24 及更早版本仍使用原平底检查**；普通盆地、雪山盆地与火山坑也继续使用旧检查。并未删除海岛水域、盆地深度或坑壁非均匀性检查。
- 将 `java/adapters/26.3-fabric/build.gradle` 中的特殊地貌验证任务说明更新为当前规则。
- **绝无实际地形生成算法修改，没有更换生成规则版本，也没有改动历史存档。**

额外读取之前真实保存的 v1.1.24 模型，验证其中 4 个旧天坑中心→内坡的高度差均在 0.00006 以内，仍满足旧平底语义；未让旧世界误走 v1.1.25 新断言。

修复后：该任务**单独复验通过**（6 个特殊地貌，其中 2 个天坑）；随后 26.3 Fabric 整套 `clean check build` **再次通过**。

## 本地构建结果

| Minecraft | Loader | 本轮验证结果 |
|---|---|---|
| **1.21.1** | **Fabric** | **`clean check build` 通过，17 个 Gradle 任务，最终重跑约 1m39s** |
| **26.3** | **Fabric** | **`clean check build` 通过，41 个 Gradle 任务，最终重跑约 5m03s** |
| 1.21.1 | NeoForge | 已有服务端真实世界验收及 Java 编译证据，本轮**未**完成 `clean check build` |
| 26.3 | NeoForge | 本轮未完成 `clean check build` |
| 26.2 | NeoForge / Fabric | 本轮未完成 `clean check build` |
| 26.1.2 | NeoForge / Fabric | 本轮未完成 `clean check build` |
| 1.20.1 | Fabric | 本轮未完成；dadm 本地缺少 JDK17，正式 CI 有 JDK17 配置 |

本轮验证均在 dadm/Linux 上执行，**并不是 GitHub Actions 的 Windows 正式 CI**；九目标静态配置 9/9 通过只代表矩阵和文件匹配，不能取代九个真实构建。

两份通过 `clean check build` 的本地 Fabric JAR 又通过独立 ZIP 验证：`fabric.mod.json`、`GeoWorldV1.class` 和 Windows/Linux/macOS × x64/ARM64 的**六套原生引擎**均齐全。正式发布不使用本地 JAR。

**注意：** 模块 Gradle 属性当前仍叫 `mod_version=0.1.4`；这只是对开发分支的本地全构建，生成的 `geoworld-0.1.4.jar` **不是**已经发布的 0.1.4 文件，绝不能拿它冒充正式版本。GitHub tag 发布会使用正式版本覆盖参数，并从相应 tag 重新构建。

## 已执行的具体检查

1.21.1 Fabric：

- `compileJava`、六平台引擎、`verifyGeoWorldAdapter`。
- `fingerprintGeoWorldBiomeDecisions`：24,576 点匹配。
- `verifyGeoWorldMiningMaterials`：9 个材料族、35 个矿区的抽查通过。
- 模型 Query 7 个历史模型、34/34 项契约和生命周期检查通过。
- 最终 `check`、`remapJar`、`assemble`、`build`：**通过**。

26.3 Fabric：

- Java 源码、六平台原生引擎及额外 Tool 跨平台构建完成。
- 生物群系指纹、矿物体积/连通性、河岸海岸过渡、固定地貌中心、Query 模型契约/生命周期通过。
- `verifyGeoWorldSpecialLandscapes` 通过、要塞/结构预览、结构适配与快照、表层群系、WebUI 启停等通过。
- 最终 `check`、`assemble`、`build`：**通过**。

## 可核验的固化证据

- `docs/acceptance/r815_rc_local_build_matrix_20261112.json`：九目标逐项状态和 JAR、日志 SHA256，声明正式 CI 未运行。
- `docs/acceptance/r815_rc_local_1211_fabric_build_evidence.txt` 与 `r815_rc_local_263_fabric_build_evidence.txt`：完成任务与关键 PASS 行及完整本机日志的 SHA256。
- `docs/acceptance/r815_rc_fabric1211_jar_inspection.json`、`r815_rc_fabric263_jar_inspection_current.json`：Loader 元数据、六原生引擎及 JAR SHA256。
- `scripts/test_r815_rc_local_build_matrix.py`：验证不会将 2/9 本地通过写成 9/9 正式 CI 通过。

## 下阶段明确门槛

剩余 **7 个版本/Loader 的完整构建**、正式 GitHub Windows 九目标 CI，以及真实 Minecraft **客户端游戏相机**对出生点和关键地貌的观感验收。真实方块的 Anvil 投影图不能冒充客户端截图。

目前保持 **HOLD：不推送、不合并 master、不打 tag、不发布 JAR**。后续先继续构建剩余版本，不重新修改已通过的 R8.1e 天坑坡面或 R8.1f 出生点算法。
