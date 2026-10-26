package geoworld.minecraft.v1_20_1common.worldgen;

import com.mojang.serialization.Codec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.v1_20_1common.Minecraft1_20_1Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ChunkGeneratorBase;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.core.HolderLookup;
import net.minecraft.server.level.WorldGenRegion;
import net.minecraft.world.level.StructureManager;
import net.minecraft.world.level.biome.BiomeManager;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.minecraft.world.level.chunk.ChunkGeneratorStructureState;
import net.minecraft.world.level.levelgen.GenerationStep;
import net.minecraft.world.level.levelgen.RandomState;
import net.minecraft.world.level.levelgen.blending.Blender;
import net.minecraft.world.level.levelgen.structure.StructureSet;

import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executor;

public final class GeoWorldChunkGenerator
    extends GeoWorldV26ChunkGeneratorBase {

    public static final Codec<GeoWorldChunkGenerator> CODEC =
        RecordCodecBuilder.create(instance -> instance.group(
            BiomeSource.CODEC
                .fieldOf("biome_source")
                .forGetter(GeoWorldChunkGenerator::getBiomeSource),
            GeoWorldCreationConfig.CODEC
                .fieldOf("config")
                .forGetter(GeoWorldChunkGenerator::config)
        ).apply(instance, GeoWorldChunkGenerator::new));

    public GeoWorldChunkGenerator(
        BiomeSource biomeSource,
        GeoWorldCreationConfig config
    ) {
        super(
            biomeSource,
            config,
            Minecraft1_20_1Profile.INSTANCE,
            Minecraft1_20_1Profile.GENERATOR_API,
            model -> {
                if (biomeSource instanceof GeoWorldBiomeSource source) {
                    source.attachModel(model);
                }
            }
        );
    }

    @Override
    protected Codec<? extends ChunkGenerator> codec() {
        return CODEC;
    }

    @Override
    protected long geoworldSeed(RandomState randomState) {
        long seed = this.boundModelSeed();
        if (seed == Long.MIN_VALUE) {
            throw new IllegalStateException(
                "GeoWorld 1.20.1 generation seed is not bound yet"
            );
        }
        return seed;
    }

    @Override
    public ChunkGeneratorStructureState createState(
        HolderLookup<StructureSet> structureSets,
        RandomState randomState,
        long levelSeed
    ) {
        this.ensureModel(levelSeed);
        this.geoworldOrigin(levelSeed);
        return super.createState(structureSets, randomState, levelSeed);
    }

    @Override
    public CompletableFuture<ChunkAccess> createBiomes(
        Executor executor,
        RandomState randomState,
        Blender blender,
        StructureManager structureManager,
        ChunkAccess protoChunk
    ) {
        GeoWorldV1 runtime = this.ensureModel(
            this.geoworldSeed(randomState)
        );
        if (this.biomeSource instanceof GeoWorldBiomeSource source) {
            source.attachModel(runtime);
            return CompletableFuture.supplyAsync(
                () -> {
                    protoChunk.fillBiomesFromNoise(
                        source,
                        randomState.sampler()
                    );
                    return protoChunk;
                },
                executor
            );
        }
        return super.createBiomes(
            executor,
            randomState,
            blender,
            structureManager,
            protoChunk
        );
    }

    @Override
    public CompletableFuture<ChunkAccess> fillFromNoise(
        Executor executor,
        Blender blender,
        RandomState randomState,
        StructureManager structureManager,
        ChunkAccess chunk
    ) {
        return this.support().populateTerrain(
            chunk,
            this.geoworldSeed(randomState),
            this::ensureModel,
            structureManager.startsForStructure(
                chunk.getPos(),
                structure -> true
            )
        );
    }

    @Override
    public void buildSurface(
        WorldGenRegion worldGenRegion,
        StructureManager structureManager,
        RandomState randomState,
        ChunkAccess chunk
    ) {
        // GeoWorld already realizes the complete surface in fillFromNoise.
    }

    @Override
    public void applyCarvers(
        WorldGenRegion worldGenRegion,
        long seed,
        RandomState randomState,
        BiomeManager biomeManager,
        StructureManager structureManager,
        ChunkAccess chunk,
        GenerationStep.Carving carving
    ) {
        // GeoWorld intentionally suppresses vanilla noise-carver terrain.
    }

    @Override
    public void addDebugScreenInfo(
        List<String> result,
        RandomState randomState,
        BlockPos feetPos
    ) {
        result.add(this.geoworldDebugLine(randomState, feetPos));
    }
}
