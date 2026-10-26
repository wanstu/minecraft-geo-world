package runtimeworld

import (
	"math"
	"path/filepath"
	"testing"

	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/worldmodelio"
)

var acceptanceSeeds = []int64{
	123456,
	-4852173505788415804,
	-563659577384754347,
	4034658472718554561,
	5228204294757649048,
}

func TestAcceptanceSeedsHaveDeterministicOuterIslandAndEndlessOcean(t *testing.T) {
	template, err := worldmodelio.Load(filepath.Join("..", "goldenvectors", "testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	if template.ClimatePreview == nil {
		t.Fatal("golden runtime model has no climate preview")
	}

	for _, seed := range acceptanceSeeds {
		seed := seed
		t.Run(seedName(seed), func(t *testing.T) {
			m, err := newSeededOuterWorld(template, seed)
			if err != nil {
				t.Fatal(err)
			}

			transition := outerTransitionBlocks(m)
			centerZ := m.minZ() + m.Geometry.SpanZBlocks*0.5

			for side := 0; side < 2; side++ {
				x, z := guaranteedIslandAnchor(m, side, transition)
				first := m.Sample(x, z)
				second := m.Sample(x, z)

				if first.World.WaterKind == reconcile.WaterOcean || !first.SourceLand {
					t.Fatalf("seed=%d side=%d anchor %.1f,%.1f is not land: water=%v sourceLand=%v biome=%s",
						seed, side, x, z, first.World.WaterKind, first.SourceLand, first.BiomeFamily)
				}
				if first.World.GroundHeight <= m.SeaLevelNormalized {
					t.Fatalf("seed=%d side=%d anchor %.1f,%.1f ground %.6f is not above sea %.6f",
						seed, side, x, z, first.World.GroundHeight, m.SeaLevelNormalized)
				}
				if first.World.WaterKind != second.World.WaterKind ||
					first.SourceLand != second.SourceLand ||
					first.BiomeFamily != second.BiomeFamily ||
					first.TerrainVariant != second.TerrainVariant ||
					math.Abs(first.World.GroundHeight-second.World.GroundHeight) > 1e-12 {
					t.Fatalf("seed=%d side=%d outer island sample is not deterministic", seed, side)
				}
			}

			// Sparse outer islands must not turn the infinite exterior back into
			// another continent. Beyond the island fade, the world is guaranteed
			// ocean indefinitely.
			far := m.Sample(m.maxX()+transition*3.0, centerZ)
			if far.World.WaterKind != reconcile.WaterOcean || far.SourceLand {
				t.Fatalf("seed=%d far exterior is not ocean: water=%v sourceLand=%v biome=%s",
					seed, far.World.WaterKind, far.SourceLand, far.BiomeFamily)
			}

			// North and south ultimately become endless frozen ocean. Sample well
			// beyond both the finite model and the sparse-island fade.
			polarDistance := math.Max(m.Geometry.SpanZBlocks, transition*3.0)
			for _, z := range []float64{
				m.minZ() - polarDistance,
				m.maxZ() + polarDistance,
			} {
				polar := m.Sample(m.minX()+m.Geometry.SpanXBlocks*0.5, z)
				if polar.World.WaterKind != reconcile.WaterOcean ||
					polar.SourceLand ||
					polar.BiomeFamily != mcprofile.BiomeFrozenOcean {
					t.Fatalf("seed=%d polar exterior z=%.1f: water=%v sourceLand=%v biome=%s; want FrozenOcean",
						seed, z, polar.World.WaterKind, polar.SourceLand, polar.BiomeFamily)
				}
			}
		})
	}
}

func newSeededOuterWorld(template *worldmodelio.RuntimeModel, seed int64) (*Model, error) {
	p := template.Config.WorldProfile.Parameters
	c := template.ClimatePreview
	return New(Config{
		Geometry:             template.Geometry,
		Seed:                 seed,
		SeaLevelNormalized:   template.Manifest.SeaLevelNormalized,
		TransitionWidth:      p.TransitionWidth,
		OuterIslandAbundance: p.OuterIslandAbundance,
		PolarTransitionWidth: p.PolarTransitionWidth,
		Finite:               template.RuntimeProfile,
		Climate: ClimateFields{
			AnnualMeanTempC: c.AnnualMeanTempC,
			AnnualPrecipMM:  c.AnnualPrecipMM,
			MonsoonIndex:    c.MonsoonIndex,
			WindX:           c.WindX,
			WindZ:           c.WindZ,
		},
	})
}

func outerTransitionBlocks(m *Model) float64 {
	cellSize := 0.5 * (m.Geometry.SpacingX() + m.Geometry.SpacingZ())
	return math.Max(2048, cellSize*24*m.TransitionWidth)
}

func guaranteedIslandAnchor(m *Model, side int, transition float64) (float64, float64) {
	key := int64(side + 1)
	distanceJitter := 0.5 + 0.5*hashValue(m.Seed+77201, key, 11)
	zJitter := hashValue(m.Seed+77203, key, 17)
	offshore := transition * (0.26 + 0.12*distanceJitter)

	x := m.minX() - offshore
	if side == 1 {
		x = m.maxX() + offshore
	}
	z := m.minZ() + m.Geometry.SpanZBlocks*0.5 + zJitter*0.22*m.Geometry.SpanZBlocks
	return x, z
}

func TestCurrentOuterIslandIsNotRadialFlatCap(t *testing.T) {
	template, err := worldmodelio.Load(filepath.Join("..", "goldenvectors", "testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := newSeededOuterWorld(template, 123456)
	if err != nil {
		t.Fatal(err)
	}
	transition := outerTransitionBlocks(m)
	cx, cz := guaranteedIslandAnchor(m, 1, transition)

	type sample struct {
		radius float64
		y      int
	}
	var samples []sample
	minY, maxY := 1<<30, -(1 << 30)
	for z := cz - 1000; z <= cz+1000; z += 16 {
		for x := cx - 1000; x <= cx+1000; x += 16 {
			s := m.Sample(x, z)
			if s.World.WaterKind != reconcile.WaterNone || !s.SourceLand {
				continue
			}
			y := minecraftBoundaryY(s.World.GroundHeight)
			samples = append(samples, sample{
				radius: math.Hypot(x-cx, z-cz),
				y:      y,
			})
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if len(samples) < 500 {
		t.Fatalf("outer island has too little sampled land: %d", len(samples))
	}
	if maxY-minY < 12 {
		t.Fatalf("outer island relief collapsed: minY=%d maxY=%d", minY, maxY)
	}

	top := 0
	var meanR, meanY float64
	for _, s := range samples {
		meanR += s.radius
		meanY += float64(s.y)
		if s.y >= maxY-1 {
			top++
		}
	}
	meanR /= float64(len(samples))
	meanY /= float64(len(samples))
	topRatio := float64(top) / float64(len(samples))
	if topRatio > 0.10 {
		t.Fatalf("outer island has flat-cap plateau: top-two-level ratio=%.3f", topRatio)
	}

	var covariance, varianceR, varianceY float64
	for _, s := range samples {
		dr := s.radius - meanR
		dy := float64(s.y) - meanY
		covariance += dr * dy
		varianceR += dr * dr
		varianceY += dy * dy
	}
	if varianceR > 0 && varianceY > 0 {
		correlation := covariance / math.Sqrt(varianceR*varianceY)
		if correlation < -0.80 {
			t.Fatalf("outer island remains radial/hemispherical: radius-height correlation=%.3f", correlation)
		}
	}
}

func TestCurrentOuterIslandCoastIsMostlyLowAndGentle(t *testing.T) {
	template, err := worldmodelio.Load(filepath.Join("..", "goldenvectors", "testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := newSeededOuterWorld(template, 123456)
	if err != nil {
		t.Fatal(err)
	}
	transition := outerTransitionBlocks(m)
	cx, cz := guaranteedIslandAnchor(m, 1, transition)
	const step = 8.0
	seaY := minecraftBoundaryY(m.SeaLevelNormalized)
	coastSamples := 0
	lowCoast := 0
	highCoast := 0
	for z := cz - 1100; z <= cz+1100; z += step {
		for x := cx - 1100; x <= cx+1100; x += step {
			s := m.Sample(x, z)
			if s.World.WaterKind != reconcile.WaterNone || !s.SourceLand {
				continue
			}
			isCoast := false
			for _, d := range [][2]float64{
				{step, 0},
				{-step, 0},
				{0, step},
				{0, -step},
			} {
				n := m.Sample(x+d[0], z+d[1])
				if n.World.WaterKind != reconcile.WaterNone || !n.SourceLand {
					isCoast = true
					break
				}
			}
			if !isCoast {
				continue
			}
			coastSamples++
			heightAboveSea := minecraftBoundaryY(s.World.GroundHeight) - seaY
			if heightAboveSea <= 4 {
				lowCoast++
			}
			if heightAboveSea > 10 {
				highCoast++
			}
		}
	}
	if coastSamples < 200 {
		t.Fatalf("outer island coast sample count too small: %d", coastSamples)
	}
	lowRatio := float64(lowCoast) / float64(coastSamples)
	highRatio := float64(highCoast) / float64(coastSamples)
	if lowRatio < 0.60 {
		t.Fatalf(
			"outer island reverted to broadly elevated coast: low-coast ratio=%.3f want>=0.60",
			lowRatio,
		)
	}
	if highRatio > 0.30 {
		t.Fatalf(
			"outer island has too much high/cliff coast: high-coast ratio=%.3f want<=0.30",
			highRatio,
		)
	}
}

func TestV116OuterOceanSeafloorKeepsLegacyFormula(t *testing.T) {
	template, err := worldmodelio.Load(filepath.Join("..", "goldenvectors", "testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := newSeededOuterWorld(template, 123456)
	if err != nil {
		t.Fatal(err)
	}
	m.GenerationRulesVersion = "geoworld-rules-v1.1.6"
	sea := m.SeaLevelNormalized
	deep := math.Max(0.08, sea-0.18)
	for _, point := range [][2]float64{
		{m.maxX() + 7000, m.minZ() + m.Geometry.SpanZBlocks*0.35},
		{m.maxX() + 19000, m.minZ() + m.Geometry.SpanZBlocks*0.55},
		{m.minX() - 31000, m.minZ() + m.Geometry.SpanZBlocks*0.68},
	} {
		want := deep + 0.018*valueNoise(m.Seed+77003, point[0], point[1], 5200)
		got := m.outerOceanSeafloor(point[0], point[1], sea, deep)
		if math.Abs(got-want) > 1e-12 {
			t.Fatalf(
				"v1.1.6 outer seafloor changed at %.1f,%.1f: got %.12f want %.12f",
				point[0], point[1], got, want,
			)
		}
	}
}

func TestV119OuterOceanSeafloorInheritsV118RegionalRelief(t *testing.T) {
	template, err := worldmodelio.Load(filepath.Join("..", "goldenvectors", "testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := newSeededOuterWorld(template, 123456)
	if err != nil {
		t.Fatal(err)
	}
	if m.GenerationRulesVersion != "geoworld-rules-v1.1.9" {
		t.Fatalf("current outer-world rules=%q want v1.1.9", m.GenerationRulesVersion)
	}

	sea := m.SeaLevelNormalized
	deep := math.Max(0.08, sea-0.18)
	points := [][2]float64{
		{m.maxX() + 9000, m.minZ() + m.Geometry.SpanZBlocks*0.31},
		{m.maxX() + 27000, m.minZ() + m.Geometry.SpanZBlocks*0.57},
		{m.minX() - 41000, m.minZ() + m.Geometry.SpanZBlocks*0.72},
	}
	current := make([]float64, len(points))
	for i, point := range points {
		current[i] = m.outerOceanSeafloor(point[0], point[1], sea, deep)
	}

	m.GenerationRulesVersion = "geoworld-rules-v1.1.8"
	for i, point := range points {
		locked := m.outerOceanSeafloor(point[0], point[1], sea, deep)
		if math.Abs(current[i]-locked) > 1e-12 {
			t.Fatalf(
				"v1.1.9 outer seafloor diverged from inherited v1.1.8 relief at %.1f,%.1f: current %.12f locked %.12f",
				point[0], point[1], current[i], locked,
			)
		}
	}
}

func TestV117OuterOceanSeafloorHasRegionalReliefWithoutEmergentLand(t *testing.T) {
	template, err := worldmodelio.Load(filepath.Join("..", "goldenvectors", "testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := newSeededOuterWorld(template, 123456)
	if err != nil {
		t.Fatal(err)
	}
	m.GenerationRulesVersion = "geoworld-rules-v1.1.7"
	if m.GenerationRulesVersion != "geoworld-rules-v1.1.7" {
		t.Fatalf("test requires v1.1.7, got %q", m.GenerationRulesVersion)
	}
	sea := m.SeaLevelNormalized
	deep := math.Max(0.08, sea-0.18)
	transition := outerTransitionBlocks(m)
	centerZ := m.minZ() + m.Geometry.SpanZBlocks*0.5
	startX := m.maxX() + transition*3.5

	minFloor := math.Inf(1)
	maxFloor := math.Inf(-1)
	for dz := -60000.0; dz <= 60000; dz += 2500 {
		for dx := 0.0; dx <= 120000; dx += 2500 {
			x := startX + dx
			z := centerZ + dz
			floor := m.outerOceanSeafloor(x, z, sea, deep)
			minFloor = math.Min(minFloor, floor)
			maxFloor = math.Max(maxFloor, floor)
			if floor >= sea-0.034999999 {
				t.Fatalf(
					"outer seafloor emerged too close to sea at %.1f,%.1f: floor=%.6f sea=%.6f",
					x, z, floor, sea,
				)
			}
		}
	}
	if maxFloor-minFloor < 0.045 {
		t.Fatalf(
			"outer seafloor lacks regional relief: min=%.6f max=%.6f range=%.6f",
			minFloor, maxFloor, maxFloor-minFloor,
		)
	}

	// Beyond the island fade, the richer seafloor must still remain infinite
	// ocean rather than accidentally creating a second continent.
	for dz := -transition; dz <= transition; dz += transition / 4 {
		for dx := 0.0; dx <= transition*2; dx += transition / 4 {
			s := m.Sample(startX+dx, centerZ+dz)
			if s.World.WaterKind != reconcile.WaterOcean || s.SourceLand {
				t.Fatalf(
					"v1.1.7 outer seafloor created land at %.1f,%.1f: water=%v sourceLand=%v",
					startX+dx, centerZ+dz, s.World.WaterKind, s.SourceLand,
				)
			}
		}
	}
}

func minecraftBoundaryY(normalized float64) int {
	h := clamp(normalized, 0, 1)
	if h <= 0.5 {
		return int(math.Round(-64 + 127*(h/0.5)))
	}
	return int(math.Round(63 + 257*((h-0.5)/0.5)))
}

func seedName(seed int64) string {
	if seed < 0 {
		return "negative_" + formatUint(uint64(-seed))
	}
	return formatUint(uint64(seed))
}

func formatUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
