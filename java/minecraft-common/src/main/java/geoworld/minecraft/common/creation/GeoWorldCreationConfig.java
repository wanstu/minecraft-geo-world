package geoworld.minecraft.common.creation;

import com.mojang.serialization.Codec;
import com.mojang.serialization.codecs.RecordCodecBuilder;

import java.util.List;

public record GeoWorldCreationConfig(
    int width,
    int height,
    int cellSizeBlocks,
    int plateCount,
    double oceanFraction,
    double continentFragmentation,
    double continentSeparation,
    ContinentPreset continentPreset
) {
    public static final int DEFAULT_WIDTH = 320;
    public static final int DEFAULT_HEIGHT = 200;
    public static final int DEFAULT_CELL_SIZE_BLOCKS = 128;
    public static final int DEFAULT_PLATE_COUNT = 18;

    public static final int MIN_DIMENSION = 16;
    public static final int MAX_DIMENSION = 1024;
    public static final int MIN_PLATE_COUNT = 2;
    public static final int MAX_PLATE_COUNT = 128;

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
            ).forGetter(GeoWorldCreationConfig::continentSeparation)
        ).apply(
            instance,
            (width, height, cellSizeBlocks, plateCount, ocean, fragmentation, separation) ->
                new GeoWorldCreationConfig(
                    width,
                    height,
                    cellSizeBlocks,
                    plateCount,
                    ocean,
                    fragmentation,
                    separation,
                    detectPreset(ocean, fragmentation, separation)
                )
        ));

    public static GeoWorldCreationConfig defaults() {
        return new GeoWorldCreationConfig(
            DEFAULT_WIDTH,
            DEFAULT_HEIGHT,
            DEFAULT_CELL_SIZE_BLOCKS,
            DEFAULT_PLATE_COUNT,
            ContinentPreset.DEFAULT.oceanFraction(),
            ContinentPreset.DEFAULT.fragmentation(),
            ContinentPreset.DEFAULT.separation(),
            ContinentPreset.DEFAULT
        );
    }

    public GeoWorldCreationConfig withPreset(ContinentPreset preset) {
        if (preset == ContinentPreset.CUSTOM) {
            return new GeoWorldCreationConfig(
                width,
                height,
                cellSizeBlocks,
                plateCount,
                oceanFraction,
                continentFragmentation,
                continentSeparation,
                ContinentPreset.CUSTOM
            );
        }
        return new GeoWorldCreationConfig(
            width,
            height,
            cellSizeBlocks,
            plateCount,
            preset.oceanFraction(),
            preset.fragmentation(),
            preset.separation(),
            preset
        );
    }

    public List<String> validationErrors() {
        java.util.ArrayList<String> errors = new java.util.ArrayList<>();
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
        return List.copyOf(errors);
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
}
