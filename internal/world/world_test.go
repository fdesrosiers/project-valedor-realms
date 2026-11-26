package world

import (
	"math"
	"testing"
)

const tolerance = 1e-3

func newTestWorld(t *testing.T, rows ...string) *World {
	t.Helper()
	g, err := ParseGrid(rows)
	if err != nil {
		t.Fatalf("parse grid: %v", err)
	}
	return New(g, Vec2{1.5, 1.5})
}

func assertNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("%s = %.4f, want %.4f", name, got, want)
	}
}

func TestJoin_PlayerSpawnsAtSpawnPoint(t *testing.T) {
	// Arrange
	w := newTestWorld(t,
		"#####",
		"#...#",
		"#####")

	// Act
	_, err := w.Join("p1", "Papa")

	// Assert
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	p, _ := w.Player("p1")
	assertNear(t, "x", p.Pos.X, 1.5)
	assertNear(t, "y", p.Pos.Y, 1.5)
}

func TestJoin_SameIDTwice_ReturnsError(t *testing.T) {
	// Arrange
	w := newTestWorld(t, "#####", "#...#", "#####")
	w.Join("p1", "Papa")

	// Act
	_, err := w.Join("p1", "Again")

	// Assert
	if err == nil {
		t.Fatal("expected error on duplicate join")
	}
}

func TestJoin_PlayersGetDifferentColors(t *testing.T) {
	// Arrange
	w := newTestWorld(t, "#####", "#...#", "#####")

	// Act
	a, _ := w.Join("p1", "Papa")
	b, _ := w.Join("p2", "Fiston")

	// Assert
	if a.Color == b.Color {
		t.Errorf("both players got color %s", a.Color)
	}
}

func TestTick_PlayerMovesInInputDirection(t *testing.T) {
	// Arrange
	w := newTestWorld(t,
		"##########",
		"#........#",
		"##########")
	w.Join("p1", "Papa")
	w.SetInput("p1", Vec2{1, 0})

	// Act
	w.Tick(0.2) // 5 tiles/s * 0.2s = 1 tile

	// Assert
	p, _ := w.Player("p1")
	assertNear(t, "x", p.Pos.X, 2.5)
	assertNear(t, "y", p.Pos.Y, 1.5)
}

func TestTick_NoInput_PlayerDoesNotMove(t *testing.T) {
	// Arrange
	w := newTestWorld(t, "#####", "#...#", "#####")
	w.Join("p1", "Papa")

	// Act
	w.Tick(1.0)

	// Assert
	p, _ := w.Player("p1")
	assertNear(t, "x", p.Pos.X, 1.5)
}

func TestTick_PlayerStopsFlushAgainstWall(t *testing.T) {
	// Arrange: floor tiles x=1..3, wall at x=4
	w := newTestWorld(t,
		"#####",
		"#...#",
		"#####")
	w.Join("p1", "Papa")
	w.SetInput("p1", Vec2{1, 0})

	// Act
	w.Tick(10) // would travel 50 tiles without walls

	// Assert: hitbox edge (x + 0.4) touches the wall at x=4
	p, _ := w.Player("p1")
	assertNear(t, "x", p.Pos.X, 4-half)
}

func TestTick_DiagonalInputIsNotFaster(t *testing.T) {
	// Arrange
	w := newTestWorld(t,
		"##########",
		"#........#",
		"#........#",
		"#........#",
		"#........#",
		"##########")
	w.Join("p1", "Papa")
	w.SetInput("p1", Vec2{1, 1})

	// Act
	w.Tick(0.2)

	// Assert: distance travelled is 1 tile, not sqrt(2)
	p, _ := w.Player("p1")
	moved := Vec2{p.Pos.X - 1.5, p.Pos.Y - 1.5}.Len()
	assertNear(t, "distance", moved, 1.0)
}

func TestTick_PlayerSlidesAlongWall(t *testing.T) {
	// Arrange: moving diagonally up-right in a 1-tile-high corridor
	w := newTestWorld(t,
		"##########",
		"#........#",
		"##########")
	w.Join("p1", "Papa")
	w.SetInput("p1", Vec2{1, -1})

	// Act
	w.Tick(0.2)

	// Assert: blocked vertically, but still progresses horizontally
	p, _ := w.Player("p1")
	if p.Pos.X <= 1.5 {
		t.Errorf("x = %.3f, expected player to slide right", p.Pos.X)
	}
	assertNear(t, "y", p.Pos.Y, 1+half)
}

func TestLeave_PlayerRemovedFromSnapshot(t *testing.T) {
	// Arrange
	w := newTestWorld(t, "#####", "#...#", "#####")
	w.Join("p1", "Papa")
	w.Join("p2", "Fiston")

	// Act
	w.Leave("p1")

	// Assert
	snap := w.Snapshot()
	if len(snap.Players) != 1 || snap.Players[0].ID != "p2" {
		t.Errorf("snapshot players = %+v, want only p2", snap.Players)
	}
}

func TestSetInput_UnknownPlayer_ReturnsFalse(t *testing.T) {
	// Arrange
	w := newTestWorld(t, "#####", "#...#", "#####")

	// Act
	ok := w.SetInput("ghost", Vec2{1, 0})

	// Assert
	if ok {
		t.Error("expected SetInput to fail for unknown player")
	}
}

func TestDefaultZone_SpawnIsWalkable(t *testing.T) {
	// Arrange
	w := New(DefaultZone(), DefaultSpawn)

	// Act
	ok := w.canOccupy(DefaultSpawn)

	// Assert
	if !ok {
		t.Error("default spawn is inside a wall")
	}
}

func TestTick_FastMovementDoesNotTunnelThroughThinWall(t *testing.T) {
	// Arrange: a 1-tile wall at x=3 with open floor on both sides
	w := newTestWorld(t,
		"#########",
		"#..#....#",
		"#########")
	w.Join("p1", "Papa")
	w.SetInput("p1", Vec2{1, 0})

	// Act
	w.Tick(1.0) // 5 tiles in one step: destination is beyond the wall

	// Assert: stopped before the wall, not on the other side
	p, _ := w.Player("p1")
	assertNear(t, "x", p.Pos.X, 3-half)
}

func TestTick_MovingLeftStopsFlushAgainstWall(t *testing.T) {
	// Arrange
	w := newTestWorld(t,
		"#####",
		"#...#",
		"#####")
	w.Join("p1", "Papa")
	w.SetInput("p1", Vec2{-1, 0})

	// Act
	w.Tick(10)

	// Assert
	p, _ := w.Player("p1")
	assertNear(t, "x", p.Pos.X, 1+half)
}
