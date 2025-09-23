package main

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type Heatmap struct {
	*mat.Dense
}

func NewHeatmap(sizeX, sizeY int) *Heatmap {
	return &Heatmap{
		Dense: mat.NewDense(sizeY, sizeX, nil),
	}
}

func (h Heatmap) Dims() (c, r int) {
	r, c = h.Dense.Dims()
	return c, r
}

func (h Heatmap) Z(c, r int) float64 {
	return h.At(r, c)
}

func (h Heatmap) X(c int) float64 { return float64(c) }
func (h Heatmap) Y(r int) float64 { return float64(r) }

func (h Heatmap) Render(min, max float64, filename string) error {
	p := plot.New()
	p.Title.Text = "System temperature under pulsed workloads"

	pal := palette.Heat(12, 1) // moreland.Kindlmann()
	/*	pal.SetMin(min)
		pal.SetMax(max)*/

	heatmapPlot := plotter.NewHeatMap(h, pal)
	p.Add(heatmapPlot)

	// Add a color bar
	/*	cb := plotter.NewColorBar(pal)
		cb.Vertical = true
		p.Add(cb)
		p.Right = append(p.Right, draw.Crop(cb, 0, -0.5*vg.Centimeter)) */

	err := p.Save(15*vg.Centimeter, 15*vg.Centimeter, filename)
	if err != nil {
		return fmt.Errorf("heatmap: %w", err)
	}

	return nil
}
