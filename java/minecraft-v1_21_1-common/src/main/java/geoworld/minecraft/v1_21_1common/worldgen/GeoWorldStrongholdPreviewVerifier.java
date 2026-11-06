package geoworld.minecraft.v1_21_1common.worldgen;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.ContinentPreset;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import geoworld.minecraft.v26common.worldgen.GeoWorldStrongholdPreviewCli;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.SharedConstants;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderLookup;
import net.minecraft.core.HolderSet;
import net.minecraft.core.registries.Registries;
import net.minecraft.data.registries.VanillaRegistries;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.resources.ResourceKey;
import net.minecraft.server.Bootstrap;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGeneratorStructureState;
import net.minecraft.world.level.levelgen.NoiseGeneratorSettings;
import net.minecraft.world.level.levelgen.RandomState;
import net.minecraft.world.level.levelgen.structure.StructureSet;
import net.minecraft.world.level.levelgen.structure.placement.ConcentricRingsStructurePlacement;
import net.minecraft.world.level.levelgen.synth.NormalNoise;

import java.io.InputStream;
import java.lang.reflect.Constructor;
import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CompletableFuture;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Build-time parity gate proving the standalone Web stronghold planner matches
 * Minecraft 1.21.1's own concentric-rings implementation for a locked GeoWorld
 * model, including the GeoWorld-biome bias search.
 */
public final class GeoWorldStrongholdPreviewVerifier {
    private static final Pattern MINECRAFT_ID =
        Pattern.compile("\"(minecraft:[a-z0-9_./-]+)\"");

    private GeoWorldStrongholdPreviewVerifier() {}

    public static void main(String[] args) throws Exception {
        long seed = 123456L;
        GeoWorldCreationConfig config = new GeoWorldCreationConfig(
            160,
            100,
            128,
            18,
            0.68,
            0.35,
            0.65,
            ContinentPreset.DEFAULT
        );
        GeoWorldV1 model = GeoWorldEngineBridge.generate(seed, config).model();

        SharedConstants.tryDetectVersion();
        Bootstrap.bootStrap();

        HolderLookup.Provider provider = VanillaRegistries.createLookup();
        HolderLookup.RegistryLookup<StructureSet> structureSets =
            provider.lookupOrThrow(Registries.STRUCTURE_SET);
        Holder<StructureSet> strongholdSet = structureSets.listElements()
            .filter(holder ->
                holder.value().placement() instanceof ConcentricRingsStructurePlacement
            )
            .findFirst()
            .orElseThrow(() ->
                new IllegalStateException("vanilla stronghold StructureSet not found")
            );
        ConcentricRingsStructurePlacement registryPlacement =
            (ConcentricRingsStructurePlacement) strongholdSet.value().placement();

        require(
            registryPlacement.distance() ==
                GeoWorldStrongholdPreviewCli.DISTANCE,
            "distance mismatch"
        );
        require(
            registryPlacement.spread() ==
                GeoWorldStrongholdPreviewCli.INITIAL_SPREAD,
            "spread mismatch"
        );
        require(
            registryPlacement.count() ==
                GeoWorldStrongholdPreviewCli.COUNT,
            "count mismatch"
        );

        String preferredTag = registryPlacement.preferredBiomes()
            .unwrapKey()
            .map(tag -> tag.location().toString())
            .orElse("");
        require(
            preferredTag.equals("minecraft:stronghold_biased_to"),
            "preferred-biome tag mismatch: " + preferredTag
        );

        Set<String> resourcePreferred = loadPreferredBiomeResource();
        require(
            resourcePreferred.equals(
                GeoWorldStrongholdPreviewCli.preferredBiomes("1.21.1")
            ),
            "preferred-biome contents mismatch: minecraft=" +
            resourcePreferred + " standalone=" +
            GeoWorldStrongholdPreviewCli.preferredBiomes("1.21.1")
        );

        HolderLookup.RegistryLookup<Biome> biomeLookup =
            provider.lookupOrThrow(Registries.BIOME);
        GeoWorldBiomeSource biomeSource =
            GeoWorldBiomeSource.createDefault(biomeLookup);
        biomeSource.attachModel(model);

        HolderSet<Biome> directPreferred = directPreferredBiomes(
            biomeLookup,
            resourcePreferred
        );
        ConcentricRingsStructurePlacement directPlacement =
            new ConcentricRingsStructurePlacement(
                GeoWorldStrongholdPreviewCli.DISTANCE,
                GeoWorldStrongholdPreviewCli.INITIAL_SPREAD,
                GeoWorldStrongholdPreviewCli.COUNT,
                directPreferred
            );

        var noises =
            provider.lookupOrThrow(Registries.NOISE);
        NoiseGeneratorSettings noiseSettings = provider
            .lookupOrThrow(Registries.NOISE_SETTINGS)
            .getOrThrow(NoiseGeneratorSettings.OVERWORLD)
            .value();
        RandomState randomState = RandomState.create(
            noiseSettings,
            noises,
            seed
        );

        List<ChunkPos> minecraft = minecraftRingPositions(
            randomState,
            biomeSource,
            strongholdSet,
            directPlacement,
            seed
        );
        List<GeoWorldStrongholdPreviewCli.Position> standalone =
            GeoWorldStrongholdPreviewCli.positions(model, "1.21.1");

        require(
            minecraft.size() == standalone.size(),
            "position count mismatch: minecraft=" + minecraft.size() +
            " standalone=" + standalone.size()
        );

        for (int i = 0; i < minecraft.size(); i++) {
            ChunkPos expected = minecraft.get(i);
            GeoWorldStrongholdPreviewCli.Position actual =
                standalone.get(i);
            if (expected.x != actual.chunkX() ||
                expected.z != actual.chunkZ()) {
                throw new IllegalStateException(
                    "stronghold parity mismatch at index " + i +
                    ": minecraft=" + expected.x + "," + expected.z +
                    " standalone=" + actual.chunkX() + "," + actual.chunkZ()
                );
            }
        }

        String setId = strongholdSet.unwrapKey()
            .map(GeoWorldMinecraftIds::resourceKeyId)
            .orElse("minecraft:strongholds");
        require(
            setId.equals(GeoWorldStrongholdPreviewCli.SET_ID),
            "stronghold set id mismatch: " + setId
        );

        System.out.println(
            "GeoWorld stronghold preview parity PASS seed=" + seed +
            " positions=" + minecraft.size() +
            " set=" + setId
        );
    }

    private static HolderSet<Biome> directPreferredBiomes(
        HolderLookup.RegistryLookup<Biome> biomeLookup,
        Set<String> ids
    ) {
        ArrayList<Holder<Biome>> holders = new ArrayList<>(ids.size());
        for (String id : ids) {
            ResourceKey<Biome> key = ResourceKey.create(
                Registries.BIOME,
                ResourceLocation.parse(id)
            );
            holders.add(biomeLookup.getOrThrow(key));
        }
        return HolderSet.direct(holders);
    }

    @SuppressWarnings("unchecked")
    private static List<ChunkPos> minecraftRingPositions(
        RandomState randomState,
        BiomeSource biomeSource,
        Holder<StructureSet> strongholdSet,
        ConcentricRingsStructurePlacement placement,
        long seed
    ) throws Exception {
        Constructor<ChunkGeneratorStructureState> constructor =
            ChunkGeneratorStructureState.class.getDeclaredConstructor(
                RandomState.class,
                BiomeSource.class,
                long.class,
                long.class,
                List.class
            );
        constructor.setAccessible(true);
        ChunkGeneratorStructureState state = constructor.newInstance(
            randomState,
            biomeSource,
            seed,
            seed,
            List.of(strongholdSet)
        );

        Method generate =
            ChunkGeneratorStructureState.class.getDeclaredMethod(
                "generateRingPositions",
                Holder.class,
                ConcentricRingsStructurePlacement.class
            );
        generate.setAccessible(true);
        CompletableFuture<List<ChunkPos>> future =
            (CompletableFuture<List<ChunkPos>>)generate.invoke(
                state,
                strongholdSet,
                placement
            );
        return future.join();
    }

    private static Set<String> loadPreferredBiomeResource() throws Exception {
        String path =
            "data/minecraft/tags/worldgen/biome/stronghold_biased_to.json";
        try (InputStream in =
                 GeoWorldStrongholdPreviewVerifier.class
                     .getClassLoader()
                     .getResourceAsStream(path)) {
            if (in == null) {
                throw new IllegalStateException(
                    "Minecraft stronghold biome tag resource not found: " +
                    path
                );
            }
            String json = new String(
                in.readAllBytes(),
                StandardCharsets.UTF_8
            );
            LinkedHashSet<String> ids = new LinkedHashSet<>();
            Matcher matcher = MINECRAFT_ID.matcher(json);
            while (matcher.find()) {
                ids.add(matcher.group(1));
            }
            require(
                !ids.isEmpty(),
                "Minecraft stronghold biome tag resource is empty"
            );
            return Set.copyOf(ids);
        }
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalStateException(message);
        }
    }
}
