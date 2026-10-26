package geoworld.runtime;

import java.util.HashSet;
import java.util.Set;

public final class GeoWorldAcceptanceRegistryVerifier {
    private GeoWorldAcceptanceRegistryVerifier() {}

    public static void main(String[] args) {
        var entries = GeoWorldAcceptanceRegistry.entries();
        if (entries.size() != 177) {
            throw new AssertionError(
                "acceptance registry size=" + entries.size() + " want=177"
            );
        }
        Set<String> ids = new HashSet<>();
        for (var entry : entries) {
            if (!ids.add(entry.id())) {
                throw new AssertionError(
                    "duplicate acceptance ID " + entry.id()
                );
            }
        }
        String[] required = {
            "CORE-01", "RIVER-04", "FLOOD-01", "RARE-01",
            "MODEL-08", "PROFILE-08", "PARAM-09", "CLIMATE-08"
        };
        for (String id : required) {
            GeoWorldAcceptanceRegistry.require(id);
        }
        System.out.printf(
            "GeoWorld Acceptance Registry PASS: version=%s entries=%d%n",
            GeoWorldAcceptanceRegistry.VERSION,
            entries.size()
        );
    }
}
