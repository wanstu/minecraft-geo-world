package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.LinkedHashSet;
import java.util.Set;

public final class GeoWorldV26BiomePresetVerifier {
    private static final String RESOURCE =
        "/data/geoworld/worldgen/world_preset/geoworld.json";
    private static final String VANILLA_OVERWORLD_RESOURCE =
        "/data/minecraft/tags/worldgen/biome/is_overworld.json";

    private GeoWorldV26BiomePresetVerifier() {}

    public static void run(
        MinecraftVersionProfile profile,
        String[] args
    ) throws Exception {
        GeoWorldV26BiomeRealizer realizer =
            new GeoWorldV26BiomeRealizer(profile);
        String presetJson = readResource(RESOURCE);
        LinkedHashSet<String> preset = extractMinecraftIds(
            presetJson,
            "biomes"
        );

        Set<String> required = realizer.requiredBiomeIds();
        LinkedHashSet<String> missingRealized =
            new LinkedHashSet<>(required);
        missingRealized.removeAll(preset);
        if (!missingRealized.isEmpty()) {
            throw new IllegalStateException(
                "GeoWorld world preset is missing realized biomes: " +
                missingRealized
            );
        }

        String vanillaJson = readResource(VANILLA_OVERWORLD_RESOURCE);
        LinkedHashSet<String> vanillaOverworld = extractMinecraftIds(
            vanillaJson,
            "values"
        );
        LinkedHashSet<String> missingVanilla =
            new LinkedHashSet<>(vanillaOverworld);
        missingVanilla.removeAll(preset);
        LinkedHashSet<String> extraPreset =
            new LinkedHashSet<>(preset);
        extraPreset.removeAll(vanillaOverworld);
        if (!missingVanilla.isEmpty() || !extraPreset.isEmpty()) {
            throw new IllegalStateException(
                "GeoWorld overworld biome coverage mismatch: missing=" +
                missingVanilla +
                " extra=" +
                extraPreset
            );
        }

        System.out.printf(
            "GeoWorld biome preset verification passed: preset=%d required=%d " +
            "vanillaOverworld=%d missing=0 extra=0%n",
            preset.size(),
            required.size(),
            vanillaOverworld.size()
        );
    }

    private static String readResource(String resource) throws Exception {
        try (InputStream input =
                 GeoWorldV26BiomePresetVerifier.class.getResourceAsStream(resource)) {
            if (input == null) {
                throw new IllegalStateException(
                    "required biome resource not found: " + resource
                );
            }
            return new String(
                input.readAllBytes(),
                StandardCharsets.UTF_8
            );
        }
    }

    private static LinkedHashSet<String> extractMinecraftIds(
        String json,
        String arrayKey
    ) {
        int key = json.indexOf("\"" + arrayKey + "\"");
        int open = key < 0 ? -1 : json.indexOf('[', key);
        int close = open < 0 ? -1 : json.indexOf(']', open);
        if (key < 0 || open < 0 || close < 0) {
            throw new IllegalStateException(
                "biome array is missing: " + arrayKey
            );
        }

        LinkedHashSet<String> ids = new LinkedHashSet<>();
        String block = json.substring(open + 1, close);
        java.util.regex.Matcher matcher = java.util.regex.Pattern
            .compile("\\\"(minecraft:[^\\\"]+)\\\"")
            .matcher(block);
        while (matcher.find()) {
            ids.add(matcher.group(1));
        }
        return ids;
    }
}
