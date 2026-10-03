package geoworld.minecraft.v26_3;

import geoworld.minecraft.nbt.Nbt;
import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.minecraft.v26_3common.Minecraft26_3ProfileMapping;
import geoworld.runtime.GeoWorldV1;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class Minecraft26_3Chunk {
    public static final int DATA_VERSION = 5023;
    public static final int MIN_SECTION_Y = -4;
    public static final int SECTION_COUNT = 24;
    public static final int MAX_SECTION_Y = MIN_SECTION_Y + SECTION_COUNT - 1;

    private static final String AIR = "minecraft:air";
    private static final String STONE = "minecraft:stone";
    private static final String BEDROCK = "minecraft:bedrock";
    private static final String WATER = "minecraft:water";
    private static final String ICE = "minecraft:ice";

    private final GeoWorldV1 model;
    private final HeightMapping heights;

    public Minecraft26_3Chunk(GeoWorldV1 model, HeightMapping heights) {
        if (model == null || heights == null) {
            throw new NullPointerException("model/heights");
        }
        this.model = model;
        this.heights = heights;
    }

    public byte[] writeChunkNbt(int chunkX, int chunkZ) {
        Column[] columns = sampleColumns(chunkX, chunkZ);

        List<Nbt.Tag> sections = new ArrayList<>(SECTION_COUNT);
        for (int sectionY = MIN_SECTION_Y; sectionY <= MAX_SECTION_Y; sectionY++) {
            sections.add(sectionTag(sectionY, chunkX, chunkZ, columns));
        }

        Nbt.CompoundTag heightmaps = Nbt.compound()
            .put("WORLD_SURFACE", Nbt.longArray(heightmap(columns, HeightmapKind.WORLD_SURFACE)))
            .put("OCEAN_FLOOR", Nbt.longArray(heightmap(columns, HeightmapKind.OCEAN_FLOOR)))
            .put("MOTION_BLOCKING", Nbt.longArray(heightmap(columns, HeightmapKind.MOTION_BLOCKING)))
            .put(
                "MOTION_BLOCKING_NO_LEAVES",
                Nbt.longArray(heightmap(columns, HeightmapKind.MOTION_BLOCKING_NO_LEAVES))
            );

        List<Nbt.Tag> postProcessing = new ArrayList<>(SECTION_COUNT);
        for (int i = 0; i < SECTION_COUNT; i++) {
            postProcessing.add(Nbt.list(List.of()));
        }

        Nbt.CompoundTag structures = Nbt.compound()
            .put("starts", Nbt.compound())
            .put("References", Nbt.compound());

        Nbt.CompoundTag root = Nbt.compound()
            .put("DataVersion", Nbt.intTag(DATA_VERSION))
            .put("xPos", Nbt.intTag(chunkX))
            .put("yPos", Nbt.intTag(MIN_SECTION_Y))
            .put("zPos", Nbt.intTag(chunkZ))
            .put("LastUpdate", Nbt.longTag(0))
            .put("InhabitedTime", Nbt.longTag(0))
            .put("Status", Nbt.string("minecraft:full"))
            .put("sections", Nbt.list(sections))
            .put("Heightmaps", heightmaps)
            .put("block_ticks", Nbt.list(List.of()))
            .put("fluid_ticks", Nbt.list(List.of()))
            .put("PostProcessing", Nbt.list(postProcessing))
            .put("block_entities", Nbt.list(List.of()))
            .put("structures", structures);

        return Nbt.writeRoot("", root);
    }

    private Column[] sampleColumns(int chunkX, int chunkZ) {
        Column[] columns = new Column[256];
        int baseX = chunkX * 16;
        int baseZ = chunkZ * 16;

        for (int z = 0; z < 16; z++) {
            for (int x = 0; x < 16; x++) {
                GeoWorldV1.RuntimeSample sample = model.sample(baseX + x, baseZ + z);
                int ground = heights.boundaryY(sample.groundHeight());
                int water = ground;
                if (sample.waterSurfaceDefined() && sample.waterSurfaceElevation() != null) {
                    int mappedWater = "River".equals(sample.waterKind())
                        ? heights.boundaryYCeil(sample.waterSurfaceElevation())
                        : heights.boundaryY(sample.waterSurfaceElevation());
                    if ("River".equals(sample.waterKind())) {
                        mappedWater = Math.max(mappedWater, ground + 1);
                    }
                    water = Math.max(ground, mappedWater);
                }
                columns[z * 16 + x] = new Column(
                    sample,
                    ground,
                    water,
                    Minecraft26_3ProfileMapping.surfaceBlock(sample.surfaceProfile()),
                    Minecraft26_3ProfileMapping.underBlock(sample.surfaceProfile()),
                    Minecraft26_3BiomeRealizer.biomeId(
                        model,
                        baseX + x,
                        baseZ + z
                    )
                );
            }
        }
        return columns;
    }

    private Nbt.CompoundTag sectionTag(
        int sectionY,
        int chunkX,
        int chunkZ,
        Column[] columns
    ) {
        int[] blockPaletteIndexes = new int[4096];
        LinkedHashMap<String, Integer> blockPalette = new LinkedHashMap<>();

        int baseY = sectionY * 16;
        for (int y = 0; y < 16; y++) {
            int worldY = baseY + y;
            for (int z = 0; z < 16; z++) {
                for (int x = 0; x < 16; x++) {
                    Column column = columns[z * 16 + x];
                    String block = blockAt(column, worldY);
                    int paletteIndex = blockPalette.computeIfAbsent(
                        block,
                        ignored -> blockPalette.size()
                    );
                    int index = (y * 16 + z) * 16 + x;
                    blockPaletteIndexes[index] = paletteIndex;
                }
            }
        }

        Nbt.CompoundTag blockStates = Nbt.compound()
            .put("palette", blockStatePalette(blockPalette));
        int blockBits = PackedValues.blockStateBits(blockPalette.size());
        if (blockBits > 0) {
            blockStates.put("data", Nbt.longArray(PackedValues.pack(blockPaletteIndexes, blockBits)));
        }

        int[] biomeIndexes = new int[64];
        LinkedHashMap<String, Integer> biomePalette = new LinkedHashMap<>();
        int baseX = chunkX * 16;
        int baseZ = chunkZ * 16;

        for (int by = 0; by < 4; by++) {
            for (int bz = 0; bz < 4; bz++) {
                for (int bx = 0; bx < 4; bx++) {
                    // Keep the first preview deliberately local-palette only:
                    // 2x2 horizontal biome samples per Chunk -> at most 4 unique IDs.
                    int sampleX = bx < 2 ? 4 : 12;
                    int sampleZ = bz < 2 ? 4 : 12;
                    int sampleY = baseY + by * 4 + 2;
                    String biome = Minecraft26_3BiomeRealizer.biomeId(
                        model,
                        baseX + sampleX,
                        sampleY,
                        baseZ + sampleZ
                    );
                    int paletteIndex = biomePalette.computeIfAbsent(
                        biome,
                        ignored -> biomePalette.size()
                    );
                    int index = (by * 4 + bz) * 4 + bx;
                    biomeIndexes[index] = paletteIndex;
                }
            }
        }

        Nbt.CompoundTag biomes = Nbt.compound()
            .put("palette", stringPalette(biomePalette));
        int biomeBits = PackedValues.biomeBits(biomePalette.size());
        if (biomeBits > 0) {
            biomes.put("data", Nbt.longArray(PackedValues.pack(biomeIndexes, biomeBits)));
        }

        return Nbt.compound()
            .put("Y", Nbt.byteTag(sectionY))
            .put("block_states", blockStates)
            .put("biomes", biomes);
    }

    private long[] heightmap(Column[] columns, HeightmapKind kind) {
        int[] values = new int[256];
        for (int z = 0; z < 16; z++) {
            for (int x = 0; x < 16; x++) {
                Column column = columns[z * 16 + x];
                int firstAvailable = switch (kind) {
                    case OCEAN_FLOOR -> Math.max(
                        HeightMapping.MIN_Y + 1,
                        column.groundBoundary()
                    );
                    case WORLD_SURFACE,
                         MOTION_BLOCKING,
                         MOTION_BLOCKING_NO_LEAVES -> Math.max(
                            HeightMapping.MIN_Y + 1,
                            Math.max(column.groundBoundary(), column.waterBoundary())
                         );
                };
                firstAvailable = Math.min(HeightMapping.MAX_Y_EXCLUSIVE, firstAvailable);
                values[z * 16 + x] = heights.relativeHeightmapValue(firstAvailable);
            }
        }

        // 384-block Overworld height => ceil(log2(385)) = 9 bits.
        return PackedValues.pack(values, 9);
    }

    private static Nbt.ListTag blockStatePalette(Map<String, Integer> palette) {
        List<Nbt.Tag> tags = new ArrayList<>(palette.size());
        for (String block : palette.keySet()) {
            tags.add(Nbt.compound().put("Name", Nbt.string(block)));
        }
        return Nbt.list(tags);
    }

    private static Nbt.ListTag stringPalette(Map<String, Integer> palette) {
        List<Nbt.Tag> tags = new ArrayList<>(palette.size());
        for (String value : palette.keySet()) {
            tags.add(Nbt.string(value));
        }
        return Nbt.list(tags);
    }

    private static String blockAt(Column column, int y) {
        if (y < HeightMapping.MIN_Y || y >= HeightMapping.MAX_Y_EXCLUSIVE) {
            return AIR;
        }
        if (y == HeightMapping.MIN_Y) {
            return BEDROCK;
        }

        if (y < column.groundBoundary()) {
            int depth = column.groundBoundary() - 1 - y;
            if (depth == 0) {
                return column.surfaceBlock();
            }
            if (depth <= 3) {
                return column.underBlock();
            }
            return STONE;
        }

        if (y < column.waterBoundary()) {
            if (
                y == column.waterBoundary() - 1 &&
                "Frozen Ocean".equals(column.sample().biomeFamily()) &&
                "Ocean".equals(column.sample().waterKind())
            ) {
                return ICE;
            }
            return WATER;
        }

        return AIR;
    }

    private static String surfaceBlock(String profile) {
        return Minecraft26_3ProfileMapping.surfaceBlock(profile);
    }

    private static String underBlock(String profile) {
        return Minecraft26_3ProfileMapping.underBlock(profile);
    }

    public static String biomeId(String family) {
        return Minecraft26_3ProfileMapping.biomeId(family);
    }

    private enum HeightmapKind {
        WORLD_SURFACE,
        OCEAN_FLOOR,
        MOTION_BLOCKING,
        MOTION_BLOCKING_NO_LEAVES
    }

    private record Column(
        GeoWorldV1.RuntimeSample sample,
        int groundBoundary,
        int waterBoundary,
        String surfaceBlock,
        String underBlock,
        String biome
    ) {}
}
