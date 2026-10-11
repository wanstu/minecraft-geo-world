package geoworld.neoforge.worldgen;

import com.mojang.serialization.MapCodec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.minecraft.v26_3common.Minecraft26_3ProfileMapping;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.util.Util;
import net.minecraft.core.BlockPos;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderLookup;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.resources.Identifier;
import net.minecraft.server.level.WorldGenRegion;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.LevelHeightAccessor;
import net.minecraft.world.level.NoiseColumn;
import net.minecraft.world.level.StructureManager;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.BiomeManager;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.minecraft.world.level.chunk.ChunkGeneratorStructureState;
import net.minecraft.world.level.levelgen.Heightmap;
import net.minecraft.world.level.levelgen.RandomState;
import net.minecraft.world.level.levelgen.blending.Blender;
import net.minecraft.world.level.levelgen.densityfunction.SamplerContext;
import net.minecraft.world.level.levelgen.structure.StructureSet;
import org.jspecify.annotations.Nullable;

import java.io.IOException;
import java.nio.file.Path;
import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.CompletableFuture;

public final class GeoWorldChunkGenerator extends ChunkGenerator {
    public static final MapCodec<GeoWorldChunkGenerator> CODEC =
        RecordCodecBuilder.mapCodec(instance -> instance.group(
            BiomeSource.CODEC
                .fieldOf("biome_source")
                .forGetter(GeoWorldChunkGenerator::getBiomeSource),
            GeoWorldCreationConfig.CODEC
                .fieldOf("config")
                .forGetter(GeoWorldChunkGenerator::config)
        ).apply(instance, GeoWorldChunkGenerator::new));

    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    private final GeoWorldCreationConfig config;
    private final Object modelLock = new Object();

    private volatile long modelSeed = Long.MIN_VALUE;
    private volatile GeoWorldV1 model;
    private volatile Path modelPath;
    private volatile long spawnOriginSeed = Long.MIN_VALUE;
    private volatile ChunkPos spawnOrigin = ChunkPos.ZERO;

    public GeoWorldChunkGenerator(
        BiomeSource biomeSource,
        GeoWorldCreationConfig config
    ) {
        super(
            biomeSource,
            biome -> GeoWorldDecorationPolicy.filter(
                biome.value().getGenerationSettings()
            )
        );
        if (config == null) {
            throw new NullPointerException("config");
        }
        List<String> errors = config.validationErrors();
        if (!errors.isEmpty()) {
            throw new IllegalArgumentException(errors.getFirst());
        }
        this.config = config;
    }

    public GeoWorldCreationConfig config() {
        return this.config;
    }

    @Override
    protected MapCodec<? extends ChunkGenerator> codec() {
        return CODEC;
    }

    @Override
    public ChunkPos getOrigin(RandomState randomState) {
        long seed = randomState.seed();
        ChunkPos current = this.spawnOrigin;
        if (this.spawnOriginSeed == seed) {
            return current;
        }

        GeoWorldV1 runtime = this.ensureModel(seed);
        GeoWorldSpawnPlanner.Candidate candidate =
            GeoWorldSpawnPlanner.find(runtime);
        ChunkPos found = new ChunkPos(
            candidate.chunkX(),
            candidate.chunkZ()
        );
        synchronized (this.modelLock) {
            this.spawnOriginSeed = seed;
            this.spawnOrigin = found;
        }
        return found;
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
            // Fill from the GeoWorld resolver directly. This avoids any chance
            // that a chunk captures a fallback resolver before the persisted
            // Runtime model is attached, which would leave river cells stored
            // as the surrounding Plains/Forest biome.
            source.attachModel(runtime);
            return CompletableFuture.supplyAsync(
                () -> {
                    protoChunk.fillBiomesFromNoise(source::getNoiseBiome);
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
        GeoWorldV1 runtime = this.ensureModel(randomState.seed());
        int baseX = chunk.getPos().getMinBlockX();
        int baseZ = chunk.getPos().getMinBlockZ();

        GeoWorldShoreline.ChunkPlan shoreline =
            GeoWorldShoreline.planChunk(runtime, baseX, baseZ);

        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();
        Heightmap oceanFloor = chunk.getOrCreateHeightmapUnprimed(
            Heightmap.Types.OCEAN_FLOOR_WG
        );
        Heightmap worldSurface = chunk.getOrCreateHeightmapUnprimed(
            Heightmap.Types.WORLD_SURFACE_WG
        );

        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                int worldX = baseX + localX;
                int worldZ = baseZ + localZ;
                GeoWorldShoreline.Column column =
                    shoreline.column(localX, localZ);
                GeoWorldV1.RuntimeSample sample = column.sample();

                int groundBoundary = column.groundBoundary();
                int waterBoundary = column.waterBoundary();

                int maxBoundary = Math.min(
                    Minecraft26_3HeightMapping.MAX_Y_EXCLUSIVE,
                    Math.max(groundBoundary, waterBoundary)
                );

                for (
                    int y = Minecraft26_3HeightMapping.MIN_Y;
                    y < maxBoundary;
                    y++
                ) {
                    BlockState state = blockAt(
                        column,
                        y
                    );
                    if (state.isAir()) {
                        continue;
                    }

                    chunk.setBlockState(
                        pos.set(localX, y, localZ),
                        state
                    );
                    oceanFloor.update(localX, y, localZ, state);
                    worldSurface.update(localX, y, localZ, state);
                }
            }
        }

        // Overlay deterministic, ecology-biased 3D mineral veins after the
        // solid/water voxels exist. The planner clips world-space seams to
        // this chunk and never replaces the surface or water blocks.
        GeoWorldMineralVeins.populate(chunk, runtime, randomState.seed(), shoreline);
        GeoWorldAmethystGeodes.populate(chunk, randomState.seed(), shoreline);

        return CompletableFuture.completedFuture(chunk);
    }

    @Override
    public int getBaseHeight(
        int x,
        int z,
        Heightmap.Types type,
        LevelHeightAccessor heightAccessor,
        RandomState randomState
    ) {
        GeoWorldShoreline.Column column = GeoWorldShoreline.plan(
            this.ensureModel(randomState.seed()),
            x,
            z
        );
        int ground = column.groundBoundary();
        int water = column.waterBoundary();

        if (
            type == Heightmap.Types.OCEAN_FLOOR ||
            type == Heightmap.Types.OCEAN_FLOOR_WG
        ) {
            return clampToHeightAccessor(ground, heightAccessor);
        }
        return clampToHeightAccessor(
            Math.max(ground, water),
            heightAccessor
        );
    }

    @Override
    public NoiseColumn getBaseColumn(
        int x,
        int z,
        LevelHeightAccessor heightAccessor,
        RandomState randomState
    ) {
        GeoWorldShoreline.Column column = GeoWorldShoreline.plan(
            this.ensureModel(randomState.seed()),
            x,
            z
        );
        int ground = column.groundBoundary();
        int water = column.waterBoundary();

        BlockState[] states = new BlockState[heightAccessor.getHeight()];
        int minY = heightAccessor.getMinY();

        for (int index = 0; index < states.length; index++) {
            int y = minY + index;
            states[index] = blockAt(
                column,
                y
            );
        }

        return new NoiseColumn(minY, states);
    }

    @Override
    public int getSpawnHeight(LevelHeightAccessor heightAccessor) {
        GeoWorldV1 current = this.model;
        ChunkPos origin = this.spawnOrigin;
        if (
            current != null &&
            this.spawnOriginSeed == this.modelSeed
        ) {
            int x = origin.getMinBlockX() + 8;
            int z = origin.getMinBlockZ() + 8;
            GeoWorldV1.RuntimeSample sample = current.sample(x, z);
            int ground = HEIGHTS.boundaryY(sample.groundHeight());
            return clampToHeightAccessor(ground, heightAccessor);
        }

        return Math.min(
            heightAccessor.getMaxY(),
            Minecraft26_3HeightMapping.SEA_LEVEL + 1
        );
    }

    @Override
    public int getSeaLevel() {
        return Minecraft26_3HeightMapping.SEA_LEVEL;
    }

    @Override
    public int getMinY() {
        return Minecraft26_3HeightMapping.MIN_Y;
    }

    @Override
    public int getGenDepth() {
        return Minecraft26_3HeightMapping.HEIGHT;
    }

    @Override
    public void spawnOriginalMobs(WorldGenRegion worldGenRegion) {
    }

    @Override
    public void addDebugScreenInfo(
        List<String> result,
        RandomState randomState,
        BlockPos feetPos,
        SamplerContext samplerContext
    ) {
        GeoWorldV1.RuntimeSample sample = this
            .ensureModel(randomState.seed())
            .sample(feetPos.getX(), feetPos.getZ());
        result.add(
            "GeoWorld ground=" +
            String.format(java.util.Locale.ROOT, "%.3f", sample.groundHeight()) +
            " water=" +
            sample.waterKind() +
            " biome=" +
            sample.biomeFamily()
        );
    }

    public GeoWorldV1 ensureModel(long seed) {
        GeoWorldV1 current = this.model;
        if (current != null && this.modelSeed == seed) {
            return current;
        }

        synchronized (this.modelLock) {
            current = this.model;
            if (current != null && this.modelSeed == seed) {
                return current;
            }

            try {
                GeoWorldEngineBridge.GeneratedModel generated =
                    GeoWorldEngineBridge.generate(seed, this.config);
                this.attachModel(
                    seed,
                    generated.model(),
                    generated.path()
                );
                return generated.model();
            } catch (IOException | InterruptedException e) {
                if (e instanceof InterruptedException) {
                    Thread.currentThread().interrupt();
                }
                throw new IllegalStateException(
                    "Failed to prepare GeoWorld model for seed " + seed,
                    e
                );
            }
        }
    }

    public void attachModel(
        long seed,
        GeoWorldV1 model,
        @Nullable Path sourcePath
    ) {
        if (model == null) {
            throw new NullPointerException("model");
        }
        if (model.manifest().seed() != seed) {
            throw new IllegalArgumentException(
                "GeoWorld model seed mismatch: model=" +
                model.manifest().seed() +
                " expected=" +
                seed
            );
        }

        synchronized (this.modelLock) {
            this.modelSeed = seed;
            this.model = model;
            this.modelPath = sourcePath;
            if (this.biomeSource instanceof GeoWorldBiomeSource source) {
                source.attachModel(model);
            }
        }
    }

    public Optional<Path> modelPath() {
        return Optional.ofNullable(this.modelPath);
    }

    private static int clampToHeightAccessor(
        int boundary,
        LevelHeightAccessor heightAccessor
    ) {
        int min = heightAccessor.getMinY();
        int maxExclusive = min + heightAccessor.getHeight();
        return Math.max(min, Math.min(maxExclusive, boundary));
    }

    private static BlockState blockAt(
        GeoWorldShoreline.Column column,
        int y
    ) {
        GeoWorldV1.RuntimeSample sample = column.sample();
        int groundBoundary = column.groundBoundary();
        int waterBoundary = column.waterBoundary();

        if (
            y < Minecraft26_3HeightMapping.MIN_Y ||
            y >= Minecraft26_3HeightMapping.MAX_Y_EXCLUSIVE
        ) {
            return Blocks.AIR.defaultBlockState();
        }
        if (y == Minecraft26_3HeightMapping.MIN_Y) {
            return Blocks.BEDROCK.defaultBlockState();
        }

        if (y < groundBoundary) {
            int depth = groundBoundary - 1 - y;
            if (depth == 0) {
                String override = column.surfaceOverride();
                return blockState(
                    override != null
                        ? override
                        : Minecraft26_3ProfileMapping.surfaceBlock(
                            sample.surfaceProfile()
                        )
                );
            }
            if (depth <= 2 && column.underOverride() != null) {
                return blockState(column.underOverride());
            }
            if (depth <= 3) {
                return blockState(
                    Minecraft26_3ProfileMapping.underBlock(
                        sample.surfaceProfile()
                    )
                );
            }
            return y < 0
                ? Blocks.DEEPSLATE.defaultBlockState()
                : Blocks.STONE.defaultBlockState();
        }

        if (y < waterBoundary) {
            if (
                y == waterBoundary - 1 &&
                "Frozen Ocean".equals(sample.biomeFamily()) &&
                "Ocean".equals(sample.waterKind())
            ) {
                return Blocks.ICE.defaultBlockState();
            }
            return Blocks.WATER.defaultBlockState();
        }

        return Blocks.AIR.defaultBlockState();
    }

    private static BlockState blockState(String id) {
        String path = id.startsWith("minecraft:")
            ? id.substring("minecraft:".length())
            : id;
        var block = BuiltInRegistries.BLOCK.getValue(
            Identifier.withDefaultNamespace(path)
        );
        if (block == null) {
            throw new IllegalArgumentException(
                "Unknown Minecraft block id: " + id
            );
        }
        return block.defaultBlockState();
    }
}
