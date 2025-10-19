package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestHeatmap(t *testing.T) {
	const limit = 8
	const steps = 8

	heatmap := Heatmap{
		Data: mat.NewDense(limit*steps, limit*steps, nil),
		Step: 10,
	}

	maxTempOverall := 96
	minTempOverall := 80

	for offTime := range limit * steps {
		for onTime := range limit * steps {
			val := math.Trunc(86 + rand.Float64()*4) // intentionally smaller range
			fmt.Printf("%.1f/%.1f=%.f ", float64(onTime)/steps, float64(offTime)/steps, val)
			heatmap.Data.Set(onTime, offTime, val)
		}
	}

	err := heatmap.Render(float64(minTempOverall), float64(maxTempOverall), float64(limit*steps), float64(limit*steps), steps,
		"System temperature under pulsed workloads ('C)",
		"idle time (s)",
		"compute time (s)",
		"heatmap.pdf")
	if err != nil {
		t.Fatal(err)
	}
}
