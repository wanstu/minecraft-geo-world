package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.v26common.MinecraftV26Profile;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Random;
import java.util.Set;

/**
 * Standalone Minecraft 26.3 stronghold planner used by the executable GeoWorld
 * Web tool. This class intentionally has no net.minecraft dependency so it can
 * run from the mod JAR outside Minecraft.
 *
 * Its ring/random/biome-bias behavior is parity-checked against Minecraft's
 * ChunkGeneratorStructureState during the Fabric build.
 */
public final class GeoWorldV26_3StrongholdPreviewCli {
    public static final String SET_ID = "minecraft:strongholds";
    public static final int DISTANCE = 32;
    public static final int INITIAL_SPREAD = 3;
    public static final int COUNT = 128;
    public static final int BIOME_SEARCH_RADIUS_BLOCKS = 112;

    private static final Set<String> PREFERRED_BIOMES = Set.of(
        "minecraft:plains",
        "minecraft:sunflower_plains",
        "minecraft:snowy_plains",
        "minecraft:ice_spikes",
        "minecraft:desert",
        "minecraft:forest",
        "minecraft:flower_forest",
        "minecraft:birch_forest",
        "minecraft:dark_forest",
        "minecraft:pale_garden",
        "minecraft:old_growth_birch_forest",
        "minecraft:old_growth_pine_taiga",
        "minecraft:old_growth_spruce_taiga",
        "minecraft:taiga",
        "minecraft:snowy_taiga",
        "minecraft:savanna",
        "minecraft:savanna_plateau",
        "minecraft:windswept_hills",
        "minecraft:windswept_gravelly_hills",
        "minecraft:windswept_forest",
        "minecraft:windswept_savanna",
        "minecraft:jungle",
        "minecraft:sparse_jungle",
        "minecraft:bamboo_jungle",
        "minecraft:badlands",
        "minecraft:eroded_badlands",
        "minecraft:wooded_badlands",
        "minecraft:meadow",
        "minecraft:cherry_grove",
        "minecraft:grove",
        "minecraft:snowy_slopes",
        "minecraft:frozen_peaks",
        "minecraft:jagged_peaks",
        "minecraft:stony_peaks",
        "minecraft:mushroom_fields",
        "minecraft:dripstone_caves",
        "minecraft:lush_caves",
        "minecraft:sulfur_caves"
    );

    private static final GeoWorldV26BiomeRealizer BIOMES =
        new GeoWorldV26BiomeRealizer(
            new MinecraftV26Profile("26.3", true, true, true)
        );

    private GeoWorldV26_3StrongholdPreviewCli() {}

    public record Position(
        int index,
        int chunkX,
        int chunkZ,
        int x,
        int z
    ) {}

    public static Set<String> preferredBiomes() {
        return PREFERRED_BIOMES;
    }

    public static List<Position> positions(GeoWorldV1 model) {
        if (model == null) throw new NullPointerException("model");

        long seed = model.manifest().seed();
        Random random = new Random(seed);
        double angle = random.nextDouble() * Math.PI * 2.0;
        int withinRing = 0;
        int ring = 0;
        int spread = INITIAL_SPREAD;
        ArrayList<Position> result = new ArrayList<>(COUNT);

        for (int index = 0; index < COUNT; index++) {
            double radius =
                4.0 * DISTANCE +
                DISTANCE * ring * 6.0 +
                (random.nextDouble() - 0.5) * DISTANCE * 2.5;

            int chunkX = (int)Math.round(Math.cos(angle) * radius);
            int chunkZ = (int)Math.round(Math.sin(angle) * radius);

            // Minecraft forks the LegacyRandomSource before the async biome
            // lookup. java.util.Random has the same 48-bit LCG and method
            // semantics as LegacyRandomSource.
            Random biomeRandom = new Random(random.nextLong());
            int[] biased = biasToPreferredBiome(
                model,
                chunkX,
                chunkZ,
                biomeRandom
            );
            chunkX = biased[0];
            chunkZ = biased[1];

            result.add(new Position(
                index,
                chunkX,
                chunkZ,
                chunkX * 16 + 8,
                chunkZ * 16 + 8
            ));

            angle += Math.PI * 2.0 / spread;
            withinRing++;
            if (withinRing == spread) {
                ring++;
                withinRing = 0;
                spread += 2 * spread / (ring + 1);
                spread = Math.min(spread, COUNT - index);
                angle += random.nextDouble() * Math.PI * 2.0;
            }
        }
        return List.copyOf(result);
    }

    private static int[] biasToPreferredBiome(
        GeoWorldV1 model,
        int chunkX,
        int chunkZ,
        Random random
    ) {
        // Mirrors BiomeSource.findBiomeHorizontal(... radius=112, step=1,
        // findClosest=false). QuartPos.fromBlock is floorDiv(block, 4).
        int centerBlockX = chunkX * 16 + 8;
        int centerBlockZ = chunkZ * 16 + 8;
        int centerQuartX = Math.floorDiv(centerBlockX, 4);
        int centerQuartZ = Math.floorDiv(centerBlockZ, 4);
        int radiusQuart = Math.floorDiv(BIOME_SEARCH_RADIUS_BLOCKS, 4);

        Integer selectedBlockX = null;
        Integer selectedBlockZ = null;
        int matches = 0;

        for (int dz = -radiusQuart; dz <= radiusQuart; dz++) {
            for (int dx = -radiusQuart; dx <= radiusQuart; dx++) {
                int quartX = centerQuartX + dx;
                int quartZ = centerQuartZ + dz;
                int blockX = quartX * 4;
                int blockZ = quartZ * 4;
                String biome = BIOMES.biomeId(model, blockX, 0, blockZ);
                if (!PREFERRED_BIOMES.contains(biome)) {
                    continue;
                }

                if (selectedBlockX == null || random.nextInt(matches + 1) == 0) {
                    selectedBlockX = blockX;
                    selectedBlockZ = blockZ;
                }
                matches++;
            }
        }

        if (selectedBlockX == null) {
            return new int[] {chunkX, chunkZ};
        }
        return new int[] {
            Math.floorDiv(selectedBlockX, 16),
            Math.floorDiv(selectedBlockZ, 16)
        };
    }

    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            throw new IllegalArgumentException("expected <model.geoworld>");
        }
        GeoWorldV1 model = GeoWorldV1.load(
            Path.of(args[0]).toAbsolutePath().normalize()
        );
        List<Position> positions = positions(model);

        StringBuilder out = new StringBuilder(positions.size() * 72);
        out.append('{');
        numberField(out, "seed", model.manifest().seed()).append(',');
        stringField(out, "setId", SET_ID).append(',');
        numberField(out, "distance", DISTANCE).append(',');
        numberField(out, "spread", INITIAL_SPREAD).append(',');
        numberField(out, "count", COUNT).append(',');
        out.append("\"positions\":[");

        for (int i = 0; i < positions.size(); i++) {
            if (i > 0) out.append(',');
            Position p = positions.get(i);
            out.append('{');
            numberField(out, "index", p.index()).append(',');
            numberField(out, "chunkX", p.chunkX()).append(',');
            numberField(out, "chunkZ", p.chunkZ()).append(',');
            numberField(out, "x", p.x()).append(',');
            numberField(out, "z", p.z());
            out.append('}');
        }
        out.append("]}");
        System.out.println(out);
    }

    private static StringBuilder stringField(
        StringBuilder out,
        String name,
        String value
    ) {
        return out.append('"').append(name).append("\":\"")
            .append(escape(value)).append('"');
    }

    private static StringBuilder numberField(
        StringBuilder out,
        String name,
        long value
    ) {
        return out.append('"').append(name).append("\":").append(value);
    }

    private static String escape(String value) {
        if (value == null) return "";
        return value
            .replace("\\", "\\\\")
            .replace("\"", "\\\"");
    }
}
