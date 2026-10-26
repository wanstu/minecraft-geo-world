package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.worldgen.GeoWorldSpawnPlanner;
import geoworld.minecraft.v26common.runtime.GeoWorldDynamicGenerator;
import geoworld.minecraft.v26common.runtime.GeoWorldV26GeneratorRuntime;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.server.level.WorldGenRegion;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.LevelHeightAccessor;
import net.minecraft.world.level.NoiseColumn;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.minecraft.world.level.levelgen.Heightmap;
import net.minecraft.world.level.levelgen.RandomState;
import java.nio.file.Path;
import java.util.Optional;
import java.util.function.Consumer;

/**
 * Shared Minecraft 26.x ChunkGenerator implementation.
 *
 * Version subclasses only adapt lifecycle callbacks whose Mojang signatures
 * differ between supported Minecraft 26.x releases.
 */
public abstract class GeoWorldV26ChunkGeneratorBase
    extends ChunkGenerator
    implements GeoWorldDynamicGenerator {

    private final GeoWorldV26GeneratorRuntime runtime;
    private final GeoWorldV26GeneratorSupport support;
    private final Object spawnLock = new Object();

    private volatile long spawnOriginSeed = Long.MIN_VALUE;
    private volatile ChunkPos spawnOrigin = ChunkPos.ZERO;

    protected GeoWorldV26ChunkGeneratorBase(
        BiomeSource biomeSource,
        GeoWorldCreationConfig config,
        MinecraftVersionProfile profile,
        GeoWorldV26GeneratorSupport.MinecraftApi api,
        Consumer<GeoWorldV1> modelAttacher
    ) {
        super(
            biomeSource,
            biome -> GeoWorldV26DecorationPolicy.filter(
                biome.value().getGenerationSettings(),
                api
            )
        );
        this.support = new GeoWorldV26GeneratorSupport(profile, api);
        this.runtime = new GeoWorldV26GeneratorRuntime(
            config,
            modelAttacher
        );
    }

    public final GeoWorldCreationConfig config() {
        return this.runtime.config();
    }

    protected final GeoWorldV26GeneratorSupport support() {
        return this.support;
    }

    protected final long boundModelSeed() {
        return this.runtime.modelSeed();
    }

    protected abstract long geoworldSeed(RandomState randomState);

    protected final ChunkPos geoworldOrigin(long seed) {
        ChunkPos current = this.spawnOrigin;
        if (this.spawnOriginSeed == seed) {
            return current;
        }

        GeoWorldV1 model = this.ensureModel(seed);
        GeoWorldSpawnPlanner.Candidate candidate =
            GeoWorldSpawnPlanner.find(
                model,
                model.worldProfile().parameters().spawnPreference()
            );
        ChunkPos found = new ChunkPos(
            candidate.chunkX(),
            candidate.chunkZ()
        );
        synchronized (this.spawnLock) {
            this.spawnOriginSeed = seed;
            this.spawnOrigin = found;
        }
        return found;
    }

    @Override
    public int getBaseHeight(
        int x,
        int z,
        Heightmap.Types type,
        LevelHeightAccessor heightAccessor,
        RandomState randomState
    ) {
        return this.support.baseHeight(
            this.ensureModel(this.geoworldSeed(randomState)),
            x,
            z,
            type,
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
        return this.support.baseColumn(
            this.ensureModel(this.geoworldSeed(randomState)),
            x,
            z,
            heightAccessor
        );
    }

    @Override
    public int getSpawnHeight(LevelHeightAccessor heightAccessor) {
        return this.support.spawnHeight(
            this.runtime.currentModel(),
            this.runtime.modelSeed(),
            this.spawnOriginSeed,
            this.spawnOrigin,
            heightAccessor
        );
    }

    @Override
    public int getSeaLevel() {
        return this.support.heights().seaLevel();
    }

    @Override
    public int getMinY() {
        return this.support.heights().minY();
    }

    @Override
    public int getGenDepth() {
        return this.support.heights().height();
    }

    @Override
    public void spawnOriginalMobs(WorldGenRegion worldGenRegion) {
    }

    protected final String geoworldDebugLine(
        RandomState randomState,
        BlockPos feetPos
    ) {
        return this.support.debugLine(
            this.ensureModel(this.geoworldSeed(randomState)),
            feetPos
        );
    }

    @Override
    public final GeoWorldV1 ensureModel(long seed) {
        return this.runtime.ensureModel(seed);
    }

    @Override
    public final void attachModel(
        long seed,
        GeoWorldV1 model,
        Path sourcePath
    ) {
        this.runtime.attachModel(seed, model, sourcePath);
    }

    @Override
    public final Optional<Path> modelPath() {
        return this.runtime.modelPath();
    }
}
