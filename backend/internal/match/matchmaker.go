package match

import (
	"sync"
	"time"

	"liars-bar/internal/config"
	"liars-bar/internal/game"
	"liars-bar/internal/logger"
	"liars-bar/internal/model"
	"liars-bar/internal/service"
	"liars-bar/internal/websocket"
)

type MatchService struct {
	Hub         *websocket.Hub
	Config      *config.GameConfig
	Queue       []MatchEntry
	mu          sync.Mutex
	aiPlayerID  uint
	aiPlayerMu  sync.Mutex
	roomService *service.RoomService
}

type MatchEntry struct {
	UserID      uint
	Nickname    string
	CharacterID string
	JoinedAt    time.Time
}

func NewMatchService(hub *websocket.Hub, cfg *config.GameConfig, roomService *service.RoomService) *MatchService {
	ms := &MatchService{
		Hub:         hub,
		Config:      cfg,
		Queue:       make([]MatchEntry, 0),
		aiPlayerID:  100000,
		roomService: roomService,
	}
	go ms.matchLoop()
	return ms
}

func (ms *MatchService) JoinQueue(userID uint, nickname string, characterID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if already in queue
	for _, entry := range ms.Queue {
		if entry.UserID == userID {
			return nil
		}
	}

	ms.Queue = append(ms.Queue, MatchEntry{
		UserID:      userID,
		Nickname:    nickname,
		CharacterID: game.NormalizeCharacterID(characterID),
		JoinedAt:    time.Now(),
	})

	logger.WithContext(map[string]interface{}{
		"user_id":      userID,
		"nickname":     nickname,
		"character_id": game.NormalizeCharacterID(characterID),
	}).Info("Player joined match queue")

	return nil
}

func (ms *MatchService) LeaveQueue(userID uint) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for i, entry := range ms.Queue {
		if entry.UserID == userID {
			ms.Queue = append(ms.Queue[:i], ms.Queue[i+1:]...)
			break
		}
	}
}

func (ms *MatchService) GetQueueStatus(userID uint) string {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, entry := range ms.Queue {
		if entry.UserID == userID {
			return "WAITING"
		}
	}

	// Check if in any room
	for client := range ms.Hub.Clients {
		if client == userID {
			return "IN_ROOM"
		}
	}
	return "NOT_MATCHING"
}

func (ms *MatchService) QueueLength() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return len(ms.Queue)
}

func (ms *MatchService) matchLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ms.tryMatch()
	}
}

func (ms *MatchService) tryMatch() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	maxPlayers := ms.Config.MaxPlayers
	if len(ms.Queue) == 0 {
		return
	}

	now := time.Now()

	// 将队列分为:仍在等真人的(fresh) 和 已超时愿意接受AI补位的(timedOut)
	var fresh, timedOut []MatchEntry
	for _, entry := range ms.Queue {
		if now.Sub(entry.JoinedAt) >= ms.Config.AIFillTimeout {
			timedOut = append(timedOut, entry)
		} else {
			fresh = append(fresh, entry)
		}
	}

	var selected []MatchEntry
	fillAI := false

	switch {
	case len(fresh) >= maxPlayers:
		// 凑齐4个真人,立即开局
		selected = fresh[:maxPlayers]
		fillAI = false
	case len(timedOut) > 0:
		// 有人等超时,用队列所有人+AI补位开局(支持单人)
		pool := append(fresh, timedOut...)
		if len(pool) > maxPlayers {
			pool = pool[:maxPlayers]
		}
		selected = pool
		fillAI = true
	default:
		return
	}

	// 从队列移除已选中的玩家
	selSet := make(map[uint]bool, len(selected))
	for _, s := range selected {
		selSet[s.UserID] = true
	}
	newQueue := make([]MatchEntry, 0, len(ms.Queue)-len(selected))
	for _, entry := range ms.Queue {
		if !selSet[entry.UserID] {
			newQueue = append(newQueue, entry)
		}
	}
	ms.Queue = newQueue

	go ms.createRoom(selected, fillAI)
}

func (ms *MatchService) createRoom(players []MatchEntry, fillAI bool) {
	if len(players) == 0 {
		return
	}

	// Persist to MySQL so the room gets a unique auto-increment ID (the old
	// len(hub.Rooms)+1 scheme collided once rooms were destroyed).
	room, err := ms.roomService.CreateRoom(players[0].UserID, "Match Room")
	if err != nil {
		logger.WithContext(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to create DB room for match")
		return
	}
	for _, entry := range players[1:] {
		if err := ms.roomService.JoinRoom(room.ID, entry.UserID); err != nil {
			logger.WithContext(map[string]interface{}{
				"room_id":   room.ID,
				"user_id":   entry.UserID,
				"error":     err.Error(),
			}).Error("Failed to add player to match room")
		}
	}
	ms.roomService.UpdateStatus(room.ID, model.RoomStatusMatched)

	gameRoom := websocket.NewGameRoom(room.ID, "Match Room", ms.Hub)

	aiCount := 0
	if fillAI {
		aiCount = 4 - len(players)
	}

	// Add human players
	for i, entry := range players {
		characterID := game.NormalizeCharacterID(entry.CharacterID)
		gameRoom.Players[entry.UserID] = &game.Player{
			ID:            entry.UserID,
			Nickname:      entry.Nickname,
			SeatIndex:     i,
			IsAlive:       true,
			IsOnline:      true,
			IsAI:          false,
			CharacterID:   characterID,
			CharacterName: game.CharacterName(characterID),
		}

		logger.WithContext(map[string]interface{}{
			"user_id":      entry.UserID,
			"seat_index":   i,
			"character_id": characterID,
		}).Info("Added human player to match room")

		if client := ms.Hub.GetClient(entry.UserID); client != nil {
			client.RoomID = gameRoom.ID
			client.SendMessage(websocket.Message{
				Type: "MATCH_FOUND",
				Payload: map[string]interface{}{
					"room_id":   gameRoom.ID,
					"room_name": gameRoom.Name,
				},
			})
		}
	}

	// Add AI players
	for i := 0; i < aiCount; i++ {
		ms.aiPlayerMu.Lock()
		ms.aiPlayerID++
		aiID := ms.aiPlayerID
		ms.aiPlayerMu.Unlock()

		gameRoom.Players[aiID] = &game.Player{
			ID:            aiID,
			Nickname:      "AI-Bot",
			SeatIndex:     len(players) + i,
			IsAlive:       true,
			IsOnline:      true,
			IsAI:          true,
			CharacterID:   game.CharacterScubby,
			CharacterName: game.CharacterName(game.CharacterScubby),
		}

		logger.WithContext(map[string]interface{}{
			"ai_id":        aiID,
			"seat_index":   len(players) + i,
			"character_id": game.CharacterScubby,
		}).Debug("Added AI player to match room")
	}

	ms.Hub.RegisterRoom(gameRoom)
	gameRoom.State.Phase = game.PhaseMatched

	logger.WithContext(map[string]interface{}{
		"room_id":      gameRoom.ID,
		"human_count":  len(players),
		"ai_count":     aiCount,
		"total_players": len(gameRoom.Players),
	}).Info("Match room created, starting game in 2 seconds")

	// Start game after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		gameRoom.HandleEvent(websocket.GameEvent{Type: "START_GAME"})
		ms.roomService.UpdateStatus(room.ID, model.RoomStatusPlaying)
	}()
}
