package plotutil

import (
	"image/color"
	"math"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/spf13/cast"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type plotLineConf struct {
	XLabel     *string
	YLabel     *string
	PlotWidth  *font.Length
	PlotHeight *font.Length
	XTickNames []string
	LineLabel  *string
}

func WithXLabel(v string) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.XLabel = &v
	}
}

func WithYLabel(v string) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.YLabel = &v
	}
}

func WithWidth(v font.Length) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.PlotWidth = &v
	}
}

func WithHeight(v font.Length) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.PlotHeight = &v
	}
}

func WithXNames(n []string) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.XTickNames = slutil.Copy(n)
	}
}

func WithLineLabel(v string) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.LineLabel = &v
	}
}

type plotLineConfFunc func(*plotLineConf)

func PlotLine(title string, plots plotter.XYs, fname string, ops ...plotLineConfFunc) error {
	pgc := &plotLineConf{}
	for _, o := range ops {
		o(pgc)
	}

	var (
		xlabel     string = "X"
		ylabel     string = "Y"
		lineLabel  string
		plotWidth  = 10 * vg.Inch
		plotHeight = plotWidth / 2
	)
	if pgc.XLabel != nil {
		xlabel = *pgc.XLabel
	}
	if pgc.YLabel != nil {
		ylabel = *pgc.YLabel
	}
	if pgc.LineLabel != nil {
		lineLabel = *pgc.LineLabel
	}
	if pgc.PlotWidth != nil {
		plotWidth = *pgc.PlotWidth
	}
	if pgc.PlotHeight != nil {
		plotHeight = *pgc.PlotHeight
	}

	p := plot.New()
	p.Title.Text = "\n" + title
	p.Title.Padding = 0.1 * vg.Inch
	p.X.Label.Text = "\n" + xlabel + "\n"
	p.X.Label.Padding = 0.1 * vg.Inch
	p.Y.Label.Text = "\n" + ylabel + "\n"
	p.Y.Label.Padding = 0.1 * vg.Inch
	p.X.Max = float64(len(plots))
	p.Y.Min = 0
	p.Y.Max = -1
	for _, v := range plots {
		if v.Y > p.Y.Max {
			p.Y.Max = v.Y
		}
	}

	if len(pgc.XTickNames) > 0 {
		p.NominalX(pgc.XTickNames...)
		p.X.Padding = 0.1 * vg.Inch
		p.X.LineStyle = draw.LineStyle{
			Color: color.Black,
			Width: vg.Points(0.5),
		}
	}

	drawLine(p, plots, 1, lineLabel)
	err := p.Save(plotWidth, plotHeight, fname)
	if err != nil {
		return err
	}
	return nil
}

func drawLine(p *plot.Plot, dat plotter.XYs, color int, lineLabel string) error {

	// find min, max
	var min, max float64 = math.MaxFloat64, 0
	var mini, maxi int
	for i, xy := range dat {
		if xy.Y < min {
			mini = i
			min = xy.Y
		}
		if xy.Y >= max {
			maxi = i
			max = xy.Y
		}
	}

	line, err := plotter.NewLine(dat)
	if err != nil {
		return errs.Wrap(err)
	}

	line.LineStyle.Color = plotutil.Color(color)
	p.Add(line)

	if min < max {
		lineLabels, err := plotter.NewLabels(plotter.XYLabels{
			XYs:    []plotter.XY{{X: float64(mini), Y: min + 1}},
			Labels: []string{cast.ToString(min)},
		})
		if err != nil {
			return err
		}
		p.Add(lineLabels)
	}

	lineLabels, err := plotter.NewLabels(plotter.XYLabels{
		XYs:    []plotter.XY{{X: float64(maxi), Y: max + 1}},
		Labels: []string{cast.ToString(max)},
	})
	if err != nil {
		return err
	}
	p.Add(lineLabels)

	lineLabels, err = plotter.NewLabels(plotter.XYLabels{
		XYs:    []plotter.XY{{X: float64(1), Y: dat[0].Y + 1}},
		Labels: []string{lineLabel},
	})
	if err != nil {
		return err
	}
	p.Add(lineLabels)

	return nil
}
