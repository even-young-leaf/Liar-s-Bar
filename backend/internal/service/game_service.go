package service

import (
	"liars-bar/internal/model"
	"liars-bar/internal/repository"
)

type GameService struct {
	repo *repository.GameRepo
}

func NewGameService() *GameService {
	return &GameService{repo: repository.NewGameRepo()}
}

type GameDetailResponse struct {
	Game    model.Game                   `json:"game"`
	Players []repository.GamePlayerInfo  `json:"players"`
	Actions []model.GameAction           `json:"actions"`
}

func (s *GameService) GetUserHistory(userID uint, page, pageSize int) ([]repository.GameHistoryItem, int64, error) {
	return s.repo.GetUserGames(userID, page, pageSize)
}

func (s *GameService) GetGameDetail(gameID uint) (*GameDetailResponse, error) {
	game, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	players, err := s.repo.GetGamePlayers(gameID)
	if err != nil {
		return nil, err
	}

	actions, err := s.repo.GetGameActions(gameID)
	if err != nil {
		return nil, err
	}

	return &GameDetailResponse{
		Game:    *game,
		Players: players,
		Actions: actions,
	}, nil
}
