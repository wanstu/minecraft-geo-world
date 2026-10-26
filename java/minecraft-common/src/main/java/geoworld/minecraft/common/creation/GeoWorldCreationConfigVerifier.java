package geoworld.minecraft.common.creation;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

/**
 * Executable acceptance checks for the semantic world-creation parameter
 * contract. It deliberately verifies only claims that can be proven without
 * subjective in-game visual judgement.
 */
public final class GeoWorldCreationConfigVerifier {
    private static final String[] PARAMETER_KEYS = {
        "mainland_scale",
        "region_scale",
        "transition_width",
        "mountain_strength",
        "plain_continuity",
        "plateau_scale",
        "cliff_rarity",
        "rare_macro_terrain",
        "river_density",
        "main_river_scale",
        "lake_abundance",
        "wetland_abundance",
        "climate_region_scale",
        "climate_transition_width",
        "rain_shadow_strength",
        "biome_continuity",
        "cave_richness",
        "large_cave_rarity",
        "mineral_regionalization",
        "spawn_preference",
        "structure_influence",
        "outer_island_abundance",
        "polar_transition_width",
        "ocean_climate_strength",
        "prevailing_wind_strength"
    };

    private static final List<GeoWorldCreationConfig.WorldPreset>
        REPRESENTATIVE_PRESETS = List.of(
            GeoWorldCreationConfig.WorldPreset.STANDARD,
            GeoWorldCreationConfig.WorldPreset.NATURAL,
            GeoWorldCreationConfig.WorldPreset.CIVILIZATION,
            GeoWorldCreationConfig.WorldPreset.EPIC
        );

    private GeoWorldCreationConfigVerifier() {}

    public static void main(String[] args) throws Exception {
        verifyDefaults();
        pass("PARAM-01");

        verifyLocalizedDescriptions();
        pass("PARAM-02");

        verifyConflictWarnings();
        pass("PARAM-04");

        verifySnapshotRoundTrip();
        pass("PARAM-05");

        verifyPresetsAreParameterBundles();
        pass("PARAM-06");

        verifyModelCapabilitiesAreSeparatedFromParameters();
        pass("PARAM-07");

        verifyRepresentativePresetsShareOneConfigContract();
        pass("PARAM-08");

        System.out.println(
            "DELEGATED PARAM-03 verifier=verifyGeoWorldParameterEffects " +
            "scope=real-engine-world-output"
        );
        System.out.println(
            "GeoWorld Creation Config Verifier: passed=7/7 local, delegated=1"
        );
    }

    private static void verifyDefaults() {
        GeoWorldCreationConfig config = GeoWorldCreationConfig.defaults();
        require(config.isValid(), "default creation config is invalid: " + config.validationErrors());
        require(
            config.worldPreset() == GeoWorldCreationConfig.WorldPreset.STANDARD,
            "default world preset is not Standard"
        );
        double[] values = config.parameters().values();
        require(values.length == 25, "default parameter count=" + values.length + " want=25");
        for (int i = 0; i < values.length; i++) {
            require(Double.isFinite(values[i]), "default parameter[" + i + "] is not finite");
            require(values[i] == 1.0, "Standard parameter[" + i + "]=" + values[i] + " want=1");
        }

        for (GeoWorldCreationConfig.WorldPreset preset :
            GeoWorldCreationConfig.WorldPreset.values()) {
            if (preset == GeoWorldCreationConfig.WorldPreset.CUSTOM) continue;
            GeoWorldCreationConfig candidate = config.withWorldPreset(preset);
            require(
                candidate.isValid(),
                preset + " preset is invalid: " + candidate.validationErrors()
            );
            require(
                candidate.parameters().values().length == 25,
                preset + " does not expose exactly 25 parameters"
            );
        }
    }

    private static void verifyLocalizedDescriptions() throws IOException {
        for (String locale : List.of("en_us", "zh_cn")) {
            String text = readResource("assets/geoworld/lang/" + locale + ".json");
            for (String key : PARAMETER_KEYS) {
                requireJsonKey(text, "geoworld.parameter." + key, locale);
                requireJsonKey(text, "geoworld.parameter." + key + ".desc", locale);
            }
            for (String warning : List.of(
                "spatial_pressure",
                "water_fragmentation",
                "structure_pressure",
                "cave_pressure"
            )) {
                requireJsonKey(text, "geoworld.warning." + warning, locale);
            }
        }
    }

    private static void verifyConflictWarnings() {
        GeoWorldCreationConfig base = GeoWorldCreationConfig.defaults();
        GeoWorldCreationConfig.Parameters standard =
            GeoWorldCreationConfig.Parameters.standard();

        GeoWorldCreationConfig.Parameters spatial =
            new GeoWorldCreationConfig.Parameters(
                new GeoWorldCreationConfig.WorldScale(0.80, 1.40, 1.40),
                standard.terrain(),
                standard.water(),
                new GeoWorldCreationConfig.ClimateParameters(
                    1.40, 1.40, 1, 1, 1, 1
                ),
                standard.underground(),
                standard.compatibility(),
                standard.ocean()
            );
        requireWarning(
            base.withParameters(spatial),
            "geoworld.warning.spatial_pressure"
        );

        GeoWorldCreationConfig.Parameters water =
            new GeoWorldCreationConfig.Parameters(
                standard.worldScale(),
                standard.terrain(),
                new GeoWorldCreationConfig.WaterParameters(1.60, 1, 1.40, 1.40),
                standard.climate(),
                standard.underground(),
                standard.compatibility(),
                standard.ocean()
            );
        requireWarning(
            base.withParameters(water),
            "geoworld.warning.water_fragmentation"
        );

        GeoWorldCreationConfig.Parameters structure =
            new GeoWorldCreationConfig.Parameters(
                standard.worldScale(),
                new GeoWorldCreationConfig.TerrainParameters(
                    1.40, 1, 1, 0.75, 1.40
                ),
                standard.water(),
                standard.climate(),
                standard.underground(),
                new GeoWorldCreationConfig.CompatibilityParameters(1, 1.60),
                standard.ocean()
            );
        requireWarning(
            base.withParameters(structure),
            "geoworld.warning.structure_pressure"
        );

        GeoWorldCreationConfig.Parameters caves =
            new GeoWorldCreationConfig.Parameters(
                standard.worldScale(),
                standard.terrain(),
                standard.water(),
                standard.climate(),
                new GeoWorldCreationConfig.UndergroundParameters(1.80, 0.70, 1),
                standard.compatibility(),
                standard.ocean()
            );
        requireWarning(
            base.withParameters(caves),
            "geoworld.warning.cave_pressure"
        );
    }

    private static void verifySnapshotRoundTrip() {
        GeoWorldCreationConfig.Parameters source =
            GeoWorldCreationConfig.Parameters.forPreset(
                GeoWorldCreationConfig.WorldPreset.EPIC
            );
        double[] values = source.values();
        require(values.length == 25, "snapshot source parameter count is not 25");

        GeoWorldCreationConfig.Parameters restored =
            GeoWorldCreationConfig.Parameters.fromValues(values);
        require(
            Arrays.equals(source.values(), restored.values()),
            "25-value parameter snapshot did not round-trip"
        );

        String csv = restored.toCsv();
        require(
            csv.split(",", -1).length == 25,
            "stable parameter CSV does not contain 25 values"
        );

        GeoWorldCreationConfig custom =
            GeoWorldCreationConfig.defaults().withParameters(restored);
        require(
            custom.worldPreset() == GeoWorldCreationConfig.WorldPreset.CUSTOM,
            "custom snapshot retained a display preset as source of truth"
        );
        require(
            Arrays.equals(custom.parameters().values(), values),
            "custom config changed its parameter snapshot"
        );
    }

    private static void verifyPresetsAreParameterBundles() {
        GeoWorldCreationConfig base = GeoWorldCreationConfig.defaults();
        Set<String> vectors = new LinkedHashSet<>();

        for (GeoWorldCreationConfig.WorldPreset preset :
            GeoWorldCreationConfig.WorldPreset.values()) {
            if (preset == GeoWorldCreationConfig.WorldPreset.CUSTOM) continue;
            GeoWorldCreationConfig.Parameters expected =
                GeoWorldCreationConfig.Parameters.forPreset(preset);
            GeoWorldCreationConfig configured = base.withWorldPreset(preset);

            require(
                configured.worldPreset() == preset,
                "preset selection did not retain display preset " + preset
            );
            require(
                Arrays.equals(
                    configured.parameters().values(),
                    expected.values()
                ),
                "preset " + preset + " contains hidden state outside its parameter bundle"
            );
            vectors.add(expected.toCsv());
        }

        require(
            vectors.size() >= 4,
            "official presets do not produce distinguishable parameter bundles"
        );
    }

    private static void verifyModelCapabilitiesAreSeparatedFromParameters() {
        // PARAM-07: creation parameters may tune magnitude/scale/frequency, but
        // must not become feature/capability switches. Keep every persisted
        // parameter component a numeric scalar and keep presets on one config
        // contract; causal capabilities remain implemented by the model.
        Class<?>[] groups = {
            GeoWorldCreationConfig.WorldScale.class,
            GeoWorldCreationConfig.TerrainParameters.class,
            GeoWorldCreationConfig.WaterParameters.class,
            GeoWorldCreationConfig.ClimateParameters.class,
            GeoWorldCreationConfig.UndergroundParameters.class,
            GeoWorldCreationConfig.CompatibilityParameters.class,
            GeoWorldCreationConfig.OceanParameters.class
        };
        int scalarCount = 0;
        for (Class<?> group : groups) {
            require(group.isRecord(), group.getSimpleName() + " is not a record");
            for (var component : group.getRecordComponents()) {
                require(
                    component.getType() == double.class,
                    "parameter capability switch detected: " +
                        group.getSimpleName() + "." + component.getName() +
                        " type=" + component.getType().getName()
                );
                scalarCount++;
            }
        }
        require(
            scalarCount == 25,
            "numeric parameter surface count=" + scalarCount + " want=25"
        );

        GeoWorldCreationConfig base = GeoWorldCreationConfig.defaults();
        for (GeoWorldCreationConfig.WorldPreset preset : REPRESENTATIVE_PRESETS) {
            GeoWorldCreationConfig configured = base.withWorldPreset(preset);
            require(
                configured.getClass() == GeoWorldCreationConfig.class,
                preset + " introduced a separate model/config capability type"
            );
            require(
                configured.parameters().values().length == scalarCount,
                preset + " changed the model capability surface instead of scalar parameters"
            );
        }
    }

    private static void verifyRepresentativePresetsShareOneConfigContract() {
        GeoWorldCreationConfig base = GeoWorldCreationConfig.defaults();
        for (GeoWorldCreationConfig.WorldPreset preset : REPRESENTATIVE_PRESETS) {
            GeoWorldCreationConfig configured = base.withWorldPreset(preset);
            require(
                configured.getClass() == GeoWorldCreationConfig.class,
                preset + " switched to a separate generator config type"
            );
            require(
                configured.parameters().values().length == 25,
                preset + " does not use the common 25-value parameter contract"
            );
        }
    }

    private static void requireWarning(
        GeoWorldCreationConfig config,
        String warning
    ) {
        require(config.isValid(), "warning test config is invalid: " + config.validationErrors());
        require(
            config.validationWarningKeys().contains(warning),
            "expected conflict warning missing: " + warning +
                " actual=" + config.validationWarningKeys()
        );
    }

    private static String readResource(String path) throws IOException {
        ClassLoader loader = GeoWorldCreationConfigVerifier.class.getClassLoader();
        try (InputStream in = loader.getResourceAsStream(path)) {
            if (in == null) {
                throw new IllegalStateException("missing classpath resource: " + path);
            }
            return new String(in.readAllBytes(), StandardCharsets.UTF_8);
        }
    }

    private static void requireJsonKey(
        String json,
        String key,
        String locale
    ) {
        require(
            json.contains("\"" + key + "\""),
            locale + " is missing translation key " + key
        );
    }

    private static void pass(String id) {
        System.out.println("PASS " + id);
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalStateException(message);
        }
    }
}
