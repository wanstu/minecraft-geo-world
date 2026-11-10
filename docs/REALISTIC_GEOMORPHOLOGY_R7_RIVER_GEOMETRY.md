# GeoWorld R7.1 — 低坡降宽河道直段改善

日期：2026-11-10  
阶段：R7.1 定向修正，准备 `v0.1.5-rc.3` 测试版（发布前仍需通过全部 CI）。

## 1. 问题来自实际存档，而不是臆测

`ttt`（MC 1.21.1 NeoForge、Seed `-1818694639628404530`、锁定规则 v1.1.20）实际区块俯视审查：

- 953 条保存河段中 213 条 `MeanderOffset == 0`（22.4%）。
- 半宽至少 8 方块的河段 652 条，其中 133 条直线（20.4%）。
- 西部 `(-3256, -1832)` 附近，segment #48 `(-3264,-1856)->(-3264,-1728)`，长度 128 方块、半宽约 16.86、原偏移 0；真实俯视存在笔直水道。
- 该形状是 `waterbed.riverMeanderOffset` 旧版的 20% 直段分支导致，**不是区块缝隙或存档损坏**。

保留的离线审查：`build/ttt-acceptance/TTT_VISUAL_ACCEPTANCE.html` 和 `TTT_R7_VISUAL_REPORT.md`。地形截图并非卫星 DEM 的真实性判定。

## 2. 最小算法调整

新世界采用 `geoworld-rules-v1.1.21`，其 `Waterbed.RiverGeometryVersion=6`；旧版 v1.1.20 始终为 5。

v6 仅对旧版 `u < 0.20` 的部分完全笔直河段施加轻微弯曲，需要同时满足：

1. 宽度：halfWidth >= 10 方块；
2. 河段长度：>= 96 方块；
3. 低坡降权重 `lowSlope >= 0.70`，高坡降河段不改变；
4. 原中心弦为轴向（水平/垂直），不动已有斜向防网格化分支；
5. `0.015 <= u < 0.20`，仍保留小比例直段，以及大量窄河/急流/不适宜弯曲的直段；
6. 河口、湖泊流入口、湖泊出口和入汇合的阻尼规则保持 v5 语义。

单个河段偏移绝对值上界：

`min(0.75*halfWidth, 0.085*length, 0.66*maxOffset*lowSlope)`

其中 `maxOffset=min(0.17*length, 5+1.35*halfWidth)`，对一般宽河不会形成反常大蛇形。

河道复用 `sin(pi*t)^2` 曲线：**起止中心线节点坐标不变**，两端新增弯曲项导数为 0；沉积、河床、自由水面和河岸收敛仍使用原有流程。

> 限制：只削减“长直段”问题。不同河段的宏观弦方向发生大角度变化时，**节点的转角仍可能明显**；不声称已完成全网段切线连续的河道改造。若确有必要，R7.2 再独立研究节点连接曲线，不在本版加大改动面。

## 3. 旧世界不变

- `ttt` 世界模型与存档没有写入或替换。
- 已有存档根据自己持久化的 `generationRulesVersion` 和 `RiverGeometryVersion` 读取，不强制转换为 v6。
- Java 阅读器保留所有旧规则识别，且添加 v21 可识别分支；客户端缓存格式从 `v24` 到 `v25` 防止新的规则误复用上次缓存。
- 新规则没有修改河网节点、河道宽度、世界生成尺度、矿区、玩法或其他维度。

## 4. 可重复验收

- `internal/waterbed/TestV121WideCardinalStraightReachCandidate`：确定性、老版本不变、窄河/陡坡/短河不受影响、节点不漂移、偏移有界。
- `internal/waterbed/TestV121ObservedTTTStraightReachGetsBoundedBend`：使用真实 `ttt` Segment #48 的 Seed / 节点身份复现。v5 仍直、v6 按条件弯曲且重复输出确定。
- `internal/generator/TestV121MeanderGeometryIsVersionLocked`：锁定 v20 vs 新 v21 固定 Seed 的完整小世界对照；节点、宽度、河口上下游连接状态与沉积模型保持一致；River Profile 不得逆坡。
- `go test ./...`：整仓回归。Windows 本地忽略目录里的独立 Go 审计脚本已加 `//go:build ignore`，避免无关的多个 main 入口污染包检查（不会进入 Git）。
- `java/runtime-reader/verify-golden.ps1 -All`：全部已提交旧模型快照跨语言验证。
- `java/adapters/1.21.1-neoforge/gradlew.bat check --offline --no-daemon`：实际 NeoForge 构建侧验收，包括河岸水面和方块几何。

## 5. 发布门槛与下一轮

本地结果（2026-11-10）：`go test -p 2 ./...` **PASS**、`java/runtime-reader/verify-golden.ps1 -All` **PASS**、1.21.1 NeoForge `gradlew check --offline --no-daemon` **PASS**。另已读取并校验原有 `ttt` 模型，仍锁定 v1.1.20 且 SHA256/模型结构有效。GitHub CI 的全部版本构建仍需单独确认。

只有完整 Go 测试、Java 跨语言测试、NeoForge 实际验收与 GitHub CI 构建通过后，才能打 `v0.1.5-rc.3` 测试标签。**不直接当作“符合现实”的正式版发布。**

旧存档可做兼容性回归，但**比较新旧地貌必须使用新建世界**；已生成的 `ttt` 地形不会自动弯曲。下一轮以可复现截图（同 Seed/坐标/规则）观察弯曲自然度，而不是无限添加生成参数。
