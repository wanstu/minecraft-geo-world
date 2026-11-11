# GeoWorld R8.1c：河口真实 Minecraft 方块验收记录

日期：2026-11-11。分支：`adm/geoworld-r81c-voxel-20261111`。
基线：`60c8286`、新世界规则 `geoworld-rules-v1.1.23`。
**结论：#509 和 #915 固定河口的实际方块专项已通过；整体 R8.1c / v0.1.5 RC 尚未批准发布。**

可重复核实的机器证据：`docs/acceptance/r81c_v23_actual_voxel_evidence.json`。

## 1. 证据来源与方法

- Minecraft 1.21.1，NeoForge 21.1.211，精确 Seed `-1818694639628404530`。
- 原版规则世界：`R81VoxelAudit`，保存的绑定规则 v1.1.22；候选规则世界：`R81cVerifiedGeoWorld`，绑定规则 v1.1.23。
- 两个世界均由 `scripts/verify_geoworld_voxel_world.py` 读取 `level.dat` 与 `data/geoworld/geoworld.properties`，确认生成器、种子、规则版本及模型存在。**没有把旧存档的模型文件替换成新版，也没有修改旧世界**。
- 以 `scripts/r81c_voxel_audit.py` 只读解析真实 `region/*.mca` 的 Anvil `minecraft:full` 区块、`sections.block_states`、`WORLD_SURFACE` 与 `OCEAN_FLOOR`。未生成区块只标 `not_generated`，绝不视为干地或断水。
- 用 Go 只读审计独立加载**实际服务器生成并持久化**的 v1.1.23 世界模型，而不拿离线候选模型替代游戏存档。

## 2. #509：原一格逆坡消失（实际方块）

固定 X=-1071、-1070、-1069、-1068，河流向 Z 负方向流动。相邻的 Z=-1026 和 Z=-1027 顶层水方块：

| X | v1.1.22（上游→下游） | v1.1.23（上游→下游） |
|---|---|---|
| -1071 | Y61 → Y62（+1） | Y62 → Y62 |
| -1070 | Y61 → Y62（+1） | Y62 → Y62 |
| -1069 | Y61 → Y62（+1） | Y62 → Y62 |
| -1068 | Y61 → Y62（+1） | Y62 → Y62 |

这些都是已保存的实际水方块，两个版本同样的固定坐标；旧版 4/4 有反向台阶，新版 0/4。沿 Z=-1016、-1026、-1027、-1038、-1050 取横断面，两个世界有 110 个已生成的可比对列，其中 54 列顶层水方块高度发生变化。检查到的相邻低岸没有新增干低岸与高水面直接贴邻的水墙。

**需记录的一处差异：** `(-1026,-1016)` 砂质底面从旧版 Y59 变为新版 Y60；其余 109 个可比对横断面列的 `OCEAN_FLOOR` 高度相同。不能把“模型预测地形不变”扩展声称“Minecraft 每个海床方块都完全不变”。这一处留作单独分析。

## 3. #915：冻结河口的正确水位读取（实际方块）

旧版模型对正式 river #915 报告过 +4 格河海水位反升（模型证据，**非旧版真实方块**）。新版真实存档 `Z=10432`、`X=1616..1700` 共 85 列均为 `minecraft:full`：

- 85/85 列在 Y62 具有一致的水面方块或覆冰表面，连续向下游无升水台阶。
- 其中 76 列的 Y62 为 `minecraft:ice`，下方 Y61 为水；少数位置 Y62 为直接露出的 `minecraft:water`。
- 一列有高出海面的蓝冰结构（例如 X=1682 顶层蓝冰 Y67），但同列 Y62 仍为水；蓝冰突出物不是河流水位从 Y62 升到 Y67，也不是“干断口”。
- 横向 7 条取样线在 `Z=10400..10476` 范围的 140/140 个已生成实际方块列均为 Y62 水或覆冰顶面；未发现水位逆升。

原始审计脚本只看 `minecraft:water`，会把冰下 Y61 水与露天 Y62 水误判为 +1，并把蓝冰上部误判为断水。现将原始液态水顶层与冰水共用水位分别统计，保留蓝冰突出物以便视觉检查；新增 5 项判定单元测试。

**注意：** 这里只能说新 v1.1.23 游戏世界中没有重现 #915 的四格断崖；没有旧版 #915 Minecraft 方块级对照，不得声称已重建并直接量化旧版的四格水墙。

## 4. 真实模型对照与未关闭问题

对游戏持久化模型再次运行 `cmd/geoworld-r81-hydrology-audit`，得到 953 河段、103 终端、88 入海和 15 入湖；88 条入海河段的模型级逆升信号为 0。**此统计是模型级，只与上述两处实际方块检查互补，不能替代其他河口的 Minecraft 验收。**

仍未关闭：

1. #509 的一处 +1 砂床差异需要确认成因，并回归海岸地貌是否存在隐性变化。
2. #535、#851 的旧模型各 +1 信号在新模型已消除，但尚无同等真实方块验收。
3. Go/Java 采样存在先前记录的 16 处干地高程差异；本轮只处理冰水审计误判，没有解决该兼容性问题。
4. 天坑 #11 与河段 #735 选址冲突及无规划排水，仍属 R8.1d；R8.2 雪山盆地、岛屿生成不在本轮。
5. 尚需完整 Java/Go 回归、多 Loader/多版本 CI、实际游戏视觉验收后，才允许发布新的 RC。

## 5. 复验命令

在仓库根目录，从正确的已保存世界检查：

```bash
python3 -m unittest discover -s scripts -p 'test_*voxel*.py' -v

python3 scripts/r81c_voxel_audit.py \
  --world java/adapters/1.21.1-neoforge/run/R81cVerifiedGeoWorld \
  --expect-rules geoworld-rules-v1.1.23 \
  --expect-seed -1818694639628404530 \
  --output .tmp/r81c-v23-actual.json
```

旧版本读样只读调用、不能升级或覆盖旧版模型：

```bash
python3 scripts/r81c_voxel_audit.py \
  --world /home/wanstu/projects/minecraft-geo-world/java/adapters/1.21.1-neoforge/run/saves/R81VoxelAudit \
  --expect-rules geoworld-rules-v1.1.22 \
  --expect-seed -1818694639628404530 \
  --output .tmp/r81-v22-voxel.json
```

两套模型与 Anvil 报告的 SHA256 和固定统计都保存在证据 JSON 中。当前仍不合并主干、不推送、不发布 JAR。
