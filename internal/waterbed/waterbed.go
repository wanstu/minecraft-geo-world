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

	type riverDraft struct {
		edge      reconcile.RiverEdgeConstraint
		x0, z0    float64
		x1, z1    float64
		halfWidth float64
	}
	drafts := make([]riverDraft, len(rec.RiverEdges))
	outgoing := make([]int, n)
	indegree := make([]int, n)
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
		drafts[id] = riverDraft{
			edge: edge,
			x0:   x0, z0: z0, x1: x1, z1: z1,
			halfWidth: riverHalfWidth(cfg, rec.Config.RiverStrengthThreshold, edge.RiverStrength),
		}
		outgoing[edge.From] = id
		indegree[edge.To]++
	}

	// A sampled upstream profile may become lower than its macro endpoint target
	// because the broad terrain itself contains a local dip between control
	// points. Propagate that actual outlet height through the river DAG before
	// constructing the downstream segment. Without this, an incoming corridor
	// can carve a confluence below the next segment's start target; as its radial
	// influence fades downstream, the final bed rises back out of the junction.
	effectiveBed := append([]float64(nil), rec.RiverBedTarget...)
	queue := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if indegree[i] == 0 {
			queue = append(queue, i)
		}
	}
	processed := 0
	for head := 0; head < len(queue); head++ {
		node := queue[head]
		edgeID := outgoing[node]
		if edgeID < 0 {
			continue
		}
		draft := drafts[edgeID]
		bed0 := effectiveBed[draft.edge.From]
		bed1 := math.Min(effectiveBed[draft.edge.To], bed0)
		widthT := riverWidthFraction(cfg, draft.halfWidth)
		profile := buildRiverProfile(
			base,
			draft.x0, draft.z0,
			draft.x1, draft.z1,
			bed0, bed1,
			cfg.RiverProfileStepBlocks,
			cfg.MinRiverBedDepthNormalized*(1+1.5*widthT),
		)
		if len(profile) == 0 {
			return nil, errors.New("waterbed: river profile is empty")
		}
		outlet := profile[len(profile)-1]
		if outlet < effectiveBed[draft.edge.To] {
			effectiveBed[draft.edge.To] = outlet
		}
		segment := RiverSegment{
			From: draft.edge.From, To: draft.edge.To,
			X0: draft.x0, Z0: draft.z0, X1: draft.x1, Z1: draft.z1,
			Bed0:      bed0,
			Bed1:      effectiveBed[draft.edge.To],
			HalfWidth: draft.halfWidth,
			Profile:   profile,
		}
		id := len(m.Rivers)
		m.Rivers = append(m.Rivers, segment)
		m.indexRiver(id, segment)
		processed++

		indegree[draft.edge.To]--
		if indegree[draft.edge.To] == 0 {
			queue = append(queue, draft.edge.To)
		}
	}
	if processed != len(drafts) {
		return nil, errors.New("waterbed: river graph contains a cycle")
	}

	spacing := base.Geometry.SpacingX()
	inner := spacing * 0.40
	outer := spacing * 0.60
	for i, defined := range rec.LakeBedCeilingDefined {
		if !defined {
			continue
		}
		lakeID := hydro.LakeID[i]
		if lakeID <= 0 || !rec.WaterSurfaceDefined[i] {
			return nil, errors.New("waterbed: lake bed constraint is missing lake water metadata")
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

	// A channel can geometrically pass through the rendered lake footprint even
	// when its Hydrology FlowTo chain does not point directly at that lake node.
	// Treat any such spatial intersection as the same receiving-water backwater
	// constraint so Runtime does not jump upward from shallow river water to the
	// lake datum at the footprint boundary.
	// Macro hydrology finds closed basins at 128-block control points, while
	// the detailed terrain adds sub-cell relief afterwards. Recheck every lake
	// against the actual detailed terrain around the union of its rendered
	// footprint. If a lower local spill route exists, lower the lake datum to
	// that rim instead of rendering a raised reservoir wall.
	capLakeSurfacesToDetailedRims(base, m.Lakes)

	lakeSurfaces := make(map[int]float64)
	for _, lake := range m.Lakes {
		if old, ok := lakeSurfaces[lake.LakeID]; !ok || lake.WaterSurface < old {
			lakeSurfaces[lake.LakeID] = lake.WaterSurface
		}
	}
	for i := range m.Rivers {
		surface, ok, err := downstreamReceiverSurface(
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
	}

	for i := range m.Rivers {
		seg := &m.Rivers[i]
		for _, lake := range m.Lakes {
			if !segmentIntersectsSquare(
				seg.X0, seg.Z0, seg.X1, seg.Z1,
				lake.X, lake.Z, lake.OuterRadius,
			) {
				continue
			}
			if !seg.ReceiverSurfaceDefined || lake.WaterSurface > seg.ReceiverSurface {
				seg.ReceiverSurfaceDefined = true
				seg.ReceiverSurface = lake.WaterSurface
			}
		}
	}

	return m, nil
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
				x := node.X + math.Cos(angle)*radius
				z := node.Z + math.Sin(angle)*radius

				// Only inspect exposed portions of the union boundary. A sample
				// still covered by a sibling node belongs to the lake interior.
				insideSibling := false
				for _, otherID := range ids {
					other := lakes[otherID]
					if math.Hypot(x-other.X, z-other.Z) < other.OuterRadius {
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
	for _, id := range m.riverBuckets[bucket] {
		seg := m.Rivers[id]
		t, distance := pointSegment(worldX, worldZ, seg.X0, seg.Z0, seg.X1, seg.Z1)
		if distance >= seg.HalfWidth {
			continue
		}
		bed := sampleRiverProfile(seg, t)
		weight := 1 - smoothstep(0, seg.HalfWidth, distance)
		if weight > sample.RiverWeight {
			sample.RiverWeight = weight
		}
		// Water level should follow the channel that actually owns this point,
		// not a deeper neighbouring branch whose bank merely overlaps here.
		// Prefer the segment with the strongest lateral weight; at an exact
		// confluence tie, prefer the higher upstream bed so water cannot vanish
		// between two connected centerlines.
		if !sample.RiverBedDefined ||
			weight > riverDatumWeight+1e-12 ||
			(math.Abs(weight-riverDatumWeight) <= 1e-12 && bed > sample.RiverBedElevation) {
			sample.RiverBedDefined = true
			sample.RiverBedElevation = bed
			sample.RiverHalfWidth = seg.HalfWidth
			riverDatumWeight = weight
		}
		if seg.ReceiverSurfaceDefined &&
			(!sample.RiverReceiverSurfaceDefined || seg.ReceiverSurface > sample.RiverReceiverSurface) {
			sample.RiverReceiverSurfaceDefined = true
			sample.RiverReceiverSurface = seg.ReceiverSurface
		}
		candidate := baseHeight - math.Max(0, baseHeight-bed)*weight
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
		dx := worldX - node.X
		dz := worldZ - node.Z
		// Hydrology is solved on a regular macro grid, but the rendered lake
		// must not inherit that grid as square shorelines. Use a radial
		// footprint and let adjacent lake nodes blend as overlapping basins.
		distance := math.Hypot(dx, dz)
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
	expansion := int(math.Ceil(seg.HalfWidth/spacing)) + 1
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
	return cfg, nil
}

func buildRiverProfile(
	base *terrain.Model,
	x0, z0, x1, z1,
	bed0, bed1,
	stepBlocks,
	minBedDepthNormalized float64,
) []float64 {
	length := math.Hypot(x1-x0, z1-z0)
	steps := maxInt(1, int(math.Ceil(length/stepBlocks)))
	profile := make([]float64, steps+1)
	for j := 0; j <= steps; j++ {
		t := float64(j) / float64(steps)
		x := lerp(x0, x1, t)
		z := lerp(z0, z1, t)
		target := lerp(bed0, bed1, t)
		h := base.SampleNormalized(x, z)
		minimumTrench := math.Max(0, h-minBedDepthNormalized)
		v := math.Min(h, math.Min(target, minimumTrench))
		if j > 0 && v > profile[j-1] {
			v = profile[j-1]
		}
		profile[j] = v
	}
	return profile
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

func downstreamReceiverSurface(
	start int,
	rec *reconcile.Model,
	hydro *hydrology.Model,
	lakeSurfaces map[int]float64,
) (float64, bool, error) {
	n := len(hydro.FlowTo)
	if start < 0 || start >= n {
		return 0, false, errors.New("waterbed: receiver search start is out of range")
	}
	node := start
	for steps := 0; steps < n; steps++ {
		if rec.WaterSurfaceDefined[node] {
			switch rec.WaterKind[node] {
			case reconcile.WaterLake:
				if surface, ok := lakeSurfaces[hydro.LakeID[node]]; ok {
					return surface, true, nil
				}
				surface := rec.WaterSurfaceElevation[node]
				if !finite(surface) || surface < 0 || surface > 1 {
					return 0, false, errors.New("waterbed: invalid downstream lake surface")
				}
				return surface, true, nil
			case reconcile.WaterOcean:
				surface := rec.WaterSurfaceElevation[node]
				if !finite(surface) || surface < 0 || surface > 1 {
					return 0, false, errors.New("waterbed: invalid downstream ocean surface")
				}
				return surface, true, nil
			}
		}

		to := hydro.FlowTo[node]
		if to < 0 {
			return 0, false, nil
		}
		if to >= n {
			return 0, false, errors.New("waterbed: receiver search flow target is out of range")
		}
		if to == node {
			return 0, false, errors.New("waterbed: receiver search encountered self cycle")
		}
		node = to
	}
	return 0, false, errors.New("waterbed: receiver search exceeded flow graph length")
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
		prevX, prevZ := seg.X0, seg.Z0
		prev := m.SampleNormalized(prevX, prevZ)
		prevOwn := sampleRiverProfile(seg, 0)
		for j := 1; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x := lerp(seg.X0, seg.X1, t)
			z := lerp(seg.Z0, seg.Z1, t)
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
		_, distance := pointSegment(worldX, worldZ, seg.X0, seg.Z0, seg.X1, seg.Z1)
		if distance < seg.HalfWidth {
			count++
		}
	}
	return count
}
