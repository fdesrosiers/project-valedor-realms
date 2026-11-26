// Package game runs the authoritative game loop: it owns the World,
// applies player commands, ticks the simulation and broadcasts state.
// It has no networking code, so it can be tested with fake clients.
package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"realm/internal/protocol"
	"realm/internal/world"
)

// Sender delivers an encoded message to one client without blocking.
// It returns false if the client's buffer is full (message dropped).
type Sender interface {
	Send(msg []byte) bool
}

// ErrGameStopped is returned when the game loop is no longer running.
var ErrGameStopped = errors.New("game: stopped")

type command interface{}

// joinResult is what the loop sends back to a waiting Join call.
type joinResult struct {
	id  world.PlayerID
	err error
}

type joinCmd struct {
	name   string
	sender Sender
	reply  chan joinResult
}

type leaveCmd struct{ id world.PlayerID }

type inputCmd struct {
	id     world.PlayerID
	dx, dy float64
}

type Game struct {
	world    *world.World
	tickRate int
	cmds     chan command
	done     chan struct{}
	clients  map[world.PlayerID]Sender
	nextID   int
}

func New(w *world.World, tickRate int) *Game {
	return &Game{
		world:    w,
		tickRate: tickRate,
		cmds:     make(chan command, 256),
		done:     make(chan struct{}),
		clients:  make(map[world.PlayerID]Sender),
	}
}

// Run is the game loop. Everything that touches the World happens here,
// on a single goroutine: no locks needed.
func (g *Game) Run(ctx context.Context) {
	defer close(g.done)
	ticker := time.NewTicker(time.Second / time.Duration(g.tickRate))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-g.cmds:
			g.handle(c)
		case <-ticker.C:
			g.step()
		}
	}
}

// Join registers a client and returns its player ID.
// It returns ErrGameStopped if the game loop is not running.
func (g *Game) Join(name string, s Sender) (world.PlayerID, error) {
	reply := make(chan joinResult, 1)
	if !g.submit(joinCmd{name: name, sender: s, reply: reply}) {
		return "", ErrGameStopped
	}
	select {
	case r := <-reply:
		return r.id, r.err
	case <-g.done:
		return "", ErrGameStopped
	}
}

func (g *Game) Leave(id world.PlayerID)                 { g.submit(leaveCmd{id: id}) }
func (g *Game) Input(id world.PlayerID, dx, dy float64) { g.submit(inputCmd{id: id, dx: dx, dy: dy}) }

func (g *Game) submit(c command) bool {
	select {
	case g.cmds <- c:
		return true
	case <-g.done:
		return false
	}
}

func (g *Game) handle(c command) {
	switch c := c.(type) {
	case joinCmd:
		g.nextID++
		id := world.PlayerID(fmt.Sprintf("p%d", g.nextID))
		name := c.name
		if name == "" {
			name = fmt.Sprintf("Hero %d", g.nextID)
		}
		if _, err := g.world.Join(id, name); err != nil {
			c.reply <- joinResult{err: fmt.Errorf("join %q: %w", name, err)}
			return
		}
		g.clients[id] = c.sender
		c.sender.Send(g.welcome(id))
		c.reply <- joinResult{id: id}
		log.Printf("%s joined as %q", id, name)
	case leaveCmd:
		g.world.Leave(c.id)
		delete(g.clients, c.id)
		log.Printf("%s left", c.id)
	case inputCmd:
		g.world.SetInput(c.id, world.Vec2{X: c.dx, Y: c.dy})
	}
}

func (g *Game) step() {
	g.world.Tick(1.0 / float64(g.tickRate))
	msg := encodeState(g.world.Snapshot())
	for _, s := range g.clients {
		s.Send(msg) // dropped if the client is too slow; next snapshot replaces it
	}
}

func (g *Game) welcome(id world.PlayerID) []byte {
	grid := g.world.Grid()
	walls := grid.Walls()
	info := protocol.MapInfo{Width: grid.Width, Height: grid.Height, Walls: make([][2]int, len(walls))}
	for i, t := range walls {
		info.Walls[i] = [2]int{t.X, t.Y}
	}
	return mustJSON(protocol.Welcome{Type: protocol.TypeWelcome, ID: string(id), TickRate: g.tickRate, Map: info})
}

func encodeState(s world.Snapshot) []byte {
	msg := protocol.State{Type: protocol.TypeState, Tick: s.Tick, Players: make([]protocol.PlayerState, len(s.Players))}
	for i, p := range s.Players {
		msg.Players[i] = protocol.PlayerState{ID: string(p.ID), Name: p.Name, Color: p.Color, X: p.Pos.X, Y: p.Pos.Y}
	}
	return mustJSON(msg)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // only fails on programmer error (unsupported types)
	}
	return b
}
