# GeoWorld R8.1l — Windows CI 验证与发布隔离

日期：2026-11-12，dadm；隔离分支：`adm/geoworld-r81l-ci-preflight-20261112`  
源基线：`dd5451405d5b9884d2f8db9277b99e3b96efd42c`，生成规则：`geoworld-rules-v1.1.26`。

## 本阶段结论

- R8.1k 已完成当前规则 **9/9 Linux dadm 本地 `clean check build`**，且有逐项构建和 JAR 检查证据。本轮没有重新构建九目标，也不把旧证据说成 Windows CI。
- 在本轮隔离工作树复跑五组证据测试，共 **39 项 PASS**；复跑特定 Go 水文版本测试通过；`r815_rc_build_matrix_preflight.py` 确认 **9/9 目标配置和本地 JDK 就绪**。
- 检查 GitHub Actions：Windows `implemented-adapters` 与 `release-build` 都有相同的九目标矩阵，Go+Java `core-tests` 在其前置。
- **修复一个 CI 发布隔离风险：** 原来的 `release-build` / `publish` 只判断 ref 是否为 `refs/tags/v*`，即便是 `workflow_dispatch` 手动选择标签也可能触发发布。现在两个任务均要求 `github.event_name == 'push'` **并且** ref 为发行标签；手动触发总是运行九目标验证，不运行发布。
- 新增 `scripts/test_r81l_windows_ci_release_isolation.py`，**8/8 PASS**，锁定手动触发与标签推送分流、发布权限范围、九目标矩阵一致性。
- 本轮**不改生成规则、Go/Java 地貌代码、存档兼容策略**；不合并、不推送、不打 tag、不发布 JAR。

## 发布安全流程

| 触发事件 | Windows 九目标验证 | 正式发布任务 |
|---|---|---|
| `workflow_dispatch`，普通分支 | 运行 | 不运行 |
| `workflow_dispatch`，旧发行标签 | 运行 | 不运行 |
| `pull_request` | 运行 | 不运行 |
| `push` 到 `master` | 运行 | 不运行 |
| `push` 发行标签 `v*` | 不运行，改走发布构建 | 运行（需经核心测试、发行构建） |

`modrinth-publish.yml` 是独立工作流，由 GitHub Release `published` 事件或明确手动触发启动，本轮未修改。**不可把本轮本地静态测试解释为 GitHub 官方 Windows CI 通过。**

## 实际游戏视角验收仍受阻

dadm 有 `DISPLAY=:0`，但读取 `xdpyinfo` 返回 `Invalid MIT-MAGIC-COOKIE-1 key`，当前 ADM 会话未取得图形会话访问权限；未找到 `minecraft-launcher`、`prism-launcher` 或 Xvfb。没有强行干扰现有图形会话、安装客户端或伪造 Minecraft 画面。

此前 R8.1g 的独立 NeoForge 新世界保存、Anvil 真实方块列检查为 **1517/1517**，相邻水面落差超限为 0；但这**不替代真实 Minecraft 客户端相机视角**。

## 后续发布门槛

1. 审核本隔离分支的 CI 安全修正；**只有获得明确许可后才推送分支**并在 GitHub 手动运行 `GeoWorld release` 的非发布验证矩阵，逐个取得 Windows 9/9 构建结果。
2. 在具有可登录图形会话和 Minecraft 客户端的机器上，使用 v1.1.26 开发 JAR 建立新世界，按已记录种子和河口、天坑、海岛坐标拍摄真实游戏截图，与 Anvil 专项验收分开登记。
3. 两项门槛完成前，**v0.1.5 RC = HOLD**。本轮原生 JAR 的元数据仍是开发用 `mod_version=0.1.4`，不得误作为发布版本。
