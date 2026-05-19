package main

import (
	"math"
	"math/rand/v2"
)

const earthRadiusM = 6_371_000.0

// GeoWalker simulates a device moving across the Earth's surface.
// Each Step advances the position by speed*stepSeconds metres in the current
// bearing, then randomly adjusts the bearing by up to ±drift degrees.
type GeoWalker struct {
	lat, lon float64 // current position, decimal degrees
	bearing  float64 // current heading, degrees (0=N, 90=E, 180=S, 270=W)
	speed    float64 // m/s
	drift    float64 // max random bearing change per step, degrees
}

func NewGeoWalker(lat, lon, bearing, speed, drift float64) *GeoWalker {
	return &GeoWalker{lat: lat, lon: lon, bearing: bearing, speed: speed, drift: drift}
}

// Step advances the walker one step and returns the new (lat, lon).
func (g *GeoWalker) Step(rng *rand.Rand, stepSeconds int) (lat, lon float64) {
	if g.drift > 0 {
		g.bearing += g.drift * (2*rng.Float64() - 1)
		g.bearing = math.Mod(g.bearing, 360)
		if g.bearing < 0 {
			g.bearing += 360
		}
	}

	dist := g.speed * float64(stepSeconds)
	φ1 := g.lat * math.Pi / 180
	λ1 := g.lon * math.Pi / 180
	θ := g.bearing * math.Pi / 180
	δ := dist / earthRadiusM

	φ2 := math.Asin(math.Sin(φ1)*math.Cos(δ) + math.Cos(φ1)*math.Sin(δ)*math.Cos(θ))
	λ2 := λ1 + math.Atan2(math.Sin(θ)*math.Sin(δ)*math.Cos(φ1), math.Cos(δ)-math.Sin(φ1)*math.Sin(φ2))
	// Normalise longitude to [-π, π]
	λ2 = math.Mod(λ2+math.Pi, 2*math.Pi) - math.Pi

	g.lat = φ2 * 180 / math.Pi
	g.lon = λ2 * 180 / math.Pi
	return g.lat, g.lon
}

// haversineDistM returns the great-circle distance in metres between two points.
func haversineDistM(lat1, lon1, lat2, lon2 float64) float64 {
	φ1, φ2 := lat1*math.Pi/180, lat2*math.Pi/180
	dφ := (lat2 - lat1) * math.Pi / 180
	dλ := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dφ/2)*math.Sin(dφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	return 2 * earthRadiusM * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
