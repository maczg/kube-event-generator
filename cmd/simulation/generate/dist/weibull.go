package dist

import (
	"math"
	"math/rand"
)

// Weibull models a Weibull distribution with shape k>0 and scale λ>0.
// X ~ Weibull(k, λ).
// The Probability density function (PDF):  f(x) = (k / λ) * (x/λ)^(k-1) * e^(-(x/λ)^k),
// for x > 0. Mean = λ * Γ(1 + 1/k).
type Weibull struct {
	rng    *rand.Rand
	k      float64 // shape parameter
	lambda float64 // scale parameter
}

// NewWeibull returns a Weibull distribution with the provided shape k and scale λ.
// The shape k must be positive, and the scale λ must be positive.
// If k=1, the Weibull distribution is equivalent to the exponential distribution.
// If k>1, the Weibull distribution is right-skewed.
// If k<1, the Weibull distribution is left-skewed.
func NewWeibull(rng *rand.Rand, shape, scale float64) *Weibull {
	return &Weibull{
		rng:    rng,
		k:      shape,
		lambda: scale,
	}
}

// Next draws a random sample from the Weibull(k, λ) distribution.
//
// Uses inverse transform sampling:
// If U ~ Uniform(0,1), then
//
//	X = λ * (-ln(U))^(1/k).
func (w *Weibull) Next() float64 {
	u := w.rng.Float64()
	return w.lambda * math.Pow(-math.Log(u), 1.0/w.k)
}
