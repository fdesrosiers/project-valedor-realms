package world

import "fmt"

// Tile is an integer tile coordinate.
type Tile struct {
	X, Y int
}

// Grid is the static collision layer of a zone.
// Later this will be loaded from map data (e.g. LDtk JSON) shared with the client.
type Grid struct {
	Width, Height int
	blocked       []bool
}

// ParseGrid builds a grid from ASCII rows: '#' = wall, anything else = walkable.
// Handy for tests and small hand-made maps.
func ParseGrid(rows []string) (*Grid, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("grid: no rows")
	}
	w := len(rows[0])
	g := &Grid{Width: w, Height: len(rows), blocked: make([]bool, w*len(rows))}
	for y, row := range rows {
		if len(row) != w {
			return nil, fmt.Errorf("grid: row %d has length %d, want %d", y, len(row), w)
		}
		for x, c := range row {
			g.blocked[y*w+x] = c == '#'
		}
	}
	return g, nil
}

// IsBlocked reports whether a tile is a wall. Out-of-bounds counts as blocked.
func (g *Grid) IsBlocked(x, y int) bool {
	if x < 0 || y < 0 || x >= g.Width || y >= g.Height {
		return true
	}
	return g.blocked[y*g.Width+x]
}

// Walls lists every blocked tile (sent to clients so they can draw the map).
func (g *Grid) Walls() []Tile {
	var walls []Tile
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			if g.blocked[y*g.Width+x] {
				walls = append(walls, Tile{x, y})
			}
		}
	}
	return walls
}

// DefaultZone is a placeholder 64x64 zone: border walls plus 2x2 pillars.
// Throwaway until the real map pipeline exists.
func DefaultZone() *Grid {
	const size = 64
	g := &Grid{Width: size, Height: size, blocked: make([]bool, size*size)}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			border := x == 0 || y == 0 || x == size-1 || y == size-1
			pillar := (x%8 == 4 || x%8 == 5) && (y%8 == 4 || y%8 == 5)
			g.blocked[y*size+x] = border || pillar
		}
	}
	return g
}

// DefaultSpawn is a walkable point in DefaultZone.
var DefaultSpawn = Vec2{32.5, 32.5}
