package websocket

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"liars-bar/internal/game"
	"liars-bar/internal/logger"
)

type GameEvent struct {
	Type     string          `json:"type"`
	PlayerID uint            `json:"player_id,omitempty"`
	AIPlayer bool            `json:"ai_player,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type GameRoom struct {
	ID          uint
	Name        string
	Players     map[uint]*game.Player
	State       *game.GameState
	Events      chan GameEvent
	Hub         *Hub
	mu          sync.RWMutex
	aiService   *AIProxy
	turnTimer   *time.Timer
	turnTimeout time.Duration
	closed      bool
	CreatedAt   time.Time

	// game recording
	GameRecordID  uint
	RoundNo       int
	TurnNo        int
	statsRecorded int32 // accessed atomically; ensures OnGameOver fires once
}

func NewGameRoom(id uint, name string, hub *Hub) *GameRoom {
	room := &GameRoom{
		ID:          id,
		Name:        name,
		Players:     make(map[uint]*game.Player),
		State:       &game.GameState{Phase: game.PhaseWaiting},
		Events:      make(chan GameEvent, 256),
		Hub:         hub,
		turnTimeout: 30 * time.Second,
		CreatedAt:   time.Now(),
	}
	go room.eventLoop()
	return room
}

func (r *GameRoom) eventLoop() {
	for evt := range r.Events {
		r.processEvent(evt)
	}
}

func (r *GameRoom) HandleEvent(evt GameEvent) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return
	}
	select {
	case r.Events <- evt:
	default:
		logger.WithContext(map[string]interface{}{
			"room_id":    r.ID,
			"event_type": evt.Type,
		}).Error("Room event channel full, dropping event")
	}
}

func (r *GameRoom) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	close(r.Events)
}

func (r *GameRoom) CanJoin(userID uint) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, exists := r.Players[userID]; exists {
		return true
	}
	if r.State.Phase == game.PhasePlaying || r.State.Phase == game.PhaseGameOver {
		return false
	}
	return len(r.Players) < 4
}

func (r *GameRoom) processEvent(evt GameEvent) {
	switch evt.Type {
	case "PLAYER_JOIN":
		r.handlePlayerJoin(evt)
	case "SET_CHARACTER":
		r.handleSetCharacter(evt)
	case "PLAYER_LEAVE":
		r.handlePlayerLeave(evt)
	case "PLAYER_READY":
		r.handlePlayerReady(evt)
	case "START_GAME":
		r.handleStartGame(evt)
	case "PLAY_CARD":
		r.handlePlayCard(evt)
	case "CHALLENGE":
		r.handleChallenge(evt)
	case "PASS":
		r.handlePass(evt)
	case "CHAT":
		r.handleChat(evt)
	case "USE_SKILL":
		r.handleUseSkill(evt)
	case "AI_ACTION":
		r.handleAIAction(evt)
	case "RECONNECT":
		r.handleReconnect(evt)
	case "GAME_OVER":
		r.handleGameOver(evt)
	}
}

func (r *GameRoom) handlePlayerJoin(evt GameEvent) {
	if r.State.Phase == game.PhasePlaying || r.State.Phase == game.PhaseGameOver {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"phase":     r.State.Phase,
		}).Warn("Player tried to join game in progress")
		r.sendError(evt.PlayerID, "game already started")
		return
	}

	nickname := fmt.Sprintf("Player %d", evt.PlayerID)
	characterID := game.CharacterScubby
	payload := struct {
		CharacterID   string `json:"character_id"`
		CharacterName string `json:"character_name"`
	}{}
	if len(evt.Payload) > 0 {
		_ = json.Unmarshal(evt.Payload, &payload)
		characterID = game.NormalizeCharacterID(payload.CharacterID)
	}
	characterName := game.CharacterName(characterID)
	if client := r.Hub.GetClient(evt.PlayerID); client != nil {
		client.RoomID = r.ID
		if client.Nickname != "" {
			nickname = client.Nickname
		}
	}

	if player, ok := r.Players[evt.PlayerID]; ok {
		// 玩家已在房间（匹配或重连），只更新在线状态和昵称，不覆盖角色
		player.IsOnline = true
		player.IsAI = false
		player.AITakeover = false
		player.Nickname = nickname
		// 只有当 payload 明确提供了 character_id 时才更新角色
		if len(evt.Payload) > 0 && payload.CharacterID != "" {
			player.CharacterID = characterID
			player.CharacterName = characterName
		}
		logger.WithContext(map[string]interface{}{
			"room_id":        r.ID,
			"player_id":      evt.PlayerID,
			"character_id":   player.CharacterID,
			"character_name": player.CharacterName,
		}).Info("Player rejoined room")
	} else {
		if len(r.Players) >= 4 {
			logger.WithContext(map[string]interface{}{
				"room_id":   r.ID,
				"player_id": evt.PlayerID,
			}).Warn("Player tried to join full room")
			r.sendError(evt.PlayerID, "room is full")
			return
		}
		r.Players[evt.PlayerID] = &game.Player{
			ID:            evt.PlayerID,
			Nickname:      nickname,
			SeatIndex:     r.nextSeatIndex(),
			IsAlive:       true,
			IsOnline:      true,
			IsReady:       false,
			CharacterID:   characterID,
			CharacterName: characterName,
		}
		logger.WithContext(map[string]interface{}{
			"room_id":        r.ID,
			"player_id":      evt.PlayerID,
			"character_id":   characterID,
			"character_name": characterName,
			"seat_index":     r.Players[evt.PlayerID].SeatIndex,
		}).Info("Player joined room")
	}

	// 获取玩家最终的角色信息（可能是保留的，也可能是新设置的）
	finalPlayer := r.Players[evt.PlayerID]
	r.broadcast(Message{
		Type: "PLAYER_JOINED",
		Payload: map[string]interface{}{
			"player_id":      evt.PlayerID,
			"nickname":       nickname,
			"character_id":   finalPlayer.CharacterID,
			"character_name": finalPlayer.CharacterName,
		},
	})
	r.broadcastRoomState()
}

func (r *GameRoom) handleSetCharacter(evt GameEvent) {
	if r.State.Phase != game.PhaseWaiting && r.State.Phase != game.PhaseMatched {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"phase":     r.State.Phase,
		}).Warn("Player tried to change character after game started")
		r.sendError(evt.PlayerID, "cannot change character after game started")
		return
	}
	player, ok := r.Players[evt.PlayerID]
	if !ok {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
		}).Warn("Player not found when setting character")
		r.sendError(evt.PlayerID, "player not in room")
		return
	}
	payload := struct {
		CharacterID string `json:"character_id"`
	}{}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"error":     err.Error(),
		}).Warn("Invalid character payload")
		r.sendError(evt.PlayerID, "invalid character payload")
		return
	}
	player.CharacterID = game.NormalizeCharacterID(payload.CharacterID)
	player.CharacterName = game.CharacterName(player.CharacterID)
	player.IsReady = false

	logger.WithContext(map[string]interface{}{
		"room_id":            r.ID,
		"player_id":          evt.PlayerID,
		"character_id_raw":   payload.CharacterID,
		"character_id":       player.CharacterID,
		"character_name":     player.CharacterName,
	}).Info("Player changed character")

	r.broadcast(Message{
		Type: "CHARACTER_SELECTED",
		Payload: map[string]interface{}{
			"player_id":      evt.PlayerID,
			"character_id":   player.CharacterID,
			"character_name": player.CharacterName,
		},
	})
	r.broadcastRoomState()
}

func (r *GameRoom) handlePlayerLeave(evt GameEvent) {
	player, ok := r.Players[evt.PlayerID]
	if !ok {
		return
	}

	// Remove the player first
	player.IsOnline = false
	delete(r.Players, evt.PlayerID)

	// Check if only AI players remain after this player leaves
	hasHuman := false
	for _, p := range r.Players {
		if !p.IsAI {
			hasHuman = true
			break
		}
	}

	// If no human players remain, clean up the room regardless of phase
	if !hasHuman {
		r.State.Phase = game.PhaseGameOver
		r.State.WinnerID = nil

		r.broadcast(Message{
			Type: "PLAYER_LEFT",
			Payload: map[string]interface{}{
				"player_id": evt.PlayerID,
				"nickname":  player.Nickname,
				"game_over": true,
				"reason":    "所有真人玩家已离开，游戏结束",
				"winner_id": nil,
			},
		})

		// Record stats and trigger database cleanup
		if atomic.CompareAndSwapInt32(&r.statsRecorded, 0, 1) {
			if r.Hub.OnGameOver != nil {
				r.Hub.OnGameOver(r.ID, 0, r.State.Players)
			}
		}

		// Destroy room after delay
		go func() {
			time.Sleep(5 * time.Second)
			r.Hub.DestroyRoom(r.ID)
		}()
		return
	}

	// If playing and a human left, end the game
	if r.State.Phase == game.PhasePlaying {
		// Pick a winner among remaining alive human players
		var winnerID *uint
		for _, p := range r.Players {
			if p.IsAlive && !p.IsAI {
				id := p.ID
				winnerID = &id
				break
			}
		}
		r.State.Phase = game.PhaseGameOver
		r.State.WinnerID = winnerID

		r.broadcast(Message{
			Type: "PLAYER_LEFT",
			Payload: map[string]interface{}{
				"player_id": evt.PlayerID,
				"nickname":  player.Nickname,
				"game_over": true,
				"reason":    "玩家退出，游戏结束",
				"winner_id": winnerID,
			},
		})

		// Record stats once
		if atomic.CompareAndSwapInt32(&r.statsRecorded, 0, 1) {
			wid := uint(0)
			if r.State.WinnerID != nil {
				wid = *r.State.WinnerID
			}
			if r.Hub.OnGameOver != nil {
				r.Hub.OnGameOver(r.ID, wid, r.State.Players)
			}
		}

		// Destroy room after delay
		go func() {
			time.Sleep(5 * time.Second)
			r.Hub.DestroyRoom(r.ID)
		}()
		return
	}

	// Pre-game leave: just broadcast the leave event
	r.broadcast(Message{
		Type: "PLAYER_LEFT",
		Payload: map[string]interface{}{
			"player_id": evt.PlayerID,
		},
	})
	if len(r.Players) == 0 {
		r.Hub.DestroyRoom(r.ID)
		return
	}
	r.broadcastRoomState()
}

func (r *GameRoom) handlePlayerReady(evt GameEvent) {
	if r.State.Phase != game.PhaseWaiting && r.State.Phase != game.PhaseMatched {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"phase":     r.State.Phase,
		}).Debug("Player ready in wrong phase")
		return
	}
	player, ok := r.Players[evt.PlayerID]
	if !ok {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
		}).Warn("Player not found when marking ready")
		return
	}
	player.IsReady = true

	logger.WithContext(map[string]interface{}{
		"room_id":   r.ID,
		"player_id": evt.PlayerID,
	}).Info("Player marked ready")

	r.broadcast(Message{
		Type: "PLAYER_READY",
		Payload: map[string]interface{}{
			"player_id": evt.PlayerID,
		},
	})
	r.broadcastRoomState()

	if len(r.Players) == 4 && r.allPlayersReady() {
		logger.WithContext(map[string]interface{}{
			"room_id": r.ID,
		}).Info("All players ready, starting game")
		r.startGame()
	}
}

func (r *GameRoom) handleStartGame(evt GameEvent) {
	if len(r.Players) != 4 {
		return
	}
	r.startGame()
}

func (r *GameRoom) startGame() {
	players := make([]*game.Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	// Sort by SeatIndex so gs.Players[i].SeatIndex == i. The frontend treats
	// GameState.CurrentPlayer (a slice index) as a seat index, so the slice
	// must be in seat order — iterating r.Players (a map) gives random order.
	sort.Slice(players, func(i, j int) bool {
		return players[i].SeatIndex < players[j].SeatIndex
	})
	r.State.InitGame(players)
	r.State.Phase = game.PhasePlaying

	r.broadcast(Message{
		Type:    "GAME_STARTED",
		Payload: r.State.ToPublic(0),
	})

	for _, p := range r.Players {
		if client := r.Hub.GetClient(p.ID); client != nil && !p.IsAI {
			client.SendMessage(Message{
				Type:    "GAME_STATE",
				Payload: r.State.ToPublic(p.ID),
			})
		}
	}

	r.processAITurns()
}

func (r *GameRoom) nextSeatIndex() int {
	used := make(map[int]bool, len(r.Players))
	for _, p := range r.Players {
		used[p.SeatIndex] = true
	}
	for i := 0; i < 4; i++ {
		if !used[i] {
			return i
		}
	}
	return len(r.Players)
}

func (r *GameRoom) allPlayersReady() bool {
	for _, p := range r.Players {
		if !p.IsReady {
			return false
		}
	}
	return true
}

func (r *GameRoom) readyCount() int {
	count := 0
	for _, p := range r.Players {
		if p.IsReady {
			count++
		}
	}
	return count
}

func (r *GameRoom) roomStatePayload() map[string]interface{} {
	players := make([]*game.Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].SeatIndex < players[j].SeatIndex
	})

	publicPlayers := make([]map[string]interface{}, 0, len(players))
	for _, p := range players {
		publicPlayers = append(publicPlayers, map[string]interface{}{
			"id":             p.ID,
			"nickname":       p.Nickname,
			"is_ai":          p.IsAI,
			"is_online":      p.IsOnline,
			"is_ready":       p.IsReady,
			"seat_index":     p.SeatIndex,
			"character_id":   p.CharacterID,
			"character_name": p.CharacterName,
		})
	}

	return map[string]interface{}{
		"id":           r.ID,
		"name":         r.Name,
		"phase":        r.State.Phase,
		"players":      publicPlayers,
		"player_count": len(players),
		"max_players":  4,
		"ready_count":  r.readyCount(),
		"can_join":     r.State.Phase != game.PhasePlaying && r.State.Phase != game.PhaseGameOver && len(players) < 4,
	}
}

func (r *GameRoom) broadcastRoomState() {
	r.broadcast(Message{Type: "ROOM_STATE", Payload: r.roomStatePayload()})
}

func (r *GameRoom) sendError(playerID uint, msg string) {
	if client := r.Hub.GetClient(playerID); client != nil {
		client.SendMessage(Message{Type: "ERROR", Payload: map[string]interface{}{"msg": msg}})
	}
}

// finalizeGameIfOver broadcasts GAME_OVER and records stats exactly once when
// the phase has transitioned to PhaseGameOver. Returns true if the game is over.
func (r *GameRoom) finalizeGameIfOver() bool {
	if r.State.Phase != game.PhaseGameOver {
		return false
	}
	winnerID := uint(0)
	if r.State.WinnerID != nil {
		winnerID = *r.State.WinnerID
	}
	r.broadcast(Message{
		Type: "GAME_OVER",
		Payload: map[string]interface{}{
			"winner_id": winnerID,
		},
	})
	if r.State.WinnerID != nil && atomic.CompareAndSwapInt32(&r.statsRecorded, 0, 1) {
		if r.Hub.OnGameOver != nil {
			r.Hub.OnGameOver(r.ID, *r.State.WinnerID, r.State.Players)
		}
	}

	// Auto-destroy room after game over (delay 5 seconds for clients to see results)
	go func() {
		time.Sleep(5 * time.Second)
		r.Hub.DestroyRoom(r.ID)
	}()

	return true
}

func (r *GameRoom) handlePlayCard(evt GameEvent) {
	if r.State.Phase != game.PhasePlaying {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"phase":     r.State.Phase,
		}).Warn("Player tried to play card in wrong phase")
		return
	}

	payload := struct {
		CardIDs []int  `json:"card_ids"`
		Claim   string `json:"claim"`
	}{}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
		}).Error("Failed to parse play card payload: %v", err)
		return
	}

	err := r.State.PlayCard(evt.PlayerID, payload.CardIDs, game.Card(payload.Claim))
	if err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"card_ids":  payload.CardIDs,
			"claim":     payload.Claim,
		}).Warn("Play card failed: %v", err)
		r.Hub.GetClient(evt.PlayerID).SendMessage(Message{
			Type:    "ERROR",
			Payload: map[string]interface{}{"msg": err.Error()},
		})
		return
	}

	logger.WithContext(map[string]interface{}{
		"room_id":    r.ID,
		"player_id":  evt.PlayerID,
		"card_count": len(payload.CardIDs),
		"claim":      payload.Claim,
	}).Info("Player played cards")

	r.broadcastGameState()
	if r.finalizeGameIfOver() {
		return
	}
	r.processAITurns()
}

func (r *GameRoom) handleChallenge(evt GameEvent) {
	if r.State.Phase != game.PhaseChallenge {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"phase":     r.State.Phase,
		}).Warn("Player tried to challenge in wrong phase")
		return
	}

	payload := struct {
		TargetPlayerID uint `json:"target_player_id"`
	}{}
	json.Unmarshal(evt.Payload, &payload)

	logger.WithContext(map[string]interface{}{
		"room_id":      r.ID,
		"challenger":   evt.PlayerID,
		"target":       payload.TargetPlayerID,
	}).Info("Player challenging")

	result, err := r.State.Challenge(evt.PlayerID)
	if err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":    r.ID,
			"player_id":  evt.PlayerID,
		}).Error("Challenge failed: %v", err)
		r.Hub.GetClient(evt.PlayerID).SendMessage(Message{
			Type:    "ERROR",
			Payload: map[string]interface{}{"msg": err.Error()},
		})
		return
	}

	logger.WithContext(map[string]interface{}{
		"room_id":    r.ID,
		"challenger": evt.PlayerID,
		"success":    result.Success,
		"loser_id":   result.LoserID,
		"survived":   result.Punishment != nil && result.Punishment.Survived,
	}).Info("Challenge result")

	r.broadcast(Message{
		Type:    "CHALLENGE_RESULT",
		Payload: result,
	})

	loser := r.State.GetPlayer(result.LoserID)
	if loser != nil {
		r.broadcast(Message{
			Type: "RUSSIAN_ROULETTE",
			Payload: map[string]interface{}{
				"player_id":        loser.ID,
				"bullet_count":     loser.PunishmentCount,
				"survived":         loser.IsAlive,
				"tor_reduced":      result.Punishment != nil && result.Punishment.TorReduced,
				"tor_immune":       result.Punishment != nil && result.Punishment.TorImmune,
				"failed_challenge": result.Punishment != nil && result.Punishment.FailedChallenge,
			},
		})

		if !loser.IsAlive {
			logger.WithContext(map[string]interface{}{
				"room_id":   r.ID,
				"player_id": loser.ID,
			}).Info("Player eliminated")
			r.broadcast(Message{
				Type: "PLAYER_ELIMINATED",
				Payload: map[string]interface{}{
					"player_id": loser.ID,
				},
			})
		}
	}

	if r.finalizeGameIfOver() {
		return
	}

	// Challenge() 已经处理了惩罚并将状态转换到 PhasePlaying
	// 不需要调用 SkipChallenge()，直接广播状态并继续游戏
	r.broadcastGameState()
	r.processAITurns()
}

func (r *GameRoom) handlePass(evt GameEvent) {
	if r.State.Phase == game.PhaseChallenge {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"phase":     r.State.Phase,
		}).Debug("Player passing challenge")

		allPassed, err := r.State.PassChallenge(evt.PlayerID)
		if err != nil {
			logger.WithContext(map[string]interface{}{
				"room_id":   r.ID,
				"player_id": evt.PlayerID,
				"error":     err.Error(),
			}).Warn("Pass challenge failed")
			r.Hub.GetClient(evt.PlayerID).SendMessage(Message{
				Type:    "ERROR",
				Payload: map[string]interface{}{"msg": err.Error()},
			})
			return
		}

		if allPassed {
			logger.WithContext(map[string]interface{}{
				"room_id": r.ID,
			}).Info("All players passed challenge, skipping to next player")
			if err := r.State.SkipChallenge(); err != nil {
				return
			}
		}

		r.broadcastGameState()
		if allPassed {
			r.processAITurns()
		}
		return
	}

	logger.WithContext(map[string]interface{}{
		"room_id":   r.ID,
		"player_id": evt.PlayerID,
		"phase":     r.State.Phase,
	}).Debug("Player passing turn")

	if err := r.State.Pass(evt.PlayerID); err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"error":     err.Error(),
		}).Warn("Pass turn failed")
		r.Hub.GetClient(evt.PlayerID).SendMessage(Message{
			Type:    "ERROR",
			Payload: map[string]interface{}{"msg": err.Error()},
		})
		return
	}
	r.broadcastGameState()
	r.processAITurns()
}

func (r *GameRoom) handleChat(evt GameEvent) {
	payload := struct {
		Content string `json:"content"`
	}{}
	json.Unmarshal(evt.Payload, &payload)

	if payload.Content == "" || len(payload.Content) > 500 {
		return
	}

	nickname := fmt.Sprintf("玩家%d", evt.PlayerID)
	if p := r.Players[evt.PlayerID]; p != nil && p.Nickname != "" {
		nickname = p.Nickname
	} else if client := r.Hub.GetClient(evt.PlayerID); client != nil && client.Nickname != "" {
		nickname = client.Nickname
	}

	r.broadcast(Message{
		Type: "CHAT",
		Payload: map[string]interface{}{
			"sender_id":   evt.PlayerID,
			"sender_name": nickname,
			"content":     payload.Content,
			"is_ai":       evt.AIPlayer,
		},
	})
}

func (r *GameRoom) handleUseSkill(evt GameEvent) {
	payload := struct {
		TargetPlayerID uint `json:"target_player_id"`
	}{}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"error":     err.Error(),
		}).Warn("Invalid skill payload")
		r.sendError(evt.PlayerID, "invalid skill payload")
		return
	}
	hand, err := r.State.UseFoxyPeek(evt.PlayerID, payload.TargetPlayerID)
	if err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
			"target_id": payload.TargetPlayerID,
			"error":     err.Error(),
		}).Warn("Skill use failed")
		r.sendError(evt.PlayerID, err.Error())
		return
	}

	logger.WithContext(map[string]interface{}{
		"room_id":   r.ID,
		"player_id": evt.PlayerID,
		"target_id": payload.TargetPlayerID,
		"skill":     "foxy_peek",
	}).Info("Player used skill")

	if client := r.Hub.GetClient(evt.PlayerID); client != nil {
		client.SendMessage(Message{
			Type: "SKILL_RESULT",
			Payload: map[string]interface{}{
				"skill":            "foxy_peek",
				"target_player_id": payload.TargetPlayerID,
				"hand":             hand,
				"duration_ms":      3000,
			},
		})
	}
	r.broadcast(Message{
		Type: "SKILL_USED",
		Payload: map[string]interface{}{
			"player_id":    evt.PlayerID,
			"character_id": game.CharacterFoxy,
			"skill":        "foxy_peek",
		},
	})
	r.broadcastGameState()
}

func (r *GameRoom) handleAIAction(evt GameEvent) {
}

func (r *GameRoom) handleReconnect(evt GameEvent) {
	player := r.State.GetPlayer(evt.PlayerID)
	if player != nil {
		player.IsOnline = true
		player.IsAI = false
		player.AITakeover = false

		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": evt.PlayerID,
		}).Info("Player reconnected")

		if client := r.Hub.GetClient(evt.PlayerID); client != nil {
			client.SendMessage(Message{
				Type:    "GAME_STATE",
				Payload: r.State.ToPublic(evt.PlayerID),
			})
		}
	}
}

func (r *GameRoom) handleGameOver(evt GameEvent) {
	r.State.Phase = game.PhaseGameOver
	r.broadcast(Message{
		Type: "GAME_OVER",
		Payload: map[string]interface{}{
			"winner_id": *r.State.WinnerID,
		},
	})
}

func (r *GameRoom) broadcast(msg Message) {
	r.Hub.BroadcastToRoom(r.ID, msg)
}

func (r *GameRoom) broadcastGameState() {
	for _, p := range r.Players {
		if !p.IsAI || !p.AITakeover {
			if client := r.Hub.GetClient(p.ID); client != nil {
				client.SendMessage(Message{
					Type:    "GAME_STATE",
					Payload: r.State.ToPublic(p.ID),
				})
			}
		}
	}
}

func (r *GameRoom) processAITurns() {
	if r.State.Phase == game.PhaseChallenge {
		r.processAIChallengePhase()
		return
	}

	if r.State.Phase != game.PhasePlaying {
		return
	}

	currentPlayer := r.State.GetCurrentPlayer()
	if currentPlayer == nil {
		return
	}

	// 如果当前玩家没有手牌，自动跳过（包括真人玩家和AI）
	if len(currentPlayer.Hand) == 0 {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": currentPlayer.ID,
			"phase":     r.State.Phase,
		}).Debug("Player has no cards, auto-skipping")
		r.State.NextPlayer()
		r.broadcastGameState()
		r.processAITurns()
		return
	}

	// 如果是AI玩家，继续AI逻辑
	if !currentPlayer.IsAI {
		return
	}

	// 延迟执行AI回合，避免race condition
	go func() {
		time.Sleep(1 * time.Second)
		r.mu.Lock()
		defer r.mu.Unlock()

		// 再次检查当前玩家状态，防止延迟期间状态变化
		cp := r.State.GetCurrentPlayer()
		if cp == nil || !cp.IsAI || len(cp.Hand) == 0 {
			return
		}

		r.executeAITurn()
	}()
}

func (r *GameRoom) processAIChallengePhase() {
	if r.State.Phase != game.PhaseChallenge {
		return
	}

	// 安全检查：如果没有 LastPlay，不应该在 challenge 阶段
	if r.State.LastPlay == nil {
		r.State.Phase = game.PhasePlaying
		r.broadcastGameState()
		r.processAITurns()
		return
	}

	// 收集所有需要决策的AI玩家
	aiPlayers := make([]*game.Player, 0)
	for _, player := range r.State.Players {
		if !player.IsAI || !player.IsAlive {
			continue
		}
		if player.ID == r.State.LastPlay.PlayerID {
			continue
		}
		if r.State.ChallengePassed[player.ID] {
			continue
		}
		aiPlayers = append(aiPlayers, player)
	}

	// 如果没有AI需要决策，检查是否所有人都已pass
	if len(aiPlayers) == 0 {
		// 计算是否所有合格玩家都已pass
		eligibleCount := 0
		passedCount := 0
		for _, p := range r.State.Players {
			if p.IsAlive && p.ID != r.State.LastPlay.PlayerID {
				eligibleCount++
				if r.State.ChallengePassed[p.ID] {
					passedCount++
				}
			}
		}
		if eligibleCount > 0 && passedCount >= eligibleCount {
			r.State.SkipChallenge()
			r.broadcastGameState()
			r.processAITurns()
		}
		return
	}

	// 让第一个AI做决策
	player := aiPlayers[0]
	go func(p *game.Player) {
		time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)
		r.mu.Lock()
		defer r.mu.Unlock()

		if r.State.Phase != game.PhaseChallenge {
			return
		}

		// 再次检查玩家是否还活着
		if !p.IsAlive {
			// 玩家已死亡，继续处理下一个AI
			r.processAIChallengePhase()
			return
		}

		// 检查LastPlay是否还存在
		if r.State.LastPlay == nil {
			return
		}

		actions := r.State.GetLegalActions(p.ID)
		hasChallenge := false
		for _, a := range actions {
			if a == "CHALLENGE" {
				hasChallenge = true
				break
			}
		}

		ai := newAIStrategy(p, r.State)
		if hasChallenge && ai.shouldChallenge() {
			r.executeChallenge(p.ID, r.State.LastPlay.PlayerID)
		} else {
			allPassed, err := r.State.PassChallenge(p.ID)
			if err == nil {
				if allPassed {
					// 所有人都pass，跳过质疑阶段
					r.State.SkipChallenge()
					r.broadcastGameState()
					r.processAITurns()
				} else {
					// 还有其他AI需要决策，不广播，继续处理下一个AI
					r.processAIChallengePhase()
				}
			} else {
				// PassChallenge失败，可能是因为状态已改变，重新处理
				r.processAIChallengePhase()
			}
		}
	}(player)
}

func (r *GameRoom) executeAITurn() {
	currentPlayer := r.State.GetCurrentPlayer()
	if currentPlayer == nil || !currentPlayer.IsAI {
		return
	}

	// 如果AI玩家没有手牌，自动跳过
	if len(currentPlayer.Hand) == 0 {
		r.State.NextPlayer()
		r.broadcastGameState()
		r.processAITurns()
		return
	}

	// Simple AI strategy
	actions := r.State.GetLegalActions(currentPlayer.ID)

	if r.aiService != nil {
		action := r.aiService.GetAction(r.State, currentPlayer.ID, actions)
		r.executeAction(action, currentPlayer.ID)
		return
	}

	// Fallback: rule-based AI
	r.simpleAITurn(currentPlayer, actions)
}

func (r *GameRoom) simpleAITurn(player *game.Player, actions []string) {
	hasChallenge := false
	hasPlay := false
	for _, a := range actions {
		if a == "CHALLENGE" {
			hasChallenge = true
		}
		if a == "PLAY_CARD" {
			hasPlay = true
		}
	}

	ai := newAIStrategy(player, r.State)

	if hasChallenge && r.State.LastPlay != nil && ai.shouldChallenge() {
		r.executeChallenge(player.ID, r.State.LastPlay.PlayerID)
	} else if hasPlay {
		// 游戏规则：只要有牌就必须出牌，没牌才能Pass
		if len(player.Hand) == 0 {
			r.executePass(player.ID)
		} else {
			r.executeAIPlaySmart(player, ai)
		}
	} else {
		r.executePass(player.ID)
	}
}

// AI Strategy types
type AIStrategyType int

const (
	AIStrategyConservative AIStrategyType = iota // 保守型：少说谎，多质疑
	AIStrategyAggressive                         // 激进型：多说谎，少质疑
	AIStrategyBalanced                           // 平衡型：适度说谎和质疑
	AIStrategyRandom                             // 随机型：不可预测
)

type aiStrategy struct {
	player       *game.Player
	state        *game.GameState
	strategyType AIStrategyType
}

func newAIStrategy(player *game.Player, state *game.GameState) *aiStrategy {
	// 随机选择策略类型
	strategyType := AIStrategyType(rand.Intn(4))
	return &aiStrategy{
		player:       player,
		state:        state,
		strategyType: strategyType,
	}
}

// 分析手牌情况
func (ai *aiStrategy) analyzeHand() (targetCards []int, wildCards []int, otherCards []int) {
	targetCards = []int{}
	wildCards = []int{}
	otherCards = []int{}

	for i, card := range ai.player.Hand {
		if card == ai.state.TargetCard {
			targetCards = append(targetCards, i)
		} else if card == game.Wild {
			wildCards = append(wildCards, i)
		} else {
			otherCards = append(otherCards, i)
		}
	}
	return
}

// 计算说谎风险
func (ai *aiStrategy) calculateLyingRisk(targetCards, wildCards []int) float64 {
	truthfulCards := len(targetCards) + len(wildCards)

	// 基础风险：手里真牌越少，风险越高
	baseRisk := 1.0 - float64(truthfulCards)/float64(len(ai.player.Hand))

	// 考虑其他玩家的质疑倾向
	alivePlayers := 0
	for _, p := range ai.state.Players {
		if p.IsAlive && p.ID != ai.player.ID {
			alivePlayers++
		}
	}

	// 玩家越少，被质疑概率越高
	riskMultiplier := 1.0 + float64(4-alivePlayers)*0.2

	return baseRisk * riskMultiplier
}

// 决定打几张牌
func (ai *aiStrategy) decidePlayCount() int {
	handSize := len(ai.player.Hand)

	// 防护：如果手牌为0，返回1（调用方会处理）
	if handSize == 0 {
		return 1
	}

	switch ai.strategyType {
	case AIStrategyConservative:
		// 保守型：少打点，降低风险
		if handSize <= 2 {
			return 1
		}
		if handSize <= 4 {
			return rand.Intn(2) + 1 // 1-2张
		}
		return rand.Intn(2) + 1 // 1-2张

	case AIStrategyAggressive:
		// 激进型：多打点，快速出牌
		if handSize <= 2 {
			return handSize
		}
		if handSize <= 4 {
			return rand.Intn(2) + 2 // 2-3张
		}
		return rand.Intn(2) + 2 // 2-3张

	case AIStrategyBalanced:
		// 平衡型：根据手牌数量适度调整
		if handSize <= 2 {
			return 1
		}
		if handSize <= 4 {
			return rand.Intn(2) + 1 // 1-2张
		}
		return rand.Intn(3) + 1 // 1-3张

	case AIStrategyRandom:
		// 随机型：完全随机
		maxPlay := 3
		if handSize < maxPlay {
			maxPlay = handSize
		}
		if maxPlay < 1 {
			return 1
		}
		return rand.Intn(maxPlay) + 1
	}

	return 1
}

// 选择要打的牌
func (ai *aiStrategy) selectCards(playCount int) []int {
	targetCards, wildCards, otherCards := ai.analyzeHand()
	indices := []int{}

	switch ai.strategyType {
	case AIStrategyConservative:
		// 保守型：优先打真牌+WILD，尽量不说谎
		for len(indices) < playCount && len(targetCards) > 0 {
			indices = append(indices, targetCards[0])
			targetCards = targetCards[1:]
		}
		for len(indices) < playCount && len(wildCards) > 0 {
			indices = append(indices, wildCards[0])
			wildCards = wildCards[1:]
		}
		// 如果必须说谎，优先在手牌多时说谎，但手牌少时也会说谎以避免返回空数组
		for len(indices) < playCount && len(otherCards) > 0 {
			indices = append(indices, otherCards[0])
			otherCards = otherCards[1:]
		}

	case AIStrategyAggressive:
		// 激进型：混合打牌，经常说谎
		// 50%概率混入假牌
		for len(indices) < playCount {
			cardAdded := false
			if rand.Float64() < 0.5 && len(targetCards) > 0 {
				indices = append(indices, targetCards[0])
				targetCards = targetCards[1:]
				cardAdded = true
			} else if len(otherCards) > 0 {
				indices = append(indices, otherCards[0])
				otherCards = otherCards[1:]
				cardAdded = true
			} else if len(targetCards) > 0 {
				indices = append(indices, targetCards[0])
				targetCards = targetCards[1:]
				cardAdded = true
			} else if len(wildCards) > 0 {
				indices = append(indices, wildCards[0])
				wildCards = wildCards[1:]
				cardAdded = true
			}
			// 如果没有可用卡牌，退出循环
			if !cardAdded {
				break
			}
		}

	case AIStrategyBalanced:
		// 平衡型：优先真牌，必要时说谎
		for len(indices) < playCount && len(targetCards) > 0 {
			indices = append(indices, targetCards[0])
			targetCards = targetCards[1:]
		}
		for len(indices) < playCount && len(wildCards) > 0 {
			indices = append(indices, wildCards[0])
			wildCards = wildCards[1:]
		}
		for len(indices) < playCount && len(otherCards) > 0 {
			indices = append(indices, otherCards[0])
			otherCards = otherCards[1:]
		}

	case AIStrategyRandom:
		// 随机型：完全随机选牌
		allIndices := []int{}
		for i := range ai.player.Hand {
			allIndices = append(allIndices, i)
		}
		rand.Shuffle(len(allIndices), func(i, j int) {
			allIndices[i], allIndices[j] = allIndices[j], allIndices[i]
		})
		for i := 0; i < playCount && i < len(allIndices); i++ {
			indices = append(indices, allIndices[i])
		}
	}

	// 确保至少有牌可打
	if len(indices) == 0 && len(ai.player.Hand) > 0 {
		indices = append(indices, 0)
	}

	return indices
}

// 决定是否质疑
func (ai *aiStrategy) shouldChallenge() bool {
	if ai.state.LastPlay == nil {
		return false
	}

	// Bristle 角色检查
	maxChallenges := 1
	if ai.player.CharacterID == game.CharacterBristle {
		maxChallenges = 2
	}
	if ai.player.ChallengeUsed >= maxChallenges {
		return false
	}

	// 统计自己手里的目标牌（包括WILD）
	myTargetCount := 0
	for _, c := range ai.player.Hand {
		if c == ai.state.TargetCard || c == game.Wild {
			myTargetCount++
		}
	}

	claimedCount := len(ai.state.LastPlay.CardIDs)
	targetPlayer := ai.state.GetPreviousPlayer()

	// 计算质疑基础概率
	baseProbability := 0.0

	// 因素1：对方出牌数量 vs 自己手牌数量
	if claimedCount >= 3 {
		if myTargetCount >= 4 {
			baseProbability = 0.8 // 对方出3张，我有4张，80%质疑
		} else if myTargetCount >= 3 {
			baseProbability = 0.6 // 对方出3张，我有3张，60%质疑
		} else if myTargetCount >= 2 {
			baseProbability = 0.3 // 对方出3张，我有2张，30%质疑
		}
	} else if claimedCount >= 2 {
		if myTargetCount >= 4 {
			baseProbability = 0.5 // 对方出2张，我有4张，50%质疑
		} else if myTargetCount >= 3 {
			baseProbability = 0.3 // 对方出2张，我有3张，30%质疑
		}
	} else {
		// 对方只出1张，很少质疑
		if myTargetCount >= 5 {
			baseProbability = 0.2
		}
	}

	// 因素2：对方手牌数量（手牌少可能着急出牌容易说谎）
	if targetPlayer != nil && targetPlayer.HandCount <= 2 && claimedCount >= 2 {
		baseProbability += 0.15
	}

	// 因素3：对方历史说谎率
	if targetPlayer != nil && targetPlayer.PlayCount > 0 {
		lieRate := float64(targetPlayer.LieCount) / float64(targetPlayer.PlayCount)
		if lieRate > 0.5 {
			baseProbability += 0.1 // 对方经常说谎，增加质疑概率
		}
	}

	// 根据策略类型调整概率
	switch ai.strategyType {
	case AIStrategyConservative:
		// 保守型：更倾向于质疑
		baseProbability *= 1.3
	case AIStrategyAggressive:
		// 激进型：较少质疑（更关注自己出牌）
		baseProbability *= 0.7
	case AIStrategyBalanced:
		// 平衡型：保持基础概率
		baseProbability *= 1.0
	case AIStrategyRandom:
		// 随机型：完全随机
		baseProbability = rand.Float64()
	}

	// 限制概率范围
	if baseProbability > 0.9 {
		baseProbability = 0.9
	}
	if baseProbability < 0.0 {
		baseProbability = 0.0
	}

	return rand.Float64() < baseProbability
}

func (r *GameRoom) executeAIPlay(player *game.Player) {
	// Choose cards to play
	playCount := 1
	if len(player.Hand) >= 3 {
		playCount = 2
	}

	indices := make([]int, playCount)
	for i := 0; i < playCount && i < len(player.Hand); i++ {
		indices[i] = i
	}

	err := r.State.PlayCard(player.ID, indices, r.State.TargetCard)
	if err == nil {
		r.broadcastGameState()
		if !r.finalizeGameIfOver() {
			r.processAITurns()
		}
	}
}

func (r *GameRoom) executeAIPlaySmart(player *game.Player, ai *aiStrategy) {
	// 首先检查手牌是否为空
	if len(player.Hand) == 0 {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": player.ID,
		}).Warn("AI has no cards, skipping turn")
		r.executePass(player.ID)
		return
	}

	playCount := ai.decidePlayCount()

	// 确保playCount不超过手牌数量
	if playCount > len(player.Hand) {
		playCount = len(player.Hand)
	}

	// 确保至少打1张牌
	if playCount < 1 {
		playCount = 1
	}

	indices := ai.selectCards(playCount)

	if len(indices) == 0 {
		// 安全保护：如果没有选中任何牌，跳过
		r.executePass(player.ID)
		return
	}

	// 验证索引有效性
	for _, idx := range indices {
		if idx < 0 || idx >= len(player.Hand) {
			logger.WithContext(map[string]interface{}{
				"room_id":   r.ID,
				"player_id": player.ID,
				"index":     idx,
				"hand_size": len(player.Hand),
			}).Error("AI selected invalid card index")
			r.executePass(player.ID)
			return
		}
	}

	err := r.State.PlayCard(player.ID, indices, r.State.TargetCard)
	if err != nil {
		logger.WithContext(map[string]interface{}{
			"room_id":   r.ID,
			"player_id": player.ID,
			"error":     err.Error(),
		}).Error("AI play card failed")
		// 如果出牌失败，尝试跳过
		r.executePass(player.ID)
		return
	}

	r.broadcastGameState()
	if !r.finalizeGameIfOver() {
		r.processAITurns()
	}
}

func (r *GameRoom) executeChallenge(challengerID, targetID uint) {
	result, err := r.State.Challenge(challengerID)
	if err != nil {
		return
	}

	r.broadcast(Message{
		Type:    "CHALLENGE_RESULT",
		Payload: result,
	})

	loser := r.State.GetPlayer(result.LoserID)
	if loser != nil {
		r.broadcast(Message{
			Type: "RUSSIAN_ROULETTE",
			Payload: map[string]interface{}{
				"player_id":        loser.ID,
				"bullet_count":     loser.PunishmentCount,
				"survived":         loser.IsAlive,
				"tor_reduced":      result.Punishment != nil && result.Punishment.TorReduced,
				"tor_immune":       result.Punishment != nil && result.Punishment.TorImmune,
				"failed_challenge": result.Punishment != nil && result.Punishment.FailedChallenge,
			},
		})
		if !loser.IsAlive {
			r.broadcast(Message{
				Type: "PLAYER_ELIMINATED",
				Payload: map[string]interface{}{
					"player_id": loser.ID,
				},
			})
		}
	}

	r.broadcastGameState()
	if !r.finalizeGameIfOver() {
		r.processAITurns()
	}
}

func (r *GameRoom) executePass(playerID uint) {
	r.State.Pass(playerID)
	r.broadcastGameState()
	if !r.finalizeGameIfOver() {
		r.processAITurns()
	}
}

func (r *GameRoom) executeAction(action AIAction, playerID uint) {
	switch action.Type {
	case "PLAY_CARD":
		r.State.PlayCard(playerID, action.CardIDs, r.State.TargetCard)
	case "CHALLENGE":
		r.State.Challenge(playerID)
	case "PASS":
		r.State.Pass(playerID)
	}
	r.broadcastGameState()
	if !r.finalizeGameIfOver() {
		r.processAITurns()
	}
}

type AIAction struct {
	Type    string `json:"type"`
	CardIDs []int  `json:"card_ids,omitempty"`
	Message string `json:"message,omitempty"`
}

type AIProxy struct {
	serviceURL string
}

func NewAIProxy(url string) *AIProxy {
	return &AIProxy{serviceURL: url}
}

func (ai *AIProxy) GetAction(state *game.GameState, playerID uint, actions []string) AIAction {
	return AIAction{Type: "PASS"}
}
