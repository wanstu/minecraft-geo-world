# GeoWorld v0.1.7 规划 — 特殊地貌选址与水文因果一致性

日期：2026-11-16  
基线：`v0.1.6` / commit `8e0daf076027ae0ea1cc4409c87dd3963d0dd470`  
v0.1.6 新世界规则：`geoworld-rules-v1.1.23`  
RiverGeometryVersion：8  
Engine cache：v27

## 1. 版本定位

v0.1.6 已经把 R8.1c 的河口接收水位、河海低岸、水墙、河道最近点地形破碎和旧模型缓存污染收敛，并通过真实 geomV8 新世界视觉验收和 9/9 发布矩阵。

v0.1.7 不继续扩大河口修补范围，而转向此前仍明确保留的问题：

1. **R8.1d：特殊地貌与正式河网的因果冲突。**
2. **R8.2：海岛 / 湖岛 / 雪山盆地等特殊地貌的选址是否符合真实地形背景。**
3. 把“模型看起来合理”进一步升级为 **Model → Runtime → Minecraft voxel → Web 地图** 的固定验收。

核心原则：

> 先决定“这个地貌能不能出现在这里”，再决定“它应该长什么样”。

## 2. 最高优先级：R8.1d 天坑与正式河网冲突

固定复现对象：

- Seed：`-1818694639628404530`
- 天坑 #11：中心约 `(-1600, 5312)`
- 河段 #735：现场约 `(-1665, 5313)`
- 当前已知模型信号：天坑足迹与正式河道影响范围相交，但没有对应排水规划；河面与坑底存在约 18～27 格高差风险。

### 2.1 先做只读证据，不先改形状

新增 R8.1d 专项验收，必须同时提取：

- SpecialFeatures 应用前后的 ground / water / river ownership；
- 河段 #735 的中心线、水面、河床、岸坡；
- 天坑中心、坑壁、坑底的纵横剖面；
- 实际 Minecraft Anvil 水方块、空气、岸高；
- Web 地图对应坐标。

先回答三个问题：

1. 天坑是否实际截断、悬高或暴露正式河流；
2. 河流是否本来就有合理绕行 / 排水路径；
3. 该天坑是否从选址阶段就不应该被允许。

### 2.2 v0.1.7 默认处理策略：不合适就拒绝 / 重选

优先级：

```text
候选天坑
→ 检查正式河流活动走廊
→ 检查湖泊 / 海岸 / 排水接收关系
→ 检查局部 relief 与可容纳深度
→ 不满足条件：拒绝候选并继续选择下一个候选
→ 满足条件：才进入地貌几何生成
```

v0.1.7 **不默认实现地下排水或凭空增加地表支流**。

只有未来已有明确地下水 / 可溶岩 / 落水洞模型时，才允许把“地下排水”作为真实因果链。当前没有这条模型链时，不用视觉补丁伪造。

### 2.3 不允许的修复

- 把原河流整体抬高或压低来迁就天坑；
- 天坑生成后再把河流硬挖穿坑壁；
- 没有 FlowTo / receiver 依据就生成一条新支流；
- 为了保留固定数量的天坑而降低选址约束；
- 用局部噪声把冲突遮住。

特殊地貌数量允许随 Seed 和地理条件变化，**不保证每个世界都必须生成同样数量**。

## 3. R8.2 特殊地貌选址

### 3.1 海岛 / 湖岛：必须是水下地形的露头

适用于：

- Mushroom Island
- Pale Garden Island
- Cherry Island
- Lake special island

新选址契约应至少检查：

- 候选位于真实 Ocean / Lake 背景，而不是只看一个离散标签；
- 岛体下方存在连续的水下高地 / 山体根部；
- 从深水到岛岸存在可解释的 submerged shoulder / slope；
- 周围保留足够水体 moat；
- 不允许出现“平海床上突然贴一个圆盘岛”；
- 不允许岛体切断正式主河口或湖泊出口。

目标不是把所有岛做成相同火山锥，而是保证：

> 岛是海底 / 湖底地形高点露出水面的结果，而不是独立贴到水面上的装饰物。

### 3.2 雪山盆地 / 火山雪坑：必须先有山地条件

`VolcanicCraterSnow` / snow basin 候选需要同时考虑：

- Mountain / Foothill / high local relief 背景；
- 足够高的基础高程；
- 冷温度 / 雪线条件；
- 坑缘外侧仍能接回原山体；
- 盆地内部若有水体，必须有明确 hydrology 语义；
- 不能在平原、低丘或孤立无山体背景处直接生成“雪山盆地”。

### 3.3 樱花 / 苍白森林特殊地貌

本轮只收紧**选址背景**，不重做生态系统：

- Cherry 特殊地貌不能只靠随机命中；
- Pale Garden 岛继续要求真实水体隔离；
- 生态类型必须与已有 Climate / Ecology 结果不冲突；
- 不为了保证特殊群系出现率破坏地形和水文。

## 4. 版本与兼容策略

如果 R8.1d / R8.2 改变新世界选址语义：

- R8.1d baseline 使用 **`geoworld-rules-v1.1.24`**；R8.2 Slice C / D 新世界使用 **`geoworld-rules-v1.1.25`**；Slice E 河口航运使用 **`geoworld-rules-v1.1.26`**；Slice F 群系分布使用 **`geoworld-rules-v1.1.27`**；
- v0.1.6 / v1.1.23 世界保持原持久化模型语义；
- 不对已经生成的 `.geoworld` 重新选择特殊地貌；
- Engine cache 继续使用 v27 的 native-engine SHA256 身份，不允许旧模型静默命中新实现；
- 必要时为 SpecialFeatures 增加独立持久化 siting version，而不是复用不相关的 RiverGeometryVersion。

## 5. 实施切片

### 当前进度（2026-11-18）

- **Slice A：完成。** 已用 v1.1.23 / geomV8 的真实 1.21.1 NeoForge Anvil 方块确认天坑 #11 与河 #735 的单侧岸坡冲突。
- **Slice B：完成。** v1.1.24 在候选阶段拒绝河道冲突天坑；固定 seed 模型冲突 1→0，真实 voxel 右岸低于河面的样本 12/12→0/12，河 #735 水面保持 Y96；旧 v1.1.23 世界实测可直接加载且模型 SHA 不变。
- **Slice C：完成并已提交。** v1.1.25 Ocean island 只落在真实水下高地；固定 seed 8 个 Ocean island 全部通过 root/moat 资格，Lake island 2→0；水下根部由旧 32 格/4 格柱壁收敛到天然基底最大 4 格/4 格，真实 Anvil 74/74 full。
- **Slice D：完成并已提交（112111d）。** Snow crater / basin 增加 mountain + relief + climate + inland-continuity / depression 背景资格；固定 seed 953 条正式河流与 4 个 sinkhole 完全不动，Cherry crater #16 / basin #20 完全不动；原完整几何验收 232/232 full，最终坐标 fresh-world spot 验收 10/10 full、12/12 模型与 voxel 0 格误差。
- **Slice E：完成并已提交（afd7499）。** v1.1.26 河道核心实际最浅 3 格；88 个 OceanMouth 的 828 个中心采样保持 River；河口影响区 Warm Ocean 降为无珊瑚航道，fresh 1.21.1 Anvil 49/49 FULL、1558 个可航核心最浅 3 格、河口影响区实际 coral=0，影响区外同一区域 coral=84；v1.1.25 已持久化世界不变。
- **Slice F：实现及验收完成。** v1.1.27 四种子平均森林类 34.38% → 27.55%，近岸珊瑚暖海 43.15% → 15.82%；NeoForge 新世界实际礁区有 1,810 个珊瑚方块，暖水对照和 #43 河口均为 0。详见 [群系覆盖率验收](REALISTIC_GEOMORPHOLOGY_R8_2_BIOME_DISTRIBUTION_ACCEPTANCE.md)。
- **Slice G：RC.1 自动构建验收通过，待 eadm 操作验收。** 9 个目标完成构建、Loader/JAR 与原生引擎校验。已核对 46 个区块和 14 个地图坐标，两处内陆河湖水面跳变还有待修复。详见 [RC.1 验收](REALISTIC_GEOMORPHOLOGY_V017_RC1_ACCEPTANCE.md)。


### Slice A — R8.1d 只读现场证据

产出：

- #11 / #735 模型剖面；
- 实际 Anvil voxel；
- Web / Model / Runtime 坐标对齐；
- 明确责任层；
- 不修改生成算法。

停止条件：没有真实 voxel 证据，不进入算法修改。

### Slice B — 天坑选址约束

实现：

- 正式河流活动走廊 clearance；
- receiver / lake / coast 冲突筛选；
- 地形 relief / 可容纳深度筛选；
- rejection reason 可审计；
- deterministic next-candidate 选择。

验收：

- #11 / #735 冲突消失；
- 固定 Seed 不出现高架河、断河或无规划排水；
- 多 Seed 不因约束加强而出现“为了凑数量强行选址”。

### Slice C — 海岛 / 湖岛根部约束

实现水下 relief / moat / slope 背景筛选。

验收重点：

- 岛体与水下地形连续；
- 岛岸不出现程序化圆盘边缘；
- 不切断河口 / 湖口；
- 旧 special-island moat 回归继续通过。

### Slice D — 雪山盆地选址

实现 mountain + relief + climate + drainage 条件。

验收重点：

- 不再在缺乏雪山地理条件的位置生成；
- 坑缘与周围山体连续；
- 内部水体和出口遵守正式 Hydrology。

### Slice E — 河流可航行性与河口群系

R8.2 Snow 收尾后，单独处理河流相关优化，不与特殊地貌选址混改。

#### 河口群系 / 航运

试玩新增证据：问题不只是 WaterKind 名称。当前低纬浅海会大量实现成 `minecraft:warm_ocean`，原版珊瑚随之进入河口附近，直接阻碍船只出海。

只读基线已经确认：

- 固定 seed 最近陆地 1 个宏格（约 128 格）的海洋中，Warm Ocean family 为 **49.15%**；
- 距岸 6 个宏格以上的外海为 **34.19%**；
- 但按纬度分层后，0–15° 与 15–30° 的近岸 / 外海都是 **100% Warm Ocean family**，30–45° 近岸甚至略低于外海；
- 因此不是 coastal current 单独把海岸加热，而是“低纬暖海气候语义”被 Minecraft realization 过度等同成“整片珊瑚礁群系”。

Slice E 的硬约束：

- 正式河流出海口、mouth flare 与其必要航运过渡带不得实现成 `minecraft:warm_ocean`；
- 正式河口河道继续保持 River 语义，水位仍可继承 sea-level backwater，不重新制造河口水墙；
- River 外侧但仍属于 OceanMouth 航运影响范围的浅海，允许保持温暖海水气候，但 Minecraft biome 必须降级为不会铺满珊瑚的过渡海洋（优先 `lukewarm_ocean`），直到退出河口影响区；
- 优先复用现有 OceanMouth / mouth expansion / bank metadata 定义缓冲区，不额外发明任意固定半径；
- 真正离开河口影响区后的海域，是否仍实现为 Warm Ocean 留给 Slice F 的珊瑚礁适生模型决定；
- 固定 #509 / #915 以及新增至少一个 Warm Ocean 河口作为 model + voxel + biome + 航运回归样本。

#### 河流水深

- 正式河流的可航行水体目标最小水深为 **3 格**；
- 验收必须读取真实 Minecraft 方块，按 `water surface Y - river bed Y` 统计，不能只看 normalized model depth；
- 河道核心不得再出现 1～2 格深的浅槽；
- 岸边允许自然渐浅，但不能让整条河的有效航道被 1～2 格浅水截断；
- 最小深度修复不能制造水墙、局部水峰、悬空水、过深直槽或破坏纵向单调性；
- 必须继续保持 RiverGeometryVersion / R8.1c 河口水位 / R8.1d 特殊地貌隔离回归。

建议先做只读 audit：
1. 统计固定 seed 全部正式河段的真实模型水深分布；
2. 找出 1 格 / 2 格浅水的典型位置；
3. 在 1.21.1 NeoForge 实际 Anvil 中复核；
4. 再决定最小修改点位于 Waterbed、WorldSample 还是 Minecraft voxel realization。

### Slice F — 生态 / 群系覆盖率校正

本项统一处理“生态气候语义合理，但 Minecraft 最终群系面积 / 装饰过量”的问题。

陆地试玩反馈：丛林、稀疏丛林和森林整体过多。固定 seed 基线已确认，上游 Ecology 的森林类区域约占陆地 **45.72%**，不是单纯 Minecraft 显示层偶然偏多。

海洋试玩反馈：很多海岸看起来几乎全是暖海，低纬浅海又会被直接实现为 `minecraft:warm_ocean`，导致珊瑚分布远比“暖海水温”本身更泛化。固定 seed 基线：最近岸 Warm Ocean family **49.15%**，外海 **34.19%**；0–30° 海洋几乎全部属于 Warm Ocean family。

先做只读分布审计，再调整：

- 陆地分开统计 Boreal / Temperate / Subtropical / Tropical Monsoon / Tropical Rainforest；
- 同时统计 Minecraft 最终 `forest` / `birch_forest` / `dark_forest` / `jungle` / `sparse_jungle` / `bamboo_jungle` 实际覆盖率；
- 检查 moisture、temperature、seasonality 的竞争边界，增加 forest ↔ grassland / shrubland / savanna 的真实过渡空间，而不是随机替换；
- 海洋分开统计 SST / Warm Ocean family / Minecraft `warm_ocean` / `lukewarm_ocean` 的近岸与外海面积；
- 明确 **warm water ≠ coral reef**：暖海气候可以连续存在，但 `minecraft:warm_ocean` 只应落在满足珊瑚礁适生条件的连贯区域；
- 珊瑚适生至少排除正式河口 / 强河流影响、明显泥沙/淡水过渡及不合适深度；其余条件优先从已有海温、海岸形态、开放度等可解释变量推导；
- 不允许为了“平均分布”破坏热带雨林、针叶林、热带海水等真实大尺度带状结构；
- 调整后必须做多 Seed 覆盖率回归，避免只修固定 seed。

### Slice G — v0.1.7 RC 收敛

必须完成：

- `go test ./...`
- Go ↔ Java Runtime parity / golden
- Save → Load 特殊地貌计划稳定
- 固定 Seed 模型专项
- Minecraft 1.21.1 NeoForge 实际 voxel
- Web 地图位置对照
- v0.1.6 五个 geomV8 视觉回归点不退化
- #509 / #915 河口水墙回归不退化
- 逐点核对全图严格河流审计的历史问题。固定 seed 的 v1.1.25 至 v1.1.27 均有 6 次逆升、27 处相邻水面高差超 1 格。
- 9 目标 `clean check build`
- 9 个候选 JAR loader metadata + 六套 native engine 校验
- 最终人工视觉验收后才允许 tag

## 6. v0.1.7 固定验收集

第一组必须保留：

| Case | 坐标 | 用途 |
|---|---:|---|
| R81-SINKHOLE-01 | -1665 / 5313 | 天坑 #11 与河段 #735 冲突 |
| R81-SINKHOLE-CENTER | -1600 / 5312 | 天坑中心 / 坑壁 / 坑底 |
| R81-OCEAN-01 | -1018 / -982 | v0.1.6 河口水位回归 |
| geomV8 radial regression | -1608 / -1082 | v0.1.6 放射状地形破碎回归 |
| v8 regression 1 | -1047 / -1076 | 河口地形连续 |
| v8 regression 2 | -1015 / -988 | 河口地形连续 |
| v8 regression 3 | -1100 / -1070 | 河口地形连续 |
| v8 regression 4 | 1653 / 10405 | 冻河 / 地形连续 |

R8.2 开始后，再固定至少：

- 1 个 Ocean island；
- 1 个 Lake island；
- 1 个 Snow basin / crater；
- 1 个“不应该生成特殊地貌”的负向控制区。

## 7. 真实地貌参考在 v0.1.7 的边界

v0.1.7 可以使用真实地貌资料帮助判断：

- 岛体应有水下根部；
- 山地盆地应依赖 relief / elevation / climate；
- 河流与天坑不能无因果地互相穿透。

但 **不在 v0.1.7 宣称完成完整 R6.4**。

仍留待后续：

- REF-01/02/03/05/06/07 全部真实 reference envelope；
- floodplain/channel width 等完整形态统计；
- 四固定 Seed 与七类真实 archetype 的全面量化对照；
- 根据稳定真实偏差再调整全局生成参数。

## 8. 明确不进入 v0.1.7 的内容

- 新物品、玩法、任务、UI；
- 地下水 / 岩溶洞穴系统；
- 全局重新设计侵蚀 / sediment；
- 内流盆地完整新语义；
- 模型边界外无限世界正式方案；
- 为了特殊地貌数量而增加随机强制生成；
- 对 v0.1.6 已持久化世界做迁移或重算。

## 9. 发布标准

v0.1.7 只有在以下条件同时满足时才可发布：

1. 天坑与正式河网冲突有真实 voxel 证据并被有限修复；
2. 特殊地貌选址约束有明确因果，不是随机形状补丁；
3. v0.1.6 河口 / geomV8 回归全部保持；
4. 新规则只影响新世界，旧持久化模型不漂移；
5. Model / Runtime / Web / Minecraft voxel 四层一致；
6. 多 Seed 没有新的系统性地形破碎；
7. 9 目标发布矩阵通过；
8. 最终由人工实际走图确认特殊地貌“像自然形成”，而不是只通过数值 audit。

## 10. 建议开发顺序

**先做 R8.1d Slice A → B。**

原因：#11 / #735 是当前唯一已经明确锁定的特殊地貌 × 正式河网冲突；先把“选址必须尊重已有水文”这一条规则建立起来，之后 R8.2 的岛屿和雪山盆地可以复用同一套“背景资格 + rejection reason + deterministic reselect”框架。

完成 R8.1d 并实机通过后，再进入 R8.2，不并行修改多个特殊地貌几何。
