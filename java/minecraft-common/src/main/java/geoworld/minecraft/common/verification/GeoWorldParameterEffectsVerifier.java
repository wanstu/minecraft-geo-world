package geoworld.minecraft.common.verification;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.ContinentPreset;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.Parameters;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.WorldPreset;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * PARAM-03 executable verification.
 *
 * Uses the real embedded engine, a fixed Seed and direct 25-value parameter
 * snapshots. Each comparison changes exactly one semantic parameter, then
 * inspects the generated .geoworld output. Subjective visual quality remains a
 * manual concern; this verifier proves that parameter changes are stable and
 * materially observable in the expected model domain.
 */
public final class GeoWorldParameterEffectsVerifier {
    private static final long SEED = 4034658472718554561L;
    private static final double LOW = 0.50;
    private static final double HIGH = 1.80;
    private static final int WIDTH = 48;
    private static final int HEIGHT = 30;
    private static final double SAMPLE_STEP = 64.0;

    private GeoWorldParameterEffectsVerifier() {}

    public static void main(String[] args) throws Exception {
        EffectResult mountain = verifyMountainStrength();
        EffectResult riverWidth = verifyMainRiverScale();
        EffectResult riverDensity = verifyRiverDensity();
        EffectResult rainShadow = verifyRainShadowStrength();

        System.out.printf(
            "PASS PARAM-03 mountain=%s mainRiver=%s riverDensity=%s rainShadow=%s%n",
            mountain.summary(),
            riverWidth.summary(),
            riverDensity.summary(),
            rainShadow.summary()
        );
        System.out.println(
            "GeoWorld Parameter Effects Verifier: passed=4/4 representative causal groups"
        );
    }

    private static EffectResult verifyMountainStrength() throws Exception {
        var low = generateWithParameter(3, LOW);
        var high = generateWithParameter(3, HIGH);
        assertPersistedParameter(low.model(), 3, LOW, "mountain strength low");
        assertPersistedParameter(high.model(), 3, HIGH, "mountain strength high");

        Stats lowStats = scan(low.model());
        Stats highStats = scan(high.model());
        double delta = meanAbsoluteGroundDifference(
            low.model(),
            high.model()
        );
        double heightGain =
            highStats.mountainMean() - lowStats.mountainMean();
        double reliefGain =
            highStats.mountainLocalRelief() - lowStats.mountainLocalRelief();
        require(
            delta > 0.00075,
            "mountain strength did not materially change terrain output: " +
                "meanAbsGroundDelta=" + delta
        );
        require(
            lowStats.mountainCells() >= 100 && highStats.mountainCells() >= 100,
            "mountain strength scan found too few Mountain/Foothills samples: " +
                lowStats.mountainCells() + "/" + highStats.mountainCells()
        );
        require(
            heightGain > 0.004,
            "higher mountain strength did not raise the mountain envelope: low=" +
                lowStats.mountainMean() + " high=" + highStats.mountainMean() +
                " gain=" + heightGain
        );
        require(
            reliefGain > 0.00025,
            "higher mountain strength did not increase local mountain relief: low=" +
                lowStats.mountainLocalRelief() +
                " high=" + highStats.mountainLocalRelief() +
                " gain=" + reliefGain
        );
        assertStable(high);
        return new EffectResult(
            "mountain_strength",
            lowStats.mountainMean(),
            highStats.mountainMean(),
            delta
        );
    }

    private static EffectResult verifyMainRiverScale() throws Exception {
        var low = generateWithParameter(9, LOW);
        var high = generateWithParameter(9, HIGH);
        assertPersistedParameter(low.model(), 9, LOW, "main river scale low");
        assertPersistedParameter(high.model(), 9, HIGH, "main river scale high");

        Stats lowStats = scan(low.model());
        Stats highStats = scan(high.model());
        int minimumGain = Math.max(4, (int)Math.ceil(lowStats.riverCells() * 0.12));
        int delta = highStats.riverCells() - lowStats.riverCells();
        require(
            delta >= minimumGain,
            "main river scale did not widen the rendered river footprint: low=" +
                lowStats.riverCells() + " high=" + highStats.riverCells() +
                " requiredGain=" + minimumGain
        );
        assertStable(high);
        return new EffectResult(
            "main_river_scale",
            lowStats.riverCells(),
            highStats.riverCells(),
            delta
        );
    }

    private static EffectResult verifyRiverDensity() throws Exception {
        var low = generateWithParameter(8, LOW);
        var high = generateWithParameter(8, HIGH);
        assertPersistedParameter(low.model(), 8, LOW, "river density low");
        assertPersistedParameter(high.model(), 8, HIGH, "river density high");

        Stats lowStats = scan(low.model());
        Stats highStats = scan(high.model());
        int minimumGain = Math.max(12, (int)Math.ceil(lowStats.riverCells() * 0.05));
        int delta = highStats.riverCells() - lowStats.riverCells();
        require(
            delta >= minimumGain,
            "river density did not increase the rendered river network: low=" +
                lowStats.riverCells() + " high=" + highStats.riverCells() +
                " requiredGain=" + minimumGain
        );
        assertStable(high);
        return new EffectResult(
            "river_density",
            lowStats.riverCells(),
            highStats.riverCells(),
            delta
        );
    }

    private static EffectResult verifyRainShadowStrength() throws Exception {
        var low = generateWithParameter(14, LOW);
        var high = generateWithParameter(14, HIGH);
        assertPersistedParameter(low.model(), 14, LOW, "rain shadow low");
        assertPersistedParameter(high.model(), 14, HIGH, "rain shadow high");

        double delta = meanAbsolutePrecipitationDifference(
            low.model(),
            high.model()
        );
        require(
            delta > 0.50,
            "rain shadow strength did not materially change precipitation: " +
                "meanAbsDeltaMM=" + delta
        );
        assertStable(high);
        return new EffectResult(
            "rain_shadow_strength",
            0.0,
            delta,
            delta
        );
    }

    private static GeoWorldEngineBridge.GeneratedModel generateWithParameter(
        int parameterIndex,
        double value
    ) throws Exception {
        double[] values = Parameters.standard().values();
        values[parameterIndex] = value;
        Parameters parameters = Parameters.fromValues(values);
        GeoWorldCreationConfig config = new GeoWorldCreationConfig(
            WIDTH,
            HEIGHT,
            128,
            12,
            ContinentPreset.DEFAULT.oceanFraction(),
            ContinentPreset.DEFAULT.fragmentation(),
            ContinentPreset.DEFAULT.separation(),
            ContinentPreset.DEFAULT,
            WorldPreset.CUSTOM,
            parameters
        );
        return GeoWorldEngineBridge.generate(SEED, config);
    }

    private static void assertPersistedParameter(
        GeoWorldV1 model,
        int index,
        double expected,
        String label
    ) {
        double[] values = model.worldProfile().parameters().values();
        require(values.length == 25, label + " snapshot length=" + values.length);
        require(
            Math.abs(values[index] - expected) <= 1.0e-9,
            label + " persisted=" + values[index] + " expected=" + expected
        );
    }

    private static void assertStable(
        GeoWorldEngineBridge.GeneratedModel first
    ) throws Exception {
        GeoWorldEngineBridge.GeneratedModel second =
            GeoWorldEngineBridge.generate(first.seed(), first.config());
        require(
            second.cached(),
            "repeat generation did not resolve the identical parameter snapshot from cache"
        );
        require(
            first.model().manifest().payloadSha256().equals(
                second.model().manifest().payloadSha256()
            ),
            "identical Seed/parameter snapshot produced a different payload"
        );
    }

    private static Stats scan(GeoWorldV1 model) {
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double minX = geometry.originX();
        double minZ = geometry.originZ();
        double maxX = minX + geometry.spanXBlocks();
        double maxZ = minZ + geometry.spanZBlocks();

        int riverCells = 0;
        int lakeCells = 0;
        List<Double> landHeights = new ArrayList<>();
        List<Double> mountainHeights = new ArrayList<>();
        int xCount = (int)Math.floor((maxX - minX) / SAMPLE_STEP) + 1;
        int zCount = (int)Math.floor((maxZ - minZ) / SAMPLE_STEP) + 1;
        double[] sampledHeights = new double[xCount * zCount];
        boolean[] mountainMask = new boolean[xCount * zCount];
        double mountainReliefSum = 0.0;
        int mountainReliefPairs = 0;
        int iz = 0;
        for (double z = minZ; z <= maxZ + 1.0e-9; z += SAMPLE_STEP, iz++) {
            int ix = 0;
            for (double x = minX; x <= maxX + 1.0e-9; x += SAMPLE_STEP, ix++) {
                GeoWorldV1.RuntimeSample sample = model.sample(x, z);
                int index = iz * xCount + ix;
                sampledHeights[index] = sample.groundHeight();
                boolean mountain =
                    sample.sourceLand() &&
                    ("Mountain".equals(sample.terrainVariant()) ||
                        "Foothills".equals(sample.terrainVariant()));
                mountainMask[index] = mountain;
                if (sample.sourceLand()) {
                    landHeights.add(sample.groundHeight());
                }
                if (mountain) {
                    mountainHeights.add(sample.groundHeight());
                    if (ix > 0 && mountainMask[index - 1]) {
                        mountainReliefSum += Math.abs(
                            sample.groundHeight() - sampledHeights[index - 1]
                        );
                        mountainReliefPairs++;
                    }
                    if (iz > 0 && mountainMask[index - xCount]) {
                        mountainReliefSum += Math.abs(
                            sample.groundHeight() - sampledHeights[index - xCount]
                        );
                        mountainReliefPairs++;
                    }
                }
                if ("River".equals(sample.waterKind())) {
                    riverCells++;
                } else if ("Lake".equals(sample.waterKind())) {
                    lakeCells++;
                }
            }
        }
        require(
            landHeights.size() >= 100,
            "parameter-effect scan found too little land: " + landHeights.size()
        );
        require(
            !mountainHeights.isEmpty() && mountainReliefPairs > 0,
            "parameter-effect scan found no usable mountain region"
        );
        Collections.sort(landHeights);
        int topCount = Math.max(16, landHeights.size() / 8);
        double sum = 0.0;
        for (int i = landHeights.size() - topCount; i < landHeights.size(); i++) {
            sum += landHeights.get(i);
        }
        double mountainSum = 0.0;
        for (double height : mountainHeights) {
            mountainSum += height;
        }
        return new Stats(
            riverCells,
            lakeCells,
            sum / topCount,
            landHeights.size(),
            mountainSum / mountainHeights.size(),
            mountainReliefSum / mountainReliefPairs,
            mountainHeights.size()
        );
    }

    private static double meanAbsoluteGroundDifference(
        GeoWorldV1 low,
        GeoWorldV1 high
    ) {
        GeoWorldV1.Geometry geometry = low.manifest().geometry();
        double minX = geometry.originX();
        double minZ = geometry.originZ();
        double maxX = minX + geometry.spanXBlocks();
        double maxZ = minZ + geometry.spanZBlocks();
        double sum = 0.0;
        int count = 0;
        for (double z = minZ; z <= maxZ + 1.0e-9; z += SAMPLE_STEP) {
            for (double x = minX; x <= maxX + 1.0e-9; x += SAMPLE_STEP) {
                GeoWorldV1.RuntimeSample a = low.sample(x, z);
                GeoWorldV1.RuntimeSample b = high.sample(x, z);
                if (!a.sourceLand() || !b.sourceLand()) {
                    continue;
                }
                sum += Math.abs(a.groundHeight() - b.groundHeight());
                count++;
            }
        }
        require(count >= 100, "parameter-effect terrain scan has too few shared land samples");
        return sum / count;
    }

    private static double meanAbsolutePrecipitationDifference(
        GeoWorldV1 low,
        GeoWorldV1 high
    ) {
        GeoWorldV1.Geometry geometry = low.manifest().geometry();
        double minX = geometry.originX();
        double minZ = geometry.originZ();
        double maxX = minX + geometry.spanXBlocks();
        double maxZ = minZ + geometry.spanZBlocks();
        double sum = 0.0;
        int count = 0;
        for (double z = minZ; z <= maxZ + 1.0e-9; z += SAMPLE_STEP) {
            for (double x = minX; x <= maxX + 1.0e-9; x += SAMPLE_STEP) {
                GeoWorldV1.ClimateSample a = low.sampleClimate(x, z);
                GeoWorldV1.ClimateSample b = high.sampleClimate(x, z);
                if (!a.available() || !b.available()) {
                    continue;
                }
                sum += Math.abs(a.annualPrecipMM() - b.annualPrecipMM());
                count++;
            }
        }
        require(count >= 100, "parameter-effect climate scan has too few samples");
        return sum / count;
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    private record Stats(
        int riverCells,
        int lakeCells,
        double topLandMean,
        int landCells,
        double mountainMean,
        double mountainLocalRelief,
        int mountainCells
    ) {}

    private record EffectResult(
        String name,
        double lowMetric,
        double highMetric,
        double delta
    ) {
        String summary() {
            return String.format(
                java.util.Locale.ROOT,
                "%s[low=%.4f,high=%.4f,delta=%.4f]",
                name,
                lowMetric,
                highMetric,
                delta
            );
        }
    }
}
