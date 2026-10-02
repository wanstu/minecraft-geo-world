package geoworld.minecraft.reference;

import com.mojang.serialization.Codec;
import com.mojang.serialization.Dynamic;
import geoworld.minecraft.anvil.RegionFile;
import net.minecraft.SharedConstants;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderLookup;
import net.minecraft.core.IdMap;
import net.minecraft.core.registries.Registries;
import net.minecraft.data.registries.VanillaRegistries;
import net.minecraft.nbt.CompoundTag;
import net.minecraft.nbt.NbtAccounter;
import net.minecraft.nbt.NbtIo;
import net.minecraft.nbt.NbtOps;
import net.minecraft.resources.Identifier;
import net.minecraft.resources.ResourceKey;
import net.minecraft.server.Bootstrap;
import net.minecraft.world.level.LevelHeightAccessor;
import net.minecraft.world.level.LevelSettings;
import net.minecraft.world.level.WorldDataConfiguration;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.Biomes;
import net.minecraft.world.level.block.Block;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.PalettedContainer;
import net.minecraft.world.level.chunk.PalettedContainerFactory;
import net.minecraft.world.level.chunk.Strategy;
import net.minecraft.world.level.chunk.storage.SerializableChunkData;
import net.minecraft.world.level.levelgen.WorldGenSettings;
import net.minecraft.world.level.storage.LevelData;
import net.minecraft.world.level.storage.LevelStorageSource;
import net.minecraft.world.level.storage.LevelVersion;

import java.io.ByteArrayInputStream;
import java.io.DataInputStream;
import java.nio.file.Path;
import java.util.IdentityHashMap;
import java.util.Iterator;
import java.util.List;

public final class Mojang26_3Verifier {
    private Mojang26_3Verifier() {}

    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            throw new IllegalArgumentException(
                "usage: Mojang26_3Verifier <world-directory> [radius]"
            );
        }
        Path world = Path.of(args[0]).toAbsolutePath();
        int radius = args.length >= 2 ? Integer.parseInt(args[1]) : 8;
        if (radius < 0 || radius > 31) {
            throw new IllegalArgumentException("radius must be in 0..31");
        }

        SharedConstants.tryDetectVersion();
        Bootstrap.bootStrap();

        CompoundTag levelRoot = NbtIo.readCompressed(
            world.resolve("level.dat"),
            NbtAccounter.unlimitedHeap()
        );
        CompoundTag levelData = levelRoot.getCompoundOrEmpty("Data");
        Dynamic<?> dynamic = new Dynamic<>(NbtOps.INSTANCE, levelData);

        LevelVersion levelVersion = LevelVersion.parse(dynamic);
        WorldDataConfiguration config = LevelStorageSource.readDataConfig(dynamic);
        LevelSettings settings = LevelSettings.parse(dynamic, config);
        LevelData.RespawnData spawn = levelData
            .read("spawn", LevelData.RespawnData.CODEC)
            .orElseThrow(() -> new IllegalStateException("level.dat has no valid spawn"));

        requireEquals("version name", levelVersion.minecraftVersionName(), "26.3");
        requireEquals(
            "DataVersion",
            levelVersion.minecraftVersion().version(),
            5023
        );
        requireEquals(
            "level storage version",
            levelVersion.levelDataVersion(),
            19133
        );

        System.out.println(
            "PASS Mojang level.dat: " +
            settings.levelName() +
            ", version=" + levelVersion.minecraftVersionName() +
            ", DataVersion=" + levelVersion.minecraftVersion().version() +
            ", spawn=" + spawn.pos()
        );

        HolderLookup.Provider worldLookup = VanillaRegistries.createWorldLookup();
        CompoundTag worldGenRoot = NbtIo.readCompressed(
            world.resolve("data").resolve("world_gen_settings.dat"),
            NbtAccounter.unlimitedHeap()
        );
        CompoundTag worldGenData = worldGenRoot.getCompoundOrEmpty("data");
        var registryOps = worldLookup.createSerializationContext(NbtOps.INSTANCE);
        WorldGenSettings worldGen = WorldGenSettings.CODEC
            .parse(registryOps, worldGenData)
            .getOrThrow();

        if (worldGen.dimensions().overworld() == null) {
            throw new AssertionError("overworld generator missing");
        }
        requireEquals("dimension count", worldGen.dimensions().dimensions().size(), 3);

        System.out.println(
            "PASS Mojang world_gen_settings.dat: seed=" +
            worldGen.options().seed() +
            ", dimensions=" + worldGen.dimensions().dimensions().keySet()
        );

        PalettedContainerFactory factory = createContainerFactory(worldLookup);
        LevelHeightAccessor height = LevelHeightAccessor.create(384, -64);

        int centerChunkX = Math.floorDiv(spawn.pos().getX(), 16);
        int centerChunkZ = Math.floorDiv(spawn.pos().getZ(), 16);
        int checked = 0;

        for (
            int chunkZ = centerChunkZ - radius;
            chunkZ <= centerChunkZ + radius;
            chunkZ++
        ) {
            for (
                int chunkX = centerChunkX - radius;
                chunkX <= centerChunkX + radius;
                chunkX++
            ) {
                Path region = RegionFile.regionPath(
                    world.resolve("region"),
                    RegionFile.regionXForChunk(chunkX),
                    RegionFile.regionZForChunk(chunkZ)
                );
                byte[] raw = RegionFile.readChunkNbt(region, chunkX, chunkZ);
                if (raw == null) {
                    throw new AssertionError(
                        "missing chunk " + chunkX + "," + chunkZ
                    );
                }

                CompoundTag chunkTag;
                try (
                    DataInputStream in = new DataInputStream(
                        new ByteArrayInputStream(raw)
                    )
                ) {
                    chunkTag = NbtIo.read(in, NbtAccounter.unlimitedHeap());
                }

                SerializableChunkData parsed = SerializableChunkData.parse(
                    height,
                    factory,
                    chunkTag
                );
                requireEquals("chunk x", parsed.chunkPos().x(), chunkX);
                requireEquals("chunk z", parsed.chunkPos().z(), chunkZ);
                checked++;
            }
        }

        int expected = (radius * 2 + 1) * (radius * 2 + 1);
        requireEquals("parsed chunk count", checked, expected);
        System.out.println(
            "PASS Mojang SerializableChunkData.parse: " +
            checked + "/" + expected + " chunks"
        );
        System.out.println("Mojang 26.3 verification passed.");
    }

    private static PalettedContainerFactory createContainerFactory(
        HolderLookup.Provider worldLookup
    ) {
        var biomeLookup = worldLookup.lookupOrThrow(Registries.BIOME);
        List<Holder<Biome>> biomeHolders = biomeLookup
            .listElements()
            .map(holder -> (Holder<Biome>) holder)
            .toList();

        IdentityHashMap<Holder<Biome>, Integer> biomeIds = new IdentityHashMap<>();
        for (int i = 0; i < biomeHolders.size(); i++) {
            biomeIds.put(biomeHolders.get(i), i);
        }

        IdMap<Holder<Biome>> biomeIdMap = new IdMap<>() {
            @Override
            public int getId(Holder<Biome> value) {
                Integer id = biomeIds.get(value);
                return id == null ? -1 : id;
            }

            @Override
            public Holder<Biome> byId(int id) {
                return id >= 0 && id < biomeHolders.size()
                    ? biomeHolders.get(id)
                    : null;
            }

            @Override
            public int size() {
                return biomeHolders.size();
            }

            @Override
            public Iterator<Holder<Biome>> iterator() {
                return biomeHolders.iterator();
            }
        };

        Strategy<BlockState> blockStrategy =
            Strategy.createForBlockStates(Block.BLOCK_STATE_REGISTRY);
        BlockState defaultBlock = Blocks.AIR.defaultBlockState();

        Strategy<Holder<Biome>> biomeStrategy =
            Strategy.createForBiomes(biomeIdMap);
        Holder<Biome> defaultBiome = biomeLookup.getOrThrow(Biomes.PLAINS);

        Codec<Holder<Biome>> biomeNameCodec = Codec.STRING.xmap(
            name -> biomeLookup.getOrThrow(
                ResourceKey.create(Registries.BIOME, Identifier.parse(name))
            ),
            holder -> holder.unwrapKey()
                .orElseThrow()
                .identifier()
                .toString()
        );

        return new PalettedContainerFactory(
            blockStrategy,
            defaultBlock,
            PalettedContainer.codecRW(
                BlockState.CODEC,
                blockStrategy,
                defaultBlock
            ),
            biomeStrategy,
            defaultBiome,
            PalettedContainer.codecRO(
                biomeNameCodec,
                biomeStrategy,
                defaultBiome
            )
        );
    }

    private static void requireEquals(String name, int got, int want) {
        if (got != want) {
            throw new AssertionError(name + " got=" + got + " want=" + want);
        }
    }

    private static void requireEquals(String name, Object got, Object want) {
        if (!got.equals(want)) {
            throw new AssertionError(name + " got=" + got + " want=" + want);
        }
    }
}
