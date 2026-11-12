# GeoWorld R8.1d 新天坑 #11 的实际方块形态复验

日期：2026-11-11  
工作分支：`adm/geoworld-r81d-sinkhole-20261111`  
新规则：`geoworld-rules-v1.1.24`  
状态：**新位置生成已证实，河道避让专项通过；新天坑的真实地貌外形仍需人工游戏视觉验收，RC 暂不发布。**

## 验收对象和证据类型

- 游戏：Minecraft 1.21.1 / NeoForge 21.1.211；精确种子 `-1818694639628404530`。
- 新世界：`java/adapters/1.21.1-neoforge/run/R81dVerifiedGeoWorld`，本轮重新启动正式服务端，从其旧有 v1.1.24 绑定模型继续生成新地点。
- 仅使用真实 `minecraft:full` Anvil 已保存的 `region/*.mca` 方块数据，不把 GeoWorld Go 模型的 groundY 当成 Minecraft 真实方块。
- 天坑 ID=#11，中心 `(320,6592)`，椭圆半径约 `153.63 × 130.49` 方块，模型保留原 ID 且已避让正式河网。
- 官方建模文件（模型级）：`docs/acceptance/r81d_new_sinkhole_v124_model_audit.json`、`r81d_new_sinkhole_v124_model_cases.json`。
- 实际 Minecraft 方块级：`docs/acceptance/r81d_new_sinkhole_actual_voxels.json`；可重复命令 `scripts/r81d_new_sinkhole_voxel_audit.py`。
- 投影视图及东西/南北剖面：`docs/acceptance/r81d_new_sinkhole_anvil_planview.png`；**不是游戏截图、不是 DEM、不是卫星图**，灰色表示未生成区块。

## 真实区块结果

沿 `Z=6592` 的东西断面 22 个采样点，与 `X=320` 的南北断面 18 个采样点：**40/40 均为实际已保存的 `minecraft:full` 方块列**，没有把未知区块计为干地。中心自然土层高度 Y=77，没有检出地表水。外缘自然地面为西 Y115、东 Y116、北 Y115、南 Y116。

| 地点 | 真实自然基底 Y |
|---|---:|
| 中心 (320,6592) | 77 |
| 东西方向 X240～400 多点 | 全部 77 |
| 南北方向 Z6528～6656 多点 | 全部 77 |
| 西外缘 X160 | 115 |
| 东外缘 X480 | 116 |
| 北外缘 Z6464 | 115 |
| 南外缘 Z6720 | 116 |

局部实测陡壁包括：东西横断面 X216→224，8 格水平距离下降 14 格；X408→424，16 格水平距离升高 30 格，绝对坡率达到 **1.875 Y/格水平距离**。这些数值不是错误坐标也不是模型插值，而是已保存的自然土石表层真实方块。

稀疏剖面的平底范围大致为东西至少 160 格、南北至少 128 格。其本质问题是底部几乎没有层次变化，坡脚窄、坑壁陡，顶面可见接近规则椭圆；**尚不能凭数据判定它足够自然**。

实际投影图尺寸及覆盖由审计 JSON 的 `visualization` 字段记录；未知地图像素保持灰色，不能用图像补全推断。

## 实现根因

原 `v1.1.23` 就已采用的沉陷地貌规则，在 `internal/specialfeatures/model.go` 的 Sinkhole 分支中：

- `floor = selected.FloorHeight + variation`：`variation` 只接受局部基础地形高度相对中心的约 8% 变化；当原始地面很平坦时，坑底几乎就是常数。
- `inner = 0.56 + 0.05 * (wallVariation - 0.5)`：在内半径约 56% 的位置一直使用坑底高度，容易生成非常宽的大平底。
- `outer = inner + 0.14 + 0.10 * wallVariation`：较窄的外部过渡带连接原地面，约 38～39 格高差被集中在有限距离内。
- Java `GeoWorldV1` 的对应 Sinkhole 采样逻辑也沿用同样的 `floor / inner / outer / smoothstep`，不是 Java 单独出错。
- v1.1.24 的新代码只为冲突天坑寻找安全选址；**并未修改这个沉陷形状公式**。因此该问题是旧地貌方案本身的真实度不足，不应被误归因到本轮河道避让。

## 结论与下一阶段决策

1. **R8.1d 的固定缺陷仍通过**：原天坑不再切穿 river #735，正式河网不变，原位置的陡直高悬河道已消失。
2. **新天坑选址结果已经存在真实方块证据**：可以检查，而不是只存在于离线 Go 模型。
3. **地貌自然度保持 HOLD，不作为通过项**：平坦大坑底、陡壁与近规则椭圆需要真实游戏相机截图对照。这是阻止断言“GeoWorld 新版地貌已全面达标”的一项客观证据。
4. 后续如确需改形状，必须采用**新规则版本**并以独立阶段处理：扩大坡脚过渡、按底层地形和侵蚀过程引入小尺度起伏与沉积过渡，同时保持正式河道避让、旧世界绑定隔离以及宏观水文不变。至少在多个种子和新旧两个固定坐标上做模型/Java/真实方块全链路回归，再考虑新 RC。
5. 本轮没有改生成算法、没有迁移或改写任何历史世界、没有合并/推送/发布 JAR；保留正式 CI 和实际游戏相机截图为发行门槛。

## 复验命令

```bash
python3 scripts/r81d_new_sinkhole_voxel_audit.py \
 --world java/adapters/1.21.1-neoforge/run/R81dVerifiedGeoWorld \
 --output docs/acceptance/r81d_new_sinkhole_actual_voxels.json \
 --image docs/acceptance/r81d_new_sinkhole_anvil_planview.png

python3 -m unittest discover -s scripts -p 'test_r81d*.py' -v
```
