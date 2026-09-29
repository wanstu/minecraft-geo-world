package world

import "errors"

// Surface describes the sea-connected inundation of one particular heightfield.
// LandMask means non-ocean; inland lakes are a separate hydrological layer.
// It is not a crust classification or the final block-level water mask.
type Surface struct {
	Width              int
	Height             int
	SeaLevelNormalized float64
	LandMask           []bool
	EnclosedBelowSea   []bool
}

// ClassifySurface floods strictly below-sea nodes from every grid edge.
// Four-neighbour connectivity conservatively prevents diagonal-only leakage.
// Nodes exactly at sea level are shoreline land, not an open water passage.
// Enclosed lowlands remain land until hydrology establishes their water balance.
func ClassifySurface(width, height int, elevation []float64, seaLevel float64) (*Surface, error) {
	n, err := NodeCount(width, height)
	if err != nil {
		return nil, err
	}
	if len(elevation) != n {
		return nil, errors.New("world: elevation size does not match dimensions")
	}
	if !finite(seaLevel) || seaLevel <= 0 || seaLevel >= 1 {
		return nil, errors.New("world: normalized sea level must be finite and inside (0,1)")
	}
	for _, v := range elevation {
		if !finite(v) || v < 0 || v > 1 {
			return nil, errors.New("world: elevation must be finite and normalized to [0,1]")
		}
	}
	s := &Surface{Width: width, Height: height, SeaLevelNormalized: seaLevel,
		LandMask: make([]bool, n), EnclosedBelowSea: make([]bool, n)}
	for i := range s.LandMask {
		s.LandMask[i] = true
	}
	queue := make([]int, 0)
	visit := func(i int) {
		if s.LandMask[i] && elevation[i] < seaLevel {
			s.LandMask[i] = false
			queue = append(queue, i)
		}
	}
	for x := 0; x < width; x++ {
		visit(x)
		visit((height-1)*width + x)
	}
	for z := 1; z < height-1; z++ {
		visit(z * width)
		visit(z*width + width - 1)
	}
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		x, z := i%width, i/width
		if x > 0 {
			visit(i - 1)
		}
		if x+1 < width {
			visit(i + 1)
		}
		if z > 0 {
			visit(i - width)
		}
		if z+1 < height {
			visit(i + width)
		}
	}
	for i, land := range s.LandMask {
		s.EnclosedBelowSea[i] = land && elevation[i] < seaLevel
	}
	return s, nil
}
