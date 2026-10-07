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
	lakeShoreAllowanceNormalized    = 0.0015
	navigationSurfaceStepNormalized = 0.009
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
		)
		length := math.Hypot(x1-x0, z1-z0)
		drafts[id] = riverDraft{
			edge: edge,
			x0:   x0, z0: z0, x1: x1, z1: z1,
			halfWidth: halfWidth,
			meanderOffset: riverMeanderOffset(
				base.Config.Seed,
				edge.From,
				edge.To,
				length,
				halfWidth,
				edge.OriginalDrop,
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
		bed1 := math.Min(effectiveBed[draft.edge.To], bed0)
		widthT := riverWidthFraction(cfg, draft.halfWidth)
		profile := buildRiverProfile(
			base,
			draft.x0, draft.z0,
			draft.x1, draft.z1,
			bed0, bed1,
			draft.halfWidth,
			draft.meanderOffset,
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
		segment := RiverSegment{
			From: draft.edge.From, To: draft.edge.To,
			X0: draft.x0, Z0: draft.z0, X1: draft.x1, Z1: draft.z1,
			Bed0:          bed0,
			Bed1:          effectiveBed[draft.edge.To],
			HalfWidth:     draft.halfWidth,
			MeanderOffset: draft.meanderOffset,
			Profile:       profile,
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
	propagateReceiverBackwater(
		m.Rivers,
		incomingSegments,
		cfg,
	)

	return m, nil
}

func buildLakeNodes(
	base *terrain.Model,
	rec *reconcile.Model,
	hydro *hydrology.Model,
	m *Model,
) (map[int]float64, error) {
	spacing := base.Geometry.SpacingX()
	inner := spacing * 0.50
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
		node := LakeNode{
			Index:  i,
			LakeID: lakeID,
			X:      x, Z: z,
			BedCeiling:   rec.LakeBedCeiling[i],
			InnerRadius:  inner,
			OuterRadius:  outer,
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

	lakeSurfaces := make(map[int]float64)
	for _, lake := range m.Lakes {
		if old, ok := lakeSurfaces[lake.LakeID]; !ok ||
			lake.WaterSurface < old {
			lakeSurfaces[lake.LakeID] = lake.WaterSurface
		}
	}
	return lakeSurfaces, nil
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

	riverDatumWeight := -1.0
	riverDatumT := 2.0
	for _, id := range m.riverBuckets[bucket] {
		seg := m.Rivers[id]
		t, distance := pointRiverSegment(worldX, worldZ, seg)
		bankRadius := seg.HalfWidth + riverBankBlendBlocks(seg.HalfWidth)
		if distance >= bankRadius {
			continue
		}
		bed := sampleRiverProfile(seg, t)

		if distance >= seg.HalfWidth {
			// Shape a short dry river valley around the navigable channel.
			// The river is the local low point; terrain rises smoothly back to
			// the untouched broad surface instead of meeting water as a noisy
			// staircase or vertical trench wall.
			surface := bed + riverCenterDepth(m.Config, seg.HalfWidth)
			innerBank := math.Min(baseHeight, surface+0.0022)
			bankWeight := 1 - smoothstep(
				seg.HalfWidth,
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
			continue
		}

		weight := 1 - smoothstep(0, seg.HalfWidth, distance)
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
			sample.RiverHalfWidth = seg.HalfWidth
			sample.RiverReceiverSurfaceDefined = seg.ReceiverSurfaceDefined
			sample.RiverReceiverSurface = seg.ReceiverSurface
			if seg.ReceiverSurfaceDefined {
				// Receiver datum is a backwater floor, not a downstream ramp.
				// Propagation during Build guarantees the floor continues
				// upstream far enough to avoid an upward step at segment joins.
				sample.RiverReceiverWeight = 1
			} else {
				sample.RiverReceiverWeight = 0
			}
			riverDatumWeight = weight
			riverDatumT = t
		}
		surface := bed + riverCenterDepth(m.Config, seg.HalfWidth)
		channelShoulder := math.Min(baseHeight, surface+0.0022)
		crossT := smoothstep(0, seg.HalfWidth, distance)
		candidate := math.Min(
			baseHeight,
			lerp(bed, channelShoulder, crossT),
		)
		if candidate < finalHeight {
			finalHeight = candidate
			sample.RiverCarve = math.Max(sample.RiverCarve, baseHeight-candidate)
		}
		if sample.WaterKind == reconcile.WaterNone {
			sample.WaterKind = reconcile.WaterRiver
		}
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
		if baseHeight > node.WaterSurface+lakeShoreAllowanceNormalized {
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
		const minLakeWaterDepthNormalized = 0.003
		if finalHeight < node.WaterSurface-minLakeWaterDepthNormalized &&
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
	supportRadius :=
		seg.HalfWidth +
			math.Abs(seg.MeanderOffset) +
			riverBankBlendBlocks(seg.HalfWidth)
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
	return cfg, nil
}

func harmonizeConfluenceProfiles(
	rivers []RiverSegment,
	ids []int,
	sharedSurface float64,
	cfg Config,
) float64 {
	if len(ids) < 2 || !finite(sharedSurface) {
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
	meanderOffset,
	stepBlocks,
	minBedDepthNormalized float64,
) []float64 {
	length := math.Hypot(x1-x0, z1-z0)
	steps := maxInt(1, int(math.Ceil(length/stepBlocks)))
	profile := make([]float64, steps+1)
	bankOffset := halfWidth + 3
	for j := 0; j <= steps; j++ {
		t := float64(j) / float64(steps)
		x, z, nx, nz := riverCenterAndNormalAt(
			x0,
			z0,
			x1,
			z1,
			meanderOffset,
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
		return append([]float64(nil), profile...)
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
	return out
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
	length,
	halfWidth,
	originalDrop float64,
) float64 {
	if length <= 1e-9 {
		return 0
	}
	u := riverHashUnit(seed, from, to)
	// Some reaches really are straight. Keeping a minority untouched also
	// avoids turning the river network into uniform decorative sine waves.
	if u < 0.20 {
		return 0
	}
	lowSlope := 1 - smoothstep(
		0.010,
		0.050,
		math.Abs(originalDrop),
	)
	if lowSlope <= 1e-6 {
		return 0
	}
	maxOffset := math.Min(length*0.17, 5+1.35*halfWidth)
	amplitude := maxOffset * lowSlope * (0.55 + 0.45*u)
	if amplitude < 1.5 {
		return 0
	}
	if riverHashUnit(seed^0x4D45414E444552, to, from) < 0.5 {
		amplitude = -amplitude
	}
	return amplitude
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
		x, z, _, _ := riverCenterAndNormalAt(
			seg.X0,
			seg.Z0,
			seg.X1,
			seg.Z1,
			seg.MeanderOffset,
			t,
		)
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

func riverHalfWidth(cfg Config, threshold, strength float64) float64 {
	denom := math.Max(1e-9, 1-threshold)
	t := clamp((strength-threshold)/denom, 0, 1)
	// sqrt gives medium/large rivers enough width without making threshold
	// rivers abruptly wide.
	t = math.Sqrt(t)
	return lerp(cfg.MinRiverHalfWidthBlocks, cfg.MaxRiverHalfWidthBlocks, t)
}

func riverWidthFraction(cfg Config, halfWidth float64) float64 {
	denom := math.Max(1e-9, cfg.MaxRiverHalfWidthBlocks-cfg.MinRiverHalfWidthBlocks)
	return clamp((halfWidth-cfg.MinRiverHalfWidthBlocks)/denom, 0, 1)
}

func riverCenterDepth(cfg Config, halfWidth float64) float64 {
	widthT := riverWidthFraction(cfg, halfWidth)
	return cfg.RiverCenterDepthNormalized * (1 + 2*math.Pow(widthT, 1.15))
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

	// Newton-refine the nearest point on the curved centerline
	// P(t)=P0+tD+N*A*sin(pi*t). Offsets are intentionally modest, so the
	// chord projection is a stable starting point.
	for iter := 0; iter < 4 && math.Abs(seg.MeanderOffset) > 1e-9; iter++ {
		phase := math.Pi * t
		s := math.Sin(phase)
		c := math.Cos(phase)
		x := seg.X0 + dx*t + nx*seg.MeanderOffset*s
		z := seg.Z0 + dz*t + nz*seg.MeanderOffset*s
		tx := dx + nx*seg.MeanderOffset*math.Pi*c
		tz := dz + nz*seg.MeanderOffset*math.Pi*c
		ax := -nx * seg.MeanderOffset * math.Pi * math.Pi * s
		az := -nz * seg.MeanderOffset * math.Pi * math.Pi * s
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

	x, z, _, _ := riverCenterAndNormalAt(
		seg.X0,
		seg.Z0,
		seg.X1,
		seg.Z1,
		seg.MeanderOffset,
		t,
	)
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

// Audit samples the actual local waterbed surface, including non-node points
// along every river centerline. This is intentionally stricter than checking
// only macro endpoints because a local terrain hump between two valid targets
// would still create an unusable longitudinal profile.
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
		prevX, prevZ, _, _ := riverCenterAndNormalAt(
			seg.X0,
			seg.Z0,
			seg.X1,
			seg.Z1,
			seg.MeanderOffset,
			0,
		)
		prev := m.SampleNormalized(prevX, prevZ)
		prevOwn := sampleRiverProfile(seg, 0)
		for j := 1; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z, _, _ := riverCenterAndNormalAt(
				seg.X0,
				seg.Z0,
				seg.X1,
				seg.Z1,
				seg.MeanderOffset,
				t,
			)
			current := m.SampleNormalized(x, z)
			own := sampleRiverProfile(seg, t)
			rise := current - prev
			d.RiverSampleSteps++
			if rise > epsilon {
				d.RiverUpslopeSteps++
				if rise > d.MaxRiverRise {
					d.MaxRiverRise = rise
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
			prev = current
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
