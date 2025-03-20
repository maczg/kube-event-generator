# Scenario Generation

Scenario Generation relies on distribution to generate random
inter-arrival times and service times (duration) for pods.

Each generation involves also the generation of resource requests for each pod of the event.


### Arrival Time Distribution

The arrival time distribution is used to generate the time between the arrival of two consecutive pods.
It follows Poisson (Exponential) distribution with a given lambda (λ) parameter.

E.g., λ = 1.0 => mean inter-arrival time = 1.0 second. If λ=0.5 (1/2) => mean 2.0 seconds

### Service Time Distribution

The service time distribution is used to generate the time a pod takes to complete its execution.
It follows Weibull distribution with a given shape (k) and scale (λ) parameters.

E.g., k=1.0, λ=1.0 => mean service time = 1.0 second. If k=2.0, λ=1.0 => mean 1.27 seconds

### Resource Request Generation

The resource request generation is used to generate the resource requests for each pod.
CPU from Weibull(1.2, 2.0) => "small",
Memory from Weibull(1.5, 512) => "larger"

