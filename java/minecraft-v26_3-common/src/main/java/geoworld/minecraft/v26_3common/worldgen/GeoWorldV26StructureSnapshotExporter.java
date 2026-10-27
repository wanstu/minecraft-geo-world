package geoworld.minecraft.v26_3common.worldgen;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.mojang.serialization.JsonOps;
import net.minecraft.SharedConstants;
import net.minecraft.core.HolderLookup;
import net.minecraft.core.registries.Registries;
import net.minecraft.data.registries.VanillaRegistries;
import net.minecraft.resources.RegistryOps;
import net.minecraft.server.Bootstrap;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.levelgen.structure.StructureSet;
import net.minecraft.world.level.levelgen.structure.placement.RandomSpreadStructurePlacement;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;

/**
 * Build-time exporter for Minecraft 26.3 vanilla StructureSet definitions.
 * The Web preview consumes this version-locked snapshot without requiring
 * Minecraft classes at runtime.
 */
public final class GeoWorldV26StructureSnapshotExporter {
    private GeoWorldV26StructureSnapshotExporter() {}

    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            throw new IllegalArgumentException(
                "usage: GeoWorldV26StructureSnapshotExporter <output.json>"
            );
        }

        SharedConstants.tryDetectVersion();
        Bootstrap.bootStrap();
        HolderLookup.Provider provider = VanillaRegistries.createWorldLookup();
        HolderLookup.RegistryLookup<StructureSet> lookup =
            provider.lookupOrThrow(Registries.STRUCTURE_SET);
        RegistryOps<JsonElement> ops =
            provider.createSerializationContext(JsonOps.INSTANCE);

        JsonObject root = new JsonObject();
        root.addProperty("minecraftVersion", "26.3");
        JsonObject sets = new JsonObject();
        JsonObject vectors = new JsonObject();
        final long vectorSeed = 123456L;
        final int[][] vectorRegions = {
            {-3, -2}, {-1, 4}, {0, 0}, {2, 1}, {5, -4}
        };

        lookup.listElements()
            .sorted(Comparator.comparing(holder -> holder.key().identifier().toString()))
            .forEach(holder -> {
                JsonElement encoded = StructureSet.DIRECT_CODEC
                    .encodeStart(ops, holder.value())
                    .getOrThrow();
                String id = holder.key().identifier().toString();
                sets.add(id, encoded);
                if (holder.value().placement() instanceof RandomSpreadStructurePlacement placement) {
                    com.google.gson.JsonArray entries = new com.google.gson.JsonArray();
                    for (int[] region : vectorRegions) {
                        int queryChunkX = region[0] * placement.spacing();
                        int queryChunkZ = region[1] * placement.spacing();
                        ChunkPos candidate = placement.getPotentialStructureChunk(
                            vectorSeed, queryChunkX, queryChunkZ
                        );
                        JsonObject vector = new JsonObject();
                        vector.addProperty("queryChunkX", queryChunkX);
                        vector.addProperty("queryChunkZ", queryChunkZ);
                        vector.addProperty("candidateChunkX", candidate.x());
                        vector.addProperty("candidateChunkZ", candidate.z());
                        vector.addProperty(
                            "frequencyAllowed",
                            placement.applyAdditionalChunkRestrictions(
                                candidate.x(), candidate.z(), vectorSeed
                            )
                        );
                        entries.add(vector);
                    }
                    vectors.add(id, entries);
                }
            });

        root.add("structureSets", sets);
        root.addProperty("vectorSeed", vectorSeed);
        root.add("randomSpreadVectors", vectors);
        Gson gson = new GsonBuilder()
            .setPrettyPrinting()
            .disableHtmlEscaping()
            .create();

        Path output = Path.of(args[0]).toAbsolutePath().normalize();
        Files.createDirectories(output.getParent());
        Files.writeString(
            output,
            gson.toJson(root) + System.lineSeparator(),
            StandardCharsets.UTF_8
        );
        System.out.println(
            "GeoWorld structure snapshot: " + sets.size() +
            " sets -> " + output
        );
    }
}
