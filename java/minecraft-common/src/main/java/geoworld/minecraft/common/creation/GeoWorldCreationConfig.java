package geoworld.minecraft.common.creation;

import com.mojang.serialization.Codec;
import com.mojang.serialization.codecs.RecordCodecBuilder;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

public record GeoWorldCreationConfig(
    int width,
    int height,
    int cellSizeBlocks,
    int plateCount,
    double oceanFraction,
    double continentFragmentation,
    double continentSeparation,
    ContinentPreset continentPreset,
    WorldPreset worldPreset,
    Parameters parameters
) {
    public static final int DEFAULT_WIDTH = 320;
    public static final int DEFAULT_HEIGHT = 200;
    public static final int DEFAULT_CELL_SIZE_BLOCKS = 128;
    public static final int DEFAULT_PLATE_COUNT = 18;

    public static final int MIN_DIMENSION = 16;
    public static final int MAX_DIMENSION = 1024;
    public static final int MIN_PLATE_COUNT = 2;
    public static final int MAX_PLATE_COUNT = 128;

    private static final Codec<WorldPreset> WORLD_PRESET_CODEC =
        Codec.STRING.xmap(WorldPreset::fromId, WorldPreset::id);

    public static final Codec<GeoWorldCreationConfig> CODEC =
        RecordCodecBuilder.create(instance -> instance.group(
            Codec.INT.optionalFieldOf("width", DEFAULT_WIDTH)
                .forGetter(GeoWorldCreationConfig::width),
            Codec.INT.optionalFieldOf("height", DEFAULT_HEIGHT)
                .forGetter(GeoWorldCreationConfig::height),
            Codec.INT.optionalFieldOf("cell_size_blocks", DEFAULT_CELL_SIZE_BLOCKS)
                .forGetter(GeoWorldCreationConfig::cellSizeBlocks),
            Codec.INT.optionalFieldOf("plate_count", DEFAULT_PLATE_COUNT)
                .forGetter(GeoWorldCreationConfig::plateCount),
            Codec.DOUBLE.optionalFieldOf(
                "ocean_fraction",
                ContinentPreset.DEFAULT.oceanFraction()
            ).forGetter(GeoWorldCreationConfig::oceanFraction),
            Codec.DOUBLE.optionalFieldOf(
                "continent_fragmentation",
                ContinentPreset.DEFAULT.fragmentation()
            ).forGetter(GeoWorldCreationConfig::continentFragmentation),
            Codec.DOUBLE.optionalFieldOf(
                "continent_separation",
                ContinentPreset.DEFAULT.separation()
            ).forGetter(GeoWorldCreationConfig::continentSeparation),
            WORLD_PRESET_CODEC.optionalFieldOf(
                "world_preset",
                WorldPreset.STANDARD
            ).forGetter(GeoWorldCreationConfig::worldPreset),
            Parameters.CODEC.optionalFieldOf(
                "parameters",
                Parameters.standard()
            ).forGetter(GeoWorldCreationConfig::parameters)
        ).apply(
            instance,
            (width, height, cellSizeBlocks, plateCount, ocean, fragmentation,
             separation, worldPreset, parameters) ->
                new GeoWorldCreationConfig(
                    width,
                    height,
                    cellSizeBlocks,
                    plateCount,
                    ocean,
                    fragmentation,
                    separation,
                    detectPreset(ocean, fragmentation, separation),
                    worldPreset,
                    parameters
                )
        ));

    /** Compatibility constructor for existing adapter/verifier call sites. */
    public GeoWorldCreationConfig(
        int width,
        int height,
        int cellSizeBlocks,
        int plateCount,
        double oceanFraction,
        double continentFragmentation,
        double continentSeparation,
        ContinentPreset continentPreset
    ) {
        this(
            width,
            height,
            cellSizeBlocks,
            plateCount,
            oceanFraction,
            continentFragmentation,
            continentSeparation,
            continentPreset,
            WorldPreset.STANDARD,
            Parameters.standard()
        );
    }

    public GeoWorldCreationConfig {
        if (continentPreset == null) {
            continentPreset = detectPreset(
                oceanFraction,
                continentFragmentation,
                continentSeparation
            );
        }
        if (worldPreset == null) {
            worldPreset = WorldPreset.STANDARD;
        }
        if (parameters == null) {
            parameters = Parameters.forPreset(worldPreset);
        }
    }

    public static GeoWorldCreationConfig defaults() {
        return new GeoWorldCreationConfig(
            DEFAULT_WIDTH,
            DEFAULT_HEIGHT,
            DEFAULT_CELL_SIZE_BLOCKS,
            DEFAULT_PLATE_COUNT,
            ContinentPreset.DEFAULT.oceanFraction(),
            ContinentPreset.DEFAULT.fragmentation(),
            ContinentPreset.DEFAULT.separation(),
            ContinentPreset.DEFAULT,
            WorldPreset.STANDARD,
            Parameters.standard()
        );
    }

    public GeoWorldCreationConfig withPreset(ContinentPreset preset) {
        if (preset == ContinentPreset.CUSTOM) {
            return new GeoWorldCreationConfig(
                width, height, cellSizeBlocks, plateCount,
                oceanFraction, continentFragmentation, continentSeparation,
                ContinentPreset.CUSTOM, worldPreset, parameters
            );
        }
        return new GeoWorldCreationConfig(
            width, height, cellSizeBlocks, plateCount,
            preset.oceanFraction(), preset.fragmentation(), preset.separation(),
            preset, worldPreset, parameters
        );
    }

    public GeoWorldCreationConfig withWorldPreset(WorldPreset preset) {
        if (preset == WorldPreset.CUSTOM) {
            return new GeoWorldCreationConfig(
                width, height, cellSizeBlocks, plateCount,
                oceanFraction, continentFragmentation, continentSeparation,
                continentPreset, WorldPreset.CUSTOM, parameters
            );
        }
        Parameters next = Parameters.forPreset(preset);
        int nextWidth = scaledDimension(DEFAULT_WIDTH, next.worldScale().mainlandScale());
        int nextHeight = scaledDimension(DEFAULT_HEIGHT, next.worldScale().mainlandScale());
        return new GeoWorldCreationConfig(
            nextWidth, nextHeight, cellSizeBlocks, plateCount,
            oceanFraction, continentFragmentation, continentSeparation,
            continentPreset, preset, next
        );
    }

    public GeoWorldCreationConfig withParameters(Parameters next) {
        return new GeoWorldCreationConfig(
            width, height, cellSizeBlocks, plateCount,
            oceanFraction, continentFragmentation, continentSeparation,
            continentPreset, WorldPreset.CUSTOM, next
        );
    }

    public List<String> validationErrors() {
        ArrayList<String> errors = new ArrayList<>();
        if (width < MIN_DIMENSION || width > MAX_DIMENSION) {
            errors.add("width must be in " + MIN_DIMENSION + ".." + MAX_DIMENSION);
        }
        if (height < MIN_DIMENSION || height > MAX_DIMENSION) {
            errors.add("height must be in " + MIN_DIMENSION + ".." + MAX_DIMENSION);
        }
        if (cellSizeBlocks != 64 && cellSizeBlocks != 128) {
            errors.add("cell size must be 64 or 128 blocks");
        }
        if (plateCount < MIN_PLATE_COUNT || plateCount > MAX_PLATE_COUNT) {
            errors.add("plate count must be in " + MIN_PLATE_COUNT + ".." + MAX_PLATE_COUNT);
        }
        validateFraction(errors, "ocean fraction", oceanFraction, 0.10, 0.95);
        validateFraction(errors, "continent fragmentation", continentFragmentation, 0.05, 1.0);
        validateFraction(errors, "continent separation", continentSeparation, 0.05, 1.0);
        parameters.validate(errors);
        return List.copyOf(errors);
    }

    public List<String> validationWarningKeys() {
        ArrayList<String> warnings = new ArrayList<>();
        WorldScale scale = parameters.worldScale();
        TerrainParameters terrain = parameters.terrain();
        WaterParameters water = parameters.water();
        ClimateParameters climate = parameters.climate();
        UndergroundParameters underground = parameters.underground();
        CompatibilityParameters compatibility = parameters.compatibility();

        if (
            scale.mainlandScale() < 0.85 &&
            (
                scale.regionScale() > 1.30 ||
                scale.transitionWidth() > 1.30 ||
                climate.climateRegionScale() > 1.30 ||
                climate.climateTransitionWidth() > 1.30
            )
        ) {
            warnings.add("geoworld.warning.spatial_pressure");
        }
        if (
            water.riverDensity() > 1.55 &&
            (
                water.lakeAbundance() > 1.35 ||
                water.wetlandAbundance() > 1.35
            )
        ) {
            warnings.add("geoworld.warning.water_fragmentation");
        }
        if (
            compatibility.structureInfluence() > 1.55 &&
            (
                terrain.mountainStrength() > 1.35 ||
                terrain.rareMacroTerrain() > 1.35 ||
                terrain.cliffRarity() < 0.80
            )
        ) {
            warnings.add("geoworld.warning.structure_pressure");
        }
        if (
            underground.caveRichness() > 1.70 &&
            underground.largeCaveRarity() < 0.75
        ) {
            warnings.add("geoworld.warning.cave_pressure");
        }
        return List.copyOf(warnings);
    }

    public boolean isValid() {
        return validationErrors().isEmpty();
    }

    public long spanXBlocks() {
        return (long)(width - 1) * cellSizeBlocks;
    }

    public long spanZBlocks() {
        return (long)(height - 1) * cellSizeBlocks;
    }

    public static ContinentPreset detectPreset(
        double oceanFraction,
        double fragmentation,
        double separation
    ) {
        for (ContinentPreset preset : ContinentPreset.values()) {
            if (preset == ContinentPreset.CUSTOM) continue;
            if (
                close(preset.oceanFraction(), oceanFraction) &&
                close(preset.fragmentation(), fragmentation) &&
                close(preset.separation(), separation)
            ) {
                return preset;
            }
        }
        return ContinentPreset.CUSTOM;
    }

    private static int scaledDimension(int base, double scale) {
        int value = (int)Math.round(base * scale);
        return Math.max(MIN_DIMENSION, Math.min(MAX_DIMENSION, value));
    }

    private static boolean close(double a, double b) {
        return Math.abs(a - b) <= 1.0e-9;
    }

    private static void validateFraction(
        List<String> errors,
        String name,
        double value,
        double min,
        double max
    ) {
        double epsilon = 1.0e-9;
        if (
            !Double.isFinite(value) ||
            value < min - epsilon ||
            value > max + epsilon
        ) {
            errors.add(name + " must be in " + min + ".." + max);
        }
    }

    private static void validateParameter(
        List<String> errors,
        String name,
        double value
    ) {
        if (!Double.isFinite(value) || value < 0.25 || value > 2.5) {
            errors.add(name + " must be in 0.25..2.5");
        }
    }

    public enum WorldPreset {
        STANDARD("standard"),
        NATURAL("natural"),
        CIVILIZATION("civilization"),
        EXPANSIVE("expansive"),
        EPIC("epic"),
        NEAR_VANILLA("near_vanilla"),
        CUSTOM("custom");

        private final String id;

        WorldPreset(String id) {
            this.id = id;
        }

        public String id() {
            return id;
        }

        public String translationKey() {
            return "geoworld.world_preset." + id;
        }

        public static WorldPreset fromId(String id) {
            for (WorldPreset preset : values()) {
                if (preset.id.equals(id)) return preset;
            }
            return CUSTOM;
        }
    }

    public enum ContinentPreset {
        DEFAULT("default", 0.68, 0.35, 0.65),
        SUPERCONTINENT("supercontinent", 0.50, 0.05, 0.05),
        MULTI_CONTINENT("multi_continent", 0.68, 0.30, 0.90),
        BROKEN_CONTINENT("broken_continent", 0.68, 0.75, 0.75),
        ARCHIPELAGO("archipelago", 0.90, 0.95, 0.95),
        CUSTOM("custom", Double.NaN, Double.NaN, Double.NaN);

        private final String id;
        private final double oceanFraction;
        private final double fragmentation;
        private final double separation;

        ContinentPreset(
            String id,
            double oceanFraction,
            double fragmentation,
            double separation
        ) {
            this.id = id;
            this.oceanFraction = oceanFraction;
            this.fragmentation = fragmentation;
            this.separation = separation;
        }

        public String id() {
            return id;
        }

        public double oceanFraction() {
            return oceanFraction;
        }

        public double fragmentation() {
            return fragmentation;
        }

        public double separation() {
            return separation;
        }

        public String translationKey() {
            return "geoworld.preset." + id;
        }
    }

    public record WorldScale(
        double mainlandScale,
        double regionScale,
        double transitionWidth
    ) {
        static final Codec<WorldScale> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("mainland_scale", 1.0).forGetter(WorldScale::mainlandScale),
            Codec.DOUBLE.optionalFieldOf("region_scale", 1.0).forGetter(WorldScale::regionScale),
            Codec.DOUBLE.optionalFieldOf("transition_width", 1.0).forGetter(WorldScale::transitionWidth)
        ).apply(instance, WorldScale::new));
    }

    public record TerrainParameters(
        double mountainStrength,
        double plainContinuity,
        double plateauScale,
        double cliffRarity,
        double rareMacroTerrain
    ) {
        static final Codec<TerrainParameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("mountain_strength", 1.0).forGetter(TerrainParameters::mountainStrength),
            Codec.DOUBLE.optionalFieldOf("plain_continuity", 1.0).forGetter(TerrainParameters::plainContinuity),
            Codec.DOUBLE.optionalFieldOf("plateau_scale", 1.0).forGetter(TerrainParameters::plateauScale),
            Codec.DOUBLE.optionalFieldOf("cliff_rarity", 1.0).forGetter(TerrainParameters::cliffRarity),
            Codec.DOUBLE.optionalFieldOf("rare_macro_terrain", 1.0).forGetter(TerrainParameters::rareMacroTerrain)
        ).apply(instance, TerrainParameters::new));
    }

    public record WaterParameters(
        double riverDensity,
        double mainRiverScale,
        double lakeAbundance,
        double wetlandAbundance
    ) {
        static final Codec<WaterParameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("river_density", 1.0).forGetter(WaterParameters::riverDensity),
            Codec.DOUBLE.optionalFieldOf("main_river_scale", 1.0).forGetter(WaterParameters::mainRiverScale),
            Codec.DOUBLE.optionalFieldOf("lake_abundance", 1.0).forGetter(WaterParameters::lakeAbundance),
            Codec.DOUBLE.optionalFieldOf("wetland_abundance", 1.0).forGetter(WaterParameters::wetlandAbundance)
        ).apply(instance, WaterParameters::new));
    }

    public record ClimateParameters(
        double climateRegionScale,
        double climateTransitionWidth,
        double rainShadowStrength,
        double biomeContinuity,
        double oceanClimateStrength,
        double prevailingWindStrength
    ) {
        static final Codec<ClimateParameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("region_scale", 1.0).forGetter(ClimateParameters::climateRegionScale),
            Codec.DOUBLE.optionalFieldOf("transition_width", 1.0).forGetter(ClimateParameters::climateTransitionWidth),
            Codec.DOUBLE.optionalFieldOf("rain_shadow_strength", 1.0).forGetter(ClimateParameters::rainShadowStrength),
            Codec.DOUBLE.optionalFieldOf("biome_continuity", 1.0).forGetter(ClimateParameters::biomeContinuity),
            Codec.DOUBLE.optionalFieldOf("ocean_climate_strength", 1.0).forGetter(ClimateParameters::oceanClimateStrength),
            Codec.DOUBLE.optionalFieldOf("prevailing_wind_strength", 1.0).forGetter(ClimateParameters::prevailingWindStrength)
        ).apply(instance, ClimateParameters::new));
    }

    public record UndergroundParameters(
        double caveRichness,
        double largeCaveRarity,
        double mineralRegionalization
    ) {
        static final Codec<UndergroundParameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("cave_richness", 1.0).forGetter(UndergroundParameters::caveRichness),
            Codec.DOUBLE.optionalFieldOf("large_cave_rarity", 1.0).forGetter(UndergroundParameters::largeCaveRarity),
            Codec.DOUBLE.optionalFieldOf("mineral_regionalization", 1.0).forGetter(UndergroundParameters::mineralRegionalization)
        ).apply(instance, UndergroundParameters::new));
    }

    public record CompatibilityParameters(
        double spawnPreference,
        double structureInfluence
    ) {
        static final Codec<CompatibilityParameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("spawn_preference", 1.0).forGetter(CompatibilityParameters::spawnPreference),
            Codec.DOUBLE.optionalFieldOf("structure_influence", 1.0).forGetter(CompatibilityParameters::structureInfluence)
        ).apply(instance, CompatibilityParameters::new));
    }

    public record OceanParameters(
        double outerIslandAbundance,
        double polarTransitionWidth
    ) {
        static final Codec<OceanParameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            Codec.DOUBLE.optionalFieldOf("outer_island_abundance", 1.0).forGetter(OceanParameters::outerIslandAbundance),
            Codec.DOUBLE.optionalFieldOf("polar_transition_width", 1.0).forGetter(OceanParameters::polarTransitionWidth)
        ).apply(instance, OceanParameters::new));
    }

    public record Parameters(
        WorldScale worldScale,
        TerrainParameters terrain,
        WaterParameters water,
        ClimateParameters climate,
        UndergroundParameters underground,
        CompatibilityParameters compatibility,
        OceanParameters ocean
    ) {
        public static final Codec<Parameters> CODEC = RecordCodecBuilder.create(instance -> instance.group(
            WorldScale.CODEC.fieldOf("world_scale").forGetter(Parameters::worldScale),
            TerrainParameters.CODEC.fieldOf("terrain").forGetter(Parameters::terrain),
            WaterParameters.CODEC.fieldOf("water").forGetter(Parameters::water),
            ClimateParameters.CODEC.fieldOf("climate").forGetter(Parameters::climate),
            UndergroundParameters.CODEC.fieldOf("underground").forGetter(Parameters::underground),
            CompatibilityParameters.CODEC.fieldOf("compatibility").forGetter(Parameters::compatibility),
            OceanParameters.CODEC.fieldOf("ocean").forGetter(Parameters::ocean)
        ).apply(instance, Parameters::new));

        public static Parameters standard() {
            return new Parameters(
                new WorldScale(1, 1, 1),
                new TerrainParameters(1, 1, 1, 1, 1),
                new WaterParameters(1, 1, 1, 1),
                new ClimateParameters(1, 1, 1, 1, 1, 1),
                new UndergroundParameters(1, 1, 1),
                new CompatibilityParameters(1, 1),
                new OceanParameters(1, 1)
            );
        }

        public static Parameters forPreset(WorldPreset preset) {
            Parameters p = standard();
            return switch (preset) {
                case STANDARD, CUSTOM -> p;
                case NATURAL -> new Parameters(
                    new WorldScale(1.15, 1.20, 1.20),
                    new TerrainParameters(0.92, 0.92, 1, 1.28, 1),
                    new WaterParameters(0.92, 1, 1, 1),
                    new ClimateParameters(1.20, 1.20, 1, 1.18, 1, 1),
                    p.underground(),
                    new CompatibilityParameters(0.85, 0.75),
                    new OceanParameters(0.75, 1)
                );
                case CIVILIZATION -> new Parameters(
                    new WorldScale(1.08, 1.05, 1),
                    new TerrainParameters(0.86, 1.34, 1.08, 1.45, 1),
                    new WaterParameters(0.95, 1.08, 0.90, 0.82),
                    p.climate(),
                    p.underground(),
                    new CompatibilityParameters(1.30, 1.20),
                    p.ocean()
                );
                case EXPANSIVE -> new Parameters(
                    new WorldScale(1.35, 1.40, 1.35),
                    p.terrain(),
                    p.water(),
                    new ClimateParameters(1.35, 1.35, 1, 1.25, 1, 1),
                    p.underground(),
                    p.compatibility(),
                    new OceanParameters(0.75, 1)
                );
                case EPIC -> new Parameters(
                    new WorldScale(1.15, 1, 1),
                    new TerrainParameters(1.30, 1, 1.18, 0.78, 1.45),
                    new WaterParameters(1.10, 1.35, 1.20, 1),
                    new ClimateParameters(1, 1, 1.15, 1, 1, 1),
                    p.underground(),
                    p.compatibility(),
                    p.ocean()
                );
                case NEAR_VANILLA -> new Parameters(
                    new WorldScale(0.88, 0.80, 0.82),
                    new TerrainParameters(0.92, 0.82, 1, 1, 1),
                    new WaterParameters(0.88, 0.82, 0.90, 1),
                    new ClimateParameters(0.82, 0.82, 1, 0.80, 1, 1),
                    new UndergroundParameters(1.35, 1, 1),
                    new CompatibilityParameters(1, 1.35),
                    p.ocean()
                );
            };
        }

        void validate(List<String> errors) {
            double[] values = {
                worldScale.mainlandScale(), worldScale.regionScale(), worldScale.transitionWidth(),
                terrain.mountainStrength(), terrain.plainContinuity(), terrain.plateauScale(),
                terrain.cliffRarity(), terrain.rareMacroTerrain(),
                water.riverDensity(), water.mainRiverScale(), water.lakeAbundance(), water.wetlandAbundance(),
                climate.climateRegionScale(), climate.climateTransitionWidth(),
                climate.rainShadowStrength(), climate.biomeContinuity(),
                underground.caveRichness(), underground.largeCaveRarity(), underground.mineralRegionalization(),
                compatibility.spawnPreference(), compatibility.structureInfluence(),
                ocean.outerIslandAbundance(), ocean.polarTransitionWidth(),
                climate.oceanClimateStrength(), climate.prevailingWindStrength()
            };
            String[] names = {
                "mainland scale", "region scale", "transition width",
                "mountain strength", "plain continuity", "plateau scale",
                "cliff rarity", "rare macro terrain",
                "river density", "main river scale", "lake abundance", "wetland abundance",
                "climate region scale", "climate transition width",
                "rain shadow strength", "biome continuity",
                "cave richness", "large cave rarity", "mineral regionalization",
                "spawn preference", "structure influence",
                "outer island abundance", "polar transition width",
                "ocean climate strength", "prevailing wind strength"
            };
            for (int i = 0; i < values.length; i++) {
                if (i == 19 || i == 20) {
                    if (!Double.isFinite(values[i]) || values[i] < 0 || values[i] > 2.5) {
                        errors.add(names[i] + " must be in 0..2.5");
                    }
                } else {
                    validateParameter(errors, names[i], values[i]);
                }
            }
        }

        public double[] values() {
            return new double[] {
                worldScale.mainlandScale(), worldScale.regionScale(), worldScale.transitionWidth(),
                terrain.mountainStrength(), terrain.plainContinuity(), terrain.plateauScale(),
                terrain.cliffRarity(), terrain.rareMacroTerrain(),
                water.riverDensity(), water.mainRiverScale(), water.lakeAbundance(), water.wetlandAbundance(),
                climate.climateRegionScale(), climate.climateTransitionWidth(),
                climate.rainShadowStrength(), climate.biomeContinuity(),
                underground.caveRichness(), underground.largeCaveRarity(), underground.mineralRegionalization(),
                compatibility.spawnPreference(), compatibility.structureInfluence(),
                ocean.outerIslandAbundance(), ocean.polarTransitionWidth(),
                climate.oceanClimateStrength(), climate.prevailingWindStrength()
            };
        }

        public static Parameters fromValues(double[] values) {
            if (values == null || values.length != 25) {
                throw new IllegalArgumentException("GeoWorld parameter vector must contain 25 values");
            }
            return new Parameters(
                new WorldScale(values[0], values[1], values[2]),
                new TerrainParameters(values[3], values[4], values[5], values[6], values[7]),
                new WaterParameters(values[8], values[9], values[10], values[11]),
                new ClimateParameters(values[12], values[13], values[14], values[15], values[23], values[24]),
                new UndergroundParameters(values[16], values[17], values[18]),
                new CompatibilityParameters(values[19], values[20]),
                new OceanParameters(values[21], values[22])
            );
        }

        public String toCsv() {
            double[] values = values();
            StringBuilder out = new StringBuilder();
            for (int i = 0; i < values.length; i++) {
                if (i > 0) out.append(',');
                out.append(String.format(Locale.ROOT, "%.6f", values[i]));
            }
            return out.toString();
        }
    }
}
