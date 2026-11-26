extends Node2D
## Walking-skeleton client: connects, sends movement intent, draws the
## server's state. The server is the authority; we only render and smooth.

const TILE := 32.0          # pixels per tile
const SMOOTHING := 15.0     # higher = snappier interpolation toward server position

@export var server_url := "ws://localhost:8080/ws"
@export var player_name := ""   # empty = server picks "Hero N"

var ws := WebSocketPeer.new()
var joined := false
var my_id := ""
var map_size := Vector2i.ZERO
var walls: Array = []
var players := {}           # id -> {name, color, pos, target}
var last_input := Vector2.ZERO

@onready var camera: Camera2D = $Camera2D


func _ready() -> void:
	var err := ws.connect_to_url(server_url)
	if err != OK:
		push_error("Could not connect to %s (error %d)" % [server_url, err])


func _process(delta: float) -> void:
	ws.poll()
	match ws.get_ready_state():
		WebSocketPeer.STATE_OPEN:
			if not joined:
				_send({"type": "join", "name": player_name})
				joined = true
			while ws.get_available_packet_count() > 0:
				_handle(ws.get_packet().get_string_from_utf8())
			_send_input()
		WebSocketPeer.STATE_CLOSED:
			push_warning("Disconnected (%d) %s" % [ws.get_close_code(), ws.get_close_reason()])
			set_process(false)

	# Smooth between 20 Hz server snapshots so movement looks fluid at 60+ fps.
	var t := clampf(delta * SMOOTHING, 0.0, 1.0)
	for id in players:
		players[id].pos = players[id].pos.lerp(players[id].target, t)

	if players.has(my_id):
		camera.position = players[my_id].pos * TILE
	queue_redraw()


func _send_input() -> void:
	var dir := Vector2(
		float(Input.is_key_pressed(KEY_D) or Input.is_key_pressed(KEY_RIGHT)) - float(Input.is_key_pressed(KEY_A) or Input.is_key_pressed(KEY_LEFT)),
		float(Input.is_key_pressed(KEY_S) or Input.is_key_pressed(KEY_DOWN)) - float(Input.is_key_pressed(KEY_W) or Input.is_key_pressed(KEY_UP)))
	if dir != last_input:   # only send when intent changes
		last_input = dir
		_send({"type": "input", "dx": dir.x, "dy": dir.y})


func _send(msg: Dictionary) -> void:
	ws.send_text(JSON.stringify(msg))


func _handle(text: String) -> void:
	var msg = JSON.parse_string(text)
	if typeof(msg) != TYPE_DICTIONARY:
		return
	match msg.get("type"):
		"welcome":
			my_id = msg.id
			map_size = Vector2i(int(msg.map.width), int(msg.map.height))
			walls = msg.map.walls
			print("Joined as ", my_id)
		"state":
			var seen := {}
			for s in msg.players:
				seen[s.id] = true
				var target := Vector2(s.x, s.y)
				if players.has(s.id):
					players[s.id].target = target
				else:
					players[s.id] = {"name": s.name, "color": Color(s.color), "pos": target, "target": target}
			for id in players.keys():
				if not seen.has(id):
					players.erase(id)


func _draw() -> void:
	# Ground and walls
	draw_rect(Rect2(Vector2.ZERO, Vector2(map_size) * TILE), Color("#2d4a2b"))
	for w in walls:
		draw_rect(Rect2(Vector2(w[0], w[1]) * TILE, Vector2(TILE, TILE)), Color("#5a5a5a"))

	# Players: hitbox is 0.8 tile, same as the server
	var size := Vector2(0.8, 0.8) * TILE
	for id in players:
		var p = players[id]
		var top_left: Vector2 = p.pos * TILE - size / 2
		draw_rect(Rect2(top_left, size), p.color)
		if id == my_id:
			draw_rect(Rect2(top_left, size), Color.WHITE, false, 2.0)
		draw_string(ThemeDB.fallback_font, top_left + Vector2(0, -4), p.name, HORIZONTAL_ALIGNMENT_LEFT, -1, 12)
