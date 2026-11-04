package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldMiningDistrictPlanner;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.Holder;
import net.minecraft.core.Registry;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.tags.TagKey;
import net.minecraft.world.level.block.Block;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;

import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * Loader-neutral common-ore-tag catalog.
 *
 * NeoForge, Fabric and modern datapacks converge on the c:ores/<family>
 * convention. GeoWorld never guesses ore blocks from names: a block only joins
 * a mining district when the game registry actually binds it to a common ore
 * family and to a stone/deepslate ground tag.
 */
public final class GeoWorldOreCatalog {
    private static final List<String> VANILLA_FAMILIES = List.of(
        "coal", "iron", "copper", "gold",
        "lapis", "redstone", "diamond", "emerald"
    );
    private static final List<String> COMMON_DICTIONARY_FAMILIES = List.of(
        "aluminum", "cobalt", "iridium", "lead", "nickel",
        "osmium", "platinum", "silver", "sulfur", "tin",
        "titanium", "tungsten", "uranium", "zinc"
    );
    private static volatile Snapshot snapshot;

    private GeoWorldOreCatalog() {}

    public static BlockState resolve(
        GeoWorldV1 model,
        GeoWorldMiningDistrictPlanner.Placement placement,
        BlockState host,
        BlockState fallback
    ) {
        Snapshot current = snapshot();
        if (current.families().isEmpty()) {
            return fallback;
        }

        GeoWorldV1.MineralDistrictAssignment assignment =
            model.mineralDistrictAssignment(placement.districtId());
        String family;
        if (assignment != null) {
            family = placement.companion()
                ? assignment.companion()
                : assignment.primary();
        } else {
            // Legacy models did not persist mineral-family resolution.
            String base = placement.mineral()
                .name()
                .toLowerCase(Locale.ROOT);
            family = placement.companion()
                ? current.companionFamily(
                    placement.districtId(),
                    base
                )
                : current.primaryFamily(
                    placement.districtId(),
                    base
                );
        }
        Family ores = current.families().get(family);
        if (ores == null) {
            // A model-bound modded ore may have been removed after world
            // creation. Preserve the model identity: do not silently replace
            // it with a newly available mineral.
            return assignment == null ? fallback : host;
        }

        boolean deepslate = isDeepslateHost(host);
        List<Entry> candidates = deepslate
            ? ores.deepslate()
            : ores.stone();
        if (candidates.isEmpty()) {
            candidates = deepslate ? ores.stone() : ores.deepslate();
        }
        if (candidates.isEmpty()) {
            return fallback;
        }

        long salt = placement.districtId() ^
            (placement.companion()
                ? 0x434F4D50414E494FL
                : 0x5052494D4152594CL) ^
            family.hashCode();
        int index = Math.floorMod(mix32(salt), candidates.size());
        return candidates.get(index).block().defaultBlockState();
    }

    public static BlockState surfaceMarkerBlock(
        GeoWorldV1 model,
        long districtId,
        boolean companion
    ) {
        GeoWorldV1.MineralDistrictAssignment assignment =
            model.mineralDistrictAssignment(districtId);
        if (assignment == null) {
            return null;
        }
        String family = companion
            ? assignment.companion()
            : assignment.primary();
        Snapshot current = snapshot();
        Family ores = current.families().get(family);
        if (ores == null) {
            return null;
        }
        List<Entry> candidates = !ores.stone().isEmpty()
            ? ores.stone()
            : ores.deepslate();
        if (candidates.isEmpty()) {
            return null;
        }
        long salt = districtId ^
            (companion
                ? 0x53555246434F4D50L
                : 0x535552465052494DL);
        int index = Math.floorMod(mix32(salt), candidates.size());
        return candidates.get(index).block().defaultBlockState();
    }

    public static Map<String, String> previewBlockIds() {
        Snapshot current = snapshot();
        Map<String, String> out = new LinkedHashMap<>();
        current.families().forEach((family, ores) -> {
            List<Entry> candidates = !ores.stone().isEmpty()
                ? ores.stone()
                : ores.deepslate();
            if (!candidates.isEmpty()) {
                out.put(family, candidates.get(0).id());
            }
        });
        return Map.copyOf(out);
    }

    public static List<String> dictionaryFamilies() {
        return snapshot().dictionaryFamilies();
    }

    public static List<String> families() {
        return snapshot().familyNames();
    }

    public static List<String> extraFamilies() {
        return snapshot().extraFamilies();
    }

    /**
     * Scans the block registry from the active world-creation context.
     * Datapack/common tags can already be bound there while BuiltInRegistries
     * still exposes the pre-reload tag view.
     */
    public static CatalogView scanWorldgenRegistry(Registry<Block> registry) {
        Snapshot scanned = scan(registry);
        return new CatalogView(
            scanned.dictionaryFamilies(),
            scanned.familyNames()
        );
    }

    public static void invalidate() {
        snapshot = null;
    }

    private static Snapshot snapshot() {
        Snapshot current = snapshot;
        if (current != null) {
            return current;
        }
        synchronized (GeoWorldOreCatalog.class) {
            current = snapshot;
            if (current == null) {
                current = scan(BuiltInRegistries.BLOCK);
                snapshot = current;
            }
            return current;
        }
    }

    private static Snapshot scan(Registry<Block> registry) {
        Map<String, MutableFamily> mutable = new HashMap<>();
        java.util.TreeSet<String> dictionary = new java.util.TreeSet<>(VANILLA_FAMILIES);
        dictionary.addAll(COMMON_DICTIONARY_FAMILIES);

        registry.stream().forEach(block -> {
            Holder<Block> holder = registry.wrapAsHolder(block);
            List<String> tags = holder.tags()
                .map(GeoWorldOreCatalog::tagId)
                .filter(id -> id != null && !id.isBlank())
                .toList();

            boolean stone =
                tags.contains("c:ores_in_ground/stone") ||
                tags.contains("forge:ores_in_ground/stone");
            boolean deepslate =
                tags.contains("c:ores_in_ground/deepslate") ||
                tags.contains("forge:ores_in_ground/deepslate");

            String blockId = registry
                .getKey(block)
                .toString();

            for (String tag : tags) {
                String family;
                if (tag.startsWith("c:ores/")) {
                    family = tag.substring("c:ores/".length());
                } else if (tag.startsWith("forge:ores/")) {
                    family = tag.substring("forge:ores/".length());
                } else {
                    continue;
                }
                if (
                    family.isBlank() ||
                    family.contains("/") ||
                    family.equals("in_ground")
                ) {
                    continue;
                }
                // Dictionary membership and runtime availability are separate.
                // A family may be known by the common ore dictionary even when
                // this world has no concrete stone/deepslate ore block for it.
                dictionary.add(family);
                if (!stone && !deepslate) {
                    continue;
                }
                MutableFamily target = mutable.computeIfAbsent(
                    family,
                    ignored -> new MutableFamily()
                );
                Entry entry = new Entry(block, blockId);
                if (stone) target.stone.add(entry);
                if (deepslate) target.deepslate.add(entry);
            }
        });

        Map<String, Family> families = new LinkedHashMap<>();
        mutable.entrySet().stream()
            .sorted(Map.Entry.comparingByKey())
            .forEach(entry -> {
                List<Entry> stone = entry.getValue().stone.stream()
                    .distinct()
                    .sorted(Comparator.comparing(Entry::id))
                    .toList();
                List<Entry> deepslate = entry.getValue().deepslate.stream()
                    .distinct()
                    .sorted(Comparator.comparing(Entry::id))
                    .toList();
                if (!stone.isEmpty() || !deepslate.isEmpty()) {
                    families.put(
                        entry.getKey(),
                        new Family(stone, deepslate)
                    );
                }
            });

        List<String> names = List.copyOf(families.keySet());
        List<String> extras = names.stream()
            .filter(name -> !VANILLA_FAMILIES.contains(name))
            .toList();
        return new Snapshot(
            Map.copyOf(families),
            List.copyOf(dictionary),
            names,
            extras
        );
    }

    private static String tagId(TagKey<?> tag) {
        for (String methodName : new String[]{"location", "identifier"}) {
            try {
                Method method = tag.getClass().getMethod(methodName);
                Object value = method.invoke(tag);
                if (value != null) return value.toString();
            } catch (ReflectiveOperationException ignored) {
            }
        }
        return "";
    }

    private static boolean isDeepslateHost(BlockState host) {
        String id = BuiltInRegistries.BLOCK
            .getKey(host.getBlock())
            .toString();
        return id.equals("minecraft:deepslate") ||
            id.contains("deepslate");
    }

    private static int mix32(long value) {
        long h = value;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;
        return (int)(h ^ (h >>> 32));
    }

    public record CatalogView(
        List<String> dictionaryFamilies,
        List<String> families
    ) {}

    private record Entry(Block block, String id) {}

    private record Family(
        List<Entry> stone,
        List<Entry> deepslate
    ) {}

    private record Snapshot(
        Map<String, Family> families,
        List<String> dictionaryFamilies,
        List<String> familyNames,
        List<String> extraFamilies
    ) {
        String primaryFamily(long districtId, String fallback) {
            if (usesExtraPrimary(districtId)) {
                return extraFamilies.get(
                    Math.floorMod(
                        mix32(districtId ^ 0x5052494D415259L),
                        extraFamilies.size()
                    )
                );
            }
            return families.containsKey(fallback)
                ? fallback
                : firstAvailableFamily(fallback);
        }

        String companionFamily(long districtId, String fallback) {
            // A district has exactly one primary family and one companion
            // family. If the district promotes a modded family to primary,
            // keep the planner's original companion as the secondary ore.
            // Otherwise a discovered modded family may become the companion.
            if (usesExtraPrimary(districtId)) {
                return families.containsKey(fallback)
                    ? fallback
                    : firstAvailableFamily(fallback);
            }
            if (!extraFamilies.isEmpty()) {
                return extraFamilies.get(
                    Math.floorMod(
                        mix32(districtId ^ 0x434F4D50414E494FL),
                        extraFamilies.size()
                    )
                );
            }
            return families.containsKey(fallback)
                ? fallback
                : firstAvailableFamily(fallback);
        }

        private boolean usesExtraPrimary(long districtId) {
            return !extraFamilies.isEmpty() &&
                Math.floorMod(
                    mix32(districtId ^ 0x4558545241505249L),
                    100
                ) < 32;
        }

        private String firstAvailableFamily(String fallback) {
            if (families.containsKey(fallback)) return fallback;
            return familyNames.isEmpty()
                ? fallback
                : familyNames.get(0);
        }
    }

    private static final class MutableFamily {
        final List<Entry> stone = new ArrayList<>();
        final List<Entry> deepslate = new ArrayList<>();
    }
}
