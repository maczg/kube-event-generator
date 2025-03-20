package generate

import (
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"math/rand"
)

type Params struct {
	Rng                 *rand.Rand
	Scenario            scenario.Scenario
	NumEvents           int
	ArrivalLambda       float64
	ArrivalScaleFactor  float64
	DurationShape       float64
	DurationK           float64
	DurationScaleFactor float64

	PodCpuScaleLambda float64
	PodCpuScaleK      float64
	PodCpuFactor      float64

	PodMemShapeK      float64
	PodMemScaleLambda float64
	PodMemFactor      float64
}

func NewGenerationParams(scn scenario.Scenario, opts ...Opt) Params {
	params := Params{
		Scenario:            scn,
		NumEvents:           50,
		ArrivalLambda:       1.0,
		ArrivalScaleFactor:  5.0,
		DurationShape:       1.5,
		DurationK:           2.0,
		DurationScaleFactor: 10.0,
		PodCpuScaleK:        1.0,
		PodCpuScaleLambda:   1.0,
		PodCpuFactor:        1.0,
		PodMemShapeK:        1.5,
		PodMemScaleLambda:   512.0,
		PodMemFactor:        1.0,
	}

	for _, opt := range opts {
		opt(&params)
	}
	return params
}

type Opt func(*Params)

func WithNumEvents(numEvents int) Opt {
	return func(params *Params) {
		params.NumEvents = numEvents
	}
}

func WithRand(r *rand.Rand) Opt {
	return func(params *Params) {
		params.Rng = r
	}
}

func WithArrivalLambda(arrivalLambda float64) Opt {
	return func(params *Params) {
		params.ArrivalLambda = arrivalLambda
	}
}

func WithArrivalScaleFactor(arrivalScaleFactor float64) Opt {
	return func(params *Params) {
		params.ArrivalScaleFactor = arrivalScaleFactor
	}
}

func WithDurationShape(durationShape float64) Opt {
	return func(params *Params) {
		params.DurationShape = durationShape
	}
}

func WithDurationK(durationK float64) Opt {
	return func(params *Params) {
		params.DurationK = durationK
	}
}

func WithDurationScaleFactor(durationScaleFactor float64) Opt {
	return func(params *Params) {
		params.DurationScaleFactor = durationScaleFactor
	}
}

func WithPodCpuShape(podCpuShape float64) Opt {
	return func(params *Params) {
		params.PodCpuScaleLambda = podCpuShape
	}
}

func WithPodCpuK(podCpuK float64) Opt {
	return func(params *Params) {
		params.PodCpuScaleK = podCpuK
	}
}

func WithPodCpuScaleFactor(podCpuScaleFactor float64) Opt {
	return func(params *Params) {
		params.PodCpuFactor = podCpuScaleFactor
	}
}

func WithPodMemShape(podMemShape float64) Opt {
	return func(params *Params) {
		params.PodMemShapeK = podMemShape
	}
}

func WithPodMemK(podMemK float64) Opt {
	return func(params *Params) {
		params.PodMemScaleLambda = podMemK
	}
}

func WithPodMemScaleFactor(podMemScaleFactor float64) Opt {
	return func(params *Params) {
		params.PodMemFactor = podMemScaleFactor
	}
}
