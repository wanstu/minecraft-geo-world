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

type TectonicFeature uint8

const (
	NoFeature TectonicFeature = iota
	CollisionOrogen
	VolcanicArc
	Trench
	IslandArc
	MidOceanRidge
	ContinentalRift
	TransformFault
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
	Config            Config
	Plates            []Plate
	RawPlateID        []int
	PlateID           []int
	Crust             []CrustType
	LandPotential     []float64
	Boundary          []BoundaryType
	OceanAge          []float64
	Feature           []TectonicFeature
	TectonicInfluence []float64
	Elevation         []float64
}
