# GeoWorld Minecraft Save Writer

这一模块负责把 GeoWorld World Model 输出为 Minecraft Java 世界存档。

## 当前目标版本

- Minecraft Java Edition 26.3
- DataVersion: 5023
- level.dat storage version: 19133
- Overworld: min_y=-64, height=384, sea_level=63

版本值和 26.3 NBT schema 均从 Mojang 26.3 官方 server jar / 内置数据核对；main writer 本身不依赖 Mojang jar。

## 已实现

- 标准 NBT 全类型写出与 gzip NBT 文件；
- 标准 UTF-8 NBT String；
- Anvil .mca Region header / location / timestamp；
- 正负 Chunk 坐标 floorDiv/floorMod；
- zlib Chunk payload 与 4 KiB sector 分配；
- 26.3 Section block_states / biomes local palette；
- 26.3 SimpleBitStorage 兼容的 non-crossing long packing；
- WORLD_SURFACE / OCEAN_FLOOR / MOTION_BLOCKING / MOTION_BLOCKING_NO_LEAVES heightmaps；
- 26.3 level.dat；
- 26.3 data/world_gen_settings.dat；
- GeoWorld V1 Runtime Sample -> 地形 / 水体 / 群系 / 表层方块；
- 自动选择非海洋、无 semantic fallback/mismatch 的安全出生点；
- 按 Chunk 半径预生成小范围 Overworld。

## 高度映射

第一版 HeightMapping 以 Minecraft 26.3 官方 Overworld sea_level=63 为锚点：

    GeoWorld 0.0 -> Y boundary -64
    GeoWorld 0.5 -> Y boundary 63
    GeoWorld 1.0 -> Y boundary 320

地面方块占 y < groundBoundary；水占 groundBoundary <= y < waterBoundary。

## 生成 Preview World

仓库根目录：

    java\minecraft-save-writer\generate-preview-world.cmd --force

默认：

- 输入：internal/goldenvectors/testdata/runtime-v1.geoworld
- 输出：output/minecraft-26.3-preview/GeoWorld-Preview
- 预生成半径：8 chunks（17x17 chunks）

可选参数：

    --model <file.geoworld>
    --output <world-directory>
    --name <level-name>
    --radius <0..31>
    --force

## 本地验证

    java\minecraft-save-writer\verify-storage.cmd

该命令会验证：

- 已知 NBT 字节向量与 UTF-8；
- Region header / zlib / sector / 负坐标；
- 26.3 HeightMapping 锚点；
- block-state / biome / heightmap bit packing；
- 3x3 Chunk preview save 生成和 Region 逐 Chunk round-trip。

## Mojang 26.3 官方 codec 验收

需要 JDK 25+（当前开发机 JDK 27）。验证脚本会先检查官方 26.3 server jar 的固定 SHA-1，再在不启动 server、不接受 EULA 的情况下调用 Mojang 26.3 官方类做反序列化验收：

    java\minecraft-save-writer\verify-mojang-26.3.cmd -World output\minecraft-26.3-preview\GeoWorld-Preview -Radius 8

已验证：

- NbtIo + LevelVersion / LevelSettings 成功读取生成的 level.dat；
- WorldGenSettings.CODEC 成功读取 data/world_gen_settings.dat，Seed 与 overworld/nether/end 三维度全部通过；
- SerializableChunkData.parse 先通过 3x3 smoke，再对正式 radius=8（17x17）的 289/289 个生成 Chunk 全量解析通过。

官方 server jar 与解包内容仅放在 output/ 调试目录，不进入 Git。

## 当前 Preview 限制

- 目标是先验证 GeoWorld 的大陆、地形、水体和群系在真实 Minecraft 世界中的空间结果；
- 表层材质是简化映射，尚未加入完整植被、树木、洞穴、矿物和 GeoWorld 专属 surface rule；
- 预生成 Chunk 不写结构、实体、block entity；
- isLightOn 不置 true，首次加载由 Minecraft 补光；
- 预生成范围之外仍使用 vanilla 26.3 normal worldgen，因此边界暂时可能出现接缝；
- 超过 255 sectors 的 Chunk 仍明确拒绝，external .mcc 尚未实现。

下一阶段先进行 Minecraft 26.3 客户端真实进入世界验收；确认存档可进且 GeoWorld 结果可读后，再继续 surface/vegetation 与边界策略，不急着接 Loader。
