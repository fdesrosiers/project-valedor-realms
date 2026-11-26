package world

import "math"

// Vec2 is a position or direction in tile units (1.0 = one tile).
type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2      { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Len() float64         { return math.Hypot(v.X, v.Y) }
func (v Vec2) IsZero() bool         { return v.X == 0 && v.Y == 0 }

// ClampLen returns v unchanged if its length is <= 1, otherwise the unit vector.
// Prevents diagonal movement from being faster than straight movement.
func (v Vec2) ClampLen() Vec2 {
	l := v.Len()
	if l <= 1 {
		return v
	}
	return v.Scale(1 / l)
}
