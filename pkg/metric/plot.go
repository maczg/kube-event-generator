package metric

import (
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type PlotOptions func(p *plot.Plot)

func newPlot(opts ...PlotOptions) *plot.Plot {
	p := plot.New()
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func WithTitle(title string) PlotOptions {
	return func(p *plot.Plot) {
		p.Title.Text = title
	}
}

func WithLabels(x, y string) PlotOptions {
	return func(p *plot.Plot) {
		p.X.Label.Text = x
		p.Y.Label.Text = y
	}
}

func GetBarChart(m Metric, xy ...string) *plot.Plot {
	pts := make(plotter.Values, len(m.Values.items))
	for i, record := range m.Values.items {
		pts[i] = record.value
	}

	bar, err := plotter.NewBarChart(pts, vg.Points(1))
	if err != nil {
		fmt.Printf("could not create bar chart: %v\n", err)
		return nil
	}

	plt := newPlot(WithTitle(m.Name), WithLabels(xy[0], xy[1]))
	plt.Add(bar)
	return plt
}

func GetLineChart(m Metric, xy ...string) *plot.Plot {
	pts := make(plotter.XYs, len(m.Values.items))
	for i, record := range m.Values.items {
		pts[i].X = float64(record.timestamp.Unix())
		pts[i].Y = record.value
	}

	line, err := plotter.NewLine(pts)
	if err != nil {
		fmt.Printf("could not create line: %v\n", err)
		return nil
	}

	plt := newPlot(WithTitle(m.Name), WithLabels(xy[0], xy[1]))
	plt.Add(line)
	return plt
}
