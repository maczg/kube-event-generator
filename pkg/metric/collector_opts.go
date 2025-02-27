package metric

type CollectorOpts func(c *Collector)

func WithResultDir(dir string) CollectorOpts {
	return func(c *Collector) {
		c.ResultDir = dir
	}
}
func WithMetric(name string) CollectorOpts {
	return func(c *Collector) {
		c.AddMetric(name)
	}
}
