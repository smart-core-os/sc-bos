package auto

import (
	"math/rand"
	"time"
)

func float32Between(min, max float32) float32 {
	return min + (max-min)*rand.Float32()
}

func float64Between(min, max float64) float64 {
	return min + (max-min)*rand.Float64()
}

func oneOf[T any](vals ...T) T {
	return vals[rand.Intn(len(vals))]
}

func durationBetween(min, max time.Duration) time.Duration {
	return time.Duration(rand.Intn(int(max-min)) + int(min))
}

func randomBool(probability float64) bool {
	return rand.Float64() < probability
}

// float32Range represents a min/max range for random value generation
type float32Range struct {
	Min float32
	Max float32
}

// Random returns a random value within the range
func (r float32Range) Random() float32 {
	return float32Between(r.Min, r.Max)
}

// durationRange represents a min/max range for random duration generation
type durationRange struct {
	Min time.Duration
	Max time.Duration
}

// Random returns a random duration within the range
func (r durationRange) Random() time.Duration {
	return durationBetween(r.Min, r.Max)
}
