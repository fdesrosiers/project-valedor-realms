package game

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"realm/internal/protocol"
	"realm/internal/world"
)

type fakeSender struct {
	msgs [][]byte
}

func (f *fakeSender) Send(msg []byte) bool {
	f.msgs = append(f.msgs, msg)
	return true
}

func (f *fakeSender) last(t *testing.T, v any) {
	t.Helper()
	if len(f.msgs) == 0 {
		t.Fatal("no message received")
	}
	if err := json.Unmarshal(f.msgs[len(f.msgs)-1], v); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func newTestGame(t *testing.T) *Game {
	t.Helper()
	g, err := world.ParseGrid([]string{
		"##########",
		"#........#",
		"##########",
	})
	if err != nil {
		t.Fatal(err)
	}
	return New(world.New(g, world.Vec2{X: 1.5, Y: 1.5}), 20)
}

// join drives the command directly, bypassing the Run loop.
func join(g *Game, name string, s Sender) world.PlayerID {
	reply := make(chan joinResult, 1)
	g.handle(joinCmd{name: name, sender: s, reply: reply})
	return (<-reply).id
}

func TestJoin_SendsWelcomeWithIDAndMap(t *testing.T) {
	// Arrange
	g := newTestGame(t)
	client := &fakeSender{}

	// Act
	id := join(g, "Papa", client)

	// Assert
	var msg protocol.Welcome
	client.last(t, &msg)
	if msg.Type != protocol.TypeWelcome || msg.ID != string(id) {
		t.Errorf("welcome = %+v, want type welcome and id %s", msg, id)
	}
	if msg.Map.Width != 10 || msg.Map.Height != 3 {
		t.Errorf("map size = %dx%d, want 10x3", msg.Map.Width, msg.Map.Height)
	}
}

func TestJoin_EmptyName_GetsDefaultName(t *testing.T) {
	// Arrange
	g := newTestGame(t)

	// Act
	id := join(g, "", &fakeSender{})

	// Assert
	p, _ := g.world.Player(id)
	if p.Name == "" {
		t.Error("expected a default name")
	}
}

func TestStep_BroadcastsStateToAllClients(t *testing.T) {
	// Arrange
	g := newTestGame(t)
	papa, fiston := &fakeSender{}, &fakeSender{}
	join(g, "Papa", papa)
	join(g, "Fiston", fiston)

	// Act
	g.step()

	// Assert
	for name, c := range map[string]*fakeSender{"papa": papa, "fiston": fiston} {
		var msg protocol.State
		c.last(t, &msg)
		if msg.Type != protocol.TypeState || len(msg.Players) != 2 {
			t.Errorf("%s got %+v, want state with 2 players", name, msg)
		}
	}
}

func TestInput_PlayerMovesOnNextStep(t *testing.T) {
	// Arrange
	g := newTestGame(t)
	client := &fakeSender{}
	id := join(g, "Papa", client)
	g.handle(inputCmd{id: id, dx: 1, dy: 0})

	// Act
	g.step()

	// Assert
	var msg protocol.State
	client.last(t, &msg)
	if msg.Players[0].X <= 1.5 {
		t.Errorf("x = %.3f, expected player to move right", msg.Players[0].X)
	}
}

func TestLeave_ClientNoLongerReceivesState(t *testing.T) {
	// Arrange
	g := newTestGame(t)
	client := &fakeSender{}
	id := join(g, "Papa", client)
	before := len(client.msgs)

	// Act
	g.handle(leaveCmd{id: id})
	g.step()

	// Assert
	if len(client.msgs) != before {
		t.Errorf("client received %d messages after leaving", len(client.msgs)-before)
	}
}

func TestJoin_WhileLoopRunning_ReturnsPlayerID(t *testing.T) {
	// Arrange
	g := newTestGame(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	// Act
	id, err := g.Join("Papa", &fakeSender{})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected a player ID")
	}
}

func TestJoin_AfterLoopStopped_ReturnsErrGameStopped(t *testing.T) {
	// Arrange
	g := newTestGame(t)
	ctx, cancel := context.WithCancel(context.Background())
	go g.Run(ctx)
	cancel()
	select {
	case <-g.done:
	case <-time.After(time.Second):
		t.Fatal("game loop did not stop")
	}

	// Act
	id, err := g.Join("Papa", &fakeSender{})

	// Assert
	if !errors.Is(err, ErrGameStopped) {
		t.Errorf("err = %v, want ErrGameStopped", err)
	}
	if id != "" {
		t.Errorf("id = %q, want empty", id)
	}
}
