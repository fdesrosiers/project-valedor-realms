// Package protocol defines the JSON messages exchanged over the WebSocket.
//
// Client -> server:
//
//	{"type":"join","name":"Papa"}           first message, exactly once
//	{"type":"input","dx":1,"dy":0}          movement intent, sent when it changes
//
// Server -> client:
//
//	{"type":"welcome","id":"p1","tickRate":20,"map":{...}}   reply to join
//	{"type":"state","tick":42,"players":[...]}               every tick
//
// JSON is fine for now; switch to MessagePack/Protobuf only if bandwidth matters.
package protocol

const (
	TypeJoin    = "join"
	TypeInput   = "input"
	TypeWelcome = "welcome"
	TypeState   = "state"
)

// ClientMessage is a union of all client->server messages, discriminated by Type.
type ClientMessage struct {
	Type string  `json:"type"`
	Name string  `json:"name,omitempty"`
	DX   float64 `json:"dx,omitempty"`
	DY   float64 `json:"dy,omitempty"`
}

type Welcome struct {
	Type     string  `json:"type"`
	ID       string  `json:"id"`
	TickRate int     `json:"tickRate"`
	Map      MapInfo `json:"map"`
}

type MapInfo struct {
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Walls  [][2]int `json:"walls"`
}

type State struct {
	Type    string        `json:"type"`
	Tick    uint64        `json:"tick"`
	Players []PlayerState `json:"players"`
}

type PlayerState struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color string  `json:"color"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}
