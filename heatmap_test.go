package main

import (
	"fmt"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestHeatmap(t *testing.T) {
	const limit = 10
	const steps = 10

	heatmap := Heatmap{
		Data: mat.NewDense(limit*steps, limit*steps, nil),
		Step: 10,
	}

	maxTempOverall := 96
	minTempOverall := 80

	for total := 0; total < limit*steps; total++ {
		for onTime := 0; onTime <= total; onTime++ {
			offTime := total - onTime
			val := float64(minTempOverall) + rand.Float64()*float64(maxTempOverall-minTempOverall)
			heatmap.Data.Set(onTime, offTime, val)
		}
	}

	err := heatmap.Render(minTempOverall, maxTempOverall, limit*steps, limit*steps,
		fmt.Sprintf("System temperature under pulsed workloads (%d-%d'C)", minTempOverall, maxTempOverall),
		"idle time (s)",
		"compute time (s)",
		"heatmap.pdf")
	if err != nil {
		t.Fatal(err)
	}
}
