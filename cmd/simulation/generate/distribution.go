package generate

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"time"
)

// For arrival times: returns a time.Duration to wait until the *next* event
func generateArrival(dist string) time.Duration {
	switch dist {
	case "exponential":
		// Exponential with an average of 10 seconds, for example
		lambda := 1.0 / 10.0
		interArrival := rng.ExpFloat64() / lambda
		return time.Duration(interArrival) * time.Second
	case "uniform":
		// Uniform between 5s and 15s
		return time.Duration(rng.Intn(10)+5) * time.Second
	// Add more distributions as needed
	default:
		// fallback
		return 10 * time.Second
	}
}

// For the duration each pod stays alive
func generateDuration(dist string) time.Duration {
	switch dist {
	case "exponential":
		lambda := 1.0 / 30.0 // average 30 seconds
		val := rng.ExpFloat64() / lambda
		return time.Duration(val) * time.Second
	case "uniform":
		// Uniform between 20s and 40s
		return time.Duration(rng.Intn(20)+20) * time.Second
	default:
		// fallback
		return 30 * time.Second
	}
}

// For CPU or memory quantity as a resource.Quantity
func generateResourceQuantity(dist string) resource.Quantity {
	switch dist {
	case "uniform":
		// Example: CPU between 50m and 250m
		val := rng.Intn(200) + 50 // integer in [50, 250]
		return resource.MustParse(fmt.Sprintf("%dm", val))
	case "normal":
		// Example: memory with mean=256Mi, stdev=64
		mean := 256.0
		stdev := 64.0
		sample := rng.NormFloat64()*stdev + mean
		// Bound it so it doesn't go negative or huge:
		if sample < 32 {
			sample = 32
		}
		return resource.MustParse(fmt.Sprintf("%dMi", int(sample)))
	default:
		// fallback
		return resource.MustParse("100m")
	}
}
