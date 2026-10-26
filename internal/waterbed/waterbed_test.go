package waterbed

import (
	"math"
	"testing"

	"geoworld/internal/hydrology"
	"geoworld/internal/reconcile"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

func TestRareMacroLakeScaleIsLockedAndPhysical(t *testing.T) {
	large := hydrology.Lake{
		ID:          7,
		BasinID:     19,
		SpillOutlet: 113,
		AreaBlocks2: 2_000_000,
	}
	if got := rareMacroLakeRadiusScale(123456, 0, large); got != 1 {
		t.Fatalf("locked pre-v1.1.7 lake scale=%v want=1", got)
	}
	small := large
	small.AreaBlocks2 = 400_000
	if got := rareMacroLakeRadiusScale(123456, 1, small); got != 1 {
		t.Fatalf("small lake unexpectedly became rare macro lake: scale=%v", got)
	}
	standard := rareMacroLakeRadiusScale(123456, 1, large)
	if standard < 3.4 || standard > 3.7 {
		t.Fatalf("large natural basin did not become inland-sea scale: %.6f", standard)
	}
	epic := rareMacroLakeRadiusScale(123456, 1.45, large)
	if epic <= standard {
		t.Fatalf("rare macro strength did not scale giant lake: standard=%.6f epic=%.6f", standard, epic)
	}
}

func TestV5FluvialDetailExtendsBeyondLegacyBank(t *testing.T) {
	legacy := RiverSegment{
		X0: 0, Z0: 0, X1: 256, Z1: 0,
		Bed0: 0.55, Bed1: 0.55,
		HalfWidth:             32,
		MouthExpansionVersion: 4,
	}
	modern := legacy
	modern.MouthExpansionVersion = 5

	legacyBank := riverMouthHalfWidth(legacy, 0.5) +
		riverBankBlendBlocks(legacy.HalfWidth)
	if got := riverFluvialReach(legacy); math.Abs(got-legacyBank) > 1e-12 {
		t.Fatalf("v4 reach changed: got %.12f want %.12f", got, legacyBank)
	}
	if strength := riverFluvialDetailStrength(legacy); strength != 0 {
		t.Fatalf("v4 unexpectedly has fluvial detail strength %.6f", strength)
	}
	if strength := riverFluvialDetailStrength(modern); strength < 0.9 {
		t.Fatalf("flat wide v5 reach should strongly support fluvial detail: %.6f", strength)
	}
	if got := riverFluvialReach(modern); got <= legacyBank+40 {
		t.Fatalf("v5 floodplain reach did not extend materially: bank=%.3f reach=%.3f", legacyBank, got)
	}
}

func TestV5NaturalLeveeIsSparseInterruptedAndLegacyLocked(t *testing.T) {
	legacy := RiverSegment{
		X0: 0, Z0: 0, X1: 256, Z1: 0,
		Bed0: 0.55, Bed1: 0.55,
		HalfWidth:             32,
		MouthExpansionVersion: 4,
	}
	if riverNaturalLeveeEnabled(legacy) {
		t.Fatal("v4 must not enable natural levee refinement")
	}
	if got := riverNaturalLeveeLift(
		legacy, 0.5, 128, 64, 0.10,
	); got != 0 {
		t.Fatalf("v4 natural levee lift=%v want=0", got)
	}

	modern := legacy
	modern.MouthExpansionVersion = 5
	found := false
	for x0 := 0; x0 < 10000; x0++ {
		modern.X0 = float64(x0)
		modern.X1 = float64(x0 + 256)
		if riverNaturalLeveeEnabled(modern) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("failed to find deterministic natural-levee reach")
	}

	maxLift := 0.0
	zeroSamples := 0
	asymmetricSamples := 0
	for i := 4; i <= 16; i++ {
		tValue := float64(i) / 20
		cx, cz, nx, nz := RiverCenterFrame(modern, tValue)
		bankRadius := riverMouthHalfWidth(modern, tValue) +
			riverBankBlendBlocks(modern.HalfWidth)
		reach := riverFluvialReach(modern)
		distance := bankRadius + 0.10*(reach-bankRadius)

		right := riverNaturalLeveeLift(
			modern,
			tValue,
			cx+nx*distance,
			cz+nz*distance,
			0.10,
		)
		left := riverNaturalLeveeLift(
			modern,
			tValue,
			cx-nx*distance,
			cz-nz*distance,
			0.10,
		)
		maxLift = math.Max(maxLift, math.Max(right, left))
		if right < 1e-9 || left < 1e-9 {
			zeroSamples++
		}
		if math.Abs(right-left) > 2e-4 {
			asymmetricSamples++
		}
	}
	if maxLift < 8e-4 {
		t.Fatalf("natural levee never became material: max=%.6f", maxLift)
	}
	if maxLift > 0.0042+1e-12 {
		t.Fatalf("natural levee exceeded bounded lift: %.6f", maxLift)
	}
	if zeroSamples == 0 {
		t.Fatal("natural levee became a continuous artificial embankment")
	}
	if asymmetricSamples == 0 {
		t.Fatal("natural levee banks are perfectly mirrored")
	}

	baseHeight := 0.62
	surface := 0.55
	tValue := 0.5
	cx, cz, nx, nz := RiverCenterFrame(modern, tValue)
	bankRadius := riverMouthHalfWidth(modern, tValue) +
		riverBankBlendBlocks(modern.HalfWidth)
	reach := riverFluvialReach(modern)
	distance := bankRadius + 0.10*(reach-bankRadius)
	got := applyRiverFluvialDryDetail(
		modern,
		tValue,
		cx+nx*distance,
		cz+nz*distance,
		baseHeight,
		baseHeight,
		surface,
		distance,
	)
	if got > baseHeight+1e-12 {
		t.Fatalf(
			"natural levee built above original terrain: got=%.6f base=%.6f",
			got,
			baseHeight,
		)
	}
}

func TestV5FluvialDetailRespectsOtherRiverBankProtection(t *testing.T) {
	parent := RiverSegment{
		X0: 0, Z0: 0, X1: 256, Z1: 0,
		Bed0: 0.55, Bed1: 0.55,
		HalfWidth:             32,
		MouthExpansionVersion: 5,
	}
	sibling := parent
	sibling.Z0 = 60
	sibling.Z1 = 60
	rivers := []RiverSegment{parent, sibling}

	worldX, worldZ := 128.0, 60.0
	parentT, parentDistance := pointRiverSegment(worldX, worldZ, parent)
	parentBank := riverMouthHalfWidth(parent, parentT) +
		riverBankBlendBlocks(parent.HalfWidth)
	if parentDistance < parentBank {
		t.Fatalf(
			"probe is unexpectedly inside parent bank: distance=%.3f bank=%.3f",
			parentDistance,
			parentBank,
		)
	}
	if parentDistance >= riverFluvialReach(parent) {
		t.Fatalf(
			"probe does not exercise parent fluvial reach: distance=%.3f reach=%.3f",
			parentDistance,
			riverFluvialReach(parent),
		)
	}
	if !riverBankProtectedAt(rivers, []int{0, 1}, worldX, worldZ) {
		t.Fatal("sibling active river bank did not protect overlapping fluvial detail")
	}
	if riverBankProtectedAt(rivers, []int{0, 1}, worldX, 140) {
		t.Fatal("point outside both active banks was incorrectly protected")
	}
}

func TestV5DeltaBranchesSplitContinuouslyAndLegacyStaysLocked(t *testing.T) {
	seg := RiverSegment{
		Z0: 0, Z1: 0,
		Bed0: 0.55, Bed1: 0.55,
		HalfWidth:             32,
		OceanMouth:            true,
		MouthExpansionVersion: 5,
	}
	found := false
	for x0 := 0; x0 < 10000; x0++ {
		seg.X0 = float64(x0)
		seg.X1 = float64(x0 + 256)
		if riverDeltaBranchCount(seg) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("failed to find deterministic delta-eligible ocean mouth")
	}

	legacy := seg
	legacy.MouthExpansionVersion = 4
	if got := riverDeltaBranchCount(legacy); got != 0 {
		t.Fatalf("v1.1.6/v4 unexpectedly gained delta branches: %d", got)
	}
	if got := riverDeltaSupportRadius(legacy); got != 0 {
		t.Fatalf("v1.1.6/v4 unexpectedly gained delta support: %.6f", got)
	}

	count := riverDeltaBranchCount(seg)
	if count < 1 || count > 2 {
		t.Fatalf("delta branch count=%d want 1..2", count)
	}

	overlapsParent := false
	for step := 47; step <= 70; step++ {
		tValue := float64(step) / 100
		offset, width, _, ok := riverDeltaBranchGeometry(seg, tValue, 0)
		if !ok {
			continue
		}
		if math.Abs(offset) <= riverMouthHalfWidth(seg, tValue)+width {
			overlapsParent = true
			break
		}
	}
	if !overlapsParent {
		t.Fatal("delta branch never overlaps the parent mouth near its split")
	}

	offset, width, bank, ok := riverDeltaBranchGeometry(seg, 1, 0)
	if !ok {
		t.Fatal("delta branch is not defined at the ocean receiver")
	}
	mainWidth := riverMouthHalfWidth(seg, 1)
	if math.Abs(offset) <= mainWidth+width {
		t.Fatalf(
			"delta branch did not separate from parent by the receiver: offset=%.3f main=%.3f branch=%.3f",
			offset,
			mainWidth,
			width,
		)
	}

	cx, cz, nx, nz := RiverCenterFrame(seg, 1)
	worldX := cx + nx*offset
	worldZ := cz + nz*offset
	branch, distance, gotWidth, gotBank, ok :=
		riverDeltaPoint(seg, 1, worldX, worldZ)
	if !ok || branch != 0 || math.Abs(distance) > 1e-9 {
		t.Fatalf(
			"delta centerline lookup failed: branch=%d distance=%.12f ok=%v",
			branch,
			distance,
			ok,
		)
	}
	if math.Abs(gotWidth-width) > 1e-12 ||
		math.Abs(gotBank-bank) > 1e-12 {
		t.Fatalf(
			"delta geometry drifted between branch and point lookup: width %.12f/%.12f bank %.12f/%.12f",
			gotWidth,
			width,
			gotBank,
			bank,
		)
	}

	mainBank := mainWidth + riverBankBlendBlocks(seg.HalfWidth)
	deltaOuterEdge := math.Abs(offset) + bank
	if deltaOuterEdge <= mainBank {
		t.Fatalf(
			"delta bank never extends beyond parent bank: outer=%.3f parentBank=%.3f",
			deltaOuterEdge,
			mainBank,
		)
	}
	protectedLateral := 0.5 * (mainBank + deltaOuterEdge)
	if offset < 0 {
		protectedLateral = -protectedLateral
	}
	protectedX := cx + nx*protectedLateral
	protectedZ := cz + nz*protectedLateral
	_, parentDistance := pointRiverSegment(protectedX, protectedZ, seg)
	if parentDistance <= mainBank {
		t.Fatalf(
			"delta-only protection probe remained inside parent bank: distance=%.3f parentBank=%.3f",
			parentDistance,
			mainBank,
		)
	}
	if !riverBankProtectedAt(
		[]RiverSegment{seg},
		[]int{0},
		protectedX,
		protectedZ,
	) {
		t.Fatal("delta distributary outer bank is not part of active river protection")
	}

	for _, tValue := range []float64{0.60, 0.80, 1.0} {
		bed := sampleRiverProfile(seg, tValue)
		naturalSurface := bed + riverCenterDepth(
			Config{
				MinRiverHalfWidthBlocks:    4,
				MaxRiverHalfWidthBlocks:    48,
				RiverCenterDepthNormalized: 0.004,
			},
			seg.HalfWidth,
		)
		surface := riverConnectionSurface(seg, tValue, naturalSurface)
		if math.Abs(surface-(0.55+riverCenterDepth(
			Config{
				MinRiverHalfWidthBlocks:    4,
				MaxRiverHalfWidthBlocks:    48,
				RiverCenterDepthNormalized: 0.004,
			},
			seg.HalfWidth,
		))) > 1e-12 {
			t.Fatalf("flat synthetic delta datum drifted at t=%.2f: %.12f", tValue, surface)
		}
	}
}

func TestV1LakeCoreCarveChoosesOnlyPlausibleDryAnchor(t *testing.T) {
	lakes := []LakeNode{
		{LakeID: 1, X: 0, WaterSurface: 0.50},
		{LakeID: 1, X: 1, WaterSurface: 0.50},
		{LakeID: 1, X: 2, WaterSurface: 0.50},
		{LakeID: 2, X: 3, WaterSurface: 0.50},
		{LakeID: 3, X: 4, WaterSurface: 0.50},
		{LakeID: 3, X: 5, WaterSurface: 0.50},
	}
	heights := map[float64]float64{
		0: 0.535,
		1: 0.508,
		2: 0.521,
		3: 0.501,
		4: 0.530,
		5: 0.525,
	}
	sampler := func(x, _ float64) float64 { return heights[x] }

	legacy := append([]LakeNode(nil), lakes...)
	markLakeCoreCarveAnchors(
		Config{LakeGeometryVersion: 0},
		legacy,
		sampler,
	)
	for i := range legacy {
		if legacy[i].CoreCarve {
			t.Fatalf("legacy lake node %d unexpectedly gained core carve", i)
		}
	}

	modern := append([]LakeNode(nil), lakes...)
	markLakeCoreCarveAnchors(
		Config{LakeGeometryVersion: 1},
		modern,
		sampler,
	)
	if !modern[1].CoreCarve {
		t.Fatal("dry lake did not choose its lowest plausible core anchor")
	}
	if modern[0].CoreCarve || modern[2].CoreCarve {
		t.Fatal("dry lake selected more than one core anchor")
	}
	if modern[3].CoreCarve {
		t.Fatal("naturally renderable lake should not force a core carve")
	}
	if modern[4].CoreCarve || modern[5].CoreCarve {
		t.Fatal("implausibly high lake group should remain unforced")
	}
}

func TestV1LakeBedCeilingGuaranteesVisibleDepthWithoutChangingLegacy(t *testing.T) {
	const (
		surface = 0.509226
		shallow = 0.509126
	)
	legacy := visibleLakeBedCeiling(
		Config{
			MinLakeWaterDepthNormalized: 0.003,
			LakeGeometryVersion:         0,
		},
		surface,
		shallow,
	)
	if legacy != shallow {
		t.Fatalf("legacy lake bed changed: %.12f want %.12f", legacy, shallow)
	}

	modernCfg := Config{
		MinLakeWaterDepthNormalized: 0.003,
		LakeGeometryVersion:         1,
	}
	modern := visibleLakeBedCeiling(modernCfg, surface, shallow)
	if modern >= surface-modernCfg.MinLakeWaterDepthNormalized {
		t.Fatalf(
			"modern lake still cannot expose water: bed=%.12f surface=%.12f minDepth=%.12f",
			modern,
			surface,
			modernCfg.MinLakeWaterDepthNormalized,
		)
	}
	if got := surface - modern; got < 0.004-1e-12 {
		t.Fatalf("modern visible lake depth=%.12f want >=0.004", got)
	}
}

func TestV5OxbowWaterIsClosedAndUsesOneFlatDatum(t *testing.T) {
	cfg := Config{
		MinRiverHalfWidthBlocks:     4,
		MaxRiverHalfWidthBlocks:     48,
		RiverCenterDepthNormalized:  0.004,
		MinLakeWaterDepthNormalized: 0.003,
	}
	seg := RiverSegment{
		Z0: 0, Z1: 0,
		Bed0: 0.55, Bed1: 0.55,
		HalfWidth:             32,
		MeanderOffset:         18,
		MouthExpansionVersion: 5,
	}
	found := false
	for x0 := 0; x0 < 10000; x0++ {
		seg.X0 = float64(x0)
		seg.X1 = float64(x0 + 256)
		key := riverFluvialKey(seg)
		if key%13 == 0 && (key/13)%3 == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("failed to find deterministic oxbow-eligible segment")
	}

	pointOnOxbow := func(tValue float64) (float64, float64) {
		cx, cz, nx, nz := RiverCenterFrame(seg, tValue)
		offset := riverAbandonedMeanderOffset(
			seg,
			tValue,
			riverFluvialKey(seg),
		)
		return cx + nx*offset, cz + nz*offset
	}

	x1, z1 := pointOnOxbow(0.35)
	_, activeDistance1 := pointRiverSegment(x1, z1, seg)
	surface1, floor1, weight1, ok1 :=
		riverOxbowWater(cfg, seg, 0.35, x1, z1, activeDistance1)
	x2, z2 := pointOnOxbow(0.65)
	_, activeDistance2 := pointRiverSegment(x2, z2, seg)
	surface2, floor2, weight2, ok2 :=
		riverOxbowWater(cfg, seg, 0.65, x2, z2, activeDistance2)
	if !ok1 || !ok2 || weight1 < 0.5 || weight2 < 0.5 {
		t.Fatalf(
			"eligible oxbow middle is not wet enough: ok=%v/%v weight=%.3f/%.3f",
			ok1, ok2, weight1, weight2,
		)
	}
	if math.Abs(surface1-surface2) > 1e-12 ||
		math.Abs(floor1-floor2) > 1e-12 {
		t.Fatalf(
			"oxbow datum slopes: surface %.12f/%.12f floor %.12f/%.12f",
			surface1, surface2, floor1, floor2,
		)
	}

	xEnd, zEnd := pointOnOxbow(0.05)
	_, activeDistanceEnd := pointRiverSegment(xEnd, zEnd, seg)
	if _, _, _, ok := riverOxbowWater(
		cfg, seg, 0.05, xEnd, zEnd, activeDistanceEnd,
	); ok {
		t.Fatal("oxbow must close before the parent reach endpoint")
	}

	tMid := 0.5
	cxMid, czMid, nxMid, nzMid := RiverCenterFrame(seg, tMid)
	halfMid := riverMouthHalfWidth(seg, tMid)
	bankMid := halfMid + riverBankBlendBlocks(seg.HalfWidth)
	insideDistance := 0.5 * (halfMid + bankMid)
	insideX := cxMid + nxMid*insideDistance
	insideZ := czMid + nzMid*insideDistance
	if _, _, _, ok := riverOxbowWater(
		cfg, seg, tMid, insideX, insideZ, insideDistance,
	); ok {
		t.Fatal("oxbow must not invade the active river bank protection corridor")
	}
	if got := applyRiverFluvialDryDetail(
		seg,
		tMid,
		insideX,
		insideZ,
		0.62,
		0.60,
		0.55,
		insideDistance,
	); math.Abs(got-0.60) > 1e-12 {
		t.Fatalf(
			"fluvial dry detail lowered active bank corridor: got %.12f want 0.60",
			got,
		)
	}

	legacy := seg
	legacy.MouthExpansionVersion = 4
	xMid, zMid := pointOnOxbow(0.5)
	_, legacyDistance := pointRiverSegment(xMid, zMid, legacy)
	if _, _, _, ok := riverOxbowWater(
		cfg, legacy, 0.5, xMid, zMid, legacyDistance,
	); ok {
		t.Fatal("v1.1.6/v4 must not gain oxbow water")
	}
}

func TestOceanMouthFlaresOnlyOnFinalReach(t *testing.T) {
	upstream := RiverSegment{HalfWidth: 6}
	mouth := RiverSegment{HalfWidth: 6, OceanMouth: true}
	if got := riverMouthHalfWidth(upstream, 1); got != 6 {
		t.Fatalf("ordinary upstream river widened: %v", got)
	}
	if got := riverMouthHalfWidth(mouth, 0.10); got != 6 {
		t.Fatalf("ocean mouth must retain ordinary upstream width: %v", got)
	}
	mid := riverMouthHalfWidth(mouth, 0.5)
	end := riverMouthHalfWidth(mouth, 0.9)
	if mid <= 6 || end <= mid || end > mouthMaxHalfWidth(mouth) {
		t.Fatalf("river mouth did not expand smoothly: mid=%v end=%v", mid, end)
	}
	if end != 15 {
		t.Fatalf("expected ocean mouth +9 blocks half width, got=%v", end)
	}
}

func TestV6LargeLakeMouthScalesBeyondLegacyFixedCap(t *testing.T) {
	legacy := RiverSegment{
		Bed0:                  0.50,
		Bed1:                  0.50,
		HalfWidth:             47.7,
		LakeMouth:             true,
		MouthExpansionVersion: 3,
	}
	modern := legacy
	modern.MouthExpansionVersion = 6

	legacyMax := mouthMaxHalfWidth(legacy)
	modernMax := mouthMaxHalfWidth(modern)
	if math.Abs(legacyMax-(legacy.HalfWidth+42)) > 1e-12 {
		t.Fatalf("v1.1.7 lake-mouth cap changed: got=%.6f", legacyMax)
	}
	if modernMax <= legacyMax {
		t.Fatalf(
			"v1.1.8 large lake mouth did not outgrow legacy cap: legacy=%.6f modern=%.6f",
			legacyMax,
			modernMax,
		)
	}

	transition := riverMouthHalfWidth(modern, 0.85)
	if ratio := transition / modern.HalfWidth; ratio < 2.0 {
		t.Fatalf(
			"v1.1.8 large lake-mouth transition remains too narrow: width=%.6f base=%.6f ratio=%.6f",
			transition,
			modern.HalfWidth,
			ratio,
		)
	}

	if got := mouthMaxHalfWidth(legacy); got != legacyMax {
		t.Fatalf("legacy v3 lake-mouth geometry became unstable: %.6f != %.6f", got, legacyMax)
	}
}

func TestV3LakeOutletOceanMouthNeverNarrows(t *testing.T) {
	seg := RiverSegment{
		X0:                    0,
		Z0:                    0,
		X1:                    128,
		Z1:                    0,
		Bed0:                  0.5,
		Bed1:                  0.5,
		HalfWidth:             45,
		LakeOutlet:            true,
		OceanMouth:            true,
		MouthExpansionVersion: 3,
	}
	prev := riverMouthHalfWidth(seg, 0)
	for i := 1; i <= 100; i++ {
		tValue := float64(i) / 100
		got := riverMouthHalfWidth(seg, tValue)
		if got+1e-12 < prev {
			t.Fatalf(
				"combined lake-outlet/ocean-mouth narrowed at t=%.2f: prev=%.6f got=%.6f",
				tValue,
				prev,
				got,
			)
		}
		prev = got
	}
	if start, end := riverMouthHalfWidth(seg, 0), riverMouthHalfWidth(seg, 1); end+1e-12 < start {
		t.Fatalf("combined estuary ends narrower than it starts: start=%.6f end=%.6f", start, end)
	}
}

func TestMajorRiverHalfWidthCanExceedOrdinaryMaximum(t *testing.T) {
	cfg := Config{
		MinRiverHalfWidthBlocks: 4,
		MaxRiverHalfWidthBlocks: 18,
	}
	ordinary := riverHalfWidth(cfg, 0.30, 1.0, 0, 5000, 0)
	major := riverHalfWidth(cfg, 0.30, 1.0, 160000, 5000, 5)
	if ordinary != cfg.MaxRiverHalfWidthBlocks {
		t.Fatalf("ordinary full-strength river width=%v want=%v", ordinary, cfg.MaxRiverHalfWidthBlocks)
	}
	if major <= cfg.MaxRiverHalfWidthBlocks {
		t.Fatalf("major river width=%v did not exceed ordinary maximum=%v", major, cfg.MaxRiverHalfWidthBlocks)
	}
	if major < 45 {
		t.Fatalf("major river width=%v is still too small for a great-river reach", major)
	}
	if major > cfg.MaxRiverHalfWidthBlocks*2.65+1e-12 {
		t.Fatalf("major river width=%v exceeded intended cap", major)
	}
}

func TestV115DiagonalStraightReachGetsAntiGridDeflection(t *testing.T) {
	const (
		seed         int64 = 123456
		from               = 24493
		to                 = 24172
		x0                 = 1728.0
		z0                 = -3008.0
		x1                 = 1600.0
		z1                 = -3136.0
		halfWidth          = 47.7
		originalDrop       = 0.009
	)

	legacy := riverMeanderOffset(
		seed,
		from,
		to,
		x0, z0,
		x1, z1,
		halfWidth,
		originalDrop,
		0,
	)
	if legacy != 0 {
		t.Fatalf("legacy diagonal straight reach meander=%v want=0", legacy)
	}

	modern := riverMeanderOffset(
		seed,
		from,
		to,
		x0, z0,
		x1, z1,
		halfWidth,
		originalDrop,
		1,
	)
	if math.Abs(modern) < 4 {
		t.Fatalf("v1.1.5 anti-grid meander=%v want magnitude >=4", modern)
	}
	if modern >= 0 {
		t.Fatalf(
			"v1.1.5 anti-grid meander=%v should favour 0/1 downstream raster steps for same-sign diagonal",
			modern,
		)
	}

	cardinal := riverMeanderOffset(
		seed,
		from,
		to,
		x0, z0,
		x0-128, z0,
		halfWidth,
		originalDrop,
		1,
	)
	if cardinal != 0 {
		t.Fatalf("cardinal legacy-straight reach meander=%v want=0", cardinal)
	}
}

func TestV115DiagonalRasterBreaksBothBankZippers(t *testing.T) {
	seg := RiverSegment{
		X0:                    1728,
		Z0:                    -3008,
		X1:                    1600,
		Z1:                    -3136,
		HalfWidth:             47.7,
		MeanderOffset:         -16.93,
		MouthExpansionVersion: 4,
	}
	leftEdges := make([]int, 0, 31)
	rightEdges := make([]int, 0, 31)
	for z := -3094; z <= -3064; z++ {
		left, right := 0, 0
		found := false
		for x := 1350; x <= 1800; x++ {
			tValue, distance := pointRiverSegment(float64(x), float64(z), seg)
			if distance > riverMouthHalfWidth(seg, tValue) {
				continue
			}
			if !found {
				left = x
				found = true
			}
			right = x
		}
		if !found {
			t.Fatalf("no wet raster row at z=%d", z)
		}
		leftEdges = append(leftEdges, left)
		rightEdges = append(rightEdges, right)
	}
	checkBank := func(name string, edges []int) {
		repeated := 0
		maxStep := 0
		longestSameStep := 0
		currentRun := 0
		previousStep := 999
		for i := 1; i < len(edges); i++ {
			step := edges[i] - edges[i-1]
			if step == 0 {
				repeated++
			}
			stepAbs := step
			if stepAbs < 0 {
				stepAbs = -stepAbs
			}
			if stepAbs > maxStep {
				maxStep = stepAbs
			}
			if step == previousStep {
				currentRun++
			} else {
				currentRun = 1
				previousStep = step
			}
			if currentRun > longestSameStep {
				longestSameStep = currentRun
			}
		}
		if repeated < 3 {
			t.Fatalf("%s bank still forms a 1x1 zipper: repeated=%d edges=%v", name, repeated, edges)
		}
		if maxStep > 2 {
			t.Fatalf("%s bank introduced a jump >2 blocks: max=%d edges=%v", name, maxStep, edges)
		}
		if longestSameStep > 5 {
			t.Fatalf("%s bank keeps one voxel phase too long: run=%d edges=%v", name, longestSameStep, edges)
		}
	}
	checkBank("right", rightEdges)
	_ = leftEdges
}

func TestV3MeanderKeepsEndpointTangentsChordAligned(t *testing.T) {
	seg := RiverSegment{
		X0:                    0,
		Z0:                    0,
		X1:                    128,
		Z1:                    64,
		MeanderOffset:         18,
		MouthExpansionVersion: 3,
	}
	length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
	wantNX := -(seg.Z1 - seg.Z0) / length
	wantNZ := (seg.X1 - seg.X0) / length
	for _, tValue := range []float64{0, 1} {
		_, _, nx, nz := riverCenterAndNormalForSegment(seg, tValue)
		if math.Abs(nx-wantNX) > 1e-12 || math.Abs(nz-wantNZ) > 1e-12 {
			t.Fatalf(
				"v3 meander endpoint t=%v changed chord tangent: normal=(%.12f,%.12f) want=(%.12f,%.12f)",
				tValue,
				nx,
				nz,
				wantNX,
				wantNZ,
			)
		}
	}
	_, _, legacyNX, legacyNZ := riverCenterAndNormalForSegment(
		RiverSegment{
			X0:            seg.X0,
			Z0:            seg.Z0,
			X1:            seg.X1,
			Z1:            seg.Z1,
			MeanderOffset: seg.MeanderOffset,
		},
		0,
	)
	if math.Abs(legacyNX-wantNX) <= 1e-6 &&
		math.Abs(legacyNZ-wantNZ) <= 1e-6 {
		t.Fatal("legacy meander unexpectedly lost its endpoint tangent behavior")
	}
}

func TestLakeOutletKeepsBroadWetNearShoreThroat(t *testing.T) {
	seg := RiverSegment{
		X0:                       0,
		Z0:                       0,
		X1:                       181,
		Z1:                       0,
		HalfWidth:                18,
		LakeOutlet:               true,
		LakeOutletSurfaceDefined: true,
		LakeOutletSurface:        0.508,
		MouthExpansionVersion:    2,
	}
	if got := riverMouthHalfWidth(seg, 0); got <= 39 {
		t.Fatalf("lake outlet source half-width=%v want >39", got)
	}
	if got := riverMouthHalfWidth(seg, 82.0/181.0); got <= 39 {
		t.Fatalf("lake outlet near-shore half-width=%v want >39", got)
	}
	if got := riverMouthHalfWidth(seg, 1); math.Abs(got-18) > 1e-12 {
		t.Fatalf("lake outlet downstream half-width=%v want=18", got)
	}
	if got := riverConnectionSurface(seg, 0, 0.504); math.Abs(got-0.508) > 1e-12 {
		t.Fatalf("lake outlet source surface=%v want=0.508", got)
	}
	if got := riverConnectionSurface(seg, 1, 0.504); math.Abs(got-0.504) > 1e-12 {
		t.Fatalf("lake outlet downstream surface=%v want=0.504", got)
	}
}

func TestLakeOutletBackwaterCarriesAcrossConfluence(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	outlet := RiverSegment{
		From:                     1,
		To:                       2,
		X0:                       0,
		Z0:                       0,
		X1:                       128,
		Z1:                       0,
		Bed0:                     0.5015,
		Bed1:                     0.5015,
		HalfWidth:                4,
		Profile:                  []float64{0.5015, 0.5015},
		LakeOutlet:               true,
		LakeOutletSurfaceDefined: true,
		LakeOutletSurface:        0.5100,
		MouthExpansionVersion:    3,
	}
	tributary := RiverSegment{
		From:      3,
		To:        2,
		X0:        0,
		Z0:        128,
		X1:        128,
		Z1:        0,
		Bed0:      0.497,
		Bed1:      0.497,
		HalfWidth: 9,
		Profile:   []float64{0.497, 0.497},
	}
	downstream := RiverSegment{
		From:      2,
		To:        4,
		X0:        128,
		Z0:        0,
		X1:        256,
		Z1:        0,
		Bed0:      0.497,
		Bed1:      0.497,
		HalfWidth: 11,
		Profile:   []float64{0.497, 0.497},
	}
	rivers := []RiverSegment{outlet, tributary, downstream}
	incoming := make([][]int, 5)
	incoming[2] = []int{0, 1}
	incoming[4] = []int{2}

	naturalOutlet :=
		sampleRiverProfile(outlet, 1) +
			riverCenterDepth(cfg, outlet.HalfWidth)
	want := riverConnectionSurface(outlet, 1, naturalOutlet)
	if want <= naturalOutlet+1e-12 {
		t.Fatalf(
			"test fixture has no surviving lake-outlet backwater: natural=%.6f effective=%.6f",
			naturalOutlet,
			want,
		)
	}

	seedLakeOutletConfluenceBackwater(rivers, incoming, cfg)
	for _, id := range []int{0, 1} {
		if !rivers[id].ReceiverSurfaceDefined ||
			math.Abs(rivers[id].ReceiverSurface-want) > 1e-12 {
			t.Fatalf(
				"incoming segment %d did not inherit shared junction datum: got=%v/%.6f want=%.6f",
				id,
				rivers[id].ReceiverSurfaceDefined,
				rivers[id].ReceiverSurface,
				want,
			)
		}
	}

	easeReceiverOutflowDrops(rivers, cfg)
	downstreamStart :=
		sampleRiverProfile(rivers[2], 0) +
			riverCenterDepth(cfg, rivers[2].HalfWidth)
	if downstreamStart < want-1e-12 {
		t.Fatalf(
			"downstream start did not inherit outlet backwater: got=%.6f want>=%.6f",
			downstreamStart,
			want,
		)
	}
}

func TestLakeOutletBackwaterRecomputedAfterProfileSettles(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	outlet := RiverSegment{
		From:                     1,
		To:                       2,
		X0:                       0,
		Z0:                       0,
		X1:                       220,
		Z1:                       0,
		Bed0:                     0.540,
		Bed1:                     0.510,
		HalfWidth:                18,
		Profile:                  []float64{0.540, 0.510},
		LakeOutlet:               true,
		LakeOutletSurfaceDefined: true,
		LakeOutletSurface:        0.545,
		MouthExpansionVersion:    3,
	}
	sibling := RiverSegment{
		From:      3,
		To:        2,
		X0:        0,
		Z0:        128,
		X1:        220,
		Z1:        0,
		Bed0:      0.511,
		Bed1:      0.510,
		HalfWidth: 12,
		Profile:   []float64{0.511, 0.510},
	}
	downstream := RiverSegment{
		From:      2,
		To:        4,
		X0:        220,
		Z0:        0,
		X1:        348,
		Z1:        0,
		Bed0:      0.510,
		Bed1:      0.509,
		HalfWidth: 16,
		Profile:   []float64{0.510, 0.509},
	}
	rivers := []RiverSegment{outlet, sibling, downstream}
	incoming := make([][]int, 5)
	incoming[2] = []int{0, 1}
	incoming[4] = []int{2}

	// Before final profile settlement the outlet is steep enough that its lake
	// datum has faded away before the downstream junction.
	seedLakeOutletConfluenceBackwater(rivers, incoming, cfg)
	if rivers[0].ReceiverSurfaceDefined {
		t.Fatal("steep pre-settlement outlet unexpectedly seeded junction backwater")
	}

	// Later confluence/cascade repairs can flatten the same reach. The final
	// basin signal must therefore be recomputed rather than keeping the stale
	// pre-repair receiver state.
	rivers[0].Bed0 = 0.512
	rivers[0].Bed1 = 0.510
	rivers[0].Profile = []float64{0.512, 0.510}
	seedLakeOutletConfluenceBackwater(rivers, incoming, cfg)
	if !rivers[0].ReceiverSurfaceDefined ||
		!rivers[1].ReceiverSurfaceDefined {
		t.Fatal("settled lake outlet did not reseed all incoming junction datums")
	}
	want := rivers[0].ReceiverSurface
	if want <= 0.510+riverCenterDepth(cfg, rivers[0].HalfWidth) {
		t.Fatalf("reseeded junction datum=%.6f is not elevated", want)
	}

	easeReceiverOutflowDrops(rivers, cfg)
	downstreamStart :=
		sampleRiverProfile(rivers[2], 0) +
			riverCenterDepth(cfg, rivers[2].HalfWidth)
	if downstreamStart < want-1e-12 {
		t.Fatalf(
			"downstream start did not inherit recomputed junction datum: got=%.6f want>=%.6f",
			downstreamStart,
			want,
		)
	}
}

func TestLowestOpenNodeBreaksEqualCostTiesByNodeID(t *testing.T) {
	open := map[int]bool{8: true, 3: true, 5: true}
	dist := map[int]float64{8: 2, 3: 2, 5: 2}
	for i := 0; i < 100; i++ {
		node, cost := lowestOpenNode(open, dist)
		if node != 3 || cost != 2 {
			t.Fatalf("equal-cost open set chose node=%d cost=%v want node=3 cost=2", node, cost)
		}
	}
}

func TestRiverCorridorHitsBedTargetAndStaysLocal(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	from, to := 3, 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: from, To: to, RiverStrength: 1,
		}},
	}
	rec.RiverBedTarget[from] = 0.65
	rec.RiverBedTarget[to] = 0.60
	h := emptyHydrology(g)

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 1 {
		t.Fatalf("river segments=%d want=1", len(m.Rivers))
	}

	x0, z0 := g.NodePosition(0, 1)
	x1, _ := g.NodePosition(1, 1)
	midX := (x0 + x1) * 0.5
	got := m.Sample(midX, z0)
	if math.Abs(got.FinalHeight-0.625) > 1e-12 {
		t.Fatalf("centerline height=%.12f want=0.625", got.FinalHeight)
	}
	if got.RiverCarve <= 0 || got.WaterKind != reconcile.WaterRiver {
		t.Fatalf("missing river carve metadata: %+v", got)
	}

	far := m.Sample(midX, z0+40)
	if math.Abs(far.FinalHeight-0.70) > 1e-12 || far.RiverCarve != 0 {
		t.Fatalf("river carve leaked outside corridor: %+v", far)
	}
}

func TestRiverCorridorNeverRaisesBroadTerrain(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.60)
	n := g.Width * g.Height
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.60),
		RiverBedTarget:        filled(n, 0.80),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: 3, To: 4, RiverStrength: 1,
		}},
	}
	h := emptyHydrology(g)

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x0, z0 := g.NodePosition(0, 1)
	x1, _ := g.NodePosition(1, 1)
	got := m.Sample((x0+x1)*0.5, z0)
	want := 0.60 - m.Config.MinRiverBedDepthNormalized*
		(1+1.5*riverWidthFraction(m.Config, m.Rivers[0].HalfWidth))
	if got.FinalHeight > want+1e-12 ||
		got.FinalHeight < want-navigationSurfaceStepNormalized-1e-12 ||
		got.RiverCarve <= 0 {
		t.Fatalf(
			"river centerline outside allowed navigable carve: got=%+v baseTarget=%.6f step=%.6f",
			got,
			want,
			navigationSurfaceStepNormalized,
		)
	}
	if got.FinalHeight > 0.60+1e-12 {
		t.Fatalf("river overlay raised broad terrain: %+v", got)
	}
}

func TestNavigableRiverProfileSnapsDownWithoutRaising(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	in := []float64{0.612, 0.610, 0.608, 0.607}
	out := snapNavigableRiverProfile(
		in,
		cfg,
		12,
		navigationSurfaceStepNormalized,
	)
	if len(out) != len(in) {
		t.Fatalf("snapped profile length=%d want=%d", len(out), len(in))
	}
	for i := range out {
		if out[i] > in[i]+1e-12 {
			t.Fatalf("navigation snap raised bed[%d]: %.12f -> %.12f", i, in[i], out[i])
		}
		if in[i]-out[i] > navigationSurfaceStepNormalized+1e-12 {
			t.Fatalf(
				"navigation snap carved too deeply at %d: %.12f -> %.12f step=%.6f",
				i,
				in[i],
				out[i],
				navigationSurfaceStepNormalized,
			)
		}
		if i > 0 && out[i] > out[i-1]+1e-12 {
			t.Fatalf("snapped river profile rose at %d: %.12f -> %.12f", i, out[i-1], out[i])
		}
	}
	want := out[len(out)-1]
	for i := range out {
		if math.Abs(out[i]-want) > 1e-12 {
			t.Fatalf("gentle navigable profile is not flat after snap: %v", out)
		}
	}
}

func TestRiverBankBlendIsContinuous(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.60),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: 3, To: 4, RiverStrength: 1,
		}},
	}
	m, err := Build(base, rec, emptyHydrology(g), Config{})
	if err != nil {
		t.Fatal(err)
	}
	seg := m.Rivers[0]
	x := (seg.X0 + seg.X1) * 0.5
	eps := 1e-4

	innerZ := seg.Z0 + seg.HalfWidth
	inner := m.SampleNormalized(x, innerZ)
	insideSlope :=
		(inner - m.SampleNormalized(x, innerZ-eps)) / eps
	outsideSlope :=
		(m.SampleNormalized(x, innerZ+eps) - inner) / eps
	if inner >= 0.70 || inner <= sampleRiverProfile(seg, 0.5) {
		t.Fatalf(
			"inner bank should sit between river bed and broad terrain: %.12f",
			inner,
		)
	}
	if math.Abs(insideSlope-outsideSlope) > 2e-4 {
		t.Fatalf(
			"inner bank slope kink: inside=%.12g outside=%.12g",
			insideSlope,
			outsideSlope,
		)
	}

	outerZ := innerZ + riverBankBlendBlocks(seg.HalfWidth)
	outer := m.SampleNormalized(x, outerZ)
	outerInsideSlope :=
		(outer - m.SampleNormalized(x, outerZ-eps)) / eps
	outerOutsideSlope :=
		(m.SampleNormalized(x, outerZ+eps) - outer) / eps
	if math.Abs(outer-0.70) > 1e-10 {
		t.Fatalf("outer bank height=%.12f want broad terrain 0.70", outer)
	}
	if math.Abs(outerInsideSlope-outerOutsideSlope) > 2e-4 {
		t.Fatalf(
			"outer bank slope kink: inside=%.12g outside=%.12g",
			outerInsideSlope,
			outerOutsideSlope,
		)
	}
}

func TestLakeBasinCarvesToCeilingAndExposesWaterSurface(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.60)
	n := g.Width * g.Height
	center := 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.55
	rec.WaterSurfaceDefined[center] = true
	rec.WaterSurfaceElevation[center] = 0.60
	h := emptyHydrology(g)
	h.LakeID[center] = 1

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x, z := g.NodePosition(1, 1)
	got := m.Sample(x, z)
	if math.Abs(got.FinalHeight-0.55) > 1e-12 {
		t.Fatalf("lake center bed=%.12f want=0.55", got.FinalHeight)
	}
	if got.WaterKind != reconcile.WaterLake || !got.WaterSurfaceDefined {
		t.Fatalf("lake water metadata wrong: %+v", got)
	}
	if math.Abs(got.WaterSurfaceElevation-m.Lakes[0].WaterSurface) > 1e-12 {
		t.Fatalf(
			"lake sample surface=%.12f model datum=%.12f",
			got.WaterSurfaceElevation, m.Lakes[0].WaterSurface,
		)
	}

	far := m.Sample(x+100, z)
	if math.Abs(far.FinalHeight-0.60) > 1e-12 || far.LakeCarve != 0 {
		t.Fatalf("lake carve leaked outside local footprint: %+v", far)
	}
}

func TestLakeDoesNotExcavateHighDetailedTerrain(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	center := 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.55
	rec.WaterKind[center] = reconcile.WaterLake
	rec.WaterSurfaceDefined[center] = true
	rec.WaterSurfaceElevation[center] = 0.60
	h := emptyHydrology(g)
	h.LakeID[center] = 1

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x, z := g.NodePosition(1, 1)
	got := m.Sample(x, z)
	if math.Abs(got.FinalHeight-0.70) > 1e-12 || got.LakeCarve != 0 {
		t.Fatalf("high detailed terrain was excavated into lake: %+v", got)
	}
	if got.WaterKind == reconcile.WaterLake || got.WaterSurfaceDefined {
		t.Fatalf("lake flooded terrain above its spill datum: %+v", got)
	}
}

func TestLakeSurfaceCapsAtDetailedSpillRim(t *testing.T) {
	g := testGeometry()
	// The macro lake claims a 0.70 datum but detailed terrain around its
	// rendered footprint is only 0.55. A physically valid lake must spill/lower
	// instead of standing as a raised wall above the surrounding land.
	base := testTerrain(t, g, 0.55)
	n := g.Width * g.Height
	center := 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.20},
		RiverBaseElevation:    filled(n, 0.55),
		RiverBedTarget:        filled(n, 0.55),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.65
	rec.WaterKind[center] = reconcile.WaterLake
	rec.WaterSurfaceDefined[center] = true
	rec.WaterSurfaceElevation[center] = 0.70

	h := emptyHydrology(g)
	h.LakeID[center] = 1
	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Lakes) != 1 {
		t.Fatalf("lake nodes=%d want=1", len(m.Lakes))
	}
	lake := m.Lakes[0]
	if lake.WaterSurface >= 0.55 {
		t.Fatalf(
			"invalid raised lake survived detailed spill rim: surface=%.6f rim=0.55",
			lake.WaterSurface,
		)
	}
	if lake.BedCeiling >= lake.WaterSurface {
		t.Fatalf(
			"lake bed not kept below capped surface: bed=%.6f surface=%.6f",
			lake.BedCeiling, lake.WaterSurface,
		)
	}
}

func testGeometry() world.GridGeometry {
	return world.GridGeometry{
		Width: 3, Height: 3,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
}

func testTerrain(t *testing.T, g world.GridGeometry, height float64) *terrain.Model {
	t.Helper()
	n := g.Width * g.Height
	zeros := make([]float64, n)
	in := terrain.Input{
		Width: g.Width, Height: g.Height, Geometry: g,
		BaseElevation:       filled(n, height),
		RollingPlainWeight:  zeros,
		HillWeight:          zeros,
		FoothillWeight:      zeros,
		MountainWeight:      zeros,
		PlateauWeight:       zeros,
		ValleyWeight:        zeros,
		FloodplainWeight:    zeros,
		AlluvialPlainWeight: zeros,
		AlluvialFanWeight:   zeros,
		CanyonWeight:        zeros,
		ValleyFloorWidth:    zeros,
		DepositionPotential: zeros,
	}
	m, err := terrain.Generate(in, terrain.Config{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func emptyHydrology(g world.GridGeometry) *hydrology.Model {
	n := g.Width * g.Height
	flow := make([]int, n)
	for i := range flow {
		flow[i] = -1
	}
	return &hydrology.Model{
		Geometry:      g,
		Width:         g.Width,
		Height:        g.Height,
		LandMask:      make([]bool, n),
		FlowTo:        flow,
		LakeID:        make([]int, n),
		RiverStrength: make([]float64, n),
	}
}

func filled(n int, value float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func TestConfluencePropagatesActualUpstreamOutlet(t *testing.T) {
	g := world.GridGeometry{
		Width: 4, Height: 3,
		SpanXBlocks: 384, SpanZBlocks: 256,
	}
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	from, junction, downstream := 4, 5, 6

	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{
			{From: from, To: junction, RiverStrength: 1},
			{From: junction, To: downstream, RiverStrength: 1},
		},
	}
	// The incoming segment's actual outlet will clamp to 0.55 even though the
	// junction macro target is 0.60. The downstream segment must inherit 0.55,
	// otherwise the overlapping incoming corridor creates a dip then a rise.
	rec.RiverBedTarget[from] = 0.55
	rec.RiverBedTarget[junction] = 0.60
	rec.RiverBedTarget[downstream] = 0.50

	h := emptyHydrology(g)
	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 2 {
		t.Fatalf("river segments=%d want=2", len(m.Rivers))
	}

	var downstreamSegment *RiverSegment
	for i := range m.Rivers {
		if m.Rivers[i].From == junction && m.Rivers[i].To == downstream {
			downstreamSegment = &m.Rivers[i]
			break
		}
	}
	if downstreamSegment == nil {
		t.Fatal("downstream segment not found")
	}
	if downstreamSegment.Bed0 > 0.55+1e-12 ||
		downstreamSegment.Bed0 < 0.55-navigationSurfaceStepNormalized-1e-12 {
		t.Fatalf(
			"downstream bed start=%.12f outside propagated/navigation range [%.12f, %.12f]",
			downstreamSegment.Bed0,
			0.55-navigationSurfaceStepNormalized,
			0.55,
		)
	}

	d := m.Audit(32, 1e-9)
	if d.RiverUpslopeSteps != 0 {
		t.Fatalf("confluence introduced %d upslope steps, max rise %.12g", d.RiverUpslopeSteps, d.MaxRiverRise)
	}
}

func TestFinalConfluencePassKeepsOverlappingTributarySurfacesEqual(t *testing.T) {
	g := world.GridGeometry{
		Width: 4, Height: 3,
		SpanXBlocks: 384, SpanZBlocks: 256,
	}
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	fromWest, fromNorth, junction, downstream := 4, 1, 5, 6

	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{
			{From: fromWest, To: junction, RiverStrength: 0.55},
			{From: fromNorth, To: junction, RiverStrength: 1.00},
			{From: junction, To: downstream, RiverStrength: 1.00},
		},
	}
	rec.RiverBedTarget[fromWest] = 0.58
	rec.RiverBedTarget[fromNorth] = 0.54
	rec.RiverBedTarget[junction] = 0.54
	rec.RiverBedTarget[downstream] = 0.50

	h := emptyHydrology(g)
	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}

	var incoming []*RiverSegment
	for i := range m.Rivers {
		if m.Rivers[i].To == junction {
			incoming = append(incoming, &m.Rivers[i])
		}
	}
	if len(incoming) != 2 {
		t.Fatalf("incoming segments=%d want=2", len(incoming))
	}

	// By the final 5% of both tributaries their channel footprints overlap
	// around the shared junction. Any later cascade/receiver adjustment must
	// therefore leave them on one free-water datum; otherwise Minecraft ceil
	// quantization can turn a sub-block seam into an isolated 1-block source
	// peak.
	const sampleT = 0.95
	surfaces := make([]float64, 0, len(incoming))
	for _, seg := range incoming {
		bed := sampleRiverProfile(*seg, sampleT)
		surface := bed + riverCenterDepth(m.Config, seg.HalfWidth)
		if seg.ReceiverSurfaceDefined && seg.ReceiverSurface > surface {
			surface = seg.ReceiverSurface
		}
		surfaces = append(surfaces, surface)
	}
	if math.Abs(surfaces[0]-surfaces[1]) > 1e-9 {
		t.Fatalf(
			"overlapping tributary surfaces diverged after final passes: %.12f vs %.12f",
			surfaces[0],
			surfaces[1],
		)
	}
}

func TestTerminalRiverSegmentInheritsOceanSurfaceAtItsEndpoint(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.55)
	n := g.Width * g.Height
	from, ocean := 3, 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.55),
		RiverBedTarget:        filled(n, 0.55),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: from, To: ocean, RiverStrength: 1,
		}},
	}
	rec.RiverBedTarget[from] = 0.53
	rec.RiverBedTarget[ocean] = 0.47
	rec.WaterKind[ocean] = reconcile.WaterOcean
	rec.WaterSurfaceDefined[ocean] = true
	rec.WaterSurfaceElevation[ocean] = 0.50

	h := emptyHydrology(g)
	h.LandMask[from] = true
	h.FlowTo[from] = ocean

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 1 {
		t.Fatalf("river segments=%d want=1", len(m.Rivers))
	}
	seg := m.Rivers[0]
	if !seg.ReceiverSurfaceDefined || math.Abs(seg.ReceiverSurface-0.50) > 1e-12 {
		t.Fatalf("terminal ocean surface not inherited: %+v", seg)
	}
	depth := riverCenterDepth(m.Config, seg.HalfWidth)
	for i, bed := range seg.Profile {
		if surface := bed + depth; surface > 0.50+1e-12 {
			t.Fatalf(
				"terminal receiving reach[%d] natural surface=%.12f above ocean 0.50",
				i,
				surface,
			)
		}
	}
}

func TestFinalCascadeSlopeIsBackedByRiverBed(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	rivers := []RiverSegment{{
		From:      0,
		To:        1,
		X0:        0,
		Z0:        0,
		X1:        128,
		Z1:        0,
		HalfWidth: 8,
		Profile: []float64{
			0.70,
			0.69,
			0.66,
			0.61,
			0.57,
			0.52,
			0.48,
			0.44,
			0.40,
		},
	}}
	enforceSupportedRiverCascades(rivers, cfg)
	seg := rivers[0]
	sampleBlocks :=
		math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) /
			float64(len(seg.Profile)-1)
	maxDrop := 0.70 * sampleBlocks / 514.0
	for i := 1; i < len(seg.Profile); i++ {
		drop := seg.Profile[i-1] - seg.Profile[i]
		if drop > maxDrop+1e-12 {
			t.Fatalf(
				"unsupported cascade step %d drop=%.12f max=%.12f profile=%v",
				i,
				drop,
				maxDrop,
				seg.Profile,
			)
		}
		if drop < -1e-12 {
			t.Fatalf("cascade rose downstream at %d: %v", i, seg.Profile)
		}
	}
}

func TestDiagonalLakeFootprintBackwatersCrossingRiver(t *testing.T) {
	g := testGeometry()
	// Keep detailed terrain close to the synthetic lake datum so the lake owns
	// the crossing without requiring excavation of high dry terrain.
	base := testTerrain(t, g, 0.60)
	n := g.Width * g.Height

	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.55),
		RiverBedTarget:        filled(n, 0.55),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: 0, To: 4, RiverStrength: 1,
		}},
	}
	rec.RiverBedTarget[0] = 0.54
	rec.RiverBedTarget[4] = 0.52

	// Node 1 is at (128,0). The diagonal segment 0 -> 4 passes through the
	// rounded-square lake footprint near (64,64). Final rendered connectivity
	// is authoritative: once river and lake actually touch, the lake datum must
	// become a backwater floor so Minecraft never renders an upward water wall.
	lake := 1
	rec.LakeBedCeilingDefined[lake] = true
	rec.LakeBedCeiling[lake] = 0.50
	rec.WaterKind[lake] = reconcile.WaterLake
	rec.WaterSurfaceDefined[lake] = true
	rec.WaterSurfaceElevation[lake] = 0.60

	h := emptyHydrology(g)
	h.FlowTo[0] = 4
	h.LakeID[lake] = 1

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 1 {
		t.Fatalf("river segments=%d want=1", len(m.Rivers))
	}
	seg := m.Rivers[0]
	if !seg.ReceiverSurfaceDefined {
		t.Fatalf("spatial lake crossing did not create backwater: %+v", seg)
	}
	if math.Abs(seg.ReceiverSurface-m.Lakes[0].WaterSurface) > 1e-12 {
		t.Fatalf(
			"crossing river receiver=%.12f lake=%.12f",
			seg.ReceiverSurface,
			m.Lakes[0].WaterSurface,
		)
	}
	mid := m.Sample(64, 64)
	if mid.WaterKind != reconcile.WaterLake || !mid.WaterSurfaceDefined {
		t.Fatalf("diagonal lake footprint is not continuous at crossing: %+v", mid)
	}
	if math.Abs(mid.WaterSurfaceElevation-m.Lakes[0].WaterSurface) > 1e-12 {
		t.Fatalf(
			"lake surface=%.12f model datum=%.12f",
			mid.WaterSurfaceElevation,
			m.Lakes[0].WaterSurface,
		)
	}
}
