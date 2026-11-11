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
