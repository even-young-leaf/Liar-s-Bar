package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type Card string

const (
	Ace   Card = "A"
	King  Card = "K"
	Queen Card = "Q"
	Jack  Card = "J"
	Wild  Card = "W"
)

var CardOrder = []Card{Ace, King, Queen, Jack}

type GamePhase string

const (
	PhaseWaiting    GamePhase = "WAITING"
	PhaseMatched    GamePhase = "MATCHED"
	PhasePlaying    GamePhase = "PLAYING"
	PhaseChallenge  GamePhase = "CHALLENGE"
	PhasePunishment GamePhase = "PUNISHMENT"
	PhaseRoundEnd   GamePhase = "ROUND_END"
	PhaseGameOver   GamePhase = "GAME_OVER"
)

type ActionType string

const (
	ActPlayCard   ActionType = "PLAY_CARD"
	ActChallenge  ActionType = "CHALLENGE"
	ActPass       ActionType = "PASS"
	ActChat       ActionType = "CHAT"
	ActPunishment ActionType = "PUNISHMENT"
	ActEliminated ActionType = "ELIMINATED"
	ActGameOver   ActionType = "GAME_OVER"
)

type RoundRecord struct {
	PlayerID   uint
	CardIDs    []int
	Claim      Card
	Truthful   bool
	Cards      []Card
	Challenged bool
}

type Player struct {
	ID               uint   `json:"id"`
	Nickname         string `json:"nickname"`
	IsAI             bool   `json:"is_ai"`
	IsOnline         bool   `json:"is_online"`
	IsAlive          bool   `json:"is_alive"`
	IsReady          bool   `json:"is_ready"`
	AITakeover       bool   `json:"ai_takeover"`
	SeatIndex        int    `json:"seat_index"`
	Hand             []Card `json:"-"`
	HandCount        int    `json:"hand_count"`
	PunishmentCount  int    `json:"punishment_count"`
	PlayCount        int    `json:"play_count"`
	LieCount         int    `json:"lie_count"`
	ChallengeCount   int    `json:"challenge_count"`
	ChallengeSuccess int    `json:"challenge_success"`
	CharacterID      string `json:"character_id"`
	CharacterName    string `json:"character_name"`
	SkillUsed        bool   `json:"skill_used"`
	ChallengeUsed    int    `json:"challenge_used"`
}

const (
	CharacterScubby  = "scubby"
	CharacterFoxy    = "foxy"
	CharacterBristle = "bristle"
	CharacterTor     = "tor"
)

func NormalizeCharacterID(id string) string {
	switch id {
	case CharacterScubby, CharacterFoxy, CharacterBristle, CharacterTor:
		return id
	default:
		return CharacterScubby
	}
}

func CharacterName(id string) string {
	switch NormalizeCharacterID(id) {
	case CharacterScubby:
		return "Scubby"
	case CharacterFoxy:
		return "Foxy"
	case CharacterBristle:
		return "Bristle"
	case CharacterTor:
		return "Tor"
	default:
		return "Scubby"
	}
}

type GameState struct {
	Phase            GamePhase     `json:"phase"`
	CurrentPlayer    int           `json:"current_player"`
	CurrentRound     int           `json:"current_round"`
	CurrentTurn      int           `json:"current_turn"`
	TargetCard       Card          `json:"target_card"`
	Players          []*Player     `json:"players"`
	LastPlay         *RoundRecord  `json:"last_play"`
	RoundHistory     []RoundRecord `json:"-"`
	Deck             []Card        `json:"-"`
	DiscardPile      []Card        `json:"-"`
	RoundCounter     int           `json:"-"`
	WinnerID         *uint         `json:"winner_id"`
	AliveCount       int           `json:"alive_count"`
	ChallengePassed  map[uint]bool `json:"-"`
}

func NewDeck() []Card {
	deck := make([]Card, 0, 24)
	cards := []Card{Ace, King, Queen, Jack}
	for _, c := range cards {
		for i := 0; i < 6; i++ {
			deck = append(deck, c)
		}
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

func DealCards(deck []Card, numPlayers int) ([][]Card, []Card) {
	hands := make([][]Card, numPlayers)
	for i := 0; i < numPlayers; i++ {
		hands[i] = make([]Card, 6)
		for j := 0; j < 6; j++ {
			hands[i][j] = deck[i*6+j]
		}
	}
	remaining := deck[numPlayers*6:]
	return hands, remaining
}

func (gs *GameState) InitGame(players []*Player) {
	gs.Players = players
	gs.Phase = PhasePlaying
	gs.CurrentRound = 1
	gs.CurrentTurn = 0
	gs.TargetCard = Ace
	gs.AliveCount = len(players)
	gs.RoundHistory = make([]RoundRecord, 0)

	deck := NewDeck()
	hands, remaining := DealCards(deck, len(players))
	for i, p := range players {
		p.Hand = hands[i]
		if NormalizeCharacterID(p.CharacterID) == CharacterScubby {
			p.Hand = append(p.Hand, Wild)
			rand.Shuffle(len(p.Hand), func(i, j int) { p.Hand[i], p.Hand[j] = p.Hand[j], p.Hand[i] })
		}
		p.HandCount = len(hands[i])
		p.IsAlive = true
		p.IsReady = false
		p.PunishmentCount = 0
		p.SkillUsed = false
		p.ChallengeUsed = 0
		if p.CharacterID == "" {
			p.CharacterID = CharacterScubby
		}
		p.CharacterID = NormalizeCharacterID(p.CharacterID)
		p.CharacterName = CharacterName(p.CharacterID)
		p.HandCount = len(p.Hand)
	}
	gs.Deck = remaining
	gs.CurrentPlayer = rand.Intn(len(players))
}

func (gs *GameState) GetPlayer(playerID uint) *Player {
	for _, p := range gs.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

func (gs *GameState) GetCurrentPlayer() *Player {
	if gs.CurrentPlayer < len(gs.Players) {
		return gs.Players[gs.CurrentPlayer]
	}
	return nil
}

func (gs *GameState) NextPlayer() {
	start := (gs.CurrentPlayer + 1) % len(gs.Players)
	for i := 0; i < len(gs.Players); i++ {
		idx := (start + i) % len(gs.Players)
		if gs.Players[idx].IsAlive && len(gs.Players[idx].Hand) > 0 {
			gs.CurrentPlayer = idx
			gs.CurrentTurn++
			return
		}
	}
	// 如果所有存活玩家都没有手牌，保持当前玩家不变
}

func (gs *GameState) GetPreviousPlayer() *Player {
	if gs.LastPlay == nil {
		return nil
	}
	for _, p := range gs.Players {
		if p.ID == gs.LastPlay.PlayerID {
			return p
		}
	}
	return nil
}

func (gs *GameState) PlayCard(playerID uint, cardIndices []int, claim Card) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if !player.IsAlive {
		return fmt.Errorf("player is eliminated")
	}
	currentPlayer := gs.GetCurrentPlayer()
	if currentPlayer == nil || currentPlayer.ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if len(cardIndices) < 1 || len(cardIndices) > 3 {
		return fmt.Errorf("must play 1-3 cards")
	}
	if gs.Phase != PhasePlaying {
		return fmt.Errorf("invalid phase")
	}
	if claim != gs.TargetCard {
		return fmt.Errorf("must claim current target card")
	}

	seen := make(map[int]bool, len(cardIndices))
	for _, idx := range cardIndices {
		if idx < 0 || idx >= len(player.Hand) {
			return fmt.Errorf("invalid card index")
		}
		if seen[idx] {
			return fmt.Errorf("duplicate card index")
		}
		seen[idx] = true
	}

	selected := make([]Card, 0, len(cardIndices))
	newHand := make([]Card, 0)
	skipSet := make(map[int]bool)
	for _, idx := range cardIndices {
		skipSet[idx] = true
	}
	for i, c := range player.Hand {
		if skipSet[i] {
			selected = append(selected, c)
		} else {
			newHand = append(newHand, c)
		}
	}

	truthful := true
	for _, c := range selected {
		if c != gs.TargetCard && c != Wild {
			truthful = false
			break
		}
	}

	player.Hand = newHand
	player.HandCount = len(newHand)
	player.PlayCount++
	if !truthful {
		player.LieCount++
	}
	gs.DiscardPile = append(gs.DiscardPile, selected...)

	gs.LastPlay = &RoundRecord{
		PlayerID: playerID,
		CardIDs:  cardIndices,
		Claim:    claim,
		Truthful: truthful,
		Cards:    selected,
	}

	gs.Phase = PhaseChallenge
	gs.ChallengePassed = make(map[uint]bool)
	gs.RecordAction(playerID, ActPlayCard, map[string]interface{}{
		"count":    len(selected),
		"claim":    claim,
		"truthful": truthful,
	})
	return nil
}

func (gs *GameState) Challenge(challengerID uint) (*ChallengeResult, error) {
	challenger := gs.GetPlayer(challengerID)
	if challenger == nil || !challenger.IsAlive {
		return nil, fmt.Errorf("invalid challenger")
	}
	if gs.Phase != PhaseChallenge {
		return nil, fmt.Errorf("not in challenge phase")
	}
	if gs.LastPlay == nil {
		return nil, fmt.Errorf("no cards to challenge")
	}
	limit := 1
	if NormalizeCharacterID(challenger.CharacterID) == CharacterBristle {
		limit = 2
	}
	if challenger.ChallengeUsed >= limit {
		return nil, fmt.Errorf("challenge limit reached for this round")
	}
	prevPlayer := gs.GetPreviousPlayer()
	if prevPlayer == nil || prevPlayer.ID == challengerID {
		return nil, fmt.Errorf("cannot challenge yourself")
	}

	challenger.ChallengeCount++
	challenger.ChallengeUsed++

	result := &ChallengeResult{
		Success: !gs.LastPlay.Truthful,
		LiarID:  0,
		LoserID: 0,
	}

	var loser *Player
	if !gs.LastPlay.Truthful {
		result.LiarID = prevPlayer.ID
		result.LoserID = prevPlayer.ID
		loser = prevPlayer
		challenger.ChallengeSuccess++
	} else {
		result.LoserID = challengerID
		loser = challenger
	}

	result.Truthful = gs.LastPlay.Truthful
	result.ChallengedCards = gs.LastPlay.Cards
	result.ChallengerID = challengerID
	gs.LastPlay.Challenged = true

	gs.Phase = PhasePunishment
	gs.RecordAction(challengerID, ActChallenge, map[string]interface{}{
		"target":  prevPlayer.ID,
		"success": result.Success,
	})

	_ = loser
	result.Punishment = gs.punishPlayer(loser, loser.ID == challengerID && gs.LastPlay.Truthful)
	return result, nil
}

func (gs *GameState) SkipChallenge() error {
	if gs.Phase != PhaseChallenge {
		return fmt.Errorf("not in challenge phase")
	}
	gs.Phase = PhasePlaying
	gs.NextPlayer()
	gs.checkEmptyHands()
	return nil
}

func (gs *GameState) PassChallenge(playerID uint) (bool, error) {
	if gs.Phase != PhaseChallenge {
		return false, fmt.Errorf("not in challenge phase")
	}
	// 安全检查：如果没有 LastPlay，不应该调用 PassChallenge
	if gs.LastPlay == nil {
		return false, fmt.Errorf("no play to challenge")
	}
	player := gs.GetPlayer(playerID)
	if player == nil || !player.IsAlive {
		return false, fmt.Errorf("invalid player")
	}
	if gs.LastPlay.PlayerID == playerID {
		return false, fmt.Errorf("cannot pass challenge on your own play")
	}

	gs.ChallengePassed[playerID] = true

	eligiblePlayers := 0
	passedPlayers := 0
	for _, p := range gs.Players {
		if p.IsAlive && p.ID != gs.LastPlay.PlayerID {
			eligiblePlayers++
			if gs.ChallengePassed[p.ID] {
				passedPlayers++
			}
		}
	}

	allPassed := eligiblePlayers > 0 && passedPlayers >= eligiblePlayers
	return allPassed, nil
}

type ChallengeResult struct {
	Success         bool              `json:"success"`
	Truthful        bool              `json:"truthful"`
	LiarID          uint              `json:"liar_id"`
	LoserID         uint              `json:"loser_id"`
	ChallengerID    uint              `json:"challenger_id"`
	ChallengedCards []Card            `json:"challenged_cards"`
	Punishment      *PunishmentResult `json:"punishment,omitempty"`
}

func (gs *GameState) Pass(playerID uint) error {
	currentPlayer := gs.GetCurrentPlayer()
	if currentPlayer == nil || currentPlayer.ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if gs.Phase != PhasePlaying {
		return fmt.Errorf("invalid phase")
	}
	player := gs.GetPlayer(playerID)
	if player != nil && len(player.Hand) > 0 {
		return fmt.Errorf("cannot pass when you have cards")
	}
	gs.NextPlayer()
	return nil
}

func (gs *GameState) UseFoxyPeek(playerID, targetID uint) ([]Card, error) {
	if gs.Phase != PhasePlaying {
		return nil, fmt.Errorf("skill can only be used during a game")
	}
	player := gs.GetPlayer(playerID)
	if player == nil || !player.IsAlive {
		return nil, fmt.Errorf("invalid player")
	}
	if NormalizeCharacterID(player.CharacterID) != CharacterFoxy {
		return nil, fmt.Errorf("only Foxy can use this skill")
	}
	if player.SkillUsed {
		return nil, fmt.Errorf("skill already used")
	}
	target := gs.GetPlayer(targetID)
	if target == nil || !target.IsAlive || target.ID == playerID {
		return nil, fmt.Errorf("invalid target")
	}
	player.SkillUsed = true
	hand := make([]Card, len(target.Hand))
	copy(hand, target.Hand)
	return hand, nil
}

type PunishmentResult struct {
	PlayerID        uint `json:"player_id"`
	BulletCount     int  `json:"bullet_count"`
	Chamber         int  `json:"chamber"`
	Survived        bool `json:"survived"`
	TorReduced      bool `json:"tor_reduced"`
	TorImmune       bool `json:"tor_immune"`
	FailedChallenge bool `json:"failed_challenge"`
}

func (gs *GameState) punishPlayer(player *Player, failedChallenge bool) *PunishmentResult {
	punishmentReduced := false
	if failedChallenge && NormalizeCharacterID(player.CharacterID) == CharacterTor && rand.Intn(100) < 50 {
		punishmentReduced = true
	} else {
		player.PunishmentCount++
	}
	bulletSlots := player.PunishmentCount
	if bulletSlots > 6 {
		bulletSlots = 6
	}

	hitChamber := rand.Intn(6) + 1
	survived := hitChamber > bulletSlots
	immune := false
	if !survived && NormalizeCharacterID(player.CharacterID) == CharacterTor && rand.Intn(100) < 30 {
		immune = true
		survived = true
	}

	result := &PunishmentResult{
		PlayerID:        player.ID,
		BulletCount:     bulletSlots,
		Chamber:         hitChamber,
		Survived:        survived,
		TorReduced:      punishmentReduced,
		TorImmune:       immune,
		FailedChallenge: failedChallenge,
	}

	gs.RecordAction(player.ID, ActPunishment, map[string]interface{}{
		"bullet_count":     bulletSlots,
		"chamber":          hitChamber,
		"survived":         survived,
		"tor_reduced":      punishmentReduced,
		"tor_immune":       immune,
		"failed_challenge": failedChallenge,
		"character_id":     player.CharacterID,
		"character_name":   player.CharacterName,
	})

	gs.LastPlay = nil
	if !survived {
		player.IsAlive = false
		gs.AliveCount--
		gs.RecordAction(player.ID, ActEliminated, nil)

		if gs.AliveCount <= 1 {
			gs.endGame()
			return result
		}
	}

	gs.Phase = PhasePlaying
	gs.NextPlayer()
	gs.checkEmptyHands()
	return result
}

func (gs *GameState) checkEmptyHands() {
	allEmpty := true
	for _, p := range gs.Players {
		if p.IsAlive && len(p.Hand) > 0 {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		gs.startNewRound()
	}
}

func (gs *GameState) startNewRound() {
	gs.RoundCounter++
	gs.CurrentRound++
	gs.TargetCard = CardOrder[gs.RoundCounter%len(CardOrder)]
	gs.LastPlay = nil

	deck := NewDeck()
	alivePlayers := make([]*Player, 0)
	for _, p := range gs.Players {
		if p.IsAlive {
			alivePlayers = append(alivePlayers, p)
		}
	}

	hands, _ := DealCards(deck, len(alivePlayers))
	for i, p := range alivePlayers {
		p.Hand = hands[i]
		if NormalizeCharacterID(p.CharacterID) == CharacterScubby {
			p.Hand = append(p.Hand, Wild)
			rand.Shuffle(len(p.Hand), func(i, j int) { p.Hand[i], p.Hand[j] = p.Hand[j], p.Hand[i] })
		}
		p.HandCount = len(hands[i])
		p.HandCount = len(p.Hand)
		p.ChallengeUsed = 0
		p.SkillUsed = false
	}
	gs.Deck = nil
	gs.DiscardPile = nil
}

func (gs *GameState) endGame() {
	gs.Phase = PhaseGameOver
	for _, p := range gs.Players {
		if p.IsAlive {
			winnerID := p.ID
			gs.WinnerID = &winnerID
			break
		}
	}
}

func (gs *GameState) RecordAction(playerID uint, actionType ActionType, data interface{}) {
}

func (gs *GameState) ToPublic(playerID uint) map[string]interface{} {
	players := make([]map[string]interface{}, len(gs.Players))
	for i, p := range gs.Players {
		showHand := false
		if p.ID == playerID {
			showHand = true
		}
		pm := map[string]interface{}{
			"id":               p.ID,
			"nickname":         p.Nickname,
			"is_ai":            p.IsAI,
			"is_online":        p.IsOnline,
			"is_alive":         p.IsAlive,
			"is_ready":         p.IsReady,
			"seat_index":       p.SeatIndex,
			"hand_count":       p.HandCount,
			"punishment_count": p.PunishmentCount,
			"character_id":     p.CharacterID,
			"character_name":   p.CharacterName,
			"skill_used":       p.SkillUsed,
			"challenge_used":   p.ChallengeUsed,
		}
		if showHand {
			pm["hand"] = p.Hand
		}
		players[i] = pm
	}

	result := map[string]interface{}{
		"phase":          gs.Phase,
		"current_player": gs.CurrentPlayer,
		"current_round":  gs.CurrentRound,
		"current_turn":   gs.CurrentTurn,
		"target_card":    gs.TargetCard,
		"players":        players,
		"alive_count":    gs.AliveCount,
	}

	if gs.LastPlay != nil {
		result["last_play"] = map[string]interface{}{
			"player_id": gs.LastPlay.PlayerID,
			"count":     len(gs.LastPlay.CardIDs),
			"claim":     gs.LastPlay.Claim,
		}
	}

	if gs.WinnerID != nil {
		result["winner_id"] = *gs.WinnerID
	}

	// 添加当前玩家的合法操作
	legalActions := gs.GetLegalActions(playerID)
	if legalActions != nil {
		result["legal_actions"] = legalActions
	}

	return result
}

func (gs *GameState) GetLegalActions(playerID uint) []string {
	player := gs.GetPlayer(playerID)
	if player == nil || !player.IsAlive {
		return nil
	}

	actions := make([]string, 0)

	if gs.Phase == PhasePlaying {
		if gs.GetCurrentPlayer().ID == playerID {
			if len(player.Hand) > 0 {
				actions = append(actions, "PLAY_CARD")
			} else {
				// 0张牌自动跳过，不给任何操作选项
				return []string{}
			}
		}
	}

	if gs.Phase == PhaseChallenge {
		if gs.LastPlay != nil && gs.LastPlay.PlayerID != playerID {
			limit := 1
			if NormalizeCharacterID(player.CharacterID) == CharacterBristle {
				limit = 2
			}
			if player.ChallengeUsed < limit {
				actions = append(actions, "CHALLENGE")
			}
		}
	}

	actions = append(actions, "PASS", "CHAT")
	return actions
}

func (gs *GameState) MarshalJSON() ([]byte, error) {
	type Alias GameState
	return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(gs)})
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
