package main

import (
	"fmt"
	"strconv"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type Heatmap struct {
	*mat.Dense
	limit float64
}

func NewHeatmap(sizeX, sizeY int) *Heatmap {
	return &Heatmap{
		Dense: mat.NewDense(sizeY, sizeX, nil),
	}
}

func (h Heatmap) Dims() (int, int) {
	r, c := h.Dense.Dims()
	return c, r
}

func (h Heatmap) Z(c, r int) float64 {
	return h.At(r, c)
}

func (h Heatmap) X(c int) float64 {
	cols, _ := h.Dims()
	return float64(c) * h.limit / float64(cols-1)
}

func (h Heatmap) Y(r int) float64 {
	_, rows := h.Dims()
	return float64(r) * h.limit / float64(rows-1)
}

func (h Heatmap) Render(mini, maxi int, limit float64, filename string) error {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("System temperature under pulsed workloads (%d-%d'C)", mini, maxi)
	p.X.Label.Text = "idle time (s)"
	p.Y.Label.Text = "compute time (s)"
	p.X.Min = 0
	p.X.Max = limit
	p.Y.Min = 0
	p.Y.Max = limit

	xticks := make([]plot.Tick, int(h.limit)+1)
	yticks := make([]plot.Tick, int(h.limit)+1)

	for i := 0; i <= int(h.limit); i++ {
		xticks[i] = plot.Tick{Value: float64(i), Label: strconv.Itoa(i)}
		yticks[i] = plot.Tick{Value: float64(i), Label: strconv.Itoa(i)}
	}

	p.X.Tick.Marker = plot.ConstantTicks(xticks)
	p.Y.Tick.Marker = plot.ConstantTicks(yticks)

	pal := moreland.ExtendedBlackBody()
	/*	pal.SetMin(min)
		pal.SetMax(max)*/

	heatmapPlot := plotter.NewHeatMap(h, pal.Palette(maxi-mini))
	p.Add(heatmapPlot)

	err := p.Save(15*vg.Centimeter, 15*vg.Centimeter, filename)
	if err != nil {
		return fmt.Errorf("heatmap: %w", err)
	}

	return nil
}
