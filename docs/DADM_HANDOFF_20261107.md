# GeoWorld — dadm 开发交接（2026-11-07）

## 开发机器与当前基线

- 日常开发：dadm（Linux Mint 22.3），Environment `env_9bcdf55fd8e9abc0`，根目录 `/home/wanstu/projects/minecraft-geo-world`。
- 旧开发机器：eadm（Windows），不再并行修改同一功能。
- `master` 与 `origin/master` 同步；HEAD `6ed80c8`，发布标签 `v0.1.4` 固定在 `04db6cb`。
- 已确认两台机器工作区无未提交文件；dadm 已 fetch 最新标签。
- `v0.1.4` 已完成 GitHub 9 JAR 发布与 Modrinth 9 版本上传校验；Modrinth 项目当时仍为 processing，不能声称已公开。

## dadm 本机检查

2026-11-07 首次检查：系统有 Git、Python 3、curl、unzip，但没有 Go、Java/JDK、Docker 或 Podman，故**尚未在 dadm 运行 Go、Gradle 或 Minecraft 回归**。旧机器和 CI 的通过记录不能代替 dadm 本机验证。

在仓库运行：

```bash
bash scripts/check-dev-prereqs.sh
```

- Go 需 1.26+，见 `go.mod`，不要使用 Mint/Ubuntu 仓库默认的 Go 1.22。
- 1.21.1 NeoForge 优先 JDK 21；26.3 Fabric 的已验收矩阵使用 JDK 25。分别明确 JAVA_HOME 后执行对应包装器的 `check` 与 `jar`。
- 建议使用用户级安装/明确路径，不擅自改系统默认 Java、sudo 安装或覆盖现有工具。
- 先完成 `go test -p 2 ./...`，再分别选 1.21.1 NeoForge 和 26.3 Fabric 进行 Java 验证；最后按发布矩阵覆盖所有 9 目标。

## 下一阶段：世界生成质量与一致性

本阶段优先改进**自然生成质量**，不增加玩法、物品、方块或跨维度生成，不改变 Query API v1 契约。

1. **水系地形一致性**：抽样排查河流进入盆地、穿山、入海口和湖岸衔接；必须保持河床贴近地面、河流连续、主河可达海，避免悬空河、锯齿或入海收窄。
2. **跨 Chunk 连续性**：同世界坐标的高度、水面、生态判定应与采样路径无关；重点覆盖 16 方块边界和水岸交界。先补回归/审计再修改算法。
3. **生态与特殊地形**：验证蘑菇岛、苍白之园、樱花林与封闭湖泊的联通规则，特殊岛屿不得意外与陆地粘连。
4. **兼容与性能**：避免扩大同步计算范围；保持世界锁定与旧存档、其他模组兼容，并验证生成耗时与缓存。

实施次序：读取当前算法与对应审计器 → 定位一条可复现失败样例 → 新增定向自动回归 → 最小范围修复 → Go/Java golden/Loader 验证。**现阶段没有证据表明这些问题在 v0.1.4 仍可复现，不能预先宣称存在缺陷。**

任何会改变生成结果的修改必须单独评估 rules version、旧存档与 golden 基线；不得直接更改历史 golden 来让测试通过。验收后再考虑下一次发布，不移动 `v0.1.4` 标签。
