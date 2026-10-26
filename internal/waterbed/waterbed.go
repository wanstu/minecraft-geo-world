package waterbed

import (
	"errors"
	"fmt"
	"math"

	"geoworld/internal/hydrology"
	"geoworld/internal/reconcile"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

const (
	lakeShoreAllowanceNormalized     = 0.0015
	lakeCoreCarveAllowanceNormalized = 0.012
	navigationSurfaceStepNormalized  = 0.009
	// v4 ordinary rivers make the full hydraulic half-width geometry-owned.
	// The dry-bank transition already lives outside halfWidth in bankRadius;
	// keeping another dry strip inside the channel lets broad-terrain noise bite
	// chunks out of the visible river again.
	riverWetCoreFraction = 1.0
)

func Build(base *terrain.Model, rec *reconcile.Model, hydro *hydrology.Model, cfg Config) (*Model, error) {
	if base == nil {
		return nil, errors.New("waterbed: base terrain is required")
	}
	if rec == nil {
		return nil, errors.New("waterbed: reconciliation model is required")
	}
	if hydro == nil {
		return nil, errors.New("waterbed: hydrology model is required")
	}
	if _, err := base.Geometry.SquareCellSize(); err != nil {
		return nil, fmt.Errorf("waterbed: base geometry: %w", err)
	}
	if !sameGeometry(base.Geometry, rec.Geometry) ||
		!sameGeometry(base.Geometry, hydro.Geometry) ||
		hydro.Width != base.Geometry.Width ||
		hydro.Height != base.Geometry.Height {
		return nil, errors.New("waterbed: model geometries do not match")
	}
	n, err := world.NodeCount(base.Geometry.Width, base.Geometry.Height)
	if err != nil {
		return nil, err
	}
	if len(rec.RiverBedTarget) != n ||
		len(rec.RiverBaseElevation) != n ||
		len(rec.LakeBedCeilingDefined) != n ||
		len(rec.LakeBedCeiling) != n ||
		len(rec.WaterSurfaceDefined) != n ||
		len(rec.WaterSurfaceElevation) != n ||
		len(hydro.LakeID) != n {
		return nil, errors.New("waterbed: reconciliation fields do not match geometry")
	}

	cfg, err = normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	m := &Model{
		Config:       cfg,
		Geometry:     base.Geometry,
		Base:         base,
		riverBuckets: make([][]int, n),
		lakeBuckets:  make([][]int, n),
	}
	lakeSurfaces, err := buildLakeNodes(base, rec, hydro, m)
	if err != nil {
		return nil, err
	}

	type riverDraft struct {
		edge          reconcile.RiverEdgeConstraint
		x0, z0        float64
		x1, z1        float64
		halfWidth     float64
		meanderOffset float64
	}
	drafts := make([]riverDraft, len(rec.RiverEdges))
	outgoing := make([]int, n)
	indegree := make([]int, n)
	incomingSegments := make([][]int, n)
	for i := range outgoing {
		outgoing[i] = -1
	}
	for id, edge := range rec.RiverEdges {
		if edge.From < 0 || edge.From >= n || edge.To < 0 || edge.To >= n {
			return nil, errors.New("waterbed: river edge index out of range")
		}
		if outgoing[edge.From] >= 0 {
			return nil, errors.New("waterbed: river node has multiple downstream edges")
		}
		x0i, z0i := edge.From%base.Geometry.Width, edge.From/base.Geometry.Width
		x1i, z1i := edge.To%base.Geometry.Width, edge.To/base.Geometry.Width
		x0, z0 := base.Geometry.NodePosition(x0i, z0i)
		x1, z1 := base.Geometry.NodePosition(x1i, z1i)
		halfWidth := riverHalfWidth(
			cfg,
			rec.Config.RiverStrengthThreshold,
			edge.RiverStrength,
			edge.FlowAccumulation,
			hydro.RiverThreshold,
			edge.RiverOrder,
		)
		drafts[id] = riverDraft{
			edge: edge,
			x0:   x0, z0: z0, x1: x1, z1: z1,
			halfWidth: halfWidth,
			meanderOffset: riverMeanderOffset(
				base.Config.Seed,
				edge.From,
				edge.To,
				x0, z0,
				x1, z1,
				halfWidth,
				edge.OriginalDrop,
				cfg.RiverGeometryVersion,
			),
		}
		outgoing[edge.From] = id
		indegree[edge.To]++
	}
	for id := range drafts {
		if indegree[drafts[id].edge.To] > 1 {
			// Approaching branches must not swing across one another long before
			// their confluence. Keep a visible bend, but tighten the final
			// approach so sibling channels only begin touching near the junction.
			drafts[id].meanderOffset *= 0.35
		}
	}

	// A sampled upstream profile may become lower than its macro endpoint target
	// because the broad terrain itself contains a local dip between control
	// points. Propagate both the actual outlet bed and the actual outlet water
	// surface through the river DAG before constructing the downstream segment.
	// The surface constraint matters when a tributary enters a wider/deeper main
	// stem: bed-only continuity can still make bed+depth jump upward.
	effectiveBed := append([]float64(nil), rec.RiverBedTarget...)
	effectiveSurface := make([]float64, n)
	effectiveSurfaceDefined := make([]bool, n)
	queue := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if indegree[i] == 0 {
			queue = append(queue, i)
		}
	}
	processed := 0
	for head := 0; head < len(queue); head++ {
		node := queue[head]
		if len(incomingSegments[node]) > 1 &&
			effectiveSurfaceDefined[node] {
			minBed := harmonizeConfluenceProfiles(
				m.Rivers,
				incomingSegments[node],
				effectiveSurface[node],
				cfg,
			)
			if minBed < effectiveBed[node] {
				effectiveBed[node] = minBed
			}
		}
		edgeID := outgoing[node]
		if edgeID < 0 {
			continue
		}
		draft := drafts[edgeID]
		if len(incomingSegments[node]) > 0 {
			// A trunk must not become narrower again immediately downstream.
			// Preserve the widest incoming channel, and let real confluences grow
			// a little beyond it according to the combined cross-section.
			maxIncoming := 0.0
			sumSquares := 0.0
			for _, incomingID := range incomingSegments[node] {
				incomingWidth := m.Rivers[incomingID].HalfWidth
				maxIncoming = math.Max(maxIncoming, incomingWidth)
				sumSquares += incomingWidth * incomingWidth
			}
			requiredWidth := maxIncoming
			if len(incomingSegments[node]) > 1 {
				requiredWidth = math.Max(
					requiredWidth,
					0.86*math.Sqrt(sumSquares),
				)
			}
			draft.halfWidth = math.Max(draft.halfWidth, requiredWidth)
		}
		centerDepth := riverCenterDepth(cfg, draft.halfWidth)
		bed0 := effectiveBed[draft.edge.From]
		if lakeID := hydro.LakeID[draft.edge.From]; lakeID > 0 {
			if surface, ok := lakeSurfaces[lakeID]; ok {
				// An exorheic lake outlet begins at the actual rendered lake
				// datum, not at the much deeper lake bed.
				bed0 = math.Max(0, surface-centerDepth)
				effectiveSurfaceDefined[draft.edge.From] = true
				effectiveSurface[draft.edge.From] = surface
			}
		} else if effectiveSurfaceDefined[draft.edge.From] {
			// Wider downstream rivers carry deeper water. Lower their bed at
			// the junction so the resulting water surface never steps upward.
			bed0 = math.Min(
				bed0,
				math.Max(0, effectiveSurface[draft.edge.From]-centerDepth),
			)
		}
		oceanMouth := draft.edge.To >= 0 &&
			draft.edge.To < len(rec.WaterKind) &&
			rec.WaterKind[draft.edge.To] == reconcile.WaterOcean
		lakeMouth := draft.edge.To >= 0 &&
			draft.edge.To < len(hydro.LakeID) &&
			hydro.LakeID[draft.edge.To] > 0 &&
			hydro.LakeID[draft.edge.From] != hydro.LakeID[draft.edge.To]
		lakeOutlet := draft.edge.From >= 0 &&
			draft.edge.From < len(hydro.LakeID) &&
			hydro.LakeID[draft.edge.From] > 0 &&
			hydro.LakeID[draft.edge.From] != hydro.LakeID[draft.edge.To]
		mouthExpansionVersion := uint8(3)
		if cfg.RiverGeometryVersion >= 1 &&
			!oceanMouth &&
			!lakeMouth &&
			!lakeOutlet {
			mouthExpansionVersion = 4
			if cfg.RiverGeometryVersion >= 2 {
				mouthExpansionVersion = 5
			}
		}

		bed1 := math.Min(effectiveBed[draft.edge.To], bed0)
		widthT := riverWidthFraction(cfg, draft.halfWidth)
		profile := buildRiverProfile(
			base,
			draft.x0, draft.z0,
			draft.x1, draft.z1,
			bed0, bed1,
			draft.halfWidth,
			draft.meanderOffset,
			mouthExpansionVersion,
			cfg.RiverProfileStepBlocks,
			cfg.MinRiverBedDepthNormalized*(1+1.5*widthT),
		)
		rawProfileDrop := profile[0] - profile[len(profile)-1]
		profile = terraceRiverProfile(
			profile,
			cfg.RiverProfileStepBlocks,
			riverNavigationReachBlocks(draft.halfWidth),
		)
		if draft.halfWidth >= 8 && rawProfileDrop <= 0.006 {
			profile = snapNavigableRiverProfile(
				profile,
				cfg,
				draft.halfWidth,
				navigationSurfaceStepNormalized,
			)
		}
		if len(profile) == 0 {
			return nil, errors.New("waterbed: river profile is empty")
		}
		outlet := profile[len(profile)-1]
		if outlet < effectiveBed[draft.edge.To] {
			effectiveBed[draft.edge.To] = outlet
		}
		outletSurface := outlet + centerDepth
		if !effectiveSurfaceDefined[draft.edge.To] ||
			outletSurface < effectiveSurface[draft.edge.To] {
			effectiveSurfaceDefined[draft.edge.To] = true
			effectiveSurface[draft.edge.To] = outletSurface
		}
		lakeOutletSurface := 0.0
		lakeOutletSurfaceDefined := false
		if lakeOutlet {
			if surface, ok := lakeSurfaces[hydro.LakeID[draft.edge.From]]; ok && finite(surface) {
				lakeOutletSurface = surface
				lakeOutletSurfaceDefined = true
			}
		}
		segment := RiverSegment{
			From: draft.edge.From, To: draft.edge.To,
			X0: draft.x0, Z0: draft.z0, X1: draft.x1, Z1: draft.z1,
			Bed0:                     bed0,
			Bed1:                     effectiveBed[draft.edge.To],
			HalfWidth:                draft.halfWidth,
			MeanderOffset:            draft.meanderOffset,
			Profile:                  profile,
			OceanMouth:               oceanMouth,
			LakeMouth:                lakeMouth,
			LakeOutlet:               lakeOutlet,
			LakeOutletSurfaceDefined: lakeOutletSurfaceDefined,
			LakeOutletSurface:        lakeOutletSurface,
			MouthExpansionVersion:    mouthExpansionVersion,
		}
		id := len(m.Rivers)
		m.Rivers = append(m.Rivers, segment)
		m.indexRiver(id, segment)
		incomingSegments[draft.edge.To] = append(
			incomingSegments[draft.edge.To],
			id,
		)
		processed++

		indegree[draft.edge.To]--
		if indegree[draft.edge.To] == 0 {
			queue = append(queue, draft.edge.To)
		}
	}
	if processed != len(drafts) {
		return nil, errors.New("waterbed: river graph contains a cycle")
	}
	softenNavigableJunctionDrops(m.Rivers, cfg)
	connectFeedersToRenderedLake(
		m,
		hydro,
		cfg,
		lakeSurfaces,
		incomingSegments,
	)

	// Lakes were constructed and spill-capped before the river DAG so
	// lake outlets can begin exactly at the final rendered lake datum.
	for i := range m.Rivers {
		surface, ok, err := directReceiverSurface(
			m.Rivers[i].To,
			rec,
			hydro,
			lakeSurfaces,
		)
		if err != nil {
			return nil, err
		}
		if ok {
			m.Rivers[i].ReceiverSurfaceDefined = true
			m.Rivers[i].ReceiverSurface = surface
		}
		if spatial, hit := spatialLakeReceiverSurface(m, m.Rivers[i]); hit &&
			(!m.Rivers[i].ReceiverSurfaceDefined ||
				spatial > m.Rivers[i].ReceiverSurface) {
			// Final rendered geometry is authoritative for connectivity. If a
			// curved river actually passes through a rendered lake footprint,
			// the lake is hydraulically connected even when the coarse FlowTo
			// topology missed that intersection. Use the lake datum as a
			// backwater floor so boats never meet an upward water wall.
			m.Rivers[i].ReceiverSurfaceDefined = true
			m.Rivers[i].ReceiverSurface = spatial
		}
	}
	seedLakeOutletConfluenceBackwater(
		m.Rivers,
		incomingSegments,
		cfg,
	)
	easeReceiverMouthDrops(
		m.Rivers,
		incomingSegments,
		cfg,
	)
	harmonizeReceiverConfluences(
		m.Rivers,
		incomingSegments,
	)
	propagateReceiverBackwater(
		m.Rivers,
		incomingSegments,
		cfg,
	)
	easeReceiverOutflowDrops(m.Rivers, cfg)
	enforceSupportedRiverCascades(m.Rivers, cfg)
	harmonizeFinalConfluences(
		m.Rivers,
		incomingSegments,
		cfg,
	)
	// Final profile repairs can change Bed0/Bed1 enough to change the v3
	// LakeOutlet basin signal and therefore how much source-lake datum survives
	// to the downstream junction. Recompute that derived backwater only after
	// the longitudinal profiles have settled; otherwise Runtime can alternate
	// block-by-block between the stale lower junction datum and the now-higher
	// LakeOutlet taper.
	seedLakeOutletConfluenceBackwater(
		m.Rivers,
		incomingSegments,
		cfg,
	)
	harmonizeReceiverConfluences(
		m.Rivers,
		incomingSegments,
	)
	propagateReceiverBackwater(
		m.Rivers,
		incomingSegments,
		cfg,
	)
	easeReceiverOutflowDrops(m.Rivers, cfg)
	enforceSupportedRiverCascades(m.Rivers, cfg)
	harmonizeFinalConfluences(
		m.Rivers,
		incomingSegments,
		cfg,
	)

	return m, nil
}

func harmonizeFinalConfluences(
	rivers []RiverSegment,
	incomingSegments [][]int,
	cfg Config,
) {
	for _, ids := range incomingSegments {
		if len(ids) < 2 {
			continue
		}
		sharedSurface := math.Inf(1)
		valid := 0
		for _, id := range ids {
			if id < 0 || id >= len(rivers) {
				continue
			}
			seg := rivers[id]
			if len(seg.Profile) == 0 {
				continue
			}
			surface :=
				seg.Profile[len(seg.Profile)-1] +
					riverCenterDepth(cfg, seg.HalfWidth)
			if seg.ReceiverSurfaceDefined &&
				seg.ReceiverSurface > surface {
				surface = seg.ReceiverSurface
			}
			if !finite(surface) {
				continue
			}
			sharedSurface = math.Min(sharedSurface, surface)
			valid++
		}
		if valid < 2 || !finite(sharedSurface) {
			continue
		}
		harmonizeConfluenceProfiles(
			rivers,
			ids,
			sharedSurface,
			cfg,
		)
	}
}

func enforceSupportedRiverCascades(
	rivers []RiverSegment,
	cfg Config,
) {
	if len(rivers) == 0 {
		return
	}
	incoming := make(map[int][]int, len(rivers))
	for id := range rivers {
		incoming[rivers[id].To] = append(
			incoming[rivers[id].To],
			id,
		)
	}

	queue := make([]int, len(rivers))
	queued := make([]bool, len(rivers))
	for id := range rivers {
		queue[id] = id
		queued[id] = true
	}
	for head := 0; head < len(queue); head++ {
		id := queue[head]
		queued[id] = false
		seg := &rivers[id]
		if len(seg.Profile) < 2 {
			continue
		}
		changed := enforceSegmentCascadeSlope(seg)
		if changed {
			seg.Bed0 = seg.Profile[0]
			seg.Bed1 = seg.Profile[len(seg.Profile)-1]
		}

		startSurface :=
			seg.Profile[0] +
				riverCenterDepth(cfg, seg.HalfWidth)
		for _, upstreamID := range incoming[seg.From] {
			if upstreamID < 0 || upstreamID >= len(rivers) {
				continue
			}
			upstream := &rivers[upstreamID]
			if len(upstream.Profile) == 0 {
				continue
			}
			upstreamDepth :=
				riverCenterDepth(cfg, upstream.HalfWidth)
			targetBed := math.Max(
				0,
				startSurface-upstreamDepth,
			)
			last := len(upstream.Profile) - 1
			if upstream.Profile[last] <= targetBed+1e-12 {
				continue
			}
			upstream.Profile[last] = targetBed
			upstream.Bed1 = targetBed
			if !queued[upstreamID] {
				queue = append(queue, upstreamID)
				queued[upstreamID] = true
			}
		}
	}
}

func enforceSegmentCascadeSlope(
	seg *RiverSegment,
) bool {
	if seg == nil || len(seg.Profile) < 2 {
		return false
	}
	chord := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
	if chord <= 1e-9 {
		return false
	}
	sampleBlocks := chord / float64(len(seg.Profile)-1)
	// Above normalized 0.5 the Minecraft vertical mapping is 514 blocks per
	// normalized unit. Use that stricter scale everywhere and keep a margin
	// below one vertical block per horizontal block. Therefore any final
	// source-water step has solid riverbed/cliff support instead of a free
	// vertical face.
	maxDrop := 0.70 * sampleBlocks / 514.0
	if maxDrop <= 0 {
		return false
	}

	changed := false
	for j := len(seg.Profile) - 1; j > 0; j-- {
		maxUpstream := seg.Profile[j] + maxDrop
		if seg.Profile[j-1] > maxUpstream+1e-12 {
			seg.Profile[j-1] = maxUpstream
			changed = true
		}
	}
	for j := 1; j < len(seg.Profile); j++ {
		if seg.Profile[j] > seg.Profile[j-1] {
			seg.Profile[j] = seg.Profile[j-1]
			changed = true
		}
	}
	return changed
}

func rareMacroLakeRadiusScale(
	seed int64,
	strength float64,
	lake hydrology.Lake,
) float64 {
	strength = clamp(strength, 0, 2.5)
	if strength <= 0 || lake.AreaBlocks2 < 550_000 {
		return 1
	}

	areaWeight := smoothstep(
		550_000,
		1_500_000,
		lake.AreaBlocks2,
	)
	guaranteed := smoothstep(
		1_000_000,
		1_500_000,
		lake.AreaBlocks2,
	)

	// Medium-large basins only occasionally become a macro landmark. Truly
	// exceptional natural basins increasingly bypass that random gate so a
	// world can still contain an inland-sea-scale lake without a second
	// artificial lake generator.
	h := uint64(seed) ^
		uint64(int64(lake.BasinID))*0x9E3779B185EBCA87 ^
		uint64(int64(lake.ID))*0xC2B2AE3D27D4EB4F ^
		uint64(int64(lake.SpillOutlet))*0x165667B19E3779F9
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	const inv53 = 1.0 / 9007199254740992.0
	u := float64(h>>11) * inv53
	randomSelection :=
		smoothstep(0.78, 0.96, u) * areaWeight
	selected := math.Max(guaranteed, randomSelection)
	if selected <= 0 {
		return 1
	}

	return 1 +
		selected*
			strength*
			(1.8+0.8*areaWeight)
}

func buildLakeNodes(
	base *terrain.Model,
	rec *reconcile.Model,
	hydro *hydrology.Model,
	m *Model,
) (map[int]float64, error) {
	spacing := base.Geometry.SpacingX()
	inner := spacing * 0.50
	lakeScale := make(map[int]float64, len(hydro.Lakes))
	for _, lake := range hydro.Lakes {
		lakeScale[lake.ID] = rareMacroLakeRadiusScale(
			base.Config.Seed,
			base.Config.RareMacroStrength,
			lake,
		)
	}
	// Lake cells are rendered with a rounded-square (L4/superellipse)
	// footprint rather than independent circles. This keeps D8-connected
	// hydrological cells continuous without exposing the raw macro-grid as
	// hard square shorelines.
	outer := spacing * 0.72
	for i, defined := range rec.LakeBedCeilingDefined {
		if !defined {
			continue
		}
		lakeID := hydro.LakeID[i]
		if lakeID <= 0 || !rec.WaterSurfaceDefined[i] {
			return nil, errors.New(
				"waterbed: lake bed constraint is missing lake water metadata",
			)
		}
		xi, zi := i%base.Geometry.Width, i/base.Geometry.Width
		x, z := base.Geometry.NodePosition(xi, zi)
		scale := lakeScale[lakeID]
		if scale <= 0 {
			scale = 1
		}
		node := LakeNode{
			Index:  i,
			LakeID: lakeID,
			X:      x, Z: z,
			BedCeiling:   rec.LakeBedCeiling[i],
			InnerRadius:  inner * scale,
			OuterRadius:  outer * scale,
			WaterSurface: rec.WaterSurfaceElevation[i],
		}
		id := len(m.Lakes)
		m.Lakes = append(m.Lakes, node)
		m.indexLake(id, node)
	}

	// Macro hydrology finds closed basins at 128-block control points, while
	// detailed terrain adds sub-cell relief afterwards. Recheck every lake
	// against the actual detailed union rim and lower the datum when a lower
	// spill route exists.
	capLakeSurfacesToDetailedRims(base, m.Lakes)
	if m.Config.LakeGeometryVersion >= 1 {
		for i := range m.Lakes {
			m.Lakes[i].BedCeiling = visibleLakeBedCeiling(
				m.Config,
				m.Lakes[i].WaterSurface,
				m.Lakes[i].BedCeiling,
			)
		}
		markLakeCoreCarveAnchors(
			m.Config,
			m.Lakes,
			base.SampleNormalized,
		)
	}

	lakeSurfaces := make(map[int]float64)
	for _, lake := range m.Lakes {
		if old, ok := lakeSurfaces[lake.LakeID]; !ok ||
			lake.WaterSurface < old {
			lakeSurfaces[lake.LakeID] = lake.WaterSurface
		}
	}
	return lakeSurfaces, nil
}

func visibleLakeBedCeiling(cfg Config, waterSurface, bedCeiling float64) float64 {
	if cfg.LakeGeometryVersion < 1 {
		return bedCeiling
	}
	// Runtime exposes a Lake only below waterSurface-MinLakeWaterDepth. Keep a
	// little extra depth so equality/rounding cannot turn a persisted lake back
	// into a dry macro candidate after reopen.
	visibleDepth := math.Max(
		0.004,
		cfg.MinLakeWaterDepthNormalized*1.10,
	)
	return math.Min(
		bedCeiling,
		math.Max(0, waterSurface-visibleDepth),
	)
}

func markLakeCoreCarveAnchors(
	cfg Config,
	lakes []LakeNode,
	sampleHeight func(worldX, worldZ float64) float64,
) {
	if cfg.LakeGeometryVersion < 1 || len(lakes) == 0 || sampleHeight == nil {
		return
	}
	groups := make(map[int][]int)
	for i := range lakes {
		lakes[i].CoreCarve = false
		groups[lakes[i].LakeID] = append(groups[lakes[i].LakeID], i)
	}
	for _, ids := range groups {
		// Do not force a core when detailed terrain already leaves any persisted
		// lake node naturally renderable. The anchor exists only to reconcile a
		// completely dry macro lake with detailed terrain.
		naturallyRenderable := false
		bestID := -1
		bestExcess := math.Inf(1)
		for _, id := range ids {
			node := lakes[id]
			h := sampleHeight(node.X, node.Z)
			if !finite(h) {
				continue
			}
			excess := h - node.WaterSurface
			if excess <= lakeShoreAllowanceNormalized {
				naturallyRenderable = true
				break
			}
			if excess < bestExcess {
				bestExcess = excess
				bestID = id
			}
		}
		if naturallyRenderable || bestID < 0 {
			continue
		}
		if bestExcess <= lakeCoreCarveAllowanceNormalized {
			lakes[bestID].CoreCarve = true
		}
	}
}

func capLakeSurfacesToDetailedRims(base *terrain.Model, lakes []LakeNode) {
	if base == nil || len(lakes) == 0 {
		return
	}
	groups := make(map[int][]int)
	for i := range lakes {
		groups[lakes[i].LakeID] = append(groups[lakes[i].LakeID], i)
	}

	spacing := base.Geometry.SpacingX()
	rimOffset := math.Max(2, spacing*0.025)
	const angularSamples = 48
	const bankClearance = 0.0007
	const minLakeDepth = 0.004

	for _, ids := range groups {
		minRim := math.Inf(1)
		for _, id := range ids {
			node := lakes[id]
			radius := node.OuterRadius + rimOffset
			for j := 0; j < angularSamples; j++ {
				angle := 2 * math.Pi * float64(j) / angularSamples
				ox, oz := lakeBoundaryOffset(angle, radius)
				x := node.X + ox
				z := node.Z + oz

				// Only inspect exposed portions of the union boundary. A sample
				// still covered by a sibling node belongs to the lake interior.
				insideSibling := false
				for _, otherID := range ids {
					other := lakes[otherID]
					if lakeDistance(x-other.X, z-other.Z) < other.OuterRadius {
						insideSibling = true
						break
					}
				}
				if insideSibling {
					continue
				}
				h := base.SampleNormalized(x, z)
				if h < minRim {
					minRim = h
				}
			}
		}
		if !finite(minRim) {
			continue
		}

		for _, id := range ids {
			node := &lakes[id]
			cappedSurface := math.Min(
				node.WaterSurface,
				math.Max(0, minRim-bankClearance),
			)
			if cappedSurface >= node.WaterSurface-1e-12 {
				continue
			}
			node.WaterSurface = cappedSurface
			maxBed := math.Max(0, cappedSurface-minLakeDepth)
			if node.BedCeiling > maxBed {
				node.BedCeiling = maxBed
			}
		}
	}
}

func (m *Model) SampleNormalized(worldX, worldZ float64) float64 {
	return m.Sample(worldX, worldZ).FinalHeight
}

func (m *Model) Sample(worldX, worldZ float64) Sample {
	baseHeight := m.Base.SampleNormalized(worldX, worldZ)
	finalHeight := baseHeight
	sample := Sample{
		BaseHeight:  baseHeight,
		FinalHeight: baseHeight,
		WaterKind:   reconcile.WaterNone,
	}

	bucket := m.bucketIndexForWorld(worldX, worldZ)
	if bucket < 0 {
		return sample
	}

	riverIDs := m.riverBuckets[bucket]
	activeRiverBankProtected := riverBankProtectedAt(
		m.Rivers,
		riverIDs,
		worldX,
		worldZ,
	)

	riverDatumWeight := -1.0
	riverDatumT := 2.0
	geometryWetOwned := false
	bestOxbowSurface := math.Inf(-1)
	for _, id := range riverIDs {
		seg := m.Rivers[id]
		t, distance := pointRiverSegment(worldX, worldZ, seg)
		halfWidth := riverMouthHalfWidth(seg, t)
		bankRadius := halfWidth + riverBankBlendBlocks(seg.HalfWidth)
		fluvialReach := math.Max(bankRadius, riverFluvialReach(seg))
		_, deltaDistance, deltaHalfWidth, deltaBankRadius, deltaOK :=
			riverDeltaPoint(seg, t, worldX, worldZ)
		deltaInBank := deltaOK && deltaDistance < deltaBankRadius
		deltaInWater := deltaOK && deltaDistance < deltaHalfWidth
		if distance >= fluvialReach && !deltaInBank {
			continue
		}
		bed := sampleRiverProfile(seg, t)
		naturalSurface := bed + riverCenterDepth(m.Config, seg.HalfWidth)
		surface := riverConnectionSurface(seg, t, naturalSurface)

		if deltaInBank && !deltaInWater {
			// Delta distributaries own a real dry bank corridor too. This keeps
			// each branch connected to the surrounding floodplain without turning
			// the split into a water trench cut through untouched terrain.
			innerBank := math.Min(baseHeight, surface+0.0018)
			bankWeight := 1 - smoothstep(
				deltaHalfWidth,
				deltaBankRadius,
				deltaDistance,
			)
			candidate := lerp(baseHeight, innerBank, bankWeight)
			if candidate < finalHeight {
				finalHeight = candidate
				sample.RiverCarve = math.Max(
					sample.RiverCarve,
					baseHeight-candidate,
				)
			}
		}

		if distance >= halfWidth && !deltaInWater {
			// Shape a short dry river valley around the navigable channel.
			// The river is the local low point; terrain rises smoothly back to
			// the untouched broad surface instead of meeting water as a noisy
			// staircase or vertical trench wall.
			if distance < bankRadius {
				innerBank := math.Min(baseHeight, surface+0.0022)
				bankWeight := 1 - smoothstep(
					halfWidth,
					bankRadius,
					distance,
				)
				candidate := lerp(baseHeight, innerBank, bankWeight)
				if candidate < finalHeight {
					finalHeight = candidate
					sample.RiverCarve = math.Max(
						sample.RiverCarve,
						baseHeight-candidate,
					)
				}
			}

			// Floodplain, abandoned-meander and oxbow detail belongs outside
			// every active river-bank protection corridor, not merely outside
			// the parent segment's bank. At confluences/overlaps an old meander
			// from one segment must never excavate another segment's live bank.
			if !activeRiverBankProtected {
				// v1.1.7 broadens a small subset of wide, low-gradient reaches
				// into a floodplain/terrace envelope. This code existed before but
				// was not connected to Runtime sampling.
				finalHeight = applyRiverFluvialDryDetail(
					seg,
					t,
					worldX,
					worldZ,
					baseHeight,
					finalHeight,
					surface,
					distance,
				)

				// A still rarer subset of those abandoned meanders retains shallow,
				// flat water. Record the candidate here but do not assign ownership
				// until every active river has been checked; a real river always wins.
				if oxbowSurface, oxbowFloor, oxbowWeight, ok :=
					riverOxbowWater(
						m.Config,
						seg,
						t,
						worldX,
						worldZ,
						distance,
					); ok && finalHeight <= oxbowSurface+0.010 {
					candidate := lerp(
						finalHeight,
						math.Min(finalHeight, oxbowFloor),
						oxbowWeight,
					)
					if candidate < finalHeight {
						finalHeight = candidate
						sample.LakeCarve = math.Max(
							sample.LakeCarve,
							baseHeight-candidate,
						)
					}
					if finalHeight <
						oxbowSurface-m.Config.MinLakeWaterDepthNormalized &&
						oxbowSurface > bestOxbowSurface {
						bestOxbowSurface = oxbowSurface
					}
				}
			}
			continue
		}

		channelHalfWidth := halfWidth
		channelDistance := distance
		weight := -1.0
		if distance < halfWidth {
			weight = 1 - smoothstep(0, halfWidth, distance)
		}
		if deltaInWater {
			deltaWeight := 1 - smoothstep(
				0,
				deltaHalfWidth,
				deltaDistance,
			)
			if deltaWeight > weight {
				weight = deltaWeight
				channelHalfWidth = deltaHalfWidth
				channelDistance = deltaDistance
			}
		}
		if weight <= 0 {
			continue
		}
		if weight > sample.RiverWeight {
			sample.RiverWeight = weight
		}
		// Water level should follow the channel that actually owns this point,
		// not a neighbouring branch whose bank merely overlaps here. Prefer the
		// strongest lateral weight. At an exact confluence centerline tie, the
		// downstream segment starts at t=0 while incoming segments end at t=1;
		// prefer that downstream owner.
		if !sample.RiverBedDefined ||
			weight > riverDatumWeight+1e-12 ||
			(math.Abs(weight-riverDatumWeight) <= 1e-12 &&
				(t < riverDatumT-1e-12 ||
					(math.Abs(t-riverDatumT) <= 1e-12 &&
						bed < sample.RiverBedElevation))) {
			sample.RiverBedDefined = true
			sample.RiverBedElevation = bed
			// Mouth flare changes the footprint, not the river's depth datum.
			sample.RiverHalfWidth = seg.HalfWidth
			sample.RiverReceiverSurfaceDefined = surface > naturalSurface+1e-12
			sample.RiverReceiverSurface = surface
			if sample.RiverReceiverSurfaceDefined {
				// The effective connection datum may be either downstream
				// backwater or the short lake-outlet source taper.
				sample.RiverReceiverWeight = 1
			} else {
				sample.RiverReceiverWeight = 0
			}
			riverDatumWeight = weight
			riverDatumT = t
		}
		channelShoulder := math.Min(baseHeight, surface+0.0022)
		if outletWeight := lakeOutletSourceWeight(seg, t); outletWeight > 0 {
			// A formal lake outlet must remain wet across its near-shore throat.
			// Blend the channel shoulder below the inherited lake surface, then
			// return to the ordinary dry-bank profile as the outlet weight fades.
			wetShoulder := math.Min(
				baseHeight,
				surface-m.Config.MinLakeWaterDepthNormalized,
			)
			channelShoulder = lerp(
				channelShoulder,
				wetShoulder,
				outletWeight,
			)
		}
		crossT := smoothstep(0, channelHalfWidth, channelDistance)
		if seg.MouthExpansionVersion >= 3 {
			// v1.1.3: great rivers keep a broad wet middle and rise only near
			// the bank. This makes the visible water width match the stored
			// hydraulic width instead of looking like a narrow channel in a
			// nominally large corridor.
			widthT := riverWidthFraction(m.Config, seg.HalfWidth)
			crossT = math.Pow(crossT, 1.65+0.95*widthT)
		} else if seg.MouthExpansionVersion >= 2 {
			widthT := riverWidthFraction(m.Config, seg.HalfWidth)
			crossT = math.Pow(crossT, 1.35+0.65*widthT)
		}
		candidate := math.Min(
			baseHeight,
			lerp(bed, channelShoulder, crossT),
		)
		v4WetCore := seg.MouthExpansionVersion >= 4 &&
			!seg.OceanMouth &&
			!seg.LakeMouth &&
			!seg.LakeOutlet
		insideWetCore :=
			!v4WetCore ||
				channelDistance <= channelHalfWidth*riverWetCoreFraction
		if v4WetCore && insideWetCore {
			geometryWetOwned = true
			// The visible bank is a geometric river boundary, not another
			// threshold against broad-terrain noise. Keep the full ordinary
			// hydraulic corridor at least half a center-depth below the surface;
			// the dry transition begins outside halfWidth in bankRadius.
			minimumWetDepth :=
				0.5 * riverCenterDepth(m.Config, seg.HalfWidth)
			candidate = math.Min(
				candidate,
				math.Max(0, surface-minimumWetDepth),
			)
		}
		if candidate < finalHeight {
			finalHeight = candidate
			sample.RiverCarve = math.Max(sample.RiverCarve, baseHeight-candidate)
		}
		if sample.WaterKind == reconcile.WaterNone {
			// The whole hydraulic corridor remains a river candidate. The v4
			// wet core above is only a minimum-depth guarantee; the outer
			// corridor still becomes wet or dry from its actual carved depth.
			sample.WaterKind = reconcile.WaterRiver
		}
	}

	if geometryWetOwned &&
		sample.WaterKind == reconcile.WaterRiver &&
		sample.RiverBedDefined {
		// Overlapping river corridors can choose a different strongest datum than
		// the segment that first established the geometric wet core. Re-apply the
		// wet-floor guarantee against the final selected owner surface so Runtime
		// cannot turn that overlap into a dry hole.
		ownerDepth := riverCenterDepth(m.Config, sample.RiverHalfWidth)
		ownerSurface := sample.RiverBedElevation + ownerDepth
		if sample.RiverReceiverSurfaceDefined &&
			sample.RiverReceiverSurface > ownerSurface {
			ownerSurface = sample.RiverReceiverSurface
		}
		finalHeight = math.Min(
			finalHeight,
			math.Max(0, ownerSurface-0.5*ownerDepth),
		)
	}

	if sample.WaterKind == reconcile.WaterNone &&
		bestOxbowSurface > math.Inf(-1) &&
		finalHeight <
			bestOxbowSurface-m.Config.MinLakeWaterDepthNormalized {
		sample.WaterKind = reconcile.WaterLake
		sample.WaterSurfaceDefined = true
		sample.WaterSurfaceElevation = bestOxbowSurface
	}

	// Explicit lakes override river classification where their raster footprint
	// overlaps a channel. A lake surface is only exposed where the carved ground
	// actually lies below that water datum.
	bestLakeSurface := math.Inf(-1)
	for _, id := range m.lakeBuckets[bucket] {
		node := m.Lakes[id]
		// A macro depression is only a lake candidate; the detailed terrain
		// still decides the actual shoreline. Never excavate terrain that
		// already stands materially above the spill-controlled lake datum,
		// otherwise a broad macro basin turns into an artificial raised
		// reservoir with vertical water walls.
		if baseHeight > node.WaterSurface+lakeShoreAllowanceNormalized &&
			!node.CoreCarve {
			continue
		}
		dx := worldX - node.X
		dz := worldZ - node.Z
		// Hydrology is solved on a regular macro grid. A rounded-square
		// superellipse preserves continuity across diagonal D8 neighbours while
		// overlapping cells and detailed terrain still soften the shoreline.
		distance := lakeDistance(dx, dz)
		if distance >= node.OuterRadius {
			continue
		}
		weight := 1.0
		if distance > node.InnerRadius {
			weight = 1 - smoothstep(node.InnerRadius, node.OuterRadius, distance)
		}
		candidate := baseHeight - math.Max(0, baseHeight-node.BedCeiling)*weight
		if candidate < finalHeight {
			finalHeight = candidate
			sample.LakeCarve = math.Max(sample.LakeCarve, baseHeight-candidate)
		}
		// Do not expose sub-block-thin water at the fringe. That looked
		// like a flood sheet creeping over otherwise dry terrain. A shallow
		// carved margin remains as a natural bank before the actual lake.
		if finalHeight < node.WaterSurface-m.Config.MinLakeWaterDepthNormalized &&
			node.WaterSurface > bestLakeSurface {
			bestLakeSurface = node.WaterSurface
			sample.WaterKind = reconcile.WaterLake
			sample.WaterSurfaceDefined = true
			sample.WaterSurfaceElevation = node.WaterSurface
		}
	}

	sample.FinalHeight = finalHeight
	return sample
}

func (m *Model) indexRiver(id int, seg RiverSegment) {
	spacing := m.Geometry.SpacingX()
	channelSupport :=
		riverMaxHalfWidth(seg) +
			riverBankBlendBlocks(seg.HalfWidth)
	supportRadius :=
		math.Max(
			math.Max(channelSupport, riverFluvialReach(seg)),
			riverDeltaSupportRadius(seg),
		) + math.Abs(seg.MeanderOffset)
	expansion := int(math.Ceil(supportRadius/spacing)) + 1
	x0, z0 := seg.From%m.Geometry.Width, seg.From/m.Geometry.Width
	x1, z1 := seg.To%m.Geometry.Width, seg.To/m.Geometry.Width
	minX, maxX := minInt(x0, x1)-expansion, maxInt(x0, x1)+expansion
	minZ, maxZ := minInt(z0, z1)-expansion, maxInt(z0, z1)+expansion
	m.addToBuckets(m.riverBuckets, id, minX, maxX, minZ, maxZ)
}

func (m *Model) indexLake(id int, node LakeNode) {
	spacing := m.Geometry.SpacingX()
	expansion := int(math.Ceil(node.OuterRadius/spacing)) + 1
	x, z := node.Index%m.Geometry.Width, node.Index/m.Geometry.Width
	m.addToBuckets(m.lakeBuckets, id, x-expansion, x+expansion, z-expansion, z+expansion)
}

func (m *Model) addToBuckets(buckets [][]int, id, minX, maxX, minZ, maxZ int) {
	minX = clampInt(minX, 0, m.Geometry.Width-1)
	maxX = clampInt(maxX, 0, m.Geometry.Width-1)
	minZ = clampInt(minZ, 0, m.Geometry.Height-1)
	maxZ = clampInt(maxZ, 0, m.Geometry.Height-1)
	for z := minZ; z <= maxZ; z++ {
		for x := minX; x <= maxX; x++ {
			i := z*m.Geometry.Width + x
			buckets[i] = append(buckets[i], id)
		}
	}
}

func (m *Model) bucketIndexForWorld(worldX, worldZ float64) int {
	gx := (worldX - m.Geometry.OriginX) / m.Geometry.SpacingX()
	gz := (worldZ - m.Geometry.OriginZ) / m.Geometry.SpacingZ()
	if gx < -1 || gz < -1 || gx > float64(m.Geometry.Width) || gz > float64(m.Geometry.Height) {
		return -1
	}
	x := clampInt(int(math.Round(gx)), 0, m.Geometry.Width-1)
	z := clampInt(int(math.Round(gz)), 0, m.Geometry.Height-1)
	return z*m.Geometry.Width + x
}

func normalizeConfig(cfg Config) (Config, error) {
	if cfg.MinRiverHalfWidthBlocks == 0 {
		cfg.MinRiverHalfWidthBlocks = 4
	}
	if cfg.MaxRiverHalfWidthBlocks == 0 {
		cfg.MaxRiverHalfWidthBlocks = 18
	}
	if cfg.RiverProfileStepBlocks == 0 {
		cfg.RiverProfileStepBlocks = 1
	}
	if cfg.MinRiverBedDepthNormalized == 0 {
		cfg.MinRiverBedDepthNormalized = 0.004
	}
	if cfg.RiverCenterDepthNormalized == 0 {
		cfg.RiverCenterDepthNormalized = 0.0025
	}
	if cfg.MinLakeWaterDepthNormalized == 0 {
		cfg.MinLakeWaterDepthNormalized = 0.003
	}
	if !finite(cfg.MinRiverHalfWidthBlocks) || !finite(cfg.MaxRiverHalfWidthBlocks) ||
		cfg.MinRiverHalfWidthBlocks <= 0 ||
		cfg.MaxRiverHalfWidthBlocks < cfg.MinRiverHalfWidthBlocks {
		return Config{}, errors.New("waterbed: river half-widths must be finite, positive and ordered")
	}
	if !finite(cfg.RiverProfileStepBlocks) || cfg.RiverProfileStepBlocks <= 0 {
		return Config{}, errors.New("waterbed: river profile step must be finite and positive")
	}
	if !finite(cfg.MinRiverBedDepthNormalized) ||
		cfg.MinRiverBedDepthNormalized <= 0 ||
		cfg.MinRiverBedDepthNormalized >= 0.05 {
		return Config{}, errors.New("waterbed: minimum river bed depth must be finite and inside (0,0.05)")
	}
	if !finite(cfg.RiverCenterDepthNormalized) ||
		cfg.RiverCenterDepthNormalized <= 0 ||
		cfg.RiverCenterDepthNormalized >= 0.05 {
		return Config{}, errors.New("waterbed: river center depth must be finite and inside (0,0.05)")
	}
	if !finite(cfg.MinLakeWaterDepthNormalized) ||
		cfg.MinLakeWaterDepthNormalized <= 0 ||
		cfg.MinLakeWaterDepthNormalized >= 0.05 {
		return Config{}, errors.New("waterbed: minimum lake water depth must be finite and inside (0,0.05)")
	}
	return cfg, nil
}

func harmonizeConfluenceProfiles(
	rivers []RiverSegment,
	ids []int,
	sharedSurface float64,
	cfg Config,
) float64 {
	if len(ids) == 0 || !finite(sharedSurface) {
		return math.Inf(1)
	}

	minBed := math.Inf(1)
	for _, id := range ids {
		seg := &rivers[id]
		if len(seg.Profile) == 0 {
			continue
		}
		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		if length <= 1e-9 {
			continue
		}

		depth := riverCenterDepth(cfg, seg.HalfWidth)
		targetBed := math.Max(0, sharedSurface-depth)

		// All incoming branches share the same downstream node. Their centerline
		// separation therefore shrinks linearly toward zero. Finish the vertical
		// convergence before the first sibling river footprints touch, rather
		// than allowing a high tributary to overlap a much lower carved branch.
		joinT := 1.0
		for _, otherID := range ids {
			if otherID == id {
				continue
			}
			other := rivers[otherID]
			sourceDistance := math.Hypot(
				seg.X0-other.X0,
				seg.Z0-other.Z0,
			)
			if sourceDistance <= 1e-9 {
				joinT = 0
				break
			}
			overlapSupport :=
				seg.HalfWidth +
					other.HalfWidth +
					math.Abs(seg.MeanderOffset) +
					math.Abs(other.MeanderOffset)
			overlapT := 1 - overlapSupport/sourceDistance
			joinT = math.Min(joinT, clamp(overlapT, 0, 1))
		}

		blendBlocks := math.Max(32, 2*seg.HalfWidth)
		startT := math.Max(0, joinT-blendBlocks/length)
		startBed := sampleRiverProfile(*seg, startT)
		for j := range seg.Profile {
			t := float64(j) / float64(len(seg.Profile)-1)
			if t < startT {
				continue
			}
			blend := smoothstep(startT, joinT, t)
			cap := lerp(startBed, targetBed, blend)
			if t >= joinT {
				cap = targetBed
			}
			if seg.Profile[j] > cap {
				seg.Profile[j] = cap
			}
			if j > 0 && seg.Profile[j] > seg.Profile[j-1] {
				seg.Profile[j] = seg.Profile[j-1]
			}
		}
		seg.Bed1 = seg.Profile[len(seg.Profile)-1]
		minBed = math.Min(minBed, seg.Bed1)
	}
	return minBed
}

func buildRiverProfile(
	base *terrain.Model,
	x0, z0, x1, z1,
	bed0, bed1,
	halfWidth,
	meanderOffset float64,
	mouthExpansionVersion uint8,
	stepBlocks,
	minBedDepthNormalized float64,
) []float64 {
	length := math.Hypot(x1-x0, z1-z0)
	steps := maxInt(1, int(math.Ceil(length/stepBlocks)))
	profile := make([]float64, steps+1)
	bankOffset := halfWidth + 3
	geometrySegment := RiverSegment{
		X0:                    x0,
		Z0:                    z0,
		X1:                    x1,
		Z1:                    z1,
		MeanderOffset:         meanderOffset,
		MouthExpansionVersion: mouthExpansionVersion,
	}
	for j := 0; j <= steps; j++ {
		t := float64(j) / float64(steps)
		x, z, nx, nz := riverCenterAndNormalForSegment(
			geometrySegment,
			t,
		)
		target := lerp(bed0, bed1, t)
		h := base.SampleNormalized(x, z)
		minimumTrench := math.Max(0, h-minBedDepthNormalized)
		v := math.Min(h, math.Min(target, minimumTrench))
		if bankOffset > 0 && length > 1e-9 {
			left := base.SampleNormalized(
				x+nx*bankOffset,
				z+nz*bankOffset,
			)
			right := base.SampleNormalized(
				x-nx*bankOffset,
				z-nz*bankOffset,
			)
			bankLow := math.Min(left, right)
			// Detailed relief may put a small ridge under the macro flow edge.
			// The curved river should still occupy the local low corridor, not
			// become an aqueduct above a lower nearby bank.
			bankGuidedTrench := math.Max(
				0,
				bankLow-minBedDepthNormalized,
			)
			v = math.Min(v, bankGuidedTrench)
		}
		if j > 0 && v > profile[j-1] {
			v = profile[j-1]
		}
		profile[j] = v
	}
	return profile
}

func terraceRiverProfile(
	profile []float64,
	stepBlocks,
	reachBlocks float64,
) []float64 {
	if len(profile) <= 1 || stepBlocks <= 0 || reachBlocks <= 0 {
		return append([]float64(nil), profile...)
	}
	totalDrop := profile[0] - profile[len(profile)-1]
	if totalDrop <= 1e-12 {
		return append([]float64(nil), profile...)
	}

	// Only terrace gentle river reaches. Steep mountain streams keep their
	// continuous descent and naturally become rapids/waterfalls; flattening a
	// steep 128-block segment to its outlet would carve an implausibly deep,
	// level canal. Gentle lowland rivers are where Minecraft's integer Y grid
	// turns a realistic continuous slope into annoying boat-stopping steps.
	if totalDrop > 0.014 {
		// Even true mountain/canyon reaches must not become stacks of
		// free-standing source-water blocks. Preserve their steep descent, but
		// spread any sub-cell jump into a bed-backed cascade.
		return spreadGentleProfileDrops(
			append([]float64(nil), profile...),
			stepBlocks,
		)
	}

	reachSamples := maxInt(
		1,
		int(math.Round(reachBlocks/stepBlocks)),
	)
	if totalDrop <= 0.006 {
		// A <=~3-block macro drop is rendered as one long navigable reach.
		reachSamples = len(profile)
	}
	out := append([]float64(nil), profile...)
	for start := 0; start < len(out); start += reachSamples {
		end := minInt(len(out), start+reachSamples)
		low := profile[start]
		for j := start + 1; j < end; j++ {
			if profile[j] < low {
				low = profile[j]
			}
		}
		for j := start; j < end; j++ {
			out[j] = low
		}
	}
	for j := 1; j < len(out); j++ {
		if out[j] > out[j-1] {
			out[j] = out[j-1]
		}
	}
	return spreadGentleProfileDrops(out, stepBlocks)
}

func spreadGentleProfileDrops(
	profile []float64,
	stepBlocks float64,
) []float64 {
	if len(profile) < 3 || stepBlocks <= 0 {
		return profile
	}
	out := append([]float64(nil), profile...)
	for j := 1; j < len(out); j++ {
		drop := out[j-1] - out[j]
		if drop <= 0 {
			continue
		}
		scale := 514.0
		if out[j-1] <= 0.5 && out[j] <= 0.5 {
			scale = 254.0
		}
		// Keep the continuous profile slope below one Minecraft vertical block
		// per horizontal block. Integer voxelization then becomes a supported
		// staircase rather than a multi-block water wall.
		maxDropPerSample := 0.82 * stepBlocks / scale
		if drop <= maxDropPerSample+1e-12 {
			continue
		}
		samples := int(math.Ceil(drop / maxDropPerSample))
		start := maxInt(0, j-samples)
		if start >= j {
			continue
		}
		startBed := out[start]
		targetBed := out[j]
		for k := start + 1; k < j; k++ {
			f := float64(k-start) / float64(j-start)
			cap := lerp(startBed, targetBed, f)
			if out[k] > cap {
				out[k] = cap
			}
		}
	}
	for j := 1; j < len(out); j++ {
		if out[j] > out[j-1] {
			out[j] = out[j-1]
		}
	}
	return out
}

func softenNavigableJunctionDrops(
	rivers []RiverSegment,
	cfg Config,
) {
	if len(rivers) == 0 {
		return
	}
	byFrom := make(map[int]int, len(rivers))
	incoming := make(map[int][]int, len(rivers))
	for id := range rivers {
		seg := rivers[id]
		byFrom[seg.From] = id
		incoming[seg.To] = append(incoming[seg.To], id)
	}

	for node, incomingIDs := range incoming {
		downID, ok := byFrom[node]
		if !ok {
			continue
		}
		down := rivers[downID]
		if len(down.Profile) == 0 {
			continue
		}
		downSurface :=
			sampleRiverProfile(down, 0) +
				riverCenterDepth(cfg, down.HalfWidth)

		for _, upID := range incomingIDs {
			up := &rivers[upID]
			if len(up.Profile) < 2 {
				continue
			}
			upDepth := riverCenterDepth(cfg, up.HalfWidth)
			upSurface :=
				sampleRiverProfile(*up, 1) + upDepth
			drop := upSurface - downSurface
			if drop <= 0 {
				continue
			}

			// Spread every junction drop over the upstream bed. Low-gradient
			// navigation terraces therefore become short one-block stair
			// sequences, while mountain/canyon drops become a bed-backed
			// cascade instead of a free-standing vertical source-water wall.
			blocksPerNormalized := 514.0
			if upSurface <= 0.5 && downSurface <= 0.5 {
				blocksPerNormalized = 254.0
			}
			dropBlocks := drop * blocksPerNormalized
			if dropBlocks <= 1.05 {
				continue
			}

			length := math.Hypot(up.X1-up.X0, up.Z1-up.Z0)
			if length <= 1e-9 {
				continue
			}
			transitionBlocks := math.Min(
				length,
				math.Max(24, 8*dropBlocks),
			)
			startT := math.Max(
				0,
				1-transitionBlocks/length,
			)
			startBed := sampleRiverProfile(*up, startT)
			targetBed := math.Max(0, downSurface-upDepth)
			if targetBed >= startBed-1e-12 {
				continue
			}

			for j := range up.Profile {
				t := float64(j) / float64(len(up.Profile)-1)
				if t < startT {
					continue
				}
				f := clamp(
					(t-startT)/math.Max(1e-12, 1-startT),
					0,
					1,
				)
				cap := lerp(startBed, targetBed, f)
				if up.Profile[j] > cap {
					up.Profile[j] = cap
				}
				if j > 0 && up.Profile[j] > up.Profile[j-1] {
					up.Profile[j] = up.Profile[j-1]
				}
			}
			up.Bed1 = up.Profile[len(up.Profile)-1]
		}
	}
}

func snapNavigableRiverProfile(
	profile []float64,
	cfg Config,
	halfWidth,
	surfaceStepNormalized float64,
) []float64 {
	if len(profile) == 0 || surfaceStepNormalized <= 0 {
		return append([]float64(nil), profile...)
	}
	depth := riverCenterDepth(cfg, halfWidth)
	surface := profile[len(profile)-1] + depth
	snappedSurface :=
		math.Floor(surface/surfaceStepNormalized+1e-12) *
			surfaceStepNormalized
	targetBed := math.Max(0, snappedSurface-depth)
	out := append([]float64(nil), profile...)
	for i := range out {
		if out[i] > targetBed {
			out[i] = targetBed
		}
		if i > 0 && out[i] > out[i-1] {
			out[i] = out[i-1]
		}
	}
	return out
}

func riverNavigationReachBlocks(halfWidth float64) float64 {
	return clamp(72+3.2*halfWidth, 80, 128)
}

func riverMeanderOffset(
	seed int64,
	from,
	to int,
	x0, z0,
	x1, z1,
	halfWidth,
	originalDrop float64,
	geometryVersion uint8,
) float64 {
	length := math.Hypot(x1-x0, z1-z0)
	if length <= 1e-9 {
		return 0
	}
	u := riverHashUnit(seed, from, to)
	lowSlope := 1 - smoothstep(
		0.010,
		0.050,
		math.Abs(originalDrop),
	)
	if lowSlope <= 1e-6 {
		return 0
	}
	maxOffset := math.Min(length*0.17, 5+1.35*halfWidth)

	// Legacy rules intentionally leave 20% of reaches perfectly straight.
	// That is visually fine for cardinal reaches, but a D8 diagonal is exactly
	// 45 degrees to Minecraft's block grid: a wide 128-block straight reach
	// then becomes a long 1x1 zipper bank. v1.1.5 keeps the "straight river"
	// character while adding only a small, low-frequency lateral deflection to
	// wide, low-gradient diagonal reaches that hit this legacy branch.
	if u < 0.20 {
		if geometryVersion < 1 ||
			halfWidth < 8 ||
			lowSlope < 0.35 ||
			!riverChordIsGridDiagonal(x0, z0, x1, z1) {
			return 0
		}
		amplitude := math.Min(
			maxOffset,
			math.Max(4, 0.55*maxOffset*lowSlope),
		)
		// Pick the anti-grid deflection side deliberately instead of randomly:
		// in the downstream half this makes an exact D8 diagonal slightly more
		// axis-favouring (0/1 voxel steps) rather than steeper (1/2 jumps).
		// Endpoints remain fixed, so this introduces no network drift.
		if (x1-x0)*(z1-z0) > 0 {
			amplitude = -amplitude
		}
		return amplitude
	}

	amplitude := maxOffset * lowSlope * (0.55 + 0.45*u)
	if amplitude < 1.5 {
		return 0
	}
	if riverHashUnit(seed^0x4D45414E444552, to, from) < 0.5 {
		amplitude = -amplitude
	}
	return amplitude
}

func riverChordIsGridDiagonal(x0, z0, x1, z1 float64) bool {
	dx := math.Abs(x1 - x0)
	dz := math.Abs(z1 - z0)
	scale := math.Max(1, math.Max(dx, dz))
	return dx > 1e-9 &&
		dz > 1e-9 &&
		math.Abs(dx-dz) <= 1e-9*scale
}

func riverHashUnit(seed int64, from, to int) float64 {
	h := uint64(seed) ^
		(uint64(from+1) * 0x9E3779B97F4A7C15) ^
		(uint64(to+1) * 0xC2B2AE3D27D4EB4F)
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	return float64(h>>11) / float64(uint64(1)<<53)
}

func riverCenterAndNormalAt(
	x0, z0, x1, z1,
	meanderOffset,
	t float64,
) (float64, float64, float64, float64) {
	t = clamp(t, 0, 1)
	dx := x1 - x0
	dz := z1 - z0
	length := math.Hypot(dx, dz)
	if length <= 1e-12 {
		return x0, z0, 1, 0
	}
	chordNX := -dz / length
	chordNZ := dx / length
	phase := math.Pi * t
	s := math.Sin(phase)
	c := math.Cos(phase)
	x := x0 + dx*t + chordNX*meanderOffset*s
	z := z0 + dz*t + chordNZ*meanderOffset*s
	tx := dx + chordNX*meanderOffset*math.Pi*c
	tz := dz + chordNZ*meanderOffset*math.Pi*c
	tangentLength := math.Hypot(tx, tz)
	if tangentLength <= 1e-12 {
		return x, z, chordNX, chordNZ
	}
	return x, z, -tz / tangentLength, tx / tangentLength
}

func riverCenterAndNormalAtV3(
	x0, z0, x1, z1,
	meanderOffset,
	t float64,
) (float64, float64, float64, float64) {
	t = clamp(t, 0, 1)
	dx := x1 - x0
	dz := z1 - z0
	length := math.Hypot(dx, dz)
	if length <= 1e-12 {
		return x0, z0, 1, 0
	}
	chordNX := -dz / length
	chordNZ := dx / length
	phase := math.Pi * t
	s := math.Sin(phase)
	c := math.Cos(phase)
	shape := s * s
	shapeDerivative := 2 * math.Pi * s * c
	x := x0 + dx*t + chordNX*meanderOffset*shape
	z := z0 + dz*t + chordNZ*meanderOffset*shape
	tx := dx + chordNX*meanderOffset*shapeDerivative
	tz := dz + chordNZ*meanderOffset*shapeDerivative
	tangentLength := math.Hypot(tx, tz)
	if tangentLength <= 1e-12 {
		return x, z, chordNX, chordNZ
	}
	return x, z, -tz / tangentLength, tx / tangentLength
}

func riverMeanderShapeForSegment(
	seg RiverSegment,
	t float64,
) (float64, float64, float64) {
	t = clamp(t, 0, 1)
	phase := math.Pi * t
	s := math.Sin(phase)
	c := math.Cos(phase)

	if seg.MouthExpansionVersion >= 3 {
		return s * s,
			2 * math.Pi * s * c,
			2 * math.Pi * math.Pi * (c*c - s*s)
	}
	return s,
		math.Pi * c,
		-math.Pi * math.Pi * s
}

func riverCenterAndNormalForSegment(
	seg RiverSegment,
	t float64,
) (float64, float64, float64, float64) {
	t = clamp(t, 0, 1)
	dx := seg.X1 - seg.X0
	dz := seg.Z1 - seg.Z0
	length := math.Hypot(dx, dz)
	if length <= 1e-12 {
		return seg.X0, seg.Z0, 1, 0
	}
	chordNX := -dz / length
	chordNZ := dx / length
	shape, shapeDerivative, _ :=
		riverMeanderShapeForSegment(seg, t)
	x := seg.X0 + dx*t + chordNX*seg.MeanderOffset*shape
	z := seg.Z0 + dz*t + chordNZ*seg.MeanderOffset*shape
	tx := dx + chordNX*seg.MeanderOffset*shapeDerivative
	tz := dz + chordNZ*seg.MeanderOffset*shapeDerivative
	tangentLength := math.Hypot(tx, tz)
	if tangentLength <= 1e-12 {
		return x, z, chordNX, chordNZ
	}
	return x, z, -tz / tangentLength, tx / tangentLength
}

// RiverCenterFrame returns the authoritative rendered centerline position and
// unit normal for a persisted river segment. Acceptance tools and previews use
// this instead of duplicating versioned meander formulas.
func RiverCenterFrame(
	seg RiverSegment,
	t float64,
) (float64, float64, float64, float64) {
	return riverCenterAndNormalForSegment(seg, t)
}

// RiverCenterPoint returns only the authoritative rendered centerline position.
func RiverCenterPoint(seg RiverSegment, t float64) (float64, float64) {
	x, z, _, _ := riverCenterAndNormalForSegment(seg, t)
	return x, z
}

func sampleRiverProfile(seg RiverSegment, t float64) float64 {
	if len(seg.Profile) == 0 {
		return lerp(seg.Bed0, seg.Bed1, t)
	}
	pos := clamp(t, 0, 1) * float64(len(seg.Profile)-1)
	i0 := int(math.Floor(pos))
	i1 := minInt(i0+1, len(seg.Profile)-1)
	f := pos - float64(i0)
	return lerp(seg.Profile[i0], seg.Profile[i1], f)
}

func connectFeedersToRenderedLake(
	m *Model,
	hydro *hydrology.Model,
	cfg Config,
	lakeSurfaces map[int]float64,
	incomingSegments [][]int,
) {
	if m == nil || hydro == nil || len(m.Rivers) == 0 {
		return
	}

	primaryCount := len(m.Rivers)
	byFrom := make(map[int]int, primaryCount)
	edgeByPair := make(map[[2]int]int, primaryCount)
	outByFrom := make(map[int]int, primaryCount)
	for id := 0; id < primaryCount; id++ {
		seg := m.Rivers[id]
		byFrom[seg.From] = id
		edgeByPair[[2]int{seg.From, seg.To}] = id
		outByFrom[seg.From] = id
	}

	// Seed each lake's connector graph with macro nodes whose final detailed
	// footprint is actually wet. A coarse LakeID cell is not sufficient: high
	// sub-cell terrain can keep that candidate dry.
	connected := make(map[int]map[int]bool)
	for _, node := range m.Lakes {
		if m.Sample(node.X, node.Z).WaterKind != reconcile.WaterLake {
			continue
		}
		if connected[node.LakeID] == nil {
			connected[node.LakeID] = make(map[int]bool)
		}
		connected[node.LakeID][node.Index] = true
	}

	for feederID := 0; feederID < primaryCount; feederID++ {
		feeder := m.Rivers[feederID]
		if feeder.From < 0 || feeder.From >= len(hydro.LakeID) ||
			feeder.To < 0 || feeder.To >= len(hydro.LakeID) {
			continue
		}
		lakeID := hydro.LakeID[feeder.To]
		if lakeID <= 0 || hydro.LakeID[feeder.From] == lakeID {
			continue
		}
		if riverPathTouchesLake(
			m,
			hydro,
			byFrom,
			feederID,
			lakeID,
		) {
			continue
		}
		surface, ok := lakeSurfaces[lakeID]
		if !ok || !finite(surface) {
			continue
		}
		targets := connected[lakeID]
		if len(targets) == 0 {
			continue
		}
		path, ok := renderedLakeConnectorPath(
			m,
			hydro,
			feeder.To,
			lakeID,
			surface,
			targets,
		)
		if !ok || len(path) < 2 {
			continue
		}
		startBed := feeder.Profile[len(feeder.Profile)-1]
		if !appendLakeConnectorPath(
			m,
			path,
			feeder.HalfWidth,
			startBed,
			surface,
			cfg,
			incomingSegments,
			edgeByPair,
			outByFrom,
		) {
			continue
		}
		for _, node := range path {
			targets[node] = true
		}
	}
}

func lowestOpenNode(open map[int]bool, dist map[int]float64) (int, float64) {
	current := -1
	best := math.Inf(1)
	for node := range open {
		d := dist[node]
		if d < best || (d == best && (current < 0 || node < current)) {
			best = d
			current = node
		}
	}
	return current, best
}

func renderedLakeConnectorPath(
	m *Model,
	hydro *hydrology.Model,
	start int,
	lakeID int,
	surface float64,
	targets map[int]bool,
) ([]int, bool) {
	n := m.Geometry.Width * m.Geometry.Height
	if start < 0 || start >= n ||
		lakeID <= 0 ||
		hydro.LakeID[start] != lakeID ||
		len(targets) == 0 {
		return nil, false
	}
	if targets[start] {
		return []int{start}, true
	}

	dist := map[int]float64{start: 0}
	previous := make(map[int]int)
	open := map[int]bool{start: true}
	closed := make(map[int]bool)
	found := -1
	directions := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for len(open) > 0 {
		current, best := lowestOpenNode(open, dist)
		if current < 0 {
			break
		}
		delete(open, current)
		if targets[current] {
			found = current
			break
		}
		closed[current] = true

		cx := current % m.Geometry.Width
		cz := current / m.Geometry.Width
		for _, direction := range directions {
			nx := cx + direction[0]
			nz := cz + direction[1]
			if nx < 0 || nx >= m.Geometry.Width ||
				nz < 0 || nz >= m.Geometry.Height {
				continue
			}
			next := nz*m.Geometry.Width + nx
			if closed[next] || hydro.LakeID[next] != lakeID {
				continue
			}
			wx, wz := m.Geometry.NodePosition(nx, nz)
			height := m.Base.SampleNormalized(wx, wz)
			aboveWater := math.Max(0, height-surface)
			step := 1.0
			if direction[0] != 0 && direction[1] != 0 {
				step = math.Sqrt2
			}
			// Prefer the lowest corridor through the coarse lake basin. The
			// path may cross dry LakeID cells, but it should follow their
			// natural lows instead of cutting a straight canal across a ridge.
			cost := step * (1 + 120*aboveWater)
			candidate := best + cost
			old, seen := dist[next]
			if !seen || candidate < old {
				dist[next] = candidate
				previous[next] = current
				open[next] = true
			}
		}
	}
	if found < 0 {
		return nil, false
	}

	path := []int{found}
	for path[len(path)-1] != start {
		prev, ok := previous[path[len(path)-1]]
		if !ok {
			return nil, false
		}
		path = append(path, prev)
		if len(path) > n {
			return nil, false
		}
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path, true
}

func appendLakeConnectorPath(
	m *Model,
	path []int,
	halfWidth,
	startBed,
	receiverSurface float64,
	cfg Config,
	incomingSegments [][]int,
	edgeByPair map[[2]int]int,
	outByFrom map[int]int,
) bool {
	if len(path) < 2 || halfWidth <= 0 {
		return false
	}
	depth := riverCenterDepth(cfg, halfWidth)
	targetBed := math.Min(
		startBed,
		math.Max(0, receiverSurface-depth),
	)
	totalLength := 0.0
	lengths := make([]float64, len(path)-1)
	for i := 0; i+1 < len(path); i++ {
		x0i, z0i := path[i]%m.Geometry.Width, path[i]/m.Geometry.Width
		x1i, z1i := path[i+1]%m.Geometry.Width, path[i+1]/m.Geometry.Width
		x0, z0 := m.Geometry.NodePosition(x0i, z0i)
		x1, z1 := m.Geometry.NodePosition(x1i, z1i)
		lengths[i] = math.Hypot(x1-x0, z1-z0)
		totalLength += lengths[i]
	}
	if totalLength <= 1e-9 {
		return false
	}

	currentBed := startBed
	traversed := 0.0
	widthT := riverWidthFraction(cfg, halfWidth)
	for i := 0; i+1 < len(path); i++ {
		from, to := path[i], path[i+1]
		key := [2]int{from, to}
		if existing, ok := edgeByPair[key]; ok {
			seg := &m.Rivers[existing]
			if !seg.ReceiverSurfaceDefined ||
				receiverSurface > seg.ReceiverSurface {
				seg.ReceiverSurfaceDefined = true
				seg.ReceiverSurface = receiverSurface
			}
			if len(seg.Profile) > 0 {
				currentBed = seg.Profile[len(seg.Profile)-1]
			}
			traversed += lengths[i]
			continue
		}
		if existing, ok := outByFrom[from]; ok {
			// Do not create a synthetic distributary. Another connector has
			// already established this node's path to the same lake.
			if existing >= 0 && existing < len(m.Rivers) {
				return false
			}
		}

		x0i, z0i := from%m.Geometry.Width, from/m.Geometry.Width
		x1i, z1i := to%m.Geometry.Width, to/m.Geometry.Width
		x0, z0 := m.Geometry.NodePosition(x0i, z0i)
		x1, z1 := m.Geometry.NodePosition(x1i, z1i)
		traversed += lengths[i]
		fraction := clamp(traversed/totalLength, 0, 1)
		desiredEnd := lerp(startBed, targetBed, fraction)
		desiredEnd = math.Min(currentBed, desiredEnd)
		originalDrop :=
			m.Base.SampleNormalized(x0, z0) -
				m.Base.SampleNormalized(x1, z1)
		lakeMouth := i == len(path)-2
		mouthExpansionVersion := uint8(3)
		if cfg.RiverGeometryVersion >= 1 && !lakeMouth {
			mouthExpansionVersion = 4
		}
		meander := riverMeanderOffset(
			m.Base.Config.Seed,
			from,
			to,
			x0, z0,
			x1, z1,
			halfWidth,
			originalDrop,
			cfg.RiverGeometryVersion,
		)
		profile := buildRiverProfile(
			m.Base,
			x0,
			z0,
			x1,
			z1,
			currentBed,
			desiredEnd,
			halfWidth,
			meander,
			mouthExpansionVersion,
			cfg.RiverProfileStepBlocks,
			cfg.MinRiverBedDepthNormalized*(1+1.5*widthT),
		)
		profile = terraceRiverProfile(
			profile,
			cfg.RiverProfileStepBlocks,
			riverNavigationReachBlocks(halfWidth),
		)
		profile = spreadGentleProfileDrops(
			profile,
			cfg.RiverProfileStepBlocks,
		)
		if len(profile) == 0 {
			return false
		}
		if i == 0 && from >= 0 && from < len(incomingSegments) {
			// buildRiverProfile may discover a lower detailed-terrain low at
			// the connector's first sample than the coarse feeder endpoint had.
			// Converge every incoming branch to that actual connector water
			// surface before the geometries touch, otherwise Minecraft sees a
			// multi-block vertical source-water wall at the join.
			harmonizeConfluenceProfiles(
				m.Rivers,
				incomingSegments[from],
				profile[0]+depth,
				cfg,
			)
		}
		segment := RiverSegment{
			From:                   from,
			To:                     to,
			X0:                     x0,
			Z0:                     z0,
			X1:                     x1,
			Z1:                     z1,
			Bed0:                   profile[0],
			Bed1:                   profile[len(profile)-1],
			HalfWidth:              halfWidth,
			MeanderOffset:          meander,
			Profile:                profile,
			ReceiverSurfaceDefined: true,
			ReceiverSurface:        receiverSurface,
			LakeMouth:              lakeMouth,
			MouthExpansionVersion:  mouthExpansionVersion,
		}
		id := len(m.Rivers)
		m.Rivers = append(m.Rivers, segment)
		m.indexRiver(id, segment)
		edgeByPair[key] = id
		outByFrom[from] = id
		if to >= 0 && to < len(incomingSegments) {
			incomingSegments[to] = append(incomingSegments[to], id)
		}
		currentBed = profile[len(profile)-1]
	}
	return true
}

func riverPathTouchesLake(
	m *Model,
	hydro *hydrology.Model,
	byFrom map[int]int,
	startSegmentID int,
	lakeID int,
) bool {
	id := startSegmentID
	visited := make(map[int]bool)
	for hops := 0; hops < 64; hops++ {
		if id < 0 || id >= len(m.Rivers) || visited[id] {
			return false
		}
		visited[id] = true
		seg := m.Rivers[id]
		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		steps := maxInt(1, int(math.Ceil(length/4)))
		for j := 0; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z, _, _ := riverCenterAndNormalForSegment(seg, t)
			if m.Sample(x, z).WaterKind == reconcile.WaterLake {
				return true
			}
		}
		node := seg.To
		if node < 0 || node >= len(hydro.LakeID) ||
			hydro.LakeID[node] != lakeID {
			return false
		}
		next, ok := byFrom[node]
		if !ok {
			return false
		}
		id = next
	}
	return false
}

func spatialLakeReceiverSurface(
	m *Model,
	seg RiverSegment,
) (float64, bool) {
	length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
	steps := maxInt(16, int(math.Ceil(length/4)))
	best := math.Inf(-1)
	found := false
	for j := 0; j <= steps; j++ {
		t := float64(j) / float64(steps)
		x, z, _, _ := riverCenterAndNormalForSegment(seg, t)
		bucket := m.bucketIndexForWorld(x, z)
		if bucket < 0 {
			continue
		}
		baseHeight := m.Base.SampleNormalized(x, z)
		for _, lakeID := range m.lakeBuckets[bucket] {
			lake := m.Lakes[lakeID]
			if baseHeight >
				lake.WaterSurface+lakeShoreAllowanceNormalized {
				continue
			}
			if lakeDistance(x-lake.X, z-lake.Z) >= lake.OuterRadius {
				continue
			}
			if !found || lake.WaterSurface > best {
				best = lake.WaterSurface
				found = true
			}
		}
	}
	return best, found
}

func easeReceiverMouthDrops(
	rivers []RiverSegment,
	incomingSegments [][]int,
	cfg Config,
) {
	for id := range rivers {
		seg := &rivers[id]
		if !seg.ReceiverSurfaceDefined ||
			!finite(seg.ReceiverSurface) ||
			len(seg.Profile) < 2 {
			continue
		}
		depth := riverCenterDepth(cfg, seg.HalfWidth)
		receiver := seg.ReceiverSurface
		naturalStart := seg.Profile[0] + depth
		naturalEnd := seg.Profile[len(seg.Profile)-1] + depth
		if naturalStart <= receiver+1e-12 &&
			naturalEnd <= receiver+1e-12 {
			continue
		}

		// The final Ocean/Lake classification may intersect this receiving
		// macro reach before its endpoint. Give the whole receiving reach one
		// water datum, then move the accumulated drop into the upstream reach
		// as bed-backed steps. This prevents a river water wall at a shoreline
		// whose exact sub-cell position is not known by the macro topology.
		targetBed := math.Max(0, receiver-depth)
		for j := range seg.Profile {
			if seg.Profile[j] > targetBed {
				seg.Profile[j] = targetBed
			}
		}
		for j := 1; j < len(seg.Profile); j++ {
			if seg.Profile[j] > seg.Profile[j-1] {
				seg.Profile[j] = seg.Profile[j-1]
			}
		}
		seg.Bed0 = seg.Profile[0]
		seg.Bed1 = seg.Profile[len(seg.Profile)-1]

		node := seg.From
		if node >= 0 && node < len(incomingSegments) {
			harmonizeConfluenceProfiles(
				rivers,
				incomingSegments[node],
				receiver,
				cfg,
			)
		}
	}
}

func seedLakeOutletConfluenceBackwater(
	rivers []RiverSegment,
	incomingSegments [][]int,
	cfg Config,
) {
	// MouthExpansionVersion 3 lets a low-gradient lake outlet carry its lake
	// datum well beyond the immediate shoreline. When that elevated surface
	// survives to the outlet segment's downstream node, every river footprint
	// meeting at that node must share the same water datum. Otherwise runtime
	// ownership can alternate block-by-block between the elevated outlet and a
	// lower sibling/downstream reach, producing a checkerboard water wall.
	for id := range rivers {
		seg := rivers[id]
		if seg.MouthExpansionVersion < 3 ||
			!seg.LakeOutlet ||
			!seg.LakeOutletSurfaceDefined ||
			lakeOutletSourceWeight(seg, 1) <= 1e-9 ||
			len(seg.Profile) == 0 {
			continue
		}

		naturalEnd :=
			sampleRiverProfile(seg, 1) +
				riverCenterDepth(cfg, seg.HalfWidth)
		sharedSurface :=
			riverConnectionSurface(seg, 1, naturalEnd)
		if sharedSurface <= naturalEnd+1e-12 {
			continue
		}

		node := seg.To
		if node < 0 || node >= len(incomingSegments) {
			continue
		}
		for _, incomingID := range incomingSegments[node] {
			if incomingID < 0 || incomingID >= len(rivers) {
				continue
			}
			incoming := &rivers[incomingID]
			if !incoming.ReceiverSurfaceDefined ||
				incoming.ReceiverSurface < sharedSurface {
				incoming.ReceiverSurfaceDefined = true
				incoming.ReceiverSurface = sharedSurface
			}
		}
	}
}

func harmonizeReceiverConfluences(
	rivers []RiverSegment,
	incomingSegments [][]int,
) {
	for _, ids := range incomingSegments {
		if len(ids) < 2 {
			continue
		}
		shared := math.Inf(-1)
		for _, id := range ids {
			if id < 0 || id >= len(rivers) {
				continue
			}
			seg := rivers[id]
			if seg.ReceiverSurfaceDefined &&
				seg.ReceiverSurface > shared {
				shared = seg.ReceiverSurface
			}
		}
		if !finite(shared) {
			continue
		}
		for _, id := range ids {
			if id < 0 || id >= len(rivers) {
				continue
			}
			seg := &rivers[id]
			if !seg.ReceiverSurfaceDefined ||
				seg.ReceiverSurface < shared {
				seg.ReceiverSurfaceDefined = true
				seg.ReceiverSurface = shared
			}
		}
	}
}

func propagateReceiverBackwater(
	rivers []RiverSegment,
	incomingSegments [][]int,
	cfg Config,
) {
	queue := make([]int, 0)
	queued := make([]bool, len(rivers))
	for id := range rivers {
		if rivers[id].ReceiverSurfaceDefined {
			queue = append(queue, id)
			queued[id] = true
		}
	}

	for head := 0; head < len(queue); head++ {
		id := queue[head]
		seg := rivers[id]
		receiver := seg.ReceiverSurface
		if !seg.ReceiverSurfaceDefined || !finite(receiver) {
			continue
		}
		startSurface :=
			sampleRiverProfile(seg, 0) +
				riverCenterDepth(cfg, seg.HalfWidth)
		if receiver <= startSurface+1e-12 {
			// Natural river surface already meets the receiver without an
			// upstream-facing step; backwater does not need to propagate farther.
			continue
		}

		node := seg.From
		if node < 0 || node >= len(incomingSegments) {
			continue
		}
		for _, upstreamID := range incomingSegments[node] {
			if upstreamID < 0 || upstreamID >= len(rivers) {
				continue
			}
			upstream := &rivers[upstreamID]
			naturalOutlet :=
				sampleRiverProfile(*upstream, 1) +
					riverCenterDepth(cfg, upstream.HalfWidth)
			if receiver <= naturalOutlet+1e-12 {
				continue
			}
			if !upstream.ReceiverSurfaceDefined ||
				receiver > upstream.ReceiverSurface {
				upstream.ReceiverSurfaceDefined = true
				upstream.ReceiverSurface = receiver
			}
			if !queued[upstreamID] {
				queue = append(queue, upstreamID)
				queued[upstreamID] = true
			}
		}
	}
}

func easeReceiverOutflowDrops(
	rivers []RiverSegment,
	cfg Config,
) {
	if len(rivers) == 0 {
		return
	}
	outgoing := make(map[int]int, len(rivers))
	receiverAtNode := make(map[int]float64)
	for id := range rivers {
		seg := rivers[id]
		outgoing[seg.From] = id
		if !seg.ReceiverSurfaceDefined {
			continue
		}
		if old, ok := receiverAtNode[seg.To]; !ok ||
			seg.ReceiverSurface > old {
			receiverAtNode[seg.To] = seg.ReceiverSurface
		}
	}

	for node, receiver := range receiverAtNode {
		outID, ok := outgoing[node]
		if !ok {
			continue
		}
		seg := &rivers[outID]
		if len(seg.Profile) < 2 || !finite(receiver) {
			continue
		}
		depth := riverCenterDepth(cfg, seg.HalfWidth)
		naturalStart := seg.Profile[0] + depth
		if receiver <= naturalStart+1e-12 {
			continue
		}

		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		if length <= 1e-9 {
			continue
		}
		scale := 514.0
		if receiver <= 0.5 && naturalStart <= 0.5 {
			scale = 254.0
		}
		dropBlocks := (receiver - naturalStart) * scale
		transitionBlocks := math.Min(
			length,
			math.Max(24, 8*dropBlocks),
		)
		endT := clamp(transitionBlocks/length, 0, 1)
		if endT <= 0 {
			continue
		}
		endBed := sampleRiverProfile(*seg, endT)
		startBed := math.Max(0, receiver-depth)
		if startBed <= seg.Profile[0]+1e-12 {
			continue
		}

		for j := range seg.Profile {
			t := float64(j) / float64(len(seg.Profile)-1)
			if t > endT {
				break
			}
			f := clamp(t/endT, 0, 1)
			floor := lerp(startBed, endBed, f)
			if seg.Profile[j] < floor {
				seg.Profile[j] = floor
			}
			if j > 0 && seg.Profile[j] > seg.Profile[j-1] {
				seg.Profile[j] = seg.Profile[j-1]
			}
		}
		seg.Bed0 = seg.Profile[0]
	}
}

func directReceiverSurface(
	node int,
	rec *reconcile.Model,
	hydro *hydrology.Model,
	lakeSurfaces map[int]float64,
) (float64, bool, error) {
	n := len(hydro.FlowTo)
	if node < 0 || node >= n {
		return 0, false, errors.New("waterbed: receiver node is out of range")
	}
	if !rec.WaterSurfaceDefined[node] {
		return 0, false, nil
	}

	switch rec.WaterKind[node] {
	case reconcile.WaterLake:
		if surface, ok := lakeSurfaces[hydro.LakeID[node]]; ok {
			return surface, true, nil
		}
		surface := rec.WaterSurfaceElevation[node]
		if !finite(surface) || surface < 0 || surface > 1 {
			return 0, false, errors.New("waterbed: invalid direct lake surface")
		}
		return surface, true, nil
	case reconcile.WaterOcean:
		surface := rec.WaterSurfaceElevation[node]
		if !finite(surface) || surface < 0 || surface > 1 {
			return 0, false, errors.New("waterbed: invalid direct ocean surface")
		}
		return surface, true, nil
	default:
		return 0, false, nil
	}
}

func lakeDistance(dx, dz float64) float64 {
	ax := math.Abs(dx)
	az := math.Abs(dz)
	return math.Pow(
		ax*ax*ax*ax+az*az*az*az,
		0.25,
	)
}

func lakeBoundaryOffset(angle, radius float64) (float64, float64) {
	c := math.Cos(angle)
	s := math.Sin(angle)
	denom := math.Pow(
		math.Pow(math.Abs(c), 4)+math.Pow(math.Abs(s), 4),
		0.25,
	)
	if denom <= 1e-12 {
		return 0, 0
	}
	scale := radius / denom
	return c * scale, s * scale
}

func riverHalfWidth(
	cfg Config,
	strengthThreshold,
	strength,
	accumulation,
	accumulationThreshold float64,
	order uint8,
) float64 {
	denom := math.Max(1e-9, 1-strengthThreshold)
	t := clamp((strength-strengthThreshold)/denom, 0, 1)

	// sqrt gives medium/large rivers enough width without making threshold
	// rivers abruptly wide.
	t = math.Sqrt(t)
	base := lerp(cfg.MinRiverHalfWidthBlocks, cfg.MaxRiverHalfWidthBlocks, t)

	// RiverStrength mainly distinguishes visible streams from ordinary rivers.
	// True trunk rivers also carry much larger accumulated runoff and higher
	// Strahler order. Let those rare reaches exceed the ordinary Max width so
	// GeoWorld can form actual large rivers without widening every tributary.
	flowSignal := 0.0
	if accumulationThreshold > 0 && accumulation > accumulationThreshold {
		flowSignal = clamp(
			math.Log2(accumulation/accumulationThreshold)/5.0,
			0,
			1,
		)
	}
	orderSignal := clamp((float64(order)-2)/3.0, 0, 1)
	majorSignal := math.Max(flowSignal, orderSignal)
	// Keep ordinary rivers close to the configured maximum, but let only the
	// very highest-flow/high-order trunk reaches become true "great rivers".
	// The quartic term stays tiny for medium rivers and rises strongly only
	// near majorSignal=1, yielding roughly 90-100 block full widths in Standard.
	majorBoost := cfg.MaxRiverHalfWidthBlocks *
		(0.55*majorSignal*majorSignal + 1.10*math.Pow(majorSignal, 4))
	return base + majorBoost
}

func riverWidthFraction(cfg Config, halfWidth float64) float64 {
	denom := math.Max(1e-9, cfg.MaxRiverHalfWidthBlocks-cfg.MinRiverHalfWidthBlocks)
	return clamp((halfWidth-cfg.MinRiverHalfWidthBlocks)/denom, 0, 1)
}

func riverCenterDepth(cfg Config, halfWidth float64) float64 {
	widthT := riverWidthFraction(cfg, halfWidth)
	return cfg.RiverCenterDepthNormalized * (1 + 2*math.Pow(widthT, 1.15))
}

func connectionBasinSignal(seg RiverSegment) float64 {
	if seg.MouthExpansionVersion < 3 {
		return 0
	}
	// Low total drop over one macro reach is a good proxy for a broad alluvial
	// lowland/backwater setting. Steep valleys keep a normal confined channel.
	drop := math.Max(0, seg.Bed0-seg.Bed1)
	return 1 - smoothstep(0.004, 0.016, drop)
}

func lakeOutletExtraHalfWidth(seg RiverSegment) float64 {
	if seg.MouthExpansionVersion < 3 {
		return math.Min(24, math.Max(14, 1.25*seg.HalfWidth))
	}
	basin := connectionBasinSignal(seg)
	factor := 1.25 + 0.85*basin
	return math.Min(48, math.Max(18, factor*seg.HalfWidth))
}

func riverDiagonalRasterWidthAmplitude(seg RiverSegment) float64 {
	if seg.MouthExpansionVersion < 4 ||
		seg.OceanMouth ||
		seg.LakeMouth ||
		seg.LakeOutlet ||
		seg.HalfWidth < 24 ||
		!riverChordIsGridDiagonal(seg.X0, seg.Z0, seg.X1, seg.Z1) {
		return 0
	}
	return clamp(0.19*seg.HalfWidth, 4, 9)
}

func riverDiagonalRasterWidthDelta(seg RiverSegment, t float64) float64 {
	amplitude := riverDiagonalRasterWidthAmplitude(seg)
	if amplitude <= 0 {
		return 0
	}
	t = clamp(t, 0, 1)
	s := math.Sin(math.Pi * t)
	return -amplitude * s * s * math.Sin(4*math.Pi*t)
}

func riverFluvialDetailStrength(seg RiverSegment) float64 {
	if seg.MouthExpansionVersion < 5 ||
		seg.OceanMouth ||
		seg.LakeMouth ||
		seg.LakeOutlet ||
		seg.HalfWidth < 14 {
		return 0
	}
	length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
	if length <= 1e-9 {
		return 0
	}
	start := sampleRiverProfile(seg, 0)
	end := sampleRiverProfile(seg, 1)
	drop := math.Max(0, start-end) * 128 / length
	widthWeight := smoothstep(14, 34, seg.HalfWidth)
	lowGradient := 1 - smoothstep(0.008, 0.030, drop)
	return clamp(widthWeight*lowGradient, 0, 1)
}

func riverFluvialReach(seg RiverSegment) float64 {
	bankRadius := riverMouthHalfWidth(seg, 0.5) +
		riverBankBlendBlocks(seg.HalfWidth)
	strength := riverFluvialDetailStrength(seg)
	if strength <= 0 {
		return bankRadius
	}
	return bankRadius + strength*(28+2.15*seg.HalfWidth)
}

func riverFluvialKey(seg RiverSegment) uint64 {
	h := uint64(1469598103934665603)
	values := [...]int64{
		int64(math.Round(seg.X0)),
		int64(math.Round(seg.Z0)),
		int64(math.Round(seg.X1)),
		int64(math.Round(seg.Z1)),
	}
	for _, value := range values {
		h ^= uint64(value)
		h *= uint64(1099511628211)
	}
	return h
}

func riverFluvialPhase(seg RiverSegment) float64 {
	const inv53 = 1.0 / 9007199254740992.0
	h := riverFluvialKey(seg)
	return float64(h>>11) * inv53 * 2 * math.Pi
}

func riverAbandonedMeanderOffset(
	seg RiverSegment,
	t float64,
	key uint64,
) float64 {
	t = clamp(t, 0, 1)
	s := math.Sin(math.Pi * t)
	side := 1.0
	if key&1 != 0 {
		side = -1
	}
	bankRadius := riverMouthHalfWidth(seg, t) +
		riverBankBlendBlocks(seg.HalfWidth)
	bulge := (0.55*seg.HalfWidth + 6) * s * s
	return side * (bankRadius + 4 + bulge)
}

func riverNaturalLeveeEnabled(seg RiverSegment) bool {
	strength := riverFluvialDetailStrength(seg)
	if strength < 0.56 {
		return false
	}
	// Only a deterministic subset of eligible wide, low-gradient reaches gets
	// levee refinement. This keeps levees as regional detail instead of the
	// default visual language of every large river.
	return riverFluvialKey(seg)%5 < 2
}

func riverNaturalLeveeLift(
	seg RiverSegment,
	t,
	worldX,
	worldZ,
	n float64,
) float64 {
	if !riverNaturalLeveeEnabled(seg) {
		return 0
	}
	t = clamp(t, 0, 1)
	n = clamp(n, 0, 1)
	strength := riverFluvialDetailStrength(seg)
	if strength <= 0 {
		return 0
	}

	// The levee is a broad shoulder immediately outside the active bank, not
	// an embankment. It fades before the middle floodplain/terrace band.
	bankBand :=
		smoothstep(0.015, 0.065, n) *
			(1 - smoothstep(0.16, 0.28, n))
	if bankBand <= 0 {
		return 0
	}

	cx, cz, nx, nz := RiverCenterFrame(seg, t)
	lateral := (worldX-cx)*nx + (worldZ-cz)*nz
	side := 1.0
	if lateral < 0 {
		side = -1
	}
	phase := riverFluvialPhase(seg)
	sectionSignal := 0.5 + 0.5*math.Sin(
		2*math.Pi*t+phase+0.82*side,
	)
	continuity := smoothstep(0.34, 0.72, sectionSignal)
	endFade :=
		smoothstep(0.04, 0.16, t) *
			(1 - smoothstep(0.84, 0.96, t))
	return 0.0042 * strength * bankBand * continuity * endFade
}

func applyRiverFluvialDryDetail(
	seg RiverSegment,
	t,
	worldX,
	worldZ,
	baseHeight,
	currentHeight,
	surface,
	distance float64,
) float64 {
	strength := riverFluvialDetailStrength(seg)
	if strength <= 0 {
		return currentHeight
	}
	halfWidth := riverMouthHalfWidth(seg, t)
	bankRadius := halfWidth + riverBankBlendBlocks(seg.HalfWidth)
	reach := riverFluvialReach(seg)
	if distance < bankRadius || distance >= reach {
		return currentHeight
	}

	phase := riverFluvialPhase(seg)
	if distance >= bankRadius {
		n := clamp(
			(distance-bankRadius)/math.Max(1e-9, reach-bankRadius),
			0,
			1,
		)
		envelope := (1 - smoothstep(0, 1, n)) * strength
		terrace := smoothstep(0.38, 0.56, n) *
			(1 - smoothstep(0.74, 0.92, n)) *
			strength
		floodTarget := surface +
			0.0055 +
			0.0011*math.Sin(2*math.Pi*t+phase)
		terraceTarget := surface +
			0.015 +
			0.0018*math.Sin(math.Pi*t+phase*0.5)
		target := lerp(floodTarget, terraceTarget, terrace)
		// Natural levees are modeled as a narrow zone that is carved slightly
		// less than the surrounding floodplain. The target is still capped by
		// baseHeight below, so this can never build an artificial wall above the
		// original terrain.
		target += riverNaturalLeveeLift(
			seg,
			t,
			worldX,
			worldZ,
			n,
		)
		weight := envelope * (0.74 - 0.24*terrace)
		candidate := lerp(
			baseHeight,
			math.Min(baseHeight, target),
			clamp(weight, 0, 1),
		)
		currentHeight = math.Min(currentHeight, candidate)
	}

	// Only a small deterministic subset of wide low-gradient reaches receives
	// an abandoned meander. It is a dry/shallow geomorphic groove, not another
	// hydrologic river edge, so the real drainage graph remains unchanged.
	key := riverFluvialKey(seg)
	if distance >= bankRadius && key%13 == 0 && strength >= 0.52 {
		cx, cz, nx, nz := RiverCenterFrame(seg, t)
		lateral := (worldX-cx)*nx + (worldZ-cz)*nz
		s := math.Sin(math.Pi * clamp(t, 0, 1))
		offset := riverAbandonedMeanderOffset(seg, t, key)
		oldDistance := math.Abs(lateral - offset)
		radius := clamp(0.18*seg.HalfWidth, 4, 11)
		grooveWeight := (1 - smoothstep(0.35*radius, radius, oldDistance)) *
			s * s * strength
		if grooveWeight > 0 {
			oldTarget := surface +
				0.0045 +
				0.0008*math.Sin(2*math.Pi*t+phase)
			candidate := lerp(
				baseHeight,
				math.Min(baseHeight, oldTarget),
				clamp(0.72*grooveWeight, 0, 1),
			)
			currentHeight = math.Min(currentHeight, candidate)
		}
	}
	return currentHeight
}

func riverOxbowWater(
	cfg Config,
	seg RiverSegment,
	t,
	worldX,
	worldZ,
	activeDistance float64,
) (surface, floor, weight float64, ok bool) {
	strength := riverFluvialDetailStrength(seg)
	bankRadius := riverMouthHalfWidth(seg, t) +
		riverBankBlendBlocks(seg.HalfWidth)
	if seg.MouthExpansionVersion < 5 ||
		seg.OceanMouth ||
		seg.LakeMouth ||
		seg.LakeOutlet ||
		seg.HalfWidth < 18 ||
		strength < 0.62 ||
		activeDistance < bankRadius {
		return 0, 0, 0, false
	}

	// Oxbows are a rarer wet subset of the already-rare abandoned meanders.
	// The unsigned river key is deterministic across regenerated/reopened worlds.
	key := riverFluvialKey(seg)
	if key%13 != 0 || (key/13)%3 != 0 {
		return 0, 0, 0, false
	}

	t = clamp(t, 0, 1)
	s := math.Sin(math.Pi * t)
	if s <= 0.18 {
		return 0, 0, 0, false
	}
	cx, cz, nx, nz := RiverCenterFrame(seg, t)
	lateral := (worldX-cx)*nx + (worldZ-cz)*nz
	offset := riverAbandonedMeanderOffset(seg, t, key)
	oldDistance := math.Abs(lateral - offset)
	radius := clamp(0.20*seg.HalfWidth, 5, 13)
	core := 1 - smoothstep(0.35*radius, radius, oldDistance)
	weight = clamp(
		core*s*s*(0.82+0.18*strength),
		0,
		1,
	)
	if weight <= 0 {
		return 0, 0, 0, false
	}

	// A disconnected oxbow has one still-water datum. Use the middle of the
	// low-gradient parent reach rather than the local t value so the crescent
	// cannot become a sloping staircase of source water.
	midBed := sampleRiverProfile(seg, 0.5)
	midNaturalSurface := midBed + riverCenterDepth(cfg, seg.HalfWidth)
	surface = riverConnectionSurface(seg, 0.5, midNaturalSurface)
	depth := math.Max(
		cfg.MinLakeWaterDepthNormalized*1.35,
		0.0042,
	) * (0.90 + 0.25*strength)
	floor = math.Max(0, surface-depth)
	return surface, floor, weight, true
}

func riverDeltaKey(seg RiverSegment) uint64 {
	return riverFluvialKey(seg) ^ 0xD371A5EED17A5EED
}

func riverDeltaStrength(seg RiverSegment) float64 {
	if seg.MouthExpansionVersion < 5 ||
		!seg.OceanMouth ||
		seg.LakeMouth ||
		seg.LakeOutlet ||
		seg.HalfWidth < 16 {
		return 0
	}
	basin := connectionBasinSignal(seg)
	if basin < 0.52 {
		return 0
	}
	key := riverDeltaKey(seg)
	if key%3 != 0 {
		return 0
	}
	width := smoothstep(16, 34, seg.HalfWidth)
	lowGradient := smoothstep(0.52, 0.90, basin)
	return clamp(width*lowGradient, 0, 1)
}

func riverDeltaBranchCount(seg RiverSegment) int {
	strength := riverDeltaStrength(seg)
	if strength <= 0 {
		return 0
	}
	key := riverDeltaKey(seg)
	if strength >= 0.72 && (key/3)%2 == 0 {
		return 2
	}
	return 1
}

func riverDeltaBranchSide(seg RiverSegment, branch int) float64 {
	count := riverDeltaBranchCount(seg)
	if count <= 0 {
		return 0
	}
	if count == 2 {
		if branch == 0 {
			return -1
		}
		return 1
	}
	if riverDeltaKey(seg)&1 != 0 {
		return -1
	}
	return 1
}

func riverDeltaCentralFlareScale(seg RiverSegment) float64 {
	strength := riverDeltaStrength(seg)
	if strength <= 0 {
		return 1
	}
	return 0.40 - 0.08*strength
}

func riverDeltaBranchGeometry(
	seg RiverSegment,
	t float64,
	branch int,
) (offset, halfWidth, bankRadius float64, ok bool) {
	strength := riverDeltaStrength(seg)
	count := riverDeltaBranchCount(seg)
	if strength <= 0 || branch < 0 || branch >= count {
		return 0, 0, 0, false
	}
	start := 0.46
	u := clamp((t-start)/(1-start), 0, 1)
	if u <= 0 {
		return 0, 0, 0, false
	}
	growth := smoothstep(0, 0.24, u)
	targetWidth := clamp(
		seg.HalfWidth*(0.22+0.07*strength),
		4.5,
		13.5,
	)
	halfWidth = targetWidth * growth
	if halfWidth < 1.25 {
		return 0, 0, 0, false
	}

	normalExtra := mouthMaxHalfWidth(seg) - seg.HalfWidth
	centralEnd := seg.HalfWidth +
		normalExtra*riverDeltaCentralFlareScale(seg)
	landGap := 3.5 + 4.5*strength
	terminalSeparation := math.Min(
		82,
		centralEnd+targetWidth+landGap,
	)
	separation := terminalSeparation * smoothstep(0, 1, u)
	offset = riverDeltaBranchSide(seg, branch) * separation
	bankRadius = halfWidth + riverBankBlendBlocks(halfWidth)
	return offset, halfWidth, bankRadius, true
}

func riverDeltaSupportRadius(seg RiverSegment) float64 {
	count := riverDeltaBranchCount(seg)
	if count <= 0 {
		return 0
	}
	maxSupport := 0.0
	for branch := 0; branch < count; branch++ {
		offset, _, bankRadius, ok :=
			riverDeltaBranchGeometry(seg, 1, branch)
		if !ok {
			continue
		}
		maxSupport = math.Max(
			maxSupport,
			math.Abs(offset)+bankRadius,
		)
	}
	return maxSupport
}

func riverDeltaPoint(
	seg RiverSegment,
	t,
	worldX,
	worldZ float64,
) (
	branch int,
	distance,
	halfWidth,
	bankRadius float64,
	ok bool,
) {
	count := riverDeltaBranchCount(seg)
	if count <= 0 {
		return 0, 0, 0, 0, false
	}
	cx, cz, nx, nz := RiverCenterFrame(seg, t)
	lateral := (worldX-cx)*nx + (worldZ-cz)*nz
	bestDistance := math.Inf(1)
	bestBranch := -1
	bestWidth := 0.0
	bestBank := 0.0
	for i := 0; i < count; i++ {
		offset, width, bank, valid :=
			riverDeltaBranchGeometry(seg, t, i)
		if !valid {
			continue
		}
		d := math.Abs(lateral - offset)
		if d < bestDistance {
			bestDistance = d
			bestBranch = i
			bestWidth = width
			bestBank = bank
		}
	}
	if bestBranch < 0 {
		return 0, 0, 0, 0, false
	}
	return bestBranch, bestDistance, bestWidth, bestBank, true
}

func riverMaxHalfWidth(seg RiverSegment) float64 {
	maxWidth := mouthMaxHalfWidth(seg) +
		riverDiagonalRasterWidthAmplitude(seg)
	if seg.MouthExpansionVersion >= 2 && seg.LakeOutlet {
		maxWidth = math.Max(
			maxWidth,
			seg.HalfWidth+lakeOutletExtraHalfWidth(seg),
		)
	}
	return maxWidth
}

func mouthMaxHalfWidth(seg RiverSegment) float64 {
	if seg.MouthExpansionVersion < 2 {
		if !seg.OceanMouth {
			return seg.HalfWidth
		}
		return seg.HalfWidth +
			math.Min(12, math.Max(9, seg.HalfWidth))
	}
	if seg.MouthExpansionVersion == 2 {
		switch {
		case seg.OceanMouth:
			extra := math.Min(18, math.Max(12, 0.85*seg.HalfWidth))
			return seg.HalfWidth + extra
		case seg.LakeMouth:
			extra := math.Min(12, math.Max(6, 0.55*seg.HalfWidth))
			return seg.HalfWidth + extra
		default:
			return seg.HalfWidth
		}
	}

	basin := connectionBasinSignal(seg)
	switch {
	case seg.OceanMouth:
		// The final reach starts from an already non-shrinking trunk and then
		// opens further into the sea; no narrow "gourd neck" is introduced.
		extra := math.Min(
			32,
			math.Max(16, (0.90+0.35*basin)*seg.HalfWidth),
		)
		return seg.HalfWidth + extra
	case seg.LakeMouth:
		// A low-gradient inlet may naturally spread into a backwater/flood basin
		// before reaching the persistent lake footprint.
		extra := math.Min(
			42,
			math.Max(16, (0.80+1.05*basin)*seg.HalfWidth),
		)
		return seg.HalfWidth + extra
	default:
		return seg.HalfWidth
	}
}

// Flare formal water connections at their receiving and source ends. A river
// entering a lake/ocean opens toward the receiver; a formal lake outlet starts
// broad enough to overlap the rendered lake shore and then narrows naturally.
// Hydrology must establish the relation first, so this never connects unrelated
// nearby water bodies merely because they are geometrically close.
func riverMouthHalfWidth(seg RiverSegment, t float64) float64 {
	if seg.MouthExpansionVersion >= 3 &&
		seg.LakeOutlet &&
		seg.OceanMouth {
		// A short lake outlet can also be the final ocean-mouth reach. Treat
		// that as one broad estuary instead of superimposing an outlet taper
		// and an ocean flare, which can create a mid-reach bulge followed by
		// narrowing at the sea. The source breadth is preserved and the final
		// reach may only hold or widen from there.
		sourceExtra := lakeOutletExtraHalfWidth(seg)
		receiverExtra := mouthMaxHalfWidth(seg) - seg.HalfWidth
		targetExtra := math.Max(sourceExtra, receiverExtra)
		progress := smoothstep(0.04, 0.90, clamp(t, 0, 1))
		return seg.HalfWidth + lerp(sourceExtra, targetExtra, progress)
	}
	if seg.MouthExpansionVersion < 2 {
		if !seg.OceanMouth {
			return seg.HalfWidth
		}
		progress := clamp((t-0.12)/0.62, 0, 1)
		return seg.HalfWidth +
			(mouthMaxHalfWidth(seg)-seg.HalfWidth)*smoothstep(0, 1, progress)
	}

	width := seg.HalfWidth + riverDiagonalRasterWidthDelta(seg, t)
	if seg.OceanMouth || seg.LakeMouth {
		start, span := 0.18, 0.70
		if seg.LakeMouth {
			start, span = 0.34, 0.56
		}
		if seg.MouthExpansionVersion >= 3 {
			if seg.OceanMouth {
				start, span = 0.04, 0.86
			} else {
				start, span = 0.18, 0.76
			}
		}
		progress := clamp((t-start)/span, 0, 1)
		receiverExtra :=
			(mouthMaxHalfWidth(seg) - seg.HalfWidth) *
				smoothstep(0, 1, progress)
		if seg.OceanMouth {
			receiverExtra *= riverDeltaCentralFlareScale(seg)
		}
		receiverWidth := seg.HalfWidth + receiverExtra
		width = math.Max(width, receiverWidth)
	}

	if seg.LakeOutlet {
		extra := lakeOutletExtraHalfWidth(seg)
		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		along := clamp(t, 0, 1) * length
		startFade, endFade := 96.0, 176.0
		if seg.MouthExpansionVersion >= 3 {
			basin := connectionBasinSignal(seg)
			startFade = 88 - 24*basin
			endFade = 190 + 90*basin
		}
		fade := 1 - smoothstep(startFade, endFade, along)
		width = math.Max(width, seg.HalfWidth+extra*fade)
	}
	return width
}

func lakeOutletSourceWeight(seg RiverSegment, t float64) float64 {
	if seg.MouthExpansionVersion < 2 ||
		!seg.LakeOutletSurfaceDefined {
		return 0
	}
	length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
	along := clamp(t, 0, 1) * length
	if seg.MouthExpansionVersion >= 3 {
		basin := connectionBasinSignal(seg)
		return 1 - smoothstep(
			72-16*basin,
			196+96*basin,
			along,
		)
	}
	return 1 - smoothstep(96, 176, along)
}

func riverConnectionSurface(seg RiverSegment, t, naturalSurface float64) float64 {
	surface := naturalSurface
	if seg.ReceiverSurfaceDefined && seg.ReceiverSurface > surface {
		surface = seg.ReceiverSurface
	}
	outletWeight := lakeOutletSourceWeight(seg, t)
	if outletWeight > 0 && seg.LakeOutletSurface > naturalSurface {
		outletSurface := lerp(
			naturalSurface,
			seg.LakeOutletSurface,
			outletWeight,
		)
		if outletSurface > surface {
			surface = outletSurface
		}
	}
	return math.Min(1, surface)
}

func riverBankBlendBlocks(halfWidth float64) float64 {
	return clamp(10+0.65*halfWidth, 12, 24)
}

func pointRiverSegment(
	px,
	pz float64,
	seg RiverSegment,
) (float64, float64) {
	dx := seg.X1 - seg.X0
	dz := seg.Z1 - seg.Z0
	len2 := dx*dx + dz*dz
	if len2 <= 1e-18 {
		return 0, math.Hypot(px-seg.X0, pz-seg.Z0)
	}
	length := math.Sqrt(len2)
	nx := -dz / length
	nz := dx / length
	t := clamp(
		((px-seg.X0)*dx+(pz-seg.Z0)*dz)/len2,
		0,
		1,
	)

	// Newton-refine the nearest point on the curved centerline. Legacy
	// reaches use sin(pi*t); v3 uses sin(pi*t)^2 so the meander has zero
	// lateral derivative at both macro-node endpoints. Offsets are modest,
	// so the chord projection remains a stable starting point.
	for iter := 0; iter < 4 && math.Abs(seg.MeanderOffset) > 1e-9; iter++ {
		shape, shapeDerivative, shapeSecondDerivative :=
			riverMeanderShapeForSegment(seg, t)
		x := seg.X0 + dx*t + nx*seg.MeanderOffset*shape
		z := seg.Z0 + dz*t + nz*seg.MeanderOffset*shape
		tx := dx + nx*seg.MeanderOffset*shapeDerivative
		tz := dz + nz*seg.MeanderOffset*shapeDerivative
		ax := nx * seg.MeanderOffset * shapeSecondDerivative
		az := nz * seg.MeanderOffset * shapeSecondDerivative
		rx := x - px
		rz := z - pz
		f := rx*tx + rz*tz
		fp := tx*tx + tz*tz + rx*ax + rz*az
		if math.Abs(fp) <= 1e-12 {
			break
		}
		next := clamp(t-f/fp, 0, 1)
		if math.Abs(next-t) <= 1e-8 {
			t = next
			break
		}
		t = next
	}

	x, z, _, _ := riverCenterAndNormalForSegment(seg, t)
	return t, math.Hypot(px-x, pz-z)
}

func pointSegment(px, pz, x0, z0, x1, z1 float64) (float64, float64) {
	dx := x1 - x0
	dz := z1 - z0
	len2 := dx*dx + dz*dz
	if len2 <= 1e-18 {
		return 0, math.Hypot(px-x0, pz-z0)
	}
	t := ((px-x0)*dx + (pz-z0)*dz) / len2
	t = clamp(t, 0, 1)
	qx := x0 + t*dx
	qz := z0 + t*dz
	return t, math.Hypot(px-qx, pz-qz)
}

func segmentIntersectsSquare(x0, z0, x1, z1, cx, cz, radius float64) bool {
	minX, maxX := cx-radius, cx+radius
	minZ, maxZ := cz-radius, cz+radius
	dx, dz := x1-x0, z1-z0
	tMin, tMax := 0.0, 1.0

	clipAxis := func(p, d, lo, hi float64) bool {
		if math.Abs(d) <= 1e-18 {
			return p >= lo && p <= hi
		}
		a := (lo - p) / d
		b := (hi - p) / d
		if a > b {
			a, b = b, a
		}
		if a > tMin {
			tMin = a
		}
		if b < tMax {
			tMax = b
		}
		return tMin <= tMax
	}

	return clipAxis(x0, dx, minX, maxX) &&
		clipAxis(z0, dz, minZ, maxZ) &&
		tMax >= 0 && tMin <= 1
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 <= edge0 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func sameGeometry(a, b world.GridGeometry) bool {
	return a.Width == b.Width &&
		a.Height == b.Height &&
		math.Abs(a.OriginX-b.OriginX) <= 1e-12 &&
		math.Abs(a.OriginZ-b.OriginZ) <= 1e-12 &&
		math.Abs(a.SpanXBlocks-b.SpanXBlocks) <= 1e-9 &&
		math.Abs(a.SpanZBlocks-b.SpanZBlocks) <= 1e-9
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// Audit samples every river segment's own longitudinal bed profile at
// non-node points. The segment profile itself may never rise downstream.
//
// The composite FinalHeight is intentionally not used for the upslope
// invariant: at connected confluences multiple channel footprints overlap and
// the final carved terrain is the minimum of all participating beds. A deeper
// tributary can therefore create a local confluence scour hole whose floor
// rises back toward the downstream segment after the overlap fades. That is a
// valid composite-bed recovery, not an upslope river profile. Runtime audits
// separately enforce continuous, non-rising free-water surfaces and non-dry
// channels.
func (m *Model) Audit(samplesPerSegment int, epsilon float64) Diagnostics {
	if samplesPerSegment < 2 {
		samplesPerSegment = 16
	}
	if epsilon < 0 {
		epsilon = 0
	}

	var d Diagnostics
	d.MaxRiseSegment = -1
	for segID, seg := range m.Rivers {
		steps := samplesPerSegment
		if profileSteps := 2 * (len(seg.Profile) - 1); profileSteps > steps {
			steps = profileSteps
		}
		prevX, prevZ, _, _ := riverCenterAndNormalForSegment(seg, 0)
		prevOwn := sampleRiverProfile(seg, 0)
		for j := 1; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z, _, _ := riverCenterAndNormalForSegment(seg, t)
			own := sampleRiverProfile(seg, t)
			ownRise := own - prevOwn
			d.RiverSampleSteps++
			if ownRise > epsilon {
				d.RiverUpslopeSteps++
				if ownRise > d.MaxRiverRise {
					d.MaxRiverRise = ownRise
					d.MaxRiseSegment = segID
					d.MaxRiseFrom = seg.From
					d.MaxRiseTo = seg.To
					d.MaxRiseT = t
					d.MaxRiseOwnProfileRise = own - prevOwn
					d.MaxRisePreviousOverlaps = m.riverOverlapCount(prevX, prevZ)
					d.MaxRiseCurrentOverlaps = m.riverOverlapCount(x, z)
				}
			}
			prevX, prevZ = x, z
			prevOwn = own
		}
	}

	for _, lake := range m.Lakes {
		// Hydrology marks macro depression candidates, but detailed terrain owns
		// the final shoreline. A candidate node whose detailed terrain stands
		// above the spill-controlled datum is intentionally dry and is not a
		// lake-bed invariant violation.
		if m.Base.SampleNormalized(lake.X, lake.Z) >
			lake.WaterSurface+lakeShoreAllowanceNormalized {
			continue
		}
		h := m.SampleNormalized(lake.X, lake.Z)
		excess := h - lake.BedCeiling
		if excess > epsilon {
			d.LakeBedViolations++
			d.MaxLakeBedExcess = math.Max(d.MaxLakeBedExcess, excess)
		}
	}
	return d
}

func riverBankProtectedAt(
	rivers []RiverSegment,
	ids []int,
	worldX,
	worldZ float64,
) bool {
	for _, id := range ids {
		if id < 0 || id >= len(rivers) {
			continue
		}
		seg := rivers[id]
		t, distance := pointRiverSegment(worldX, worldZ, seg)
		bankRadius := riverMouthHalfWidth(seg, t) +
			riverBankBlendBlocks(seg.HalfWidth)
		if distance < bankRadius {
			return true
		}
		_, deltaDistance, _, deltaBankRadius, deltaOK :=
			riverDeltaPoint(seg, t, worldX, worldZ)
		if deltaOK && deltaDistance < deltaBankRadius {
			return true
		}
	}
	return false
}

func (m *Model) riverOverlapCount(worldX, worldZ float64) int {
	bucket := m.bucketIndexForWorld(worldX, worldZ)
	if bucket < 0 {
		return 0
	}
	count := 0
	for _, id := range m.riverBuckets[bucket] {
		seg := m.Rivers[id]
		_, distance := pointRiverSegment(worldX, worldZ, seg)
		if distance < seg.HalfWidth {
			count++
		}
	}
	return count
}
