# GeoWorld Minecraft Save Writer

这一模块负责 GeoWorld -> Minecraft 世界存档的版本隔离写出层。

当前第一切片只实现版本无关的底层存储协议：

- 标准 NBT 全类型写出；
- 标准 UTF-8 NBT String；
- Anvil .mca Region header；
- 32x32 Chunk slot 映射；
- 正负 Chunk 坐标 floorDiv/floorMod；
- zlib Chunk payload；
- 4 KiB sector 分配与 timestamp table；
- 超过 255 sectors 的 Chunk 明确拒绝，尚不写 external .mcc。

当前还没有：

- level.dat；
- Minecraft DataVersion；
- 具体版本 Chunk NBT schema；
- block state palette；
- biome palette；
- heightmap/light/entities；
- 实际可进入游戏的 save。

验证：

    java\minecraft-save-writer\verify-storage.cmd

验证会检查已知 NBT 字节向量、UTF-8 字符串、Region header/sector/zlib round-trip，以及负 Chunk 坐标映射。

下一切片将在这个稳定存储层之上增加 MinecraftVersionAdapter，再输出小范围真实世界存档。
