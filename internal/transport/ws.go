// Package transport connects WebSocket clients to the game loop.
package transport

import (
	"context"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"realm/internal/game"
	"realm/internal/protocol"
)

// sendBuffer is how many outgoing messages a slow client can lag behind
// before new ones are dropped (at 20 ticks/s, 64 is ~3 seconds).
const sendBuffer = 64

type conn struct {
	out chan []byte
}

func (c *conn) Send(msg []byte) bool {
	select {
	case c.out <- msg:
		return true
	default:
		return false
	}
}

// Handler upgrades HTTP requests to WebSocket and runs one player session.
func Handler(g *game.Game) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Dev only: accept any Origin. Native Godot clients send none anyway;
			// restrict with OriginPatterns before exposing a web build.
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("accept: %v", err)
			return
		}
		defer ws.CloseNow()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// The first message must be a join.
		var hello protocol.ClientMessage
		if err := wsjson.Read(ctx, ws, &hello); err != nil || hello.Type != protocol.TypeJoin {
			ws.Close(websocket.StatusPolicyViolation, "expected join")
			return
		}

		c := &conn{out: make(chan []byte, sendBuffer)}
		id, err := g.Join(hello.Name, c)
		if err != nil {
			log.Printf("join: %v", err)
			ws.Close(websocket.StatusTryAgainLater, "server unavailable")
			return
		}
		defer g.Leave(id)

		go writeLoop(ctx, ws, c.out)

		for {
			var msg protocol.ClientMessage
			if err := wsjson.Read(ctx, ws, &msg); err != nil {
				return // disconnected
			}
			switch msg.Type {
			case protocol.TypeInput:
				g.Input(id, msg.DX, msg.DY)
			default:
				log.Printf("%s: unknown message type %q", id, msg.Type)
			}
		}
	}
}

func writeLoop(ctx context.Context, ws *websocket.Conn, out <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			if err := ws.Write(ctx, websocket.MessageText, msg); err != nil {
				ws.CloseNow() // unblocks the read loop, which then cleans up
				return
			}
		}
	}
}
