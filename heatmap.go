package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgpdf"
)

type Heatmap struct {
	XOffset, YOffset float64
	Step             float64
	Data             *mat.Dense
}

type integerTicks struct{}

func (h Heatmap) Dims() (int, int) {
	r, c := h.Data.Dims()
	return c, r
}

func (h Heatmap) Z(c, r int) float64 {
	return h.Data.At(r, c)
}

func (h Heatmap) X(c int) float64 {
	_, n := h.Data.Dims()
	if c < 0 || c >= n {
		panic("range check")
	}
	return float64(c) + h.XOffset
}

func (h Heatmap) Y(r int) float64 {
	m, _ := h.Data.Dims()
	if r < 0 || r >= m {
		panic("range check")
	}
	return float64(r) + h.YOffset
}

func (integerTicks) Ticks(min, max float64) []plot.Tick {
	var t []plot.Tick

	for i := min; i <= max; i++ {
		j := (i + 0.5) / 10.

		name := ""
		if j*2 == math.Trunc(j*2) {
			name = fmt.Sprintf("%0.1f", j)
		}

		t = append(t, plot.Tick{Value: i, Label: name})
	}
	return t
}

func (h Heatmap) Render(mini, maxi int, xMax, yMax float64, title, xLabel, yLabel, filename string) error {
	pal := moreland.ExtendedBlackBody().Palette(maxi - mini + 1)
	g := plotter.NewHeatMap(h, pal)

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	p.X.Tick.Marker = integerTicks{}
	p.Y.Tick.Marker = integerTicks{}

	p.Add(g)

	l := plot.NewLegend()
	thumbs := plotter.PaletteThumbnailers(pal)

	for i := range thumbs {
		l.Add(strconv.Itoa(i+mini), thumbs[i])
	}

	p.X.Max = xMax
	p.Y.Max = yMax
	img := vgpdf.New(800, 600)
	dc := draw.New(img)

	l.Top = true

	r := l.Rectangle(dc)
	legendWidth := r.Max.X - r.Min.X
	l.YOffs = -p.Title.TextStyle.FontExtents().Height

	l.Draw(dc)
	// allow space for legend
	dc = draw.Crop(dc, 0, -legendWidth-vg.Millimeter, 0, 0)
	p.Draw(dc)

	w, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = img.WriteTo(w)
	if err != nil {
		return err
	}

	return nil
}
