package repository

import (
	"liars-bar/internal/database"
	"liars-bar/internal/model"
)

type GameRepo struct{}

func NewGameRepo() *GameRepo { return &GameRepo{} }

type GameHistoryItem struct {
	GameID       uint    `json:"game_id"`
	GameUUID     string  `json:"game_uuid"`
	RoomID       uint    `json:"room_id"`
	WinnerUserID *uint   `json:"winner_user_id"`
	IsWin        bool    `json:"is_win"`
	FinalRank    int     `json:"final_rank"`
	Survived     bool    `json:"survived"`
	TotalRounds  int     `json:"total_rounds"`
	TotalTurns   int     `json:"total_turns"`
	AICount      int     `json:"ai_count"`
	StartTime    string  `json:"start_time"`
	EndTime      *string `json:"end_time"`
	ScoreChange  int     `json:"score_change"`
}

type GamePlayerInfo struct {
	model.GamePlayer
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

func (r *GameRepo) Create(game *model.Game) error {
	return database.DB.Create(game).Error
}

func (r *GameRepo) FindByID(id uint) (*model.Game, error) {
	var game model.Game
	err := database.DB.First(&game, id).Error
	return &game, err
}

func (r *GameRepo) FindByUUID(uuid string) (*model.Game, error) {
	var game model.Game
	err := database.DB.Where("game_uuid = ?", uuid).First(&game).Error
	return &game, err
}

func (r *GameRepo) FindByUserID(userID uint, limit int) ([]model.Game, error) {
	var games []model.Game
	err := database.DB.
		Joins("JOIN game_players ON game_players.game_id = games.id").
		Where("game_players.user_id = ?", userID).
		Order("games.start_time DESC").
		Limit(limit).
		Find(&games).Error
	return games, err
}

func (r *GameRepo) Update(game *model.Game) error {
	return database.DB.Save(game).Error
}

func (r *GameRepo) CreateGamePlayer(gp *model.GamePlayer) error {
	return database.DB.Create(gp).Error
}

func (r *GameRepo) GetGamePlayers(gameID uint) ([]model.GamePlayer, error) {
	var players []model.GamePlayer
	err := database.DB.Where("game_id = ?", gameID).Find(&players).Error
	return players, err
}

func (r *GameRepo) CreateAction(action *model.GameAction) error {
	return database.DB.Create(action).Error
}

func (r *GameRepo) GetActions(gameID uint) ([]model.GameAction, error) {
	var actions []model.GameAction
	err := database.DB.Where("game_id = ?", gameID).Order("created_at ASC").Find(&actions).Error
	return actions, err
}

func (r *GameRepo) CreateChat(chat *model.ChatRecord) error {
	return database.DB.Create(chat).Error
}

func (r *GameRepo) GetChats(roomID uint, limit int) ([]model.ChatRecord, error) {
	var chats []model.ChatRecord
	err := database.DB.Where("room_id = ?", roomID).Order("created_at DESC").Limit(limit).Find(&chats).Error
	return chats, err
}

func (r *GameRepo) GetUserGames(userID uint, page, pageSize int) ([]GameHistoryItem, int64, error) {
	var total int64
	offset := (page - 1) * pageSize

	// Count total games
	database.DB.Table("game_players").Where("user_id = ?", userID).Count(&total)

	// Query with JOIN
	var items []GameHistoryItem
	err := database.DB.Table("game_players").
		Select(`games.id as game_id, games.game_uuid, games.room_id, games.winner_user_id,
			games.total_rounds, games.total_turns, games.ai_count,
			games.start_time, games.end_time,
			game_players.final_rank, game_players.survived, game_players.score_change,
			CASE WHEN games.winner_user_id = ? THEN 1 ELSE 0 END as is_win`, userID).
		Joins("JOIN games ON games.id = game_players.game_id").
		Where("game_players.user_id = ?", userID).
		Order("games.start_time DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&items).Error

	return items, total, err
}

func (r *GameRepo) GetGameByID(gameID uint) (*model.Game, error) {
	var game model.Game
	err := database.DB.First(&game, gameID).Error
	return &game, err
}

func (r *GameRepo) GetGamePlayers(gameID uint) ([]GamePlayerInfo, error) {
	var players []GamePlayerInfo
	err := database.DB.Table("game_players").
		Select("game_players.*, users.username, users.nickname, users.avatar_url").
		Joins("LEFT JOIN users ON users.id = game_players.user_id").
		Where("game_players.game_id = ?", gameID).
		Order("game_players.final_rank ASC").
		Scan(&players).Error
	return players, err
}

func (r *GameRepo) GetGameActions(gameID uint) ([]model.GameAction, error) {
	var actions []model.GameAction
	err := database.DB.Where("game_id = ?", gameID).
		Order("round_no ASC, turn_no ASC, created_at ASC").
		Find(&actions).Error
	return actions, err
}
