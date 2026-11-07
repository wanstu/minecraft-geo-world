import geoworld.runtime.GeoWorldV1;
import geoworld.minecraft.common.worldgen.GeoWorldMiningDistrictPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;

import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;

/**
 * Read-only acceptance helper; reads a locked model, not Minecraft chunks.
 * Java 21+: java -cp <GeoWorld.jar> --source 21 MiningTargets.java <model> [family]
 * This tool uses internal model classes and is not part of the stable Query API.
 */
public final class MiningTargets {
    private MiningTargets() {}

    public static void main(String[] args) throws Exception {
        if (args.length < 1 || args.length > 2) {
            throw new IllegalArgumentException("expected <model.geoworld> [mineral-family]");
        }
        GeoWorldV1 model = GeoWorldV1.load(Path.of(args[0]));
        var heights = new MinecraftV26HeightMapping();
        var planner = new GeoWorldMiningDistrictPlanner(heights);
        var shoreline = new GeoWorldShorelinePlanner(heights);
        var geometry = model.manifest().geometry();
        List<GeoWorldMiningDistrictPlanner.District> districts = new ArrayList<>();
        planner.planDistricts(
            model, model.manifest().seed(),
            (int)geometry.originX(), (int)(geometry.originX() + geometry.spanXBlocks()),
            (int)geometry.originZ(), (int)(geometry.originZ() + geometry.spanZBlocks()),
            district -> {
                if (model.mineralDistrictAssignment(district.id()) != null) {
                    districts.add(district);
                }
            }
        );
        String family = args.length == 2 ? args[1] : "zinc";
        Comparator<GeoWorldMiningDistrictPlanner.District> nearest =
            Comparator.<GeoWorldMiningDistrictPlanner.District>comparingInt(
                district -> "None".equals(model.sample(
                    district.centerX(), district.centerZ()).waterKind()) ? 0 : 1
            ).thenComparingDouble(district ->
                district.centerX() * district.centerX() +
                district.centerZ() * district.centerZ());
        var targets = new LinkedHashMap<String, GeoWorldMiningDistrictPlanner.District>();
        districts.stream().filter(district ->
            family.equals(model.mineralDistrictAssignment(district.id()).primary())
        ).min(nearest).ifPresent(district -> targets.put("mod-primary", district));
        districts.stream().filter(district ->
            family.equals(model.mineralDistrictAssignment(district.id()).companion())
        ).min(nearest).ifPresent(district -> targets.put("mod-companion", district));
        districts.stream().filter(district -> !"None".equals(
            model.sample(district.centerX(), district.centerZ()).waterKind()
        )).min(Comparator.comparingDouble(district ->
            district.centerX() * district.centerX() +
            district.centerZ() * district.centerZ()
        )).ifPresent(district -> targets.put("water-floor", district));

        System.out.printf(Locale.ROOT,
            "{\"seed\":\"%d\",\"rules\":\"%s\",\"surfaceOrePile\":%s," +
            "\"districts\":%d,\"source\":\"locked_model\",\"targets\":[",
            model.manifest().seed(), escape(model.manifest().generationRulesVersion()),
            model.surfaceOrePileEnabled(), districts.size());
        boolean comma = false;
        for (var entry : targets.entrySet()) {
            var district = entry.getValue();
            var assignment = model.mineralDistrictAssignment(district.id());
            int x = (int)Math.round(district.centerX());
            int z = (int)Math.round(district.centerZ());
            var column = shoreline.planModel(model, x, z);
            int viewY = Math.min(heights.maxYExclusive() - 8, column.groundBoundary() + 12);
            if (comma) System.out.print(',');
            comma = true;
            System.out.printf(Locale.ROOT,
                "{\"case\":\"%s\",\"districtId\":\"%d\",\"primary\":\"%s\"," +
                "\"companion\":\"%s\",\"x\":%d,\"floorY\":%d,\"z\":%d," +
                "\"water\":\"%s\",\"waterY\":%d,\"tp\":\"/tp @s %d %d %d\"}",
                entry.getKey(), district.id(), escape(assignment.primary()),
                escape(assignment.companion()), x, column.groundBoundary() - 1, z,
                escape(column.sample().waterKind()), column.waterBoundary() - 1, x, viewY, z);
        }
        System.out.println("]}");
    }

    private static String escape(String value) {
        return value.replace("\\", "\\\\").replace("\"", "\\\"");
    }
}
