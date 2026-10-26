package geoworld.minecraft.v26_3common.worldgen;

import com.mojang.serialization.MapCodec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ChunkGeneratorBase;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderLookup;
import net.minecraft.server.level.WorldGenRegion;
import net.minecraft.util.Util;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.StructureManager;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.BiomeManager;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.minecraft.world.level.chunk.ChunkGeneratorStructureState;
import net.minecraft.world.level.levelgen.RandomState;
import net.minecraft.world.level.levelgen.blending.Blender;
import net.minecraft.world.level.levelgen.densityfunction.SamplerContext;
import net.minecraft.world.level.levelgen.structure.StructureSet;
import org.jspecify.annotations.Nullable;

import java.util.List;
import java.util.Set;
import java.util.concurrent.CompletableFuture;

public final class GeoWorldChunkGenerator
    extends GeoWorldV26ChunkGeneratorBase {

    public static final MapCodec<GeoWorldChunkGenerator> CODEC =
        RecordCodecBuilder.mapCodec(instance -> instance.group(
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
            Minecraft26_3Profile.INSTANCE,
            Minecraft26_3Profile.GENERATOR_API,
            model -> {
                if (biomeSource instanceof GeoWorldBiomeSource source) {
                    source.attachModel(model);
                }
            }
        );
    }

    @Override
    protected MapCodec<? extends ChunkGenerator> codec() {
        return CODEC;
    }

    @Override
    protected long geoworldSeed(RandomState randomState) {
        return randomState.seed();
    }

    @Override
    public ChunkPos getOrigin(RandomState randomState) {
        return this.geoworldOrigin(randomState.seed());
    }

    @Override
    public ChunkGeneratorStructureState createState(
        HolderLookup<StructureSet> structureSets,
        RandomState randomState,
        long levelSeed
    ) {
        this.ensureModel(levelSeed);
        return super.createState(structureSets, randomState, levelSeed);
    }

    @Override
    public CompletableFuture<ChunkAccess> createBiomes(
        RandomState randomState,
        Blender blender,
        StructureManager structureManager,
        ChunkAccess protoChunk
    ) {
        GeoWorldV1 runtime = this.ensureModel(randomState.seed());
        if (this.biomeSource instanceof GeoWorldBiomeSource source) {
            source.attachModel(runtime);
            return CompletableFuture.supplyAsync(
                () -> {
                    protoChunk.fillBiomesFromNoise(
                        source::getNoiseBiome
                    );
                    return protoChunk;
                },
                Util.backgroundExecutor().forName("geoworld-biomes")
            );
        }
        return super.createBiomes(
            randomState,
            blender,
            structureManager,
            protoChunk
        );
    }

    @Override
    public CompletableFuture<ChunkAccess> buildTerrain(
        ChunkAccess chunk,
        Blender blender,
        RandomState randomState,
        StructureManager structureManager,
        BiomeManager biomeManager,
        @Nullable WorldGenRegion carverBiomeRegion,
        Set<Holder<Biome>> possibleBiomes
    ) {
        return this.support().populateTerrain(
            chunk,
            randomState.seed(),
            this::ensureModel,
            structureManager.startsForStructure(
                chunk.getPos().x(),
                chunk.getPos().z(),
                structure -> true
            )
        );
    }

    @Override
    public void addDebugScreenInfo(
        List<String> result,
        RandomState randomState,
        BlockPos feetPos,
        SamplerContext samplerContext
    ) {
        result.add(this.geoworldDebugLine(randomState, feetPos));
    }
}
