package territory

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
	mathrand "math/rand"
)

// NewSeedRef creates a fresh random seed reference. Store it on the mission
// (random_seed_ref) so staff can audit and replay every roll of a
// resolution.
func NewSeedRef() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand never fails on supported platforms; fall back to a
		// zero seed rather than panicking mid-request.
		return "seed_0000000000000000"
	}
	return "seed_" + hex.EncodeToString(buf)
}

// Roll is a deterministic random source derived from a seed reference.
// Replaying NewRoll with the same seedRef reproduces the exact same
// sequence of rolls, which lets staff audit resolved missions.
type Roll struct {
	seedRef string
	rng     *mathrand.Rand
}

// NewRoll builds a reproducible roll sequence from a seed reference.
func NewRoll(seedRef string) *Roll {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(seedRef))
	seed := int64(binary.BigEndian.Uint64(hasher.Sum(nil)))
	return &Roll{
		seedRef: seedRef,
		rng:     mathrand.New(mathrand.NewSource(seed)),
	}
}

// SeedRef returns the seed reference this roll sequence was created from.
func (r *Roll) SeedRef() string {
	return r.seedRef
}

// Percent returns a uniform integer in [0, 100).
func (r *Roll) Percent() int {
	return r.rng.Intn(100)
}

// Float returns a uniform float in [0, 1).
func (r *Roll) Float() float64 {
	return r.rng.Float64()
}

// IntN returns a uniform integer in [0, n). n must be positive.
func (r *Roll) IntN(n int) int {
	return r.rng.Intn(n)
}

// Jitter returns a multiplier in [1-fraction, 1+fraction].
func (r *Roll) Jitter(fraction float64) float64 {
	if fraction <= 0 {
		return 1
	}
	return 1 + (r.rng.Float64()*2-1)*fraction
}
