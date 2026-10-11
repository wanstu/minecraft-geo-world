package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"geoworld/internal/generator"
	"geoworld/internal/world"
)

type stat struct {
	min float64
	max float64
	sum float64
	n   int
}

func (s *stat) add(v float64) {
	if s.n == 0 || v < s.min {
		s.min = v
	}
	if s.n == 0 || v > s.max {
		s.max = v
	}
	s.sum += v
	s.n++
}

func (s stat) avg() float64 {
	if s.n == 0 {
		return 0
	}
	return s.sum / float64(s.n)
}

type candidate struct {
	x      float64
	z      float64
	weight float64
}

type categoryStats struct {
	points  int
	edges   int
	steep35 int
	cliff50 int
	maxDeg  float64
	slopes  []float64
}

type mountainStats struct {
	centers int
	points  int
	edges   int

	steep35 int
	cliff50 int
	cliff60 int

	saturated int
	peaks     int
	strong    int

	minY      float64
	maxY      float64
	maxRelief float64

	maxSlopeDeg float64
	maxSlopeX   float64
	maxSlopeZ   float64
	highestX    float64
	highestZ    float64
	lowestX     float64
	lowestZ     float64

	slopes []float64
}

func main() {
	count := flag.Int("count", 3, "number of deterministic fallback seeds when -seeds is empty")
	seedsText := flag.String(
		"seeds",
		"123456,731894205,10400",
		"comma-separated seeds; empty uses 7000+n*6151",
	)
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	cellSize := flag.Float64("cell-size", 128, "Minecraft blocks per macro cell")
	centered := flag.Bool("centered", false, "center the model at the Minecraft origin, as the MOD does")
	flag.Parse()

	seeds, err := parseSeeds(*seedsText, *count)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var minHeight, maxHeight, maxAbsDetail, maxValley stat
	var chunkJump stat
	failed := false

	for _, seed := range seeds {
		geometry := world.GridGeometry{
			Width:       *width,
			Height:      *height,
			SpanXBlocks: float64(*width-1) * *cellSize,
			SpanZBlocks: float64(*height-1) * *cellSize,
		}
		if *centered {
			geometry.OriginX = -geometry.SpanXBlocks / 2
			geometry.OriginZ = -geometry.SpanZBlocks / 2
		}
		model, err := generator.GenerateWorld(context.Background(), generator.Config{
			Seed:          seed,
			Geometry:      geometry,
			PlateCount:    18,
			OceanFraction: 0.68,
		}, nil)
		if err != nil {
			panic(err)
		}

		thisMin := math.Inf(1)
		thisMax := math.Inf(-1)
		thisDetail := 0.0
		thisValley := 0.0
		baseTopN := 0
		finalTopN := 0
		sampleN := 0

		for z := 0; z < geometry.Height; z++ {
			for x := 0; x < geometry.Width; x++ {
				wx, wz := geometry.NodePosition(x, z)
				raw := model.Terrain.Sample(wx, wz)
				final := model.CoastTerrain.SampleNormalized(wx, wz)
				if !finite(final) {
					panic(fmt.Sprintf(
						"seed %d invalid terrain at %.1f,%.1f",
						seed,
						wx,
						wz,
					))
				}
				thisMin = math.Min(thisMin, final)
				thisMax = math.Max(thisMax, final)
				thisDetail = math.Max(thisDetail, math.Abs(raw.RegionalDetail))
				thisValley = math.Max(thisValley, raw.ValleyCarve)
				if raw.BaseElevation >= 0.999 {
					baseTopN++
				}
				if final >= 0.999 {
					finalTopN++
				}
				sampleN++
			}
		}

		jump := auditChunkSeams(
			model.CoastTerrain.SampleNormalized,
			geometry,
		)
		categories := auditWorldCategories(model)
		centers := chooseMountainCenters(model, 12)
		mountains := auditMountainWindows(model, centers)

		minHeight.add(thisMin)
		maxHeight.add(thisMax)
		maxAbsDetail.add(thisDetail)
		maxValley.add(thisValley)
		chunkJump.add(jump)

		fmt.Printf(
			"seed=%d height=%.4f..%.4f detail<=%.5f valley<=%.5f "+
				"top(base/final)=%.3f%%/%.3f%% chunk-jump=%.9f\n",
			seed,
			thisMin,
			thisMax,
			thisDetail,
			thisValley,
			percent(baseTopN, sampleN),
			percent(finalTopN, sampleN),
			jump,
		)

		fmt.Printf(
			"  mountains centers=%d y=%.1f..%.1f relief<=%.1f "+
				"slope[p50=%.1f p90=%.1f p95=%.1f p99=%.1f max=%.1f] "+
				"steep35=%.3f%% cliff50=%.3f%% cliff60=%.3f%% "+
				"peaks=%d strong=%d top>=318=%.4f%%\n",
			mountains.centers,
			mountains.minY,
			mountains.maxY,
			mountains.maxRelief,
			percentile(mountains.slopes, 0.50),
			percentile(mountains.slopes, 0.90),
			percentile(mountains.slopes, 0.95),
			percentile(mountains.slopes, 0.99),
			mountains.maxSlopeDeg,
			percent(mountains.steep35, mountains.edges),
			percent(mountains.cliff50, mountains.edges),
			percent(mountains.cliff60, mountains.edges),
			mountains.peaks,
			mountains.strong,
			percent(mountains.saturated, mountains.points),
		)
		fmt.Printf(
			"  checkpoints high=(%.0f,%.0f y=%.1f) low=(%.0f,%.0f y=%.1f) "+
				"steep=(%.0f,%.0f %.1f°)\n",
			mountains.highestX,
			mountains.highestZ,
			mountains.maxY,
			mountains.lowestX,
			mountains.lowestZ,
			mountains.minY,
			mountains.maxSlopeX,
			mountains.maxSlopeZ,
			mountains.maxSlopeDeg,
		)

		for _, name := range []string{"plain", "hill", "foothill", "mountain", "plateau"} {
			c := categories[name]
			if c == nil || c.points == 0 {
				continue
			}
			fmt.Printf(
				"  %-9s points=%6d slope[p50=%4.1f p90=%4.1f p99=%4.1f max=%4.1f] "+
					"steep35=%6.3f%% cliff50=%6.3f%%\n",
				name,
				c.points,
				percentile(c.slopes, 0.50),
				percentile(c.slopes, 0.90),
				percentile(c.slopes, 0.99),
				c.maxDeg,
				percent(c.steep35, c.edges),
				percent(c.cliff50, c.edges),
			)
		}

		if jump > 1e-5 {
			fmt.Fprintf(os.Stderr, "seed %d chunk seam too large: %.9f\n", seed, jump)
			failed = true
		}
		if !mountainGuardrails(centers, mountains, categories) {
			fmt.Fprintf(os.Stderr, "seed %d mountain morphology outside guardrails\n", seed)
			failed = true
		}
	}

	fmt.Printf(
		"\nGeoWorld Terrain Morphology Audit: seeds=%d grid=%dx%d cell=%.0f blocks\n",
		len(seeds),
		*width,
		*height,
		*cellSize,
	)
	printStat("minimum height", minHeight)
	printStat("maximum height", maxHeight)
	printStat("maximum |regional detail|", maxAbsDetail)
	printStat("maximum valley carve", maxValley)
	printStat("chunk epsilon jump", chunkJump)

	if failed {
		fmt.Fprintln(os.Stderr, "terrain morphology audit failed guardrails")
		os.Exit(1)
	}
}

func parseSeeds(raw string, count int) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if count < 1 {
			count = 1
		}
		out := make([]int64, 0, count)
		for n := 0; n < count; n++ {
			out = append(out, int64(7000+n*6151))
		}
		return out, nil
	}

	var out []int64
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		seed, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid seed %q: %w", part, err)
		}
		out = append(out, seed)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no seeds supplied")
	}
	return out, nil
}

func chooseMountainCenters(
	model *generator.WorldModel,
	maxCenters int,
) []candidate {
	g := model.Geometry
	candidates := make([]candidate, 0)

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			weight := model.Landform.MountainWeight[i]
			if weight < 0.55 {
				continue
			}
			wx, wz := g.NodePosition(x, z)
			if wx < g.OriginX+768 ||
				wz < g.OriginZ+768 ||
				wx > g.OriginX+g.SpanXBlocks-768 ||
				wz > g.OriginZ+g.SpanZBlocks-768 {
				continue
			}
			candidates = append(candidates, candidate{
				x:      wx,
				z:      wz,
				weight: weight,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].weight > candidates[j].weight
	})

	selected := make([]candidate, 0, maxCenters)
	for _, c := range candidates {
		farEnough := true
		for _, existing := range selected {
			if math.Hypot(c.x-existing.x, c.z-existing.z) < 1400 {
				farEnough = false
				break
			}
		}
		if !farEnough {
			continue
		}
		selected = append(selected, c)
		if len(selected) >= maxCenters {
			break
		}
	}
	return selected
}

func auditMountainWindows(
	model *generator.WorldModel,
	centers []candidate,
) mountainStats {
	const (
		radius = 512.0
		step   = 16.0
	)

	result := mountainStats{
		centers: len(centers),
		minY:    math.Inf(1),
		maxY:    math.Inf(-1),
	}

	for _, center := range centers {
		n := int(2*radius/step) + 1
		ys := make([]float64, n*n)
		at := func(x, z int) float64 {
			return ys[z*n+x]
		}

		for z := 0; z < n; z++ {
			worldZ := center.z - radius + float64(z)*step
			for x := 0; x < n; x++ {
				worldX := center.x - radius + float64(x)*step
				y := minecraftY(
					model.CoastTerrain.SampleNormalized(worldX, worldZ),
				)
				ys[z*n+x] = y
				result.points++

				if y < result.minY {
					result.minY = y
					result.lowestX = worldX
					result.lowestZ = worldZ
				}
				if y > result.maxY {
					result.maxY = y
					result.highestX = worldX
					result.highestZ = worldZ
				}
				if y >= 318 {
					result.saturated++
				}
			}
		}

		windowMin := math.Inf(1)
		windowMax := math.Inf(-1)
		for _, y := range ys {
			windowMin = math.Min(windowMin, y)
			windowMax = math.Max(windowMax, y)
		}
		result.maxRelief = math.Max(
			result.maxRelief,
			windowMax-windowMin,
		)

		for z := 0; z < n; z++ {
			for x := 0; x < n; x++ {
				y := at(x, z)
				worldX := center.x - radius + float64(x)*step
				worldZ := center.z - radius + float64(z)*step

				if x+1 < n {
					addSlope(
						&result,
						worldX,
						worldZ,
						math.Abs(at(x+1, z)-y),
						step,
					)
				}
				if z+1 < n {
					addSlope(
						&result,
						worldX,
						worldZ,
						math.Abs(at(x, z+1)-y),
						step,
					)
				}

				if x < 2 || z < 2 || x+2 >= n || z+2 >= n {
					continue
				}
				localMax := true
				ringMin := math.Inf(1)
				for dz := -2; dz <= 2; dz++ {
					for dx := -2; dx <= 2; dx++ {
						if dx == 0 && dz == 0 {
							continue
						}
						neighbor := at(x+dx, z+dz)
						if neighbor >= y {
							localMax = false
						}
						ringMin = math.Min(ringMin, neighbor)
					}
				}
				if localMax {
					result.peaks++
					if y-ringMin >= 6 {
						result.strong++
					}
				}
			}
		}
	}

	return result
}

func addSlope(
	stats *mountainStats,
	x float64,
	z float64,
	deltaY float64,
	run float64,
) {
	degrees := math.Atan2(deltaY, run) * 180 / math.Pi
	stats.edges++
	stats.slopes = append(stats.slopes, degrees)

	if degrees >= 35 {
		stats.steep35++
	}
	if degrees >= 50 {
		stats.cliff50++
	}
	if degrees >= 60 {
		stats.cliff60++
	}
	if degrees > stats.maxSlopeDeg {
		stats.maxSlopeDeg = degrees
		stats.maxSlopeX = x
		stats.maxSlopeZ = z
	}
}

func auditWorldCategories(
	model *generator.WorldModel,
) map[string]*categoryStats {
	result := map[string]*categoryStats{}
	get := func(name string) *categoryStats {
		if result[name] == nil {
			result[name] = &categoryStats{}
		}
		return result[name]
	}

	g := model.Geometry
	const (
		stride = 128.0
		edge   = 16.0
	)
	endX := g.OriginX + g.SpanXBlocks
	endZ := g.OriginZ + g.SpanZBlocks

	for z := g.OriginZ + 64; z <= endZ-64; z += stride {
		for x := g.OriginX + 64; x <= endX-64; x += stride {
			height := model.CoastTerrain.SampleNormalized(x, z)
			if height <= model.Config.SeaLevelNormalized {
				continue
			}

			sample := model.Terrain.Sample(x, z)
			weights := []struct {
				name   string
				weight float64
			}{
				{"plain", sample.RollingPlainWeight},
				{"hill", sample.HillWeight},
				{"foothill", sample.FoothillWeight},
				{"mountain", sample.MountainWeight},
				{"plateau", sample.PlateauWeight},
			}
			best := weights[0]
			for _, candidate := range weights[1:] {
				if candidate.weight > best.weight {
					best = candidate
				}
			}

			stats := get(best.name)
			stats.points++
			y := minecraftY(height)

			for _, neighbor := range [][2]float64{
				{x + edge, z},
				{x, z + edge},
			} {
				neighborHeight := model.CoastTerrain.SampleNormalized(
					neighbor[0],
					neighbor[1],
				)
				if neighborHeight <= model.Config.SeaLevelNormalized {
					continue
				}
				y2 := minecraftY(neighborHeight)
				degrees := math.Atan2(math.Abs(y2-y), edge) * 180 / math.Pi
				stats.edges++
				stats.slopes = append(stats.slopes, degrees)
				if degrees >= 35 {
					stats.steep35++
				}
				if degrees >= 50 {
					stats.cliff50++
				}
				stats.maxDeg = math.Max(stats.maxDeg, degrees)
			}
		}
	}

	return result
}

func mountainGuardrails(
	centers []candidate,
	mountains mountainStats,
	categories map[string]*categoryStats,
) bool {
	if len(centers) < 4 || mountains.points == 0 || mountains.edges == 0 {
		return false
	}

	// Strong mountain belts should contain recognizable summits and substantial
	// local relief, but high cliffs must remain sparse rather than becoming the
	// default walking surface.
	mountainP90 := percentile(mountains.slopes, 0.90)
	mountainP99 := percentile(mountains.slopes, 0.99)
	if mountainP90 < 8 || mountainP90 > 25 {
		return false
	}
	if mountainP99 < 20 || mountainP99 > 55 {
		return false
	}
	if mountains.maxSlopeDeg < 50 || mountains.maxSlopeDeg > 80 {
		return false
	}
	if percent(mountains.steep35, mountains.edges) > 4 {
		return false
	}
	if percent(mountains.cliff50, mountains.edges) > 1.5 {
		return false
	}
	if mountains.maxRelief < 100 {
		return false
	}
	if mountains.strong < len(centers)*3 {
		return false
	}
	if percent(mountains.saturated, mountains.points) > 1 {
		return false
	}

	// Mountain shaping must stay spatially contained. Plains and plateaus
	// remain broadly traversable even when nearby mountain belts gain cliffs.
	if plain := categories["plain"]; plain != nil && plain.edges > 0 {
		if percentile(plain.slopes, 0.99) > 10 ||
			plain.maxDeg > 20 ||
			plain.steep35 != 0 ||
			plain.cliff50 != 0 {
			return false
		}
	}
	if hill := categories["hill"]; hill != nil && hill.edges > 0 {
		if percentile(hill.slopes, 0.99) > 20 ||
			hill.maxDeg > 45 ||
			hill.cliff50 != 0 {
			return false
		}
	}
	if foothill := categories["foothill"]; foothill != nil && foothill.edges > 0 {
		if percentile(foothill.slopes, 0.99) > 25 ||
			foothill.maxDeg > 65 ||
			percent(foothill.steep35, foothill.edges) > 0.2 ||
			percent(foothill.cliff50, foothill.edges) > 0.1 {
			return false
		}
	}
	if plateau := categories["plateau"]; plateau != nil && plateau.edges > 0 {
		if percentile(plateau.slopes, 0.99) > 6 ||
			plateau.maxDeg > 25 {
			return false
		}
	}
	if mountain := categories["mountain"]; mountain != nil && mountain.edges > 0 {
		if percentile(mountain.slopes, 0.90) < 8 ||
			percentile(mountain.slopes, 0.99) < 18 {
			return false
		}
	}
	return true
}

func auditChunkSeams(
	sample func(float64, float64) float64,
	geometry world.GridGeometry,
) float64 {
	maxJump := 0.0
	epsilon := 1e-4
	endX := geometry.OriginX + geometry.SpanXBlocks
	endZ := geometry.OriginZ + geometry.SpanZBlocks

	for x := geometry.OriginX + 16; x < endX; x += 512 {
		for z := geometry.OriginZ; z <= endZ; z += math.Max(256, geometry.SpanZBlocks/40) {
			at := sample(x, z)
			maxJump = math.Max(
				maxJump,
				math.Abs(sample(x-epsilon, z)-at),
			)
			maxJump = math.Max(
				maxJump,
				math.Abs(sample(x+epsilon, z)-at),
			)
		}
	}
	return maxJump
}

func minecraftY(height float64) float64 {
	height = math.Max(0, math.Min(1, height))
	if height <= 0.5 {
		return -64 + 127*(height/0.5)
	}
	return 63 + 257*((height-0.5)/0.5)
}

func percentile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	index := int(math.Round(
		math.Max(0, math.Min(1, quantile)) *
			float64(len(sorted)-1),
	))
	return sorted[index]
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func printStat(name string, s stat) {
	fmt.Printf(
		"%-32s min=%12.9f avg=%12.9f max=%12.9f\n",
		name,
		s.min,
		s.avg(),
		s.max,
	)
}

func percent(count, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) / float64(total) * 100
}
