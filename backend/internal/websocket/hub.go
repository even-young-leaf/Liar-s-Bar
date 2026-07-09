package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"liars-bar/internal/game"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Client struct {
	UserID   uint
	Username string
	Nickname string
	Conn     *websocket.Conn
	Hub      *Hub
	Send     chan []byte
	RoomID   uint
	IsAI     bool
	mu       sync.Mutex
}

func NewClient(userID uint, username, nickname string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		UserID:   userID,
		Username: username,
		Nickname: nickname,
		Conn:     conn,
		Hub:      hub,
		Send:     make(chan []byte, 256),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "PLAYER_JOIN" {
			roomID, err := roomIDFromPayload(msg.Payload)
			if err != nil {
				c.SendMessage(Message{Type: "ERROR", Payload: map[string]interface{}{"msg": err.Error()}})
				continue
			}
			if err := c.Hub.JoinRoom(roomID, c); err != nil {
				c.SendMessage(Message{Type: "ERROR", Payload: map[string]interface{}{"msg": err.Error()}})
			}
			continue
		}

		if c.RoomID > 0 {
			c.Hub.RouteMessage(c.RoomID, c.UserID, msg)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.Send <- data:
	default:
	}
}

type Hub struct {
	Clients    map[uint]*Client
	Rooms      map[uint]*GameRoom
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex

	// OnGameOver is invoked exactly once per game when it ends, so that the
	// caller can persist stats / records. Set by main.go.
	OnGameOver func(roomID uint, winnerID uint, players []*game.Player)

	// RoomService is used for database cleanup
	RoomService interface {
		CleanupStaleRooms(maxAgeMinutes int) (int, error)
	}
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[uint]*Client),
		Rooms:      make(map[uint]*GameRoom),
		Register:   make(chan *Client, 256),
		Unregister: make(chan *Client, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("Client registered: %d (%s)", client.UserID, client.Nickname)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.UserID]; ok {
				delete(h.Clients, client.UserID)
				close(client.Send)
				if client.RoomID > 0 {
					if room, exists := h.Rooms[client.RoomID]; exists {
						room.HandleEvent(GameEvent{
							Type:     "PLAYER_LEAVE",
							PlayerID: client.UserID,
						})
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %d", client.UserID)
		}
	}
}

func (h *Hub) GetClient(userID uint) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Clients[userID]
}

func (h *Hub) IsOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.Clients[userID]
	return ok
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}

func (h *Hub) RegisterRoom(room *GameRoom) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Rooms[room.ID] = room
}

func (h *Hub) UnregisterRoom(roomID uint) {
	h.DestroyRoom(roomID)
}

func (h *Hub) DestroyRoom(roomID uint) {
	h.mu.Lock()
	room, ok := h.Rooms[roomID]
	if ok {
		delete(h.Rooms, roomID)
	}
	h.mu.Unlock()
	if ok {
		room.Close()
		log.Printf("Room destroyed: %d", roomID)
	}
}

func (h *Hub) GetRoom(roomID uint) *GameRoom {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Rooms[roomID]
}

func (h *Hub) JoinRoom(roomID uint, client *Client) error {
	h.mu.RLock()
	room, ok := h.Rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("room not found")
	}
	if !room.CanJoin(client.UserID) {
		return fmt.Errorf("room is full or already started")
	}

	client.mu.Lock()
	client.RoomID = roomID
	client.mu.Unlock()

	room.HandleEvent(GameEvent{
		Type:     "PLAYER_JOIN",
		PlayerID: client.UserID,
	})
	return nil
}

func roomIDFromPayload(payload json.RawMessage) (uint, error) {
	var numeric struct {
		RoomID uint `json:"room_id"`
	}
	if err := json.Unmarshal(payload, &numeric); err == nil && numeric.RoomID > 0 {
		return numeric.RoomID, nil
	}

	var text struct {
		RoomID string `json:"room_id"`
	}
	if err := json.Unmarshal(payload, &text); err == nil && text.RoomID != "" {
		id, err := strconv.ParseUint(text.RoomID, 10, 64)
		if err == nil && id > 0 {
			return uint(id), nil
		}
	}

	return 0, fmt.Errorf("invalid room_id")
}

func (h *Hub) RouteMessage(roomID uint, playerID uint, msg WSMessage) {
	h.mu.RLock()
	room, ok := h.Rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	event := GameEvent{
		Type:     msg.Type,
		PlayerID: playerID,
		Payload:  msg.Payload,
	}
	room.HandleEvent(event)
}

func (h *Hub) BroadcastToRoom(roomID uint, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.Rooms[roomID]
	if !ok {
		return
	}
	for _, p := range room.Players {
		if client, exists := h.Clients[p.ID]; exists && !p.IsAI {
			client.SendMessage(msg)
		}
	}
}

func (h *Hub) BroadcastToAll(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.Clients {
		client.SendMessage(msg)
	}
}

func (h *Hub) ActiveRooms() []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]map[string]interface{}, 0, len(h.Rooms))
	for _, r := range h.Rooms {
		humanCount := 0
		aiCount := 0
		readyCount := 0
		for _, p := range r.Players {
			if p.IsAI {
				aiCount++
			} else {
				humanCount++
			}
			if p.IsReady {
				readyCount++
			}
		}
		playerCount := humanCount + aiCount
		canJoin := r.State.Phase != "PLAYING" && r.State.Phase != "GAME_OVER" && playerCount < 4
		rooms = append(rooms, map[string]interface{}{
			"id":          r.ID,
			"name":        r.Name,
			"phase":       r.State.Phase,
			"players":     playerCount,
			"max_players": 4,
			"human_count": humanCount,
			"ready_count": readyCount,
			"can_join":    canJoin,
		})
	}
	return rooms
}

// CleanupStaleRooms removes rooms that have existed for more than the specified duration
func (h *Hub) CleanupStaleRooms(maxAge time.Duration) {
	h.mu.Lock()
	now := time.Now()
	staleRooms := make([]uint, 0)

	log.Printf("Cleanup: Checking %d rooms in memory (maxAge: %v)", len(h.Rooms), maxAge)

	for roomID, room := range h.Rooms {
		age := now.Sub(room.CreatedAt)
		if age > maxAge {
			staleRooms = append(staleRooms, roomID)
			log.Printf("Cleanup: Room %d is stale (age: %v > maxAge: %v)", roomID, age, maxAge)
		}
	}

	for _, roomID := range staleRooms {
		room := h.Rooms[roomID]
		delete(h.Rooms, roomID)

		log.Printf("Cleanup: Removing stale room %d from memory (age: %v)", roomID, now.Sub(room.CreatedAt))

		// Trigger database cleanup via OnGameOver callback
		if h.OnGameOver != nil {
			h.OnGameOver(roomID, 0, room.State.Players)
		}

		// Close the room
		go room.Close()
	}
	h.mu.Unlock()

	if len(staleRooms) > 0 {
		log.Printf("Cleanup: Removed %d stale rooms from memory", len(staleRooms))
	}

	// Also clean up database directly for orphaned records
	if h.RoomService != nil {
		maxAgeMinutes := int(maxAge.Minutes())
		deletedCount, err := h.RoomService.CleanupStaleRooms(maxAgeMinutes)
		if err != nil {
			log.Printf("Cleanup: Database cleanup failed: %v", err)
		} else if deletedCount > 0 {
			log.Printf("Cleanup: Removed %d stale rooms from database", deletedCount)
		} else {
			log.Printf("Cleanup: No stale rooms found in database")
		}
	}
}

// StartCleanupTask starts a background goroutine that periodically cleans up stale rooms
func (h *Hub) StartCleanupTask(interval time.Duration, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			h.CleanupStaleRooms(maxAge)
		}
	}()
	log.Printf("Room cleanup task started: checking every %v, removing rooms older than %v", interval, maxAge)
}
