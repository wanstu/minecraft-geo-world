# R8.1c Minecraft 存档验收预检与独立测试记录

日期：2026-11-11。工作树：`adm/geoworld-r81c-voxel-20261111`。
基线提交：`60c8286`，规则 `geoworld-rules-v1.1.23`。
**状态：模型、Java 编译和存档预检通过；河口实际 Minecraft 方块验收未通过，未发布。**

## 为什么添加预检

原本在 dadm 正式工程目录创建的 `R81cFreshVoxelAudit`：
- `level.dat` 为 `minecraft:noise`，虽然精确种子为 `-1818694639628404530`；
- 无 `data/geoworld/geoworld.properties`，也无保存的 GeoWorld 模型；
- 运行日志明确记载 `not a GeoWorld world; leaving vanilla worldgen untouched`。

因此不能把此存档中的方块当作修复后的 GeoWorld 真实方块证据。另一个正式的 `R81VoxelAudit` 则是 `geoworld:geoworld`、固定种子及锁定规则 v1.1.22，能通过相同预检。

## 工具与用法

`scripts/verify_geoworld_voxel_world.py` 采用 Python 标准库，只读解析 `level.dat` NBT 和保存的绑定，检查：

1. Overworld chunk generator 必须是 `geoworld:geoworld`，不能只是普通 `minecraft:noise`；
2. 精确 64 位有符号种子必须一致；
3. GeoWorld 绑定存在、规则版本匹配，绑定种子与世界种子相同；
4. 绑定模型文件存在，且 Anvil region 已保存。

示例（在仓库根目录执行，世界名称可替换）：

```bash
python3 scripts/verify_geoworld_voxel_world.py \
  java/adapters/1.21.1-neoforge/run/saves/R81cFreshVoxelAudit \
  --expect-rules geoworld-rules-v1.1.23 \
  --expect-seed -1818694639628404530
```

退出码 0 表示**具备开始方块验收的条件**，不是实际河口已通过。退出码 2 表示不能用于验收。若绑定规则版本不匹配，不允许直接覆盖旧模型或改写旧存档绑定。

## 本轮已完成

- Go：`go test -p 2 -count=1 ./internal/worldsample ./internal/waterbed ./internal/generator ./cmd/geoworld-r81-hydrology-audit` 全部通过；生成器测试用时约 242 秒。
- Java：NeoForge 1.21.1 `./gradlew --offline compileJava` 通过。
- Python：6 个单元测试全部通过（有效存档、原版生成器、旧规则、不同种子、无绑定、无模型）；旧 v1.1.22 GeoWorld 真实存档通过预检；错误的 v1.1.23 测试存档被拒绝。

## 独立服务器实验（没有正式验收）

在隔离工作树下以 `level-type=geoworld:geoworld`、指定精确种子启动 NeoForge 1.21.1 专用测试服务，生成 `R81cVerifiedGeoWorld`。

- `level.dat` 的生成器类型和种子都正确。
- 服务进入了 GeoWorld Runtime 初始化，但观察时尚未创建完整的 GeoWorld 模型绑定；Anvil region 文件仍为 0 字节。
- 该启动过程尚未形成可验收的已保存区块，实验已经停止，避免持续消耗 dadm 资源。
- 不能推断当前 R8.1c 水面改动的实际方块正确性，也不能从此实验得出 Java 运行时必然有 bug 的结论。

## 下一道必须通过的门槛

在**真正新建的规则 v1.1.23 GeoWorld 世界**完成模型绑定后，重新生成河段 #509 的 `(-1069,-1026)` 与 `(-1069,-1027)` 及相邻四列真实方块；比较旧版 +1 方块水面逆坡、两侧低岸、海床、海水连通性。接着检查 #915 原先 +4 格的异常河口，同时保留旧 v1.1.22 不迁移。没有实际 Anvil `minecraft:full` 区块与横断面报告时不得发布 RC。

天坑 #11 / 河流 #735 的冲突仍属 R8.1d，雪山盆地与岛屿选址仍属 R8.2，不在本轮顺手修改。
