package geomorphaudit

import (
	"errors"
	"math"
	"sort"

	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/waterbed"
)

const EvidenceClassInvestigation = "investigation_signal"

type Config struct {
	RouteSampleStepBlocks float64
	TransitionProbeBlocks float64
	MaxDetailRecords      int
}

type Report struct {
	EvidenceClass string `json:"evidenceClass"`
	Seed          int64  `json:"seed"`
	RulesVersion  string `json:"rulesVersion"`

	River      RiverReport      `json:"river"`
	Transition TransitionReport `json:"transition"`
}

type RiverReport struct {
	Segments int `json:"segments"`
	Routes   int `json:"routes"`

	RouteDrySamples               int               `json:"routeDrySamples"`
	TraversedSurfaceRiseSteps     int               `json:"traversedSurfaceRiseSteps"`
	TraversedInternalRiseSteps    int               `json:"traversedInternalRiseSteps"`
	TraversedReceiverRiseSteps    int               `json:"traversedReceiverRiseSteps"`
	UniqueSurfaceRiseSteps        int               `json:"uniqueSurfaceRiseSteps"`
	UniqueInternalRiseSteps       int               `json:"uniqueInternalRiseSteps"`
	UniqueReceiverRiseSteps       int               `json:"uniqueReceiverRiseSteps"`
	InternalRiseAboveQuarterBlock int               `json:"internalRiseAboveQuarterBlock"`
	InternalRiseAboveOneBlock     int               `json:"internalRiseAboveOneBlock"`
	MaxInternalRiseBlocks         float64           `json:"maxInternalRiseBlocks"`
	MaxReceiverRiseBlocks         float64           `json:"maxReceiverRiseBlocks"`
	DownstreamSteeperRoutes       int               `json:"downstreamSteeperRoutes"`
	RouteProfiles                 []RouteProfile    `json:"routeProfiles,omitempty"`
	InternalRiseSignals           []RiverRiseSignal `json:"internalRiseSignals,omitempty"`
	ReceiverRiseSignals           []RiverRiseSignal `json:"receiverRiseSignals,omitempty"`

	Confluences                int                `json:"confluences"`
	ConfluenceSurfaceRises     int                `json:"confluenceSurfaceRises"`
	ConfluenceWidthRegressions int                `json:"confluenceWidthRegressions"`
	ConfluenceSignals          []ConfluenceSignal `json:"confluenceSignals,omitempty"`

	OceanMouths           int           `json:"oceanMouths"`
	MouthApproachRises    int           `json:"mouthApproachRises"`
	MouthNarrowingSignals int           `json:"mouthNarrowingSignals"`
	MouthSignals          []MouthSignal `json:"mouthSignals,omitempty"`

	LakeMouths  int `json:"lakeMouths"`
	LakeOutlets int `json:"lakeOutlets"`
}

type RouteProfile struct {
	SourceNode        int     `json:"sourceNode"`
	EndNode           int     `json:"endNode"`
	Segments          int     `json:"segments"`
	LengthBlocks      float64 `json:"lengthBlocks"`
	TotalDropBlocks   float64 `json:"totalDropBlocks"`
	UpperDropPerKm    float64 `json:"upperDropPerKm"`
	MiddleDropPerKm   float64 `json:"middleDropPerKm"`
	LowerDropPerKm    float64 `json:"lowerDropPerKm"`
	DrySamples        int     `json:"drySamples"`
	SurfaceRiseSteps  int     `json:"surfaceRiseSteps"`
	InternalRiseSteps int     `json:"internalRiseSteps"`
	ReceiverRiseSteps int     `json:"receiverRiseSteps"`
	DownstreamSteeper bool    `json:"downstreamSteeper"`
}

type RiverRiseSignal struct {
	Segment                int     `json:"segment"`
	T                      float64 `json:"t"`
	X                      float64 `json:"x"`
	Z                      float64 `json:"z"`
	RiseBlocks             float64 `json:"riseBlocks"`
	PreviousKind           string  `json:"previousKind"`
	CurrentKind            string  `json:"currentKind"`
	PreviousSpecialFeature string  `json:"previousSpecialFeature,omitempty"`
	CurrentSpecialFeature  string  `json:"currentSpecialFeature,omitempty"`
	ReceiverRelated        bool    `json:"receiverRelated"`
}

type ConfluenceSignal struct {
	Node                      int     `json:"node"`
	IncomingSegments          int     `json:"incomingSegments"`
	DownstreamSegment         int     `json:"downstreamSegment"`
	MaxSurfaceRiseBlocks      float64 `json:"maxSurfaceRiseBlocks"`
	RequiredHalfWidthBlocks   float64 `json:"requiredHalfWidthBlocks"`
	DownstreamHalfWidthBlocks float64 `json:"downstreamHalfWidthBlocks"`
	WidthRatio                float64 `json:"widthRatio"`
}

type MouthSignal struct {
	Segment                   int     `json:"segment"`
	ApproachSurfaceRiseBlocks float64 `json:"approachSurfaceRiseBlocks"`
	WidthAt80Blocks           float64 `json:"widthAt80Blocks"`
	WidthAt98Blocks           float64 `json:"widthAt98Blocks"`
	WidthRatio                float64 `json:"widthRatio"`
	KindAt98                  string  `json:"kindAt98"`
}

type TransitionReport struct {
	Samples int                    `json:"samples"`
	Classes []TransitionClassStats `json:"classes,omitempty"`
}

type TransitionClassStats struct {
	From               string  `json:"from"`
	To                 string  `json:"to"`
	Count              int     `json:"count"`
	SlopeP50Deg        float64 `json:"slopeP50Deg"`
	SlopeP90Deg        float64 `json:"slopeP90Deg"`
	SlopeP99Deg        float64 `json:"slopeP99Deg"`
	MaxSlopeDeg        float64 `json:"maxSlopeDeg"`
	CurvatureP90Blocks float64 `json:"curvatureP90Blocks"`
	MaxCurvatureBlocks float64 `json:"maxCurvatureBlocks"`
}

type routePoint struct {
	distance        float64
	surfaceY        float64
	wet             bool
	receiverRelated bool
	segment         int
	t               float64
	x               float64
	z               float64
	waterKind       reconcile.WaterKind
	specialFeature  string
}

type transitionAccumulator struct {
	from         string
	to           string
	slopes       []float64
	curvatures   []float64
	maxSlope     float64
	maxCurvature float64
}

func DefaultConfig() Config {
	return Config{
		RouteSampleStepBlocks: 16,
		TransitionProbeBlocks: 16,
		MaxDetailRecords:      64,
	}
}

func Analyze(model *generator.WorldModel, cfg Config) (Report, error) {
	if model == nil ||
		model.Waterbed == nil ||
		model.RuntimeProfile == nil ||
		model.Terrain == nil ||
		model.CoastTerrain == nil {
		return Report{}, errors.New("geomorphaudit: complete generated world model is required")
	}
	cfg = normalizeConfig(cfg)
	report := Report{
		EvidenceClass: EvidenceClassInvestigation,
		Seed:          model.Config.Seed,
		RulesVersion:  model.Config.WorldProfile.GenerationRulesVersion,
	}
	auditRivers(model, cfg, &report.River)
	auditTransitions(model, cfg, &report.Transition)
	return report, nil
}

func normalizeConfig(cfg Config) Config {
	d := DefaultConfig()
	if !finite(cfg.RouteSampleStepBlocks) || cfg.RouteSampleStepBlocks <= 0 {
		cfg.RouteSampleStepBlocks = d.RouteSampleStepBlocks
	}
	if !finite(cfg.TransitionProbeBlocks) || cfg.TransitionProbeBlocks <= 0 {
		cfg.TransitionProbeBlocks = d.TransitionProbeBlocks
	}
	if cfg.MaxDetailRecords <= 0 {
		cfg.MaxDetailRecords = d.MaxDetailRecords
	}
	return cfg
}

func auditRivers(model *generator.WorldModel, cfg Config, report *RiverReport) {
	report.Segments = len(model.Waterbed.Rivers)
	incoming := make(map[int][]int)
	outgoing := make(map[int]int)
	type riseKey struct {
		segment int
		t       int64
	}
	uniqueRises := make(map[riseKey]RiverRiseSignal)
	for id, seg := range model.Waterbed.Rivers {
		incoming[seg.To] = append(incoming[seg.To], id)
		if previous, exists := outgoing[seg.From]; !exists ||
			model.Waterbed.Rivers[previous].HalfWidth < seg.HalfWidth {
			outgoing[seg.From] = id
		}
		if seg.OceanMouth {
			report.OceanMouths++
		}
		if seg.LakeMouth {
			report.LakeMouths++
		}
		if seg.LakeOutlet {
			report.LakeOutlets++
		}
	}

	for id, first := range model.Waterbed.Rivers {
		if len(incoming[first.From]) != 0 {
			continue
		}
		route := traceRoute(model.Waterbed.Rivers, id, outgoing)
		if len(route) == 0 {
			continue
		}
		profile, rises := profileRoute(model, cfg, route)
		report.Routes++
		for _, rise := range rises {
			key := riseKey{segment: rise.Segment, t: int64(math.Round(rise.T * 1e9))}
			if _, exists := uniqueRises[key]; exists {
				continue
			}
			uniqueRises[key] = rise
		}
		report.RouteDrySamples += profile.DrySamples
		report.TraversedSurfaceRiseSteps += profile.SurfaceRiseSteps
		report.TraversedInternalRiseSteps += profile.InternalRiseSteps
		report.TraversedReceiverRiseSteps += profile.ReceiverRiseSteps
		if profile.DownstreamSteeper {
			report.DownstreamSteeperRoutes++
		}
		if len(report.RouteProfiles) < cfg.MaxDetailRecords {
			report.RouteProfiles = append(report.RouteProfiles, profile)
		}
	}

	for _, rise := range uniqueRises {
		report.UniqueSurfaceRiseSteps++
		if rise.ReceiverRelated {
			report.UniqueReceiverRiseSteps++
			report.MaxReceiverRiseBlocks = math.Max(report.MaxReceiverRiseBlocks, rise.RiseBlocks)
			if len(report.ReceiverRiseSignals) < cfg.MaxDetailRecords {
				report.ReceiverRiseSignals = append(report.ReceiverRiseSignals, rise)
			}
			continue
		}
		report.UniqueInternalRiseSteps++
		report.MaxInternalRiseBlocks = math.Max(report.MaxInternalRiseBlocks, rise.RiseBlocks)
		if rise.RiseBlocks >= 0.25 {
			report.InternalRiseAboveQuarterBlock++
		}
		if rise.RiseBlocks >= 1.0 {
			report.InternalRiseAboveOneBlock++
		}
		if len(report.InternalRiseSignals) < cfg.MaxDetailRecords {
			report.InternalRiseSignals = append(report.InternalRiseSignals, rise)
		}
	}

	sort.Slice(report.InternalRiseSignals, func(i, j int) bool {
		return report.InternalRiseSignals[i].RiseBlocks > report.InternalRiseSignals[j].RiseBlocks
	})
	sort.Slice(report.ReceiverRiseSignals, func(i, j int) bool {
		return report.ReceiverRiseSignals[i].RiseBlocks > report.ReceiverRiseSignals[j].RiseBlocks
	})

	auditConfluences(model, cfg, incoming, outgoing, report)
	auditMouths(model, cfg, report)
}

func traceRoute(segments []waterbed.RiverSegment, start int, outgoing map[int]int) []int {
	route := make([]int, 0, 8)
	seen := make(map[int]bool)
	current := start
	for current >= 0 && current < len(segments) && !seen[current] {
		seen[current] = true
		route = append(route, current)
		next, ok := outgoing[segments[current].To]
		if !ok {
			break
		}
		current = next
	}
	return route
}

func profileRoute(model *generator.WorldModel, cfg Config, route []int) (RouteProfile, []RiverRiseSignal) {
	first := model.Waterbed.Rivers[route[0]]
	last := model.Waterbed.Rivers[route[len(route)-1]]
	points := make([]routePoint, 0, len(route)*16)
	totalDistance := 0.0
	havePrevious := false
	var previousX, previousZ float64

	for _, id := range route {
		seg := model.Waterbed.Rivers[id]
		lengthUpperBound := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) +
			math.Pi*math.Abs(seg.MeanderOffset)
		steps := int(math.Ceil(lengthUpperBound / cfg.RouteSampleStepBlocks))
		if steps < 2 {
			steps = 2
		}
		for j := 0; j <= steps; j++ {
			if len(points) > 0 && j == 0 {
				continue
			}
			t := float64(j) / float64(steps)
			x, z := waterbed.RiverCenterPoint(seg, t)
			if havePrevious {
				totalDistance += math.Hypot(x-previousX, z-previousZ)
			}
			previousX, previousZ = x, z
			havePrevious = true
			sample := finalRuntimeSample(model, x, z)
			ground := model.Waterbed.Sample(x, z)
			wet := sample.World.WaterSurfaceDefined &&
				sample.World.WaterKind != reconcile.WaterNone
			surfaceY := 0.0
			if wet {
				surfaceY = minecraftY(sample.World.WaterSurfaceElevation)
			}
			points = append(points, routePoint{
				distance: totalDistance,
				surfaceY: surfaceY,
				wet:      wet,
				receiverRelated: ground.RiverReceiverSurfaceDefined ||
					sample.World.WaterKind == reconcile.WaterLake ||
					sample.World.WaterKind == reconcile.WaterOcean,
				segment:        id,
				t:              t,
				x:              x,
				z:              z,
				waterKind:      sample.World.WaterKind,
				specialFeature: sample.SpecialFeature,
			})
		}
	}

	out := RouteProfile{
		SourceNode:   first.From,
		EndNode:      last.To,
		Segments:     len(route),
		LengthBlocks: totalDistance,
	}
	if len(points) == 0 || totalDistance <= 1e-9 {
		return out, nil
	}

	var rises []RiverRiseSignal
	var previousWet bool
	var previousSurface float64
	var previousReceiverRelated bool
	var previousPoint routePoint
	for _, p := range points {
		if !p.wet {
			out.DrySamples++
			previousWet = false
			continue
		}
		if previousWet && p.surfaceY > previousSurface+0.01 {
			out.SurfaceRiseSteps++
			receiverRelated := previousReceiverRelated || p.receiverRelated
			if receiverRelated {
				out.ReceiverRiseSteps++
			} else {
				out.InternalRiseSteps++
			}
			rises = append(rises, RiverRiseSignal{
				Segment:                p.segment,
				T:                      p.t,
				X:                      p.x,
				Z:                      p.z,
				RiseBlocks:             p.surfaceY - previousSurface,
				PreviousKind:           waterKindName(previousPoint.waterKind),
				CurrentKind:            waterKindName(p.waterKind),
				PreviousSpecialFeature: previousPoint.specialFeature,
				CurrentSpecialFeature:  p.specialFeature,
				ReceiverRelated:        receiverRelated,
			})
		}
		previousSurface = p.surfaceY
		previousReceiverRelated = p.receiverRelated
		previousPoint = p
		previousWet = true
	}

	startY, okStart := surfaceAt(points, 0)
	endY, okEnd := surfaceAt(points, totalDistance)
	q1Y, okQ1 := surfaceAt(points, totalDistance/3)
	q2Y, okQ2 := surfaceAt(points, 2*totalDistance/3)
	if okStart && okEnd {
		out.TotalDropBlocks = startY - endY
	}
	thirdKm := totalDistance / 3 / 1000
	if thirdKm > 1e-9 {
		if okStart && okQ1 {
			out.UpperDropPerKm = (startY - q1Y) / thirdKm
		}
		if okQ1 && okQ2 {
			out.MiddleDropPerKm = (q1Y - q2Y) / thirdKm
		}
		if okQ2 && okEnd {
			out.LowerDropPerKm = (q2Y - endY) / thirdKm
		}
	}
	out.DownstreamSteeper =
		out.LowerDropPerKm > out.UpperDropPerKm*1.75 &&
			out.LowerDropPerKm-out.UpperDropPerKm > 0.5
	return out, rises
}

func surfaceAt(points []routePoint, distance float64) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	if distance <= 0 {
		for _, p := range points {
			if p.wet {
				return p.surfaceY, true
			}
		}
		return 0, false
	}
	for i := 1; i < len(points); i++ {
		if points[i].distance < distance {
			continue
		}
		a, b := points[i-1], points[i]
		if !a.wet || !b.wet {
			continue
		}
		span := b.distance - a.distance
		if span <= 1e-9 {
			return b.surfaceY, true
		}
		t := (distance - a.distance) / span
		return a.surfaceY + (b.surfaceY-a.surfaceY)*t, true
	}
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].wet {
			return points[i].surfaceY, true
		}
	}
	return 0, false
}

func auditConfluences(
	model *generator.WorldModel,
	cfg Config,
	incoming map[int][]int,
	outgoing map[int]int,
	report *RiverReport,
) {
	for node, ids := range incoming {
		if len(ids) < 2 {
			continue
		}
		downID, ok := outgoing[node]
		if !ok {
			continue
		}
		report.Confluences++
		down := model.Waterbed.Rivers[downID]
		dx, dz := waterbed.RiverCenterPoint(down, 0.04)
		downSample := finalRuntimeSample(model, dx, dz)
		// The confluence contract is persisted on the downstream segment itself.
		// NavigationSampleAt can select an overlapping incoming segment at the
		// exact junction because both bank distances are zero, producing a false
		// width regression signal.
		downWidth := down.HalfWidth

		sumSquares := 0.0
		maxIncoming := 0.0
		maxRiseBlocks := 0.0
		for _, inID := range ids {
			inSeg := model.Waterbed.Rivers[inID]
			sumSquares += inSeg.HalfWidth * inSeg.HalfWidth
			maxIncoming = math.Max(maxIncoming, inSeg.HalfWidth)
			x, z := waterbed.RiverCenterPoint(inSeg, 0.96)
			inSample := finalRuntimeSample(model, x, z)
			if !wetRuntime(inSample) || !wetRuntime(downSample) {
				continue
			}
			rise := minecraftY(downSample.World.WaterSurfaceElevation) -
				minecraftY(inSample.World.WaterSurfaceElevation)
			maxRiseBlocks = math.Max(maxRiseBlocks, rise)
		}
		required := math.Max(maxIncoming, 0.86*math.Sqrt(sumSquares))
		ratio := 1.0
		if required > 1e-9 {
			ratio = downWidth / required
		}
		surfaceSignal := maxRiseBlocks > 0.25
		widthSignal := ratio < 1
		if surfaceSignal {
			report.ConfluenceSurfaceRises++
		}
		if widthSignal {
			report.ConfluenceWidthRegressions++
		}
		if (surfaceSignal || widthSignal) &&
			len(report.ConfluenceSignals) < cfg.MaxDetailRecords {
			report.ConfluenceSignals = append(report.ConfluenceSignals, ConfluenceSignal{
				Node:                      node,
				IncomingSegments:          len(ids),
				DownstreamSegment:         downID,
				MaxSurfaceRiseBlocks:      maxRiseBlocks,
				RequiredHalfWidthBlocks:   required,
				DownstreamHalfWidthBlocks: downWidth,
				WidthRatio:                ratio,
			})
		}
	}
}

func auditMouths(model *generator.WorldModel, cfg Config, report *RiverReport) {
	for id, seg := range model.Waterbed.Rivers {
		if !seg.OceanMouth {
			continue
		}
		x80, z80 := waterbed.RiverCenterPoint(seg, 0.80)
		x98, z98 := waterbed.RiverCenterPoint(seg, 0.98)
		s80 := finalRuntimeSample(model, x80, z80)
		s98 := finalRuntimeSample(model, x98, z98)

		rise := 0.0
		if wetRuntime(s80) && wetRuntime(s98) {
			rise = minecraftY(s98.World.WaterSurfaceElevation) -
				minecraftY(s80.World.WaterSurfaceElevation)
		}
		w80 := seg.HalfWidth
		w98 := seg.HalfWidth
		if nav, ok := model.Waterbed.NavigationSampleAt(x80, z80); ok {
			w80 = nav.HalfWidth
		}
		if nav, ok := model.Waterbed.NavigationSampleAt(x98, z98); ok {
			w98 = nav.HalfWidth
		}
		ratio := 1.0
		if w80 > 1e-9 {
			ratio = w98 / w80
		}
		riseSignal := rise > 0.25
		// If the approach sample is already owned by Ocean, the river channel
		// width is no longer the visible receiving-water width. Treating that
		// handoff as a narrowed river is a false signal.
		narrowSignal := ratio < 0.95 && s98.World.WaterKind == reconcile.WaterRiver
		if riseSignal {
			report.MouthApproachRises++
		}
		if narrowSignal {
			report.MouthNarrowingSignals++
		}
		if (riseSignal || narrowSignal) && len(report.MouthSignals) < cfg.MaxDetailRecords {
			report.MouthSignals = append(report.MouthSignals, MouthSignal{
				Segment:                   id,
				ApproachSurfaceRiseBlocks: rise,
				WidthAt80Blocks:           w80,
				WidthAt98Blocks:           w98,
				WidthRatio:                ratio,
				KindAt98:                  waterKindName(s98.World.WaterKind),
			})
		}
	}
}

func auditTransitions(model *generator.WorldModel, cfg Config, report *TransitionReport) {
	g := model.Geometry
	acc := make(map[string]*transitionAccumulator)
	add := func(aClass, bClass string, x0, z0, x1, z1 float64) {
		if aClass == bClass {
			return
		}
		from, to := orderedClasses(aClass, bClass)
		key := from + "->" + to
		item := acc[key]
		if item == nil {
			item = &transitionAccumulator{from: from, to: to}
			acc[key] = item
		}
		mx := (x0 + x1) * 0.5
		mz := (z0 + z1) * 0.5
		dx := x1 - x0
		dz := z1 - z0
		length := math.Hypot(dx, dz)
		if length <= 1e-9 {
			return
		}
		ux, uz := dx/length, dz/length
		probe := math.Min(cfg.TransitionProbeBlocks, length*0.25)
		left := finalRuntimeSample(model, mx-ux*probe, mz-uz*probe)
		middle := finalRuntimeSample(model, mx, mz)
		right := finalRuntimeSample(model, mx+ux*probe, mz+uz*probe)
		y0 := minecraftY(left.World.GroundHeight)
		ym := minecraftY(middle.World.GroundHeight)
		y1 := minecraftY(right.World.GroundHeight)
		slope := math.Atan2(math.Abs(y1-y0), 2*probe) * 180 / math.Pi
		curvature := math.Abs(y0 - 2*ym + y1)
		item.slopes = append(item.slopes, slope)
		item.curvatures = append(item.curvatures, curvature)
		item.maxSlope = math.Max(item.maxSlope, slope)
		item.maxCurvature = math.Max(item.maxCurvature, curvature)
		report.Samples++
	}

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			if !model.Landform.LandMask[i] {
				continue
			}
			wx, wz := g.NodePosition(x, z)
			class := landformClass(model, i)
			if x+1 < g.Width {
				j := z*g.Width + x + 1
				if model.Landform.LandMask[j] {
					nx, nz := g.NodePosition(x+1, z)
					add(class, landformClass(model, j), wx, wz, nx, nz)
				}
			}
			if z+1 < g.Height {
				j := (z+1)*g.Width + x
				if model.Landform.LandMask[j] {
					nx, nz := g.NodePosition(x, z+1)
					add(class, landformClass(model, j), wx, wz, nx, nz)
				}
			}
		}
	}

	keys := make([]string, 0, len(acc))
	for key := range acc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		a := acc[key]
		report.Classes = append(report.Classes, TransitionClassStats{
			From:               a.from,
			To:                 a.to,
			Count:              len(a.slopes),
			SlopeP50Deg:        percentile(a.slopes, 0.50),
			SlopeP90Deg:        percentile(a.slopes, 0.90),
			SlopeP99Deg:        percentile(a.slopes, 0.99),
			MaxSlopeDeg:        a.maxSlope,
			CurvatureP90Blocks: percentile(a.curvatures, 0.90),
			MaxCurvatureBlocks: a.maxCurvature,
		})
	}
}

func landformClass(model *generator.WorldModel, i int) string {
	type candidate struct {
		name   string
		weight float64
	}
	values := []candidate{
		{"plain", model.Landform.RollingPlainWeight[i]},
		{"hill", model.Landform.HillWeight[i]},
		{"foothill", model.Landform.FoothillWeight[i]},
		{"mountain", model.Landform.MountainWeight[i]},
		{"plateau", model.Landform.PlateauWeight[i]},
		{"valley", model.Landform.ValleyWeight[i]},
	}
	best := values[0]
	for _, v := range values[1:] {
		if v.weight > best.weight {
			best = v
		}
	}
	return best.name
}

func orderedClasses(a, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}

func finalRuntimeSample(model *generator.WorldModel, x, z float64) runtimeprofile.Sample {
	base := model.RuntimeProfile.Sample(x, z)
	if len(model.SpecialFeatures) == 0 {
		return base
	}
	return specialfeatures.Apply(
		model.SpecialFeatures,
		base,
		model.Config.SeaLevelNormalized,
		x,
		z,
		model.Config.WorldProfile.GenerationRulesVersion,
	)
}

func wetRuntime(s runtimeprofile.Sample) bool {
	return s.World.WaterSurfaceDefined &&
		s.World.WaterKind != reconcile.WaterNone
}

func waterKindName(kind reconcile.WaterKind) string {
	switch kind {
	case reconcile.WaterRiver:
		return "River"
	case reconcile.WaterLake:
		return "Lake"
	case reconcile.WaterOcean:
		return "Ocean"
	default:
		return "None"
	}
}

func minecraftY(height float64) float64 {
	height = math.Max(0, math.Min(1, height))
	if height <= 0.5 {
		return -64 + 127*(height/0.5)
	}
	return 63 + 257*((height-0.5)/0.5)
}

func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	pos := math.Max(0, math.Min(1, q)) * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	t := pos - float64(lo)
	return sorted[lo] + (sorted[hi]-sorted[lo])*t
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
