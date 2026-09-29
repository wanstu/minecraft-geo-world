package phase1

type CrustType uint8

const (
	Continental CrustType = iota
	Oceanic
)

type BoundaryType uint8

const (
	NoBoundary BoundaryType = iota
	Convergent
	Divergent
	Transform
)

type Vec2 struct {
	X float64
	Y float64
}

type Plate struct {
	ID       int
	Center   Vec2
	Velocity Vec2
}

type Config struct {
	Seed          int64
	Width         int
	Height        int
	PlateCount    int
	OceanFraction float64
}

type World struct {
	Config        Config
	Plates        []Plate
	PlateID       []int
	Crust         []CrustType
	LandPotential []float64
	Boundary      []BoundaryType
	Elevation     []float64
}
