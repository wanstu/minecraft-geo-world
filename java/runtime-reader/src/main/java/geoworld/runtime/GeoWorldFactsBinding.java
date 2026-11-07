package geoworld.runtime;

import geoworld.runtime.GeoWorldQueryService.NaturalField;
import geoworld.runtime.GeoWorldQueryService.NaturalResult;
import geoworld.runtime.GeoWorldQueryService.QueryRequest;
import geoworld.runtime.GeoWorldQueryService.Resolution;
import geoworld.runtime.api.WorldFacts;

import java.lang.ref.WeakReference;
import java.util.EnumSet;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Set;
import java.util.WeakHashMap;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Internal adapter lifecycle hook, deliberately outside the stable API package.
 * Only the server WorldService binds models; consumers use GeoWorldFacade.
 */
public final class GeoWorldFactsBinding {
    private static final Map<Object, WorldFacts> BINDINGS = new WeakHashMap<>();
    private static final AtomicReference<Binding> CURRENT = new AtomicReference<>();
    private record Binding(WeakReference<Object> owner, WorldFacts facts) {}

    private GeoWorldFactsBinding() {}

    public static void bind(Object worldIdentity, GeoWorldV1 model) {
        Objects.requireNonNull(worldIdentity, "worldIdentity");
        Objects.requireNonNull(model, "model");
        // Prepare before publishing. currentWorld never performs this probe,
        // and callers cannot observe a partly initialized support set.
        WorldFacts facts = new BoundWorldFacts(model);
        synchronized (BINDINGS) {
            BINDINGS.put(worldIdentity, facts);
            CURRENT.set(new Binding(new WeakReference<>(worldIdentity), facts));
        }
    }

    public static void unbind(Object worldIdentity) {
        if (worldIdentity == null) return;
        synchronized (BINDINGS) {
            BINDINGS.remove(worldIdentity);
            Binding current = CURRENT.get();
            if (current != null && current.owner.get() == worldIdentity) {
                CURRENT.compareAndSet(current, null);
            }
        }
    }

    public static Optional<WorldFacts> currentWorld() {
        Binding current = CURRENT.get();
        if (current == null) return Optional.empty();
        if (current.owner.get() == null) {
            CURRENT.compareAndSet(current, null);
            return Optional.empty();
        }
        return Optional.of(current.facts);
    }

    private static final class BoundWorldFacts implements WorldFacts {
        private final GeoWorldQueryService query;
        private final Set<NaturalField> supported;
        private final String profileVersion;
        private final String rulesVersion;
        private final String modelVersion;

        private BoundWorldFacts(GeoWorldV1 model) {
            // Avoid the legacy shared cache: this query lives only as long as
            // its binding or an explicitly retained immutable view.
            query = new GeoWorldQueryService(model);
            var profile = model.worldProfile();
            profileVersion = profile.worldProfileVersion();
            rulesVersion = profile.generationRulesVersion();
            modelVersion = profile.worldModelVersion();
            var geometry = model.manifest().geometry();
            // All other v1 fields are unconditional, or legitimately missing
            // at a point (e.g. dry-land water surface / no nearby river).
            // Probe only model-wide optional groups, avoiding regional
            // aggregation and nearest-feature scans on the server thread.
            Set<NaturalField> optionalGroups = EnumSet.of(
                NaturalField.ANNUAL_MEAN_TEMP_C, NaturalField.ANNUAL_PRECIP_MM,
                NaturalField.MONSOON_INDEX, NaturalField.WIND_X, NaturalField.WIND_Z,
                NaturalField.FLOW_ACCUMULATION, NaturalField.RIVER_ORDER
            );
            NaturalResult probe = query.queryNatural(new QueryRequest(
                geometry.originX() + geometry.spanXBlocks() * 0.5,
                geometry.originZ() + geometry.spanZBlocks() * 0.5,
                Resolution.BLOCK_16, optionalGroups
            ));
            EnumSet<NaturalField> fields = EnumSet.allOf(NaturalField.class);
            fields.removeAll(probe.unavailableFields());
            supported = Set.copyOf(fields);
        }

        @Override public NaturalResult queryNatural(QueryRequest request) {
            return query.queryNatural(request);
        }
        @Override public boolean supports(NaturalField field) {
            return field != null && supported.contains(field);
        }
        @Override public String worldProfileVersion() { return profileVersion; }
        @Override public String generationRulesVersion() { return rulesVersion; }
        @Override public String worldModelVersion() { return modelVersion; }
    }
}
