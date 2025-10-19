package main

import (
	"fmt"
	"image/color"
	"os"

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

type ticks struct {
	divisor int
}

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

func (t ticks) Ticks(min, max float64) []plot.Tick {
	var ts []plot.Tick

	for i := min + 0.5; i <= max; i += float64(t.divisor) / 2 {
		j := i / float64(t.divisor)
		name := fmt.Sprintf("%0.1f", j)
		ts = append(ts, plot.Tick{Value: i, Label: name})
	}
	return ts
}

func (h Heatmap) Render(mini, maxi, xMax, yMax float64, divisor int, title, xLabel, yLabel, filename string) error {
	pal := moreland.ExtendedBlackBody().Palette(int(maxi - mini + 1))
	g := plotter.NewHeatMap(h, pal)
	g.Min = mini
	g.Max = maxi
	g.Rasterized = false
	g.NaN = color.RGBA{0, 255, 0, 255}
	g.Underflow = g.NaN
	g.Overflow = g.NaN

	p := plot.New()
	/* p.TextHandler = text.Plain{
		Fonts: font.DefaultCache,
	} */

	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	p.X.Tick.Marker = ticks{divisor: divisor}
	p.Y.Tick.Marker = ticks{divisor: divisor}

	p.Add(g)

	l := plot.NewLegend()
	thumbs := plotter.PaletteThumbnailers(pal)

	for i := range thumbs {
		l.Add(fmt.Sprintf("%.f", float64(i)+mini), thumbs[i])
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
