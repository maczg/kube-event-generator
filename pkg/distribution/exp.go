package distribution

import "math/rand"

// Exponential models an exponential distribution X ~ Exp(λ).
// The mean is 1/λ.
type Exponential struct {
	rng    *rand.Rand
	lambda float64
}

// NewExponential returns an Exponential distribution with parameter λ.
func NewExponential(rng *rand.Rand, lambda float64) *Exponential {
	return &Exponential{rng: rng, lambda: lambda}
}

// Next draws the next sample from Exp(λ).
func (e *Exponential) Next() float64 {
	// Rand.ExpFloat64() returns an exponentially distributed float64 with mean 1.
	// Dividing by λ scales it so the mean is 1/λ.
	return e.rng.ExpFloat64() / e.lambda
}
