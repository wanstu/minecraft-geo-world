package geoworld.runtime.api;

import geoworld.runtime.GeoWorldQueryService.NaturalField;
import geoworld.runtime.GeoWorldQueryService.NaturalResult;
import geoworld.runtime.GeoWorldQueryService.QueryRequest;

/**
 * Immutable view of one locked Overworld model on the logical server.
 * Results describe natural facts, not loaded blocks or current gameplay state.
 * Retaining this view does not make a closed world current again.
 *
 * The existing QueryRequest, NaturalResult, NaturalField and Resolution value
 * types are part of the v1 contract; the query service itself is internal.
 */
public interface WorldFacts {
    NaturalResult queryNatural(QueryRequest request);
    boolean supports(NaturalField field);
    String worldProfileVersion();
    String generationRulesVersion();
    String worldModelVersion();
}
