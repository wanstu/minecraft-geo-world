package geoworld.minecraft.v1_21_1common.worldgen;

import com.mojang.serialization.Lifecycle;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.common.worldgen.GeoWorldMineralRules;
import geoworld.minecraft.common.worldgen.GeoWorldMiningDistrictPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;
import geoworld.minecraft.v26common.worldgen.GeoWorldOreCatalog;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26MineralVeins;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.SharedConstants;
import net.minecraft.core.BlockPos;
import net.minecraft.core.Holder;
import net.minecraft.core.MappedRegistry;
import net.minecraft.core.Registry;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.core.registries.Registries;
import net.minecraft.data.registries.VanillaRegistries;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.server.Bootstrap;
import net.minecraft.tags.TagKey;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.LevelHeightAccessor;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.Biomes;
import net.minecraft.world.level.block.Block;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ProtoChunk;
import net.minecraft.world.level.chunk.UpgradeData;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Isolated native Minecraft acceptance for ore tags -> frozen model families
 * -> real chunk blocks. Fixture tags use existing vanilla ore blocks, so no
 * content mod or third-party registry is required and no game save is opened.
 */
public final class GeoWorldMiningMaterialVerifier {
    private static final String EXTRA = "catalog_fixture";
    private static final MinecraftV26HeightMapping HEIGHTS = new MinecraftV26HeightMapping();
    private static final GeoWorldShorelinePlanner SHORE = new GeoWorldShorelinePlanner(HEIGHTS);
    private static final GeoWorldMiningDistrictPlanner DISTRICTS =
        new GeoWorldMiningDistrictPlanner(HEIGHTS);
    private static final Map<String, Block[]> MATERIALS = new LinkedHashMap<>();
    private static int materialChecks;
    private static int nativeOreBlocks;
    private static int nativeExtraBlocks;
    private static int pileBlocks;
    private static int wetPileBlocks;
    private static int protectedChecks;
    private static int peakChecks;
    private static final Map<Long, int[]> CENTERS = new LinkedHashMap<>();

    private GeoWorldMiningMaterialVerifier() {}

    public static void main(String[] args) throws Exception {
        SharedConstants.tryDetectVersion();
        Bootstrap.bootStrap();
        MATERIALS.put("coal", new Block[]{Blocks.COAL_ORE, Blocks.DEEPSLATE_COAL_ORE});
        MATERIALS.put("iron", new Block[]{Blocks.IRON_ORE, Blocks.DEEPSLATE_IRON_ORE});
        MATERIALS.put("copper", new Block[]{Blocks.COPPER_ORE, Blocks.DEEPSLATE_COPPER_ORE});
        MATERIALS.put("gold", new Block[]{Blocks.GOLD_ORE, Blocks.DEEPSLATE_GOLD_ORE});
        MATERIALS.put("lapis", new Block[]{Blocks.LAPIS_ORE, Blocks.DEEPSLATE_LAPIS_ORE});
        MATERIALS.put("redstone", new Block[]{Blocks.REDSTONE_ORE, Blocks.DEEPSLATE_REDSTONE_ORE});
        MATERIALS.put("diamond", new Block[]{Blocks.DIAMOND_ORE, Blocks.DEEPSLATE_DIAMOND_ORE});
        MATERIALS.put("emerald", new Block[]{Blocks.EMERALD_ORE, Blocks.DEEPSLATE_EMERALD_ORE});
        // Deliberately unrelated block IDs prove that family tags, not names
        // or a hard-coded content-mod dependency, decide material membership.
        MATERIALS.put(EXTRA, new Block[]{Blocks.NETHER_QUARTZ_ORE, Blocks.NETHER_GOLD_ORE});
        installTags(true);
        var catalog = GeoWorldOreCatalog.prepareGenerationContext();
        require(catalog.families().size() == 9 && catalog.families().contains(EXTRA),
            "tagged fixture family not available: " + catalog.families());
        require(catalog.dictionaryFamilies().contains("dictionary_only") &&
            !catalog.families().contains("dictionary_only"),
            "a dictionary-only family must not become a placeable mineral");

        long seed = 123456L;
        var config = GeoWorldCreationConfig.defaults().withSurfaceOrePile(true);
        GeoWorldV1 model = GeoWorldEngineBridge.generate(seed, config).model();
        var frozen = List.copyOf(model.mineralDistrictAssignments());
        require(!frozen.isEmpty(), "no persisted district assignments");
        List<GeoWorldMiningDistrictPlanner.District> districts = districts(model, seed);
        districts.forEach(d -> CENTERS.put(d.id(), new int[]{
            (int)Math.round(d.centerX()), (int)Math.round(d.centerZ())}));
        int extraPrimary = 0;
        int extraCompanion = 0;
        for (var district : districts) {
            var assignment = model.mineralDistrictAssignment(district.id());
            require(assignment != null, "district missing frozen family assignment");
            if (EXTRA.equals(assignment.primary())) extraPrimary++;
            if (EXTRA.equals(assignment.companion())) extraCompanion++;
            for (boolean companion : new boolean[]{false, true}) {
                String family = companion ? assignment.companion() : assignment.primary();
                for (boolean deep : new boolean[]{false, true}) {
                    var placement = placement(district.id(), companion, deep ? -30 : 20);
                    BlockState host = deep ? Blocks.DEEPSLATE.defaultBlockState() :
                        Blocks.STONE.defaultBlockState();
                    Block actual = GeoWorldOreCatalog.resolve(model, placement, host,
                        Blocks.IRON_ORE.defaultBlockState()).getBlock();
                    require(actual == MATERIALS.get(family)[deep ? 1 : 0],
                        "wrong district material family=" + family + " companion=" +
                        companion + " deep=" + deep + " actual=" + id(actual));
                    materialChecks++;
                }
                BlockState marker = GeoWorldOreCatalog.surfaceMarkerBlock(
                    model, district.id(), companion);
                require(marker != null && marker.getBlock() == MATERIALS.get(family)[0],
                    "surface marker must be the model's ore family, not a storage block");
            }
        }
        require(extraPrimary > 0 && extraCompanion > 0,
            "fixture must be exercised as primary and companion: " +
            extraPrimary + "/" + extraCompanion);

        Registry<Biome> biomes = biomeRegistry();
        Set<Long> probes = new LinkedHashSet<>();
        var extra = districts.stream().filter(d -> {
            var a = model.mineralDistrictAssignment(d.id());
            return EXTRA.equals(a.primary()) || EXTRA.equals(a.companion());
        }).findFirst().orElseThrow();
        probes.add(chunkKey(extra));
        districts.stream().filter(d -> "None".equals(
            model.sample(d.centerX(), d.centerZ()).waterKind()
        )).findFirst().ifPresent(d -> probes.add(chunkKey(d)));
        var wet = districts.stream().filter(d -> !"None".equals(
            model.sample(d.centerX(), d.centerZ()).waterKind()
        )).findFirst().orElseThrow(() -> new AssertionError("no water-floor fixture"));
        probes.add(chunkKey(wet));
        for (long key : probes) {
            verifyChunk(model, seed, biomes, (int)(key >> 32), (int)key, true);
        }
        require(nativeOreBlocks > 1000 && nativeExtraBlocks > 0,
            "tagged minerals did not reach native chunk blocks");
        require(peakChecks > 0, "no native 5-8-block peak checked");
        require(pileBlocks > 0 && wetPileBlocks > 0,
            "land/water-floor marker blocks were not placed");

        GeoWorldV1 disabled = GeoWorldEngineBridge.generate(
            seed, config.withSurfaceOrePile(false)).model();
        long offProbe = chunkKey(extra);
        verifyChunk(disabled, seed, biomes,
            (int)(offProbe >> 32), (int)offProbe, false);
        require(!disabled.surfaceOrePileEnabled(), "disabled option changed");

        installTags(false);
        GeoWorldOreCatalog.prepareGenerationContext();
        require(!GeoWorldOreCatalog.families().contains(EXTRA),
            "removed fixture family still available");
        int removedChecks = 0;
        for (var district : districts) {
            var a = model.mineralDistrictAssignment(district.id());
            for (boolean companion : new boolean[]{false, true}) {
                if (!EXTRA.equals(companion ? a.companion() : a.primary())) continue;
                BlockState host = Blocks.STONE.defaultBlockState();
                BlockState resolved = GeoWorldOreCatalog.resolve(model,
                    placement(district.id(), companion, 20), host,
                    Blocks.IRON_ORE.defaultBlockState());
                require(resolved == host, "removed family silently reassigned");
                require(GeoWorldOreCatalog.surfaceMarkerBlock(
                    model, district.id(), companion) == null,
                    "removed family created a replacement marker");
                removedChecks++;
            }
        }
        require(removedChecks > 0 && frozen.equals(model.mineralDistrictAssignments()),
            "registry changes mutated frozen model assignments");
        System.out.printf(
            "MINING_MATERIALS PASS seed=%d families=9 districts=%d mappings=%d " +
            "extraPrimary=%d extraCompanion=%d nativeOre=%d nativeExtra=%d " +
            "pileBlocks=%d waterFloorPileBlocks=%d protected=%d peaks=%d removed=%d " +
            "disabled=PASS frozen=PASS%n",
            seed, districts.size(), materialChecks, extraPrimary, extraCompanion,
            nativeOreBlocks, nativeExtraBlocks, pileBlocks, wetPileBlocks,
            protectedChecks, peakChecks, removedChecks);
    }

    private static void verifyChunk(GeoWorldV1 model, long seed, Registry<Biome> biomes,
                                    int cx, int cz, boolean pilesEnabled) {
        int minX = cx * 16, minZ = cz * 16;
        var plan = SHORE.planChunkModel(model, minX, minZ);
        var chunk = new ProtoChunk(new ChunkPos(cx, cz), UpgradeData.EMPTY,
            LevelHeightAccessor.create(HEIGHTS.minY(), HEIGHTS.height()), biomes, null);
        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();
        for (int z = 0; z < 16; z++) for (int x = 0; x < 16; x++) {
            var col = plan.column(x, z);
            for (int y = HEIGHTS.minY(); y < Math.max(
                col.groundBoundary(), col.waterBoundary()); y++) {
                BlockState state = y == HEIGHTS.minY() ? Blocks.BEDROCK.defaultBlockState() :
                    y < col.groundBoundary() ?
                        (y < 0 ? Blocks.DEEPSLATE : Blocks.STONE).defaultBlockState() :
                        Blocks.WATER.defaultBlockState();
                chunk.setBlockState(pos.set(minX + x, y, minZ + z), state, false);
            }
        }
        Map<BlockPos, GeoWorldMiningDistrictPlanner.Placement> placements = new LinkedHashMap<>();
        DISTRICTS.plan(model, seed, minX, minX + 15, minZ, minZ + 15, p ->
            placements.putIfAbsent(new BlockPos(p.x(), p.y(), p.z()), p));
        Map<BlockPos, BlockState> protectedBlocks = new LinkedHashMap<>();
        BlockState[] protections = {Blocks.GOLD_BLOCK.defaultBlockState(),
            Blocks.AIR.defaultBlockState(), Blocks.WATER.defaultBlockState()};
        for (var entry : placements.entrySet()) {
            var p = entry.getValue();
            if (p.y() >= plan.column(p.x() - minX, p.z() - minZ).groundBoundary() - 4)
                continue;
            BlockState state = protections[protectedBlocks.size()];
            chunk.setBlockState(entry.getKey(), state, false);
            protectedBlocks.put(entry.getKey(), state);
            if (protectedBlocks.size() == protections.length) break;
        }
        require(protectedBlocks.size() == protections.length, "no host-protection samples");
        GeoWorldV26MineralVeins.populate(chunk, model, seed, plan,
            (target, at, state) -> target.setBlockState(at, state, false));
        for (int z = 0; z < 16; z++) for (int x = 0; x < 16; x++) {
            require(chunk.getBlockState(pos.set(minX + x, HEIGHTS.minY(), minZ + z))
                .is(Blocks.BEDROCK), "mineral placement overwrote bedrock");
        }
        for (var entry : protectedBlocks.entrySet()) {
            require(chunk.getBlockState(entry.getKey()) == entry.getValue(),
                "mineral placement overwrote a protected/non-rock block at " + entry.getKey());
            protectedChecks++;
        }
        for (var entry : placements.entrySet()) {
            var p = entry.getValue();
            if (protectedBlocks.containsKey(entry.getKey()) ||
                p.y() >= plan.column(p.x() - minX, p.z() - minZ).groundBoundary() - 4)
                continue;
            var a = model.mineralDistrictAssignment(p.districtId());
            String family = p.companion() ? a.companion() : a.primary();
            Block expected = MATERIALS.get(family)[p.y() < 0 ? 1 : 0];
            Block actual = chunk.getBlockState(entry.getKey()).getBlock();
            require(actual == expected, "native block mismatch at " + entry.getKey() +
                " family=" + family + " expected=" + id(expected) + " actual=" + id(actual));
            nativeOreBlocks++;
            if (EXTRA.equals(family)) nativeExtraBlocks++;
        }
        List<GeoWorldV26MineralVeins.SurfaceOrePileColumn> piles = new ArrayList<>();
        // Disabled models must leave the same marker positions as air/water.
        GeoWorldV26MineralVeins.planSurfaceOrePiles(
            model, seed, minX, minX + 15, minZ, minZ + 15, piles::add);
        if (!pilesEnabled) {
            require(piles.isEmpty(), "disabled model planned marker piles");
            for (int z = 0; z < 16; z++) for (int x = 0; x < 16; x++) {
                var col = plan.column(x, z);
                for (int y = col.groundBoundary();
                     y < Math.min(HEIGHTS.maxYExclusive(), col.groundBoundary() + 8); y++) {
                    Block expected = y < col.waterBoundary() ? Blocks.WATER : Blocks.AIR;
                    require(chunk.getBlockState(pos.set(minX + x, y, minZ + z)).is(expected),
                        "disabled model wrote an above-ground marker");
                }
            }
            return;
        }
        require(!piles.isEmpty(), "probe chunk missed its district marker");
        for (var pile : piles) {
            var col = plan.column(pile.x() - minX, pile.z() - minZ);
            var a = model.mineralDistrictAssignment(pile.districtId());
            String family = pile.companion() ? a.companion() : a.primary();
            Block expected = MATERIALS.get(family)[0];
            int[] center = CENTERS.get(pile.districtId());
            if (pile.x() == center[0] && pile.z() == center[1]) {
                require(pile.levels() >= 5 && pile.levels() <= 8 &&
                    col.groundBoundary() + pile.levels() <= HEIGHTS.maxYExclusive(),
                    "native pile peak must retain 5-8 blocks above its local floor");
                int aboveY = col.groundBoundary() + pile.levels();
                Block above = aboveY < col.waterBoundary() ? Blocks.WATER : Blocks.AIR;
                require(chunk.getBlockState(pos.set(pile.x(), aboveY, pile.z())).is(above),
                    "native pile peak extends above its planned height");
                peakChecks++;
            }
            for (int level = 0; level < pile.levels(); level++) {
                int y = col.groundBoundary() + level;
                if (y >= HEIGHTS.maxYExclusive()) break;
                Block actual = chunk.getBlockState(pos.set(pile.x(), y, pile.z())).getBlock();
                require(actual == expected, "native marker material/height mismatch");
                pileBlocks++;
                if (y < col.waterBoundary()) wetPileBlocks++;
            }
        }
    }

    private static Registry<Biome> biomeRegistry() {
        var provider = VanillaRegistries.createLookup();
        Biome plains = provider.lookupOrThrow(Registries.BIOME).getOrThrow(Biomes.PLAINS).value();
        MappedRegistry<Biome> biomes = new MappedRegistry<>(Registries.BIOME, Lifecycle.stable());
        Registry.register(biomes, Biomes.PLAINS, plains);
        return biomes.freeze();
    }

    private static List<GeoWorldMiningDistrictPlanner.District> districts(GeoWorldV1 model, long seed) {
        var geo = model.manifest().geometry();
        List<GeoWorldMiningDistrictPlanner.District> out = new ArrayList<>();
        DISTRICTS.planDistricts(model, seed, (int)geo.originX(),
            (int)(geo.originX() + geo.spanXBlocks()), (int)geo.originZ(),
            (int)(geo.originZ() + geo.spanZBlocks()), out::add);
        return out;
    }

    private static GeoWorldMiningDistrictPlanner.Placement placement(
        long id, boolean companion, int y
    ) {
        return new GeoWorldMiningDistrictPlanner.Placement(
            0, y, 0, GeoWorldMineralRules.Kind.IRON, id, companion);
    }

    private static long chunkKey(GeoWorldMiningDistrictPlanner.District district) {
        int x = Math.floorDiv((int)Math.round(district.centerX()), 16);
        int z = Math.floorDiv((int)Math.round(district.centerZ()), 16);
        return ((long)x << 32) | (z & 0xffffffffL);
    }

    private static void installTags(boolean extra) {
        Map<TagKey<Block>, List<Holder<Block>>> tags = new LinkedHashMap<>();
        List<Block> stone = new ArrayList<>(), deep = new ArrayList<>();
        for (var entry : MATERIALS.entrySet()) {
            boolean fixture = EXTRA.equals(entry.getKey());
            List<Block> ores = fixture && !extra ? List.of() :
                List.of(entry.getValue()[0], entry.getValue()[1]);
            tags.put(tag((fixture ? "forge:" : "c:") + "ores/" + entry.getKey()), holders(ores));
            if (!ores.isEmpty()) {
                stone.add(entry.getValue()[0]);
                deep.add(entry.getValue()[1]);
            }
        }
        tags.put(tag("c:ores_in_ground/stone"), holders(stone));
        tags.put(tag("forge:ores_in_ground/deepslate"), holders(deep));
        tags.put(tag("c:ores/dictionary_only"), holders(List.of(Blocks.DIAMOND_BLOCK)));
        BuiltInRegistries.BLOCK.getTagNames().forEach(key ->
            tags.putIfAbsent(key, List.of()));
        BuiltInRegistries.BLOCK.bindTags(tags);
        GeoWorldOreCatalog.invalidate();
    }

    private static TagKey<Block> tag(String id) {
        return TagKey.create(Registries.BLOCK, ResourceLocation.parse(id));
    }

    private static List<Holder<Block>> holders(List<Block> blocks) {
        return blocks.stream().map(BuiltInRegistries.BLOCK::wrapAsHolder).toList();
    }

    private static String id(Block block) {
        return BuiltInRegistries.BLOCK.getKey(block).toString();
    }

    private static void require(boolean condition, String message) {
        if (!condition) throw new AssertionError(message);
    }
}
