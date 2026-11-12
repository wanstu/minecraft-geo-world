package main

import (
	"encoding/json"
	"fmt"
	"geoworld/internal/waterbed"
	"geoworld/internal/worldmodelio"
	"os"
)

type Point struct {
	T float64 `json:"t"`
	X float64 `json:"x"`
	Z float64 `json:"z"`
}
type River struct {
	ID         int     `json:"id"`
	OceanMouth bool    `json:"oceanMouth"`
	HalfWidth  float64 `json:"halfWidth"`
	Points     []Point `json:"points"`
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: main.go <model.geoworld> <out.json>")
	}
	model, err := worldmodelio.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	ids := []int{509, 535, 851, 915}
	out := make([]River, 0, len(ids))
	for _, id := range ids {
		if id >= len(model.Waterbed.Rivers) {
			panic(fmt.Sprintf("missing river %d", id))
		}
		segment := model.Waterbed.Rivers[id]
		river := River{ID: id, OceanMouth: segment.OceanMouth, HalfWidth: segment.HalfWidth}
		for i := 0; i <= 128; i++ {
			t := float64(i) / 128
			x, z := waterbed.RiverCenterPoint(segment, t)
			river.Points = append(river.Points, Point{T: t, X: x, Z: z})
		}
		out = append(out, river)
	}
	buf, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(os.Args[2], append(buf, '\n'), 0644); err != nil {
		panic(err)
	}
	for _, v := range out {
		fmt.Printf("river=%d ocean=%v width=%.2f from=%.1f,%.1f to=%.1f,%.1f\n", v.ID, v.OceanMouth, v.HalfWidth, v.Points[0].X, v.Points[0].Z, v.Points[len(v.Points)-1].X, v.Points[len(v.Points)-1].Z)
	}
}
