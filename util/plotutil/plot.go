package plotutil

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"os"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/spf13/cast"
	"golang.org/x/image/font/opentype"
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
	Format     *string
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

func WithFormat(v string) plotLineConfFunc {
	return func(pgc *plotLineConf) {
		pgc.Format = &v
	}
}

type plotLineConfFunc func(*plotLineConf)

func PlotLine(title string, plots plotter.XYs, w io.Writer, ops ...plotLineConfFunc) error {
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
		format     = "png"
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

	// draw line on plot
	drawLine(p, plots, 1, lineLabel)

	c, err := p.WriterTo(plotWidth, plotHeight, format)
	if err != nil {
		return err
	}
	_, err = c.WriteTo(w)
	return err
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
			XYs:    []plotter.XY{{X: float64(mini), Y: min}},
			Labels: []string{cast.ToString(min)},
		})
		if err != nil {
			return err
		}
		p.Add(lineLabels)
	}

	lineLabels, err := plotter.NewLabels(plotter.XYLabels{
		XYs:    []plotter.XY{{X: float64(maxi), Y: max}},
		Labels: []string{cast.ToString(max)},
	})
	if err != nil {
		return err
	}
	p.Add(lineLabels)

	lineLabels, err = plotter.NewLabels(plotter.XYLabels{
		XYs:    []plotter.XY{{X: float64(1), Y: dat[0].Y}},
		Labels: []string{lineLabel},
	})
	if err != nil {
		return err
	}
	p.Add(lineLabels)

	return nil
}

func LoadFontFile(path string) (*font.Font, error) {
	ttf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fontTTF, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse '%s' as font: %w", path, err)
	}

	f := font.Font{}
	font.DefaultCache.Add([]font.Face{
		{
			Font: f,
			Face: fontTTF,
		},
	})
	return &f, nil
}

func LoadFont(byt []byte) (*font.Font, error) {
	fontTTF, err := opentype.Parse(byt)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	f := font.Font{}
	font.DefaultCache.Add([]font.Face{
		{
			Font: f,
			Face: fontTTF,
		},
	})
	if !font.DefaultCache.Has(f) {
		return nil, errs.NewErrf("Font not loaded")
	}
	return &f, nil
}

func ChangeDefaultFont(f *font.Font) {
	plot.DefaultFont = *f
}
