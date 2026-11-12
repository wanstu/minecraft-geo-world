# GeoWorld v0.1.5 RC 兼容性与真实方块俯视检查（2026-11-11）

> **当前状态：不可发布。** R8.1c 河口专项、R8.1d 天坑冲突专项通过，并不等于全部地貌、九构建目标和 Minecraft 游戏视觉已验收。
> 本记录为独立分支上的 **0.1.5-rc.4-dev 本地验证**，不是 release、不是已经发布的 RC。

基线：`2982977`，规则 `geoworld-rules-v1.1.24`。  
环境：dadm，Linux，8 GB 内存，独立 worktree `adm/geoworld-r81d-sinkhole-20261111`。

## 固定缺陷的当前证据

- **R8.1c 河口**：#509、#535、#851、#915 的关键固定点实际 Minecraft Anvil 采样符合预期。无全世界 88 河口的全部 Minecraft 块级验收结论。
- **R8.1d 天坑 #11 与河段 #735**：新旧两个 1.21.1 NeoForge 新世界的 30 个有效同坐标方块对照全部保存；核心 4/4 河心水面 Y96 不变，12/12 河岸低地恢复到水面以上，原天坑中心自然基底从 Y69 回到 Y103。外围 2 个样本仍有未生成区块。
- **新旧持久化模型**：953 个 Waterbed 河段完全一致，20 个特殊地貌只改变 #11 的位置，其他 19 个完整保存值一致；原天坑从 (-1600,5312) 换到 (320,6592)。
- Go 核心回归：`internal/generator`、`specialfeatures`、`runtimeworld`、`worldmodelio`、`worldsample`、`waterbed`、`cmd/geoworld-r81-hydrology-audit` 通过；仅代表这些包。此外 `go test -p 2 -run '^$' ./...` **所有 Go 包与测试源码编译通过（不执行测试）**。仍不能称 `go test ./...` 全量测试完成。
- 已有专项 Python 测试：17 项通过，额外 `scripts/r815_rc_build_matrix_preflight.py` 验证 9 个 CI 目标的版本配置和 Loader 元数据源路径全部正确。

## 新增方块俯视证据（非实际游戏截图）

`docs/acceptance/r81d_voxel_planview.png` 和 `docs/acceptance/r81d_voxel_planview_coverage.json`

- **来源是两套真实的 `minecraft:full` Anvil 已保存方块数据**，不会用模型水位生成贴图。
- X=-1760..-1544 / Z=5250..5466，旧、新、`WORLD_SURFACE` Y 变化三联图；相同 Y 区间和同坐标。
- 完整旧方块列 29,312，新方块列 24,560，可成对比对 24,560；未知 22,529，全部灰色，不计通过。
- 可成对比对区域的旧版 `WORLD_SURFACE` Y<=80 列 14,976，新版 0；这佐证旧坑底异常下切已消失，但不证明地貌视觉自然。
- **重要局限：** 投影图不是 Minecraft 游戏内相机视图；`WORLD_SURFACE` 包含树木、植被等，不能直接当作天然地表高度或真实 DEM。图中灰色边缘尚未生成。仍需真实游戏俯视和横断面截图。

生成工具：`scripts/r815_rc_anvil_planview.py`。输入为两个独立真实存档的 `--legacy-world` / `--candidate-world`，绝不写入旧存档。

## 9 目标兼容检查进度

正式 CI 定义位于 `.github/workflows/release.yml`。静态九目标配置预检为 **9/9 PASS**；本地 JDK21 和 JDK25 可用，JDK17 缺失。

| Minecraft / Loader | 本地 JDK | 本轮结果 | 是否 CI 发布验收 |
|---|---|---|---|
| 26.3 / NeoForge | 25 | 离线缺少 ModDevGradle 2.0.147 插件，源码尚未编译 | 否 |
| 26.3 / Fabric | 25 | **`compileJava` 与 `assemble` 通过；本地 JAR 六平台原生引擎、Loader 元数据检查通过** | 否（未做完整 check/build） |
| 26.2 / NeoForge | 25 | 尚未构建 | 否 |
| 26.2 / Fabric | 25 | 尚未构建 | 否 |
| 26.1.2 / NeoForge | 25 | 尚未构建 | 否 |
| 26.1.2 / Fabric | 25 | 尚未构建 | 否 |
| 1.21.1 / NeoForge | 21 | `assemble` 通过；`check build` 在 8 分钟上限超时 | 否 |
| 1.21.1 / Fabric | 21 | Loom 1.7.4 已缓存，但离线仍缺 Minecraft 编译类路径 26 个依赖 JAR；未完成编译 | 否 |
| 1.20.1 / Fabric | 17 | dadm 本地 JDK17 缺失 | 否 |

### 1.21.1 NeoForge 开发候选产物

- 通过 `-Pmod_version=0.1.5-rc.4-dev` 覆盖构建版本（源文件的正式发布版本未修改）。
- `bash ./gradlew --offline --no-daemon --max-workers=1 ... assemble` 通过。
- 本地候选 `java/adapters/1.21.1-neoforge/build/libs/geoworld-0.1.5-rc.4-dev.jar`，约 29.1 MB。Loader 元数据版本一致，Java Runtime class 存在，Windows/Linux/macOS x64/ARM64 六套原生引擎均存在。
- JAR SHA256：`3de08da233f5826ef38383181dc2854edc1b207cdb9e091b19a28429f216f026`。
- 产物验证结果：`docs/acceptance/r815_rc_neoforge_1211_jar_inspection.json`，重复验证工具 `scripts/r815_rc_verify_jar.py`。
- `check build` 未在本地限定运行时间内结束。**不称此为 check/build PASS**，不上传、不发布此产物。



### 26.3 Fabric 开发候选 JAR（本轮新增）

- `bash ./gradlew --offline --no-daemon --max-workers=1 ... -Pmod_version=0.1.5-rc.4-dev assemble`：**BUILD SUCCESSFUL**。
- `java/adapters/26.3-fabric/build/libs/geoworld-0.1.5-rc.4-dev.jar`：29,144,410 字节。
- Loader 元数据 ID `geoworld` 与版本 `0.1.5-rc.4-dev` 正确，Java Runtime 与 Windows/Linux/macOS x64/ARM64 六套原生引擎均存在。
- SHA256：`4b9837194eae1fb7396ca5cd78b91079a5f57590800c459ce0c773c65048d3df`。
- 证据：`docs/acceptance/r815_rc_fabric263_jar_inspection.json`。
- **没有运行此版本的全部游戏测试和完整 `check build`；不能作为九目标发布验收替代。**

## RC 仍需的明确验收门槛

1. 9 个正式 CI 目标都需编译、`check build`、元数据及原生引擎验证（推荐在 GitHub Actions Windows Matrix，避免强行在 dadm 同时运行所有 Gradle/JVM）。
2. Minecraft 客户端真实游戏镜头复验：#735 河谷、#509/#535/#851/#915 河口，及 #11 新坑所在地形；从河岸、俯视和低角度检查河水、坡度、河口连续性。
3. 将可复现截图、固定坐标和失败清单纳入验收。**不能用上述俯视投影代替游戏截图**。
4. 禁止覆盖旧世界模型和旧版本生成规则；不修改旧存档；新规则只对新世界生效。

此阶段不合并 master、不推送、不打 tag、不发 RC。


## 本轮补充：新天坑 #11 新位置真实方块与地貌自然度

详见 `docs/REALISTIC_GEOMORPHOLOGY_R81D_NEW_SINKHOLE_SHAPE_REVIEW.md`。
在固定种子 v1.1.24 的真实 1.21.1 NeoForge 保存世界中，围绕新址 `(320,6592)` 的
40/40 横断面方块列全部 `minecraft:full`。坑底自然基底 Y77，四向外缘 Y115～116；
东西约 160 格和南北约 128 格的抽样范围几乎都完全平坦，
部分坑壁绝对坡率达 1.875 格高度/水平格。数据与模型、Go/Java 原有
`Sinkhole` 恒定 floor + 狭窄 smoothstep wall 公式吻合。

**重要：** R8.1d 的河流避让固定问题通过，不意味着改址后大平底/陡壁的自然度已通过。
当前 `shapeRealism=HOLD_FOR_GAMEPLAY_VISUAL_REVIEW`，不能批准 RC 发布。
真实方块投影图在 `docs/acceptance/r81d_new_sinkhole_anvil_planview.png`，
**这不是 Minecraft 游戏相机截图**。是否改变天坑形状应进入下一独立地貌阶段，
并采用新规则版本、明确多种子/多加载器兼容性和回归验收，不直接改已通过的河口/河道算法。


## R8.1e 追加记录（规则 v1.1.25，独立新候选）

在 R8.1d 河道避让已通过之后，针对新址 #11 天坑大面积平底和陡坡，
独立实现 v1.1.25 连续沉陷坡面。详情：
`docs/REALISTIC_GEOMORPHOLOGY_R81E_SINKHOLE_SLOPE_ACCEPTANCE.md`。

- 两个实际 Minecraft 1.21.1 NeoForge 新世界的保存模型除 `config.WorldProfile` 外，**全部相同**（953 河段、20 特殊地貌、矿物及宏观地形均不变）。
- 新版 40/40 真实 Anvil 采样完整，东西/南北最大采样坡率分别从 1.875/2.0 降为 0.5/0.5，中心及外围地面均不变。
- 旧/新世界均有 41/41 Go/Java 连续高度一致采样；Go 7 组相关包完整测试通过，全部 Go 包编译通过，相关 R8.1 Python 测试 29 项通过。
- NeoForge 1.21.1 和 Fabric 26.3 本轮源码编译通过，**不是 9 Loader CI 全部通过**。
- **R8.1e 固定坡面专项通过，地貌整体真实度与游戏内截图仍未通过，正式 v0.1.5 RC 发布继续 HOLD。**
- 不合并、不推送、不发布 JAR，不覆盖旧世界；历史 v1.1.24/更早规则不变。


## 2026-11-12 本地九目标 CI 预检补充

**请以本节替代早期的构建进度表：** 本文较早的 26.3 Fabric / 1.21.1 Fabric 状态记录是 11 月 11 日的阶段快照，并非现在的最终状态。

- 本地 `1.21.1 Fabric`：在**本分支当前源文件**上重跑 `clean check build`，**17/17 任务通过**。
- 本地 `26.3 Fabric`：先发现 `verifyGeoWorldSpecialLandscapes` 还在误要求 v1.1.25 天坑有平底；升级为区分新规则连续坡面/旧规则平底的严格验证后，单项通过，完整 `clean check build` **41/41 任务通过**。
- 两份本地 Fabric JAR 都经过独立压缩包检查（Loader 元数据、`GeoWorldV1.class`、6 个原生引擎），仅作为**开发分支测试产物，不是正式 RC**。
- 九目标静态配置仍是 9/9 通过；但实际完成本地 `clean check build` 的只有 **2/9**；没有执行 GitHub Windows 官方九目标 CI。
- 本阶段继续 **不发布、不推送、不合并**，游戏相机视觉验收仍未完成。

详细报告：`docs/REALISTIC_GEOMORPHOLOGY_R815_RC_LOCAL_MATRIX_20261112.md`。
完整机器可验收状态：`docs/acceptance/r815_rc_local_build_matrix_20261112.json`。


## 2026-11-12 NeoForge 完整构建验收结果更新

当前分支完成本地完整 `clean check build` 的仍只有 **2/9**（1.21.1 Fabric 与 26.3 Fabric）。

1.21.1 NeoForge 全量任务在 `verifyGeoWorldWaterOwnership` 遇到真实的 River 规划水面断层：77 个高差超过 1 格的相邻水面，首次例子 `(10815,2467)` Y66 到 `(10814,2467)` Y63。该证据来自 Java 方块列规划，**不是 Minecraft 已保存 Anvil**。未放宽验证，也未修改算法。

26.3 NeoForge 首次在线初始化反编译资源超出 dadm 当前内存/Swap 容量，安全取消，该目标视为资源阻塞，不能标记通过。

完整报告 `docs/REALISTIC_GEOMORPHOLOGY_R815_RC_NEOFORGE_BLOCKERS_20261112.md`；机器可复查记录 `docs/acceptance/r815_rc_neoforge1211_water_ownership_failure.json`。

**正式 RC 继续 HOLD，不合并、不推送、不发布。**


## R8.1g（2026-11-12）新河口水面版本修复验收

**新的规则候选是 v1.1.26，历史 v1.1.25 保持原语义、不执行迁移。**

- 前次 NeoForge 水文验证发现 77 处水面自由落差，已追溯到海面回水在低于海平面的河床上无条件将较高河面压成 Y63。新规则保留河面，待接近海面半格范围再接管。
- 固定河口 144×144 方块规划：相邻 River 水面 >1 格落差 **78→0**。
- 正式 Java 水域归属任务：**1183 区块、302848 列，原 77 项 freeFallWaterFaces 降为 0**，其余违规计数均为 0。
- v1.1.25、v1.1.26 各 **141** 个 Go/Java 水文数值采样完全一致。
- 新建独立 Minecraft 1.21.1 NeoForge 存档：**1517/1517 真正 `minecraft:full` 方块列、2762 对相邻真实水面、0 处超过一格落差**；原失败点真实水面 Y65/Y65。
- **1.21.1 NeoForge v1.1.26 本地 `clean check build` 通过**（5m15s，45 Gradle 任务），JAR 内容及六套原生引擎通过检查；Go 7 组相关包全测试通过。
- 26.3 Fabric v1.1.26 `compileJava` 通过，但旧版 v1.1.25 的两次 Fabric 全构建**不能充当** v1.1.26 的全构建，当前完整本地构建仅 **1/9**，Windows 官方九目标 CI 未跑。
- 仍未执行 Minecraft 客户端游戏内相机验收；不合并、不推送、不发布。

完整报告 `docs/REALISTIC_GEOMORPHOLOGY_R81G_ESTUARY_CONTINUITY.md`；
模型/真实方块证据 `docs/acceptance/r81g_model_and_runtime_acceptance.json`；
本轮严格九目标状态 `docs/acceptance/r81g_current_rules_local_matrix.json`。


## R8.1h（2026-11-12）最新版本完整构建状态

**R8.1g 采用新生成规则 `geoworld-rules-v1.1.26`，上文的 v1.1.25 / R8.1g 1/9 状态是历史阶段快照。当前版本最新结果：本地完整 `clean check build` 通过 3/9。**

- Minecraft **1.21.1 NeoForge**：R8.1g 已通过（5m15s、45 个任务含一项缓存）。
- Minecraft **1.21.1 Fabric**：R8.1h 重新执行新规则完整构建，通过（1m38s、17/17）。
- Minecraft **26.3 Fabric**：R8.1h 重新执行新规则完整构建，通过（3m54s、41/41）。
- Fabric 两份 JAR 的 Loader 元数据、六平台原生引擎完整；字节码中的 `GeoWorldEngineBridge` 明确使用 v1.1.26。开发模块默认 0.1.4 文件名不是发布版。
- 26.3 NeoForge 依旧本机资源受限；另五个适配目标未完成完整构建。
- 正式 Windows 九目标 GitHub CI、真实游戏客户端视觉验收仍未做。**不合并、不推送、不发布。**

详细报告 `docs/REALISTIC_GEOMORPHOLOGY_R81H_FABRIC_FULL_BUILD_20261112.md`，最新机器可核验矩阵 `docs/acceptance/r81h_current_rules_local_matrix.json`。
