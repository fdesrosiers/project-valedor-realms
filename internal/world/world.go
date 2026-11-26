package world

import (
	"fmt"
	"math"
	"sort"
)

type PlayerID string

const (
	// PlayerSize is the side of a player's square hitbox, in tiles.
	PlayerSize = 0.8
	// DefaultSpeed is movement speed in tiles per second.
	DefaultSpeed = 5.0

	half    = PlayerSize / 2
	epsilon = 1e-6
)

var palette = []string{"#e6194b", "#3cb44b", "#4363d8", "#f58231", "#911eb4", "#46f0f0", "#f032e6", "#ffe119"}

type Player struct {
	ID    PlayerID
	Name  string
	Color string
	Pos   Vec2
	input Vec2 // desired direction, length <= 1
}

// World is the authoritative simulation of one zone.
// It is NOT safe for concurrent use: a single goroutine (the game loop) owns it.
type World struct {
	grid      *Grid
	spawn     Vec2
	speed     float64
	players   map[PlayerID]*Player
	tick      uint64
	nextColor int
}

func New(grid *Grid, spawn Vec2) *World {
	return &World{
		grid:    grid,
		spawn:   spawn,
		speed:   DefaultSpeed,
		players: make(map[PlayerID]*Player),
	}
}

func (w *World) Grid() *Grid { return w.grid }

// Join adds a player at the spawn point.
func (w *World) Join(id PlayerID, name string) (*Player, error) {
	if _, exists := w.players[id]; exists {
		return nil, fmt.Errorf("world: player %q already joined", id)
	}
	p := &Player{
		ID:    id,
		Name:  name,
		Color: palette[w.nextColor%len(palette)],
		Pos:   w.spawn,
	}
	w.nextColor++
	w.players[id] = p
	return p, nil
}

func (w *World) Leave(id PlayerID) {
	delete(w.players, id)
}

// SetInput records the direction a player wants to move. The client sends
// intent, never positions: the server decides where the player ends up.
func (w *World) SetInput(id PlayerID, dir Vec2) bool {
	p, ok := w.players[id]
	if !ok {
		return false
	}
	p.input = dir.ClampLen()
	return true
}

// Tick advances the simulation by dt seconds.
func (w *World) Tick(dt float64) {
	w.tick++
	for _, p := range w.players {
		if p.input.IsZero() {
			continue
		}
		delta := p.input.Scale(w.speed * dt)
		// Resolve each axis separately so players slide along walls.
		p.Pos = w.moveX(p.Pos, delta.X)
		p.Pos = w.moveY(p.Pos, delta.Y)
	}
}

func (w *World) moveX(pos Vec2, dx float64) Vec2 {
	minY, maxY := tileSpan(pos.Y)
	pos.X = w.sweep(pos.X, dx, func(col int) bool { return w.columnBlocked(col, minY, maxY) })
	return pos
}

func (w *World) moveY(pos Vec2, dy float64) Vec2 {
	minX, maxX := tileSpan(pos.X)
	pos.Y = w.sweep(pos.Y, dy, func(row int) bool { return w.rowBlocked(row, minX, maxX) })
	return pos
}

// sweep moves a hitbox centred at c by d along one axis, checking every tile
// line it would enter, and stops flush against the first blocked one.
// Checking each line (instead of only the destination) means fast movement
// can never tunnel through a wall.
func (w *World) sweep(c, d float64, blocked func(line int) bool) float64 {
	if d > 0 {
		from, to := int(math.Floor(c+half)), int(math.Floor(c+d+half))
		for line := from + 1; line <= to; line++ {
			if blocked(line) {
				return float64(line) - half - epsilon
			}
		}
	} else if d < 0 {
		from, to := int(math.Floor(c-half)), int(math.Floor(c+d-half))
		for line := from - 1; line >= to; line-- {
			if blocked(line) {
				return float64(line+1) + half + epsilon
			}
		}
	}
	return c + d
}

func (w *World) columnBlocked(x, minY, maxY int) bool {
	for y := minY; y <= maxY; y++ {
		if w.grid.IsBlocked(x, y) {
			return true
		}
	}
	return false
}

func (w *World) rowBlocked(y, minX, maxX int) bool {
	for x := minX; x <= maxX; x++ {
		if w.grid.IsBlocked(x, y) {
			return true
		}
	}
	return false
}

// tileSpan returns the range of tiles a hitbox centred at c covers on one axis.
func tileSpan(c float64) (int, int) {
	return int(math.Floor(c - half)), int(math.Floor(c + half))
}

// canOccupy checks the four corners of the player's hitbox against the grid.
func (w *World) canOccupy(pos Vec2) bool {
	minX, maxX := tileSpan(pos.X)
	minY, maxY := tileSpan(pos.Y)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if w.grid.IsBlocked(x, y) {
				return false
			}
		}
	}
	return true
}

type PlayerState struct {
	ID    PlayerID
	Name  string
	Color string
	Pos   Vec2
}

type Snapshot struct {
	Tick    uint64
	Players []PlayerState // sorted by ID for deterministic output
}

func (w *World) Snapshot() Snapshot {
	s := Snapshot{Tick: w.tick, Players: make([]PlayerState, 0, len(w.players))}
	for _, p := range w.players {
		s.Players = append(s.Players, PlayerState{ID: p.ID, Name: p.Name, Color: p.Color, Pos: p.Pos})
	}
	sort.Slice(s.Players, func(i, j int) bool { return s.Players[i].ID < s.Players[j].ID })
	return s
}

// Player returns a copy of a player's state, for tests and debugging.
func (w *World) Player(id PlayerID) (PlayerState, bool) {
	p, ok := w.players[id]
	if !ok {
		return PlayerState{}, false
	}
	return PlayerState{ID: p.ID, Name: p.Name, Color: p.Color, Pos: p.Pos}, true
}
