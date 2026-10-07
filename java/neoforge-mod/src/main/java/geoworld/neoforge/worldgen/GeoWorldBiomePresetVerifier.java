package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.LinkedHashSet;
import java.util.Set;

public final class GeoWorldBiomePresetVerifier {
    private static final String RESOURCE =
        "/data/geoworld/worldgen/world_preset/geoworld.json";

    private GeoWorldBiomePresetVerifier() {}

    public static void main(String[] args) throws Exception {
        String json;
        try (InputStream input =
                 GeoWorldBiomePresetVerifier.class.getResourceAsStream(RESOURCE)) {
            if (input == null) {
                throw new IllegalStateException(
                    "GeoWorld world preset resource not found: " + RESOURCE
                );
            }
            json = new String(input.readAllBytes(), StandardCharsets.UTF_8);
        }

        int key = json.indexOf("\"biomes\"");
        int open = key < 0 ? -1 : json.indexOf('[', key);
        int close = open < 0 ? -1 : json.indexOf(']', open);
        if (key < 0 || open < 0 || close < 0) {
            throw new IllegalStateException(
                "GeoWorld overworld biome list is missing from world preset"
            );
        }

        LinkedHashSet<String> preset = new LinkedHashSet<>();
        String biomeBlock = json.substring(open + 1, close);
        for (String line : biomeBlock.split("\\R")) {
            String trimmed = line.trim();
            if (!trimmed.startsWith("\"minecraft:")) {
                continue;
            }
            int end = trimmed.indexOf('"', 1);
            if (end > 1) {
                preset.add(trimmed.substring(1, end));
            }
        }

        Set<String> required = Minecraft26_3BiomeRealizer.requiredBiomeIds();
        LinkedHashSet<String> missing = new LinkedHashSet<>(required);
        missing.removeAll(preset);
        if (!missing.isEmpty()) {
            throw new IllegalStateException(
                "GeoWorld world preset is missing realized biomes: " + missing
            );
        }

        System.out.printf(
            "GeoWorld biome preset verification passed: preset=%d required=%d%n",
            preset.size(),
            required.size()
        );
    }
}
