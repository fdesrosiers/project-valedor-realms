# Realm — walking skeleton

Two players connect, move with WASD or the arrow keys, and see each other move. The server is the authority over everything: clients send *intent* (a direction), and the server decides positions.

## Run

```bash
go mod tidy                 # fetches github.com/coder/websocket, writes go.sum
go test ./...               # engine tests (Arrange / Act / Assert)
go run ./cmd/server         # listens on :8080, 20 ticks/s
```

Client: open `client/` in Godot 4.3+ and press F5.
To test with two players on one machine: **Debug → Customize Run Instances → Enable Multiple Instances → 2**.
To play from another computer on your network, change `server_url` on the `Main` node to `ws://<server-ip>:8080/ws`.

## Structure

```
cmd/server/            entry point (flags, HTTP, graceful shutdown)
internal/world/        pure simulation: grid, players, movement, collision (no I/O)
internal/game/         game loop: owns the World on one goroutine, commands in, snapshots out
internal/transport/    WebSocket <-> game (the only package that knows about the network)
internal/protocol/     JSON message types
client/                Godot 4 client (one script, draws with _draw)
```

**Rule:** only the game loop goroutine touches `World`. Connections talk to it through commands on a channel, so no locks are needed. Keep it that way as systems are added.

## Protocol

| Direction | Message |
|---|---|
| C→S | `{"type":"join","name":"Papa"}` (first message, exactly once) |
| C→S | `{"type":"input","dx":1,"dy":0}` (sent only when the direction changes) |
| S→C | `{"type":"welcome","id":"p1","tickRate":20,"map":{"width":64,"height":64,"walls":[[0,0],...]}}` |
| S→C | `{"type":"state","tick":42,"players":[{"id":"p1","name":"Papa","color":"#e6194b","x":32.5,"y":32.5}]}` (every tick) |

Units are tiles; the client multiplies by 32 px.

## Next steps

1. Headless Go bot client that plays scripted scenarios over the real protocol.
2. First monster plus combat: health, auto-attack, death and respawn (start with an `Entity` that both players and monsters use).
3. Replace `DefaultZone()` with an LDtk map shared by the server and the client.
