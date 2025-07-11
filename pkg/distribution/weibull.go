package distribution

import (
	"math"
	"math/rand"
)

// Weibull models a Weibull distribution with shape shape>0 and scale λ>0.
// X ~ Weibull(shape, λ).
// The Probability density function (PDF):  f(x) = (shape / λ) * (x/λ)^(shape-1) * e^(-(x/λ)^shape),
// for x > 0. Mean = λ * Γ(1 + 1/shape).
type Weibull struct {
	rng *rand.Rand
	// shape is the shape parameter (k) of the Weibull distribution.
	shape float64
	// scale is the scale parameter (λ) of the Weibull distribution.
	scale float64 // scale parameter
}

// NewWeibull returns a Weibull distribution with the provided shape and scale λ.
// The shape must be positive, and the scale λ must be positive.
// If shape=1, the Weibull distribution is equivalent to the exponential distribution.
// If shape>1, the Weibull distribution is right-skewed.
// If shape<1, the Weibull distribution is left-skewed.
func NewWeibull(rng *rand.Rand, shape, scale float64) *Weibull {
	return &Weibull{
		rng:   rng,
		shape: shape,
		scale: scale,
	}
}

// Next draws a random sample from the Weibull(shape, λ) distribution.
//
// Uses inverse transform sampling:
// If U ~ Uniform(0,1), then
//
//	X = λ * (-ln(U))^(1/shape).
func (w *Weibull) Next() float64 {
	u := w.rng.Float64()
	return w.scale * math.Pow(-math.Log(u), 1.0/w.shape)
}
