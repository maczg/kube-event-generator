package metric

import "gonum.org/v1/plot"

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
