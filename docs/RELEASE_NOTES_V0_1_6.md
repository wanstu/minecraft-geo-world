# GeoWorld v0.1.6 — 发布候选说明

状态：**自动化发布门槛已通过，人工游戏内视觉验收待完成，当前禁止正式发布。**

## 核心变化

v0.1.6 聚焦 R8.1c 河口接收水位与低岸水体一致性，不扩展新的世界生成玩法。规则版本升级为 `geoworld-rules-v1.1.23`，RiverGeometryVersion 保持 7。

本轮修复正式 OceanMouth 接收器与局部 Lake 所有权的优先级冲突。此前 #509 河口的真实 Anvil 方块仍存在 50 对相邻湿格水面高差超过 1 格的水墙；修复后 #509 与 #915 全区域扫描均为 0，且旧世界负向对照仍能稳定检出 50 对，证明验收脚本能够捕捉该缺陷。

## 兼容性与发布矩阵

旧 `v1.1.22` 及更早世界不启用 CoastalBackwaterVersion=1，不迁移旧世界语义。发布矩阵共 9 个目标，全部完成 `clean check build`：

- Minecraft 26.3：NeoForge、Fabric
- Minecraft 26.2：NeoForge、Fabric
- Minecraft 26.1.2：NeoForge、Fabric
- Minecraft 1.21.1：NeoForge、Fabric
- Minecraft 1.20.1：Fabric

9 个实际 v0.1.6 候选 JAR 均已确认 Loader 元数据版本为 0.1.6，并包含 Windows、Linux、macOS 的 x64/arm64 六套 GeoWorld 原生引擎。

## 当前唯一未完成门槛

仍需在 Minecraft 客户端对 R8.1c 重点区域做视觉验收，确认不存在“数值连续但视觉不自然”的河口、海岸或低岸回水形态。重点坐标：

- #509 河口核心约 `(-1069, -1026/-1027)`
- #509 原水墙条带约 `(-1093, -998)`
- #915 控制区中心约 `(1664, 10432)`

在该人工验收通过前，不创建 `v0.1.6` tag，不发布 GitHub Release。
