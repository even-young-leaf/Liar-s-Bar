package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"liars-bar/internal/config"
	"liars-bar/internal/game"
	"liars-bar/internal/middleware"
	"liars-bar/internal/model"
	"liars-bar/internal/repository"
	"liars-bar/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepo
	cfg  *config.Config
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func NewUserService(cfg *config.Config) *UserService {
	return &UserService{repo: repository.NewUserRepo(), cfg: cfg}
}

func (s *UserService) Register(username, password, nickname string) (*model.User, error) {
	username = strings.TrimSpace(username)
	nickname = strings.TrimSpace(nickname)
	if username == "" || !usernamePattern.MatchString(username) {
		return nil, errors.New("username can only contain letters, numbers, underscores and hyphens")
	}
	if nickname == "" {
		return nil, errors.New("nickname is required")
	}

	existing, err := s.repo.FindByUsername(username)
	if err == nil && existing != nil && existing.ID != 0 {
		return nil, errors.New("username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Nickname:     nickname,
		EloRating:    1000,
		Status:       model.UserStatusOffline,
	}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Login(username, password string) (string, *model.User, error) {
	username = strings.TrimSpace(username)
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := middleware.GenerateToken(&s.cfg.JWT, user)
	if err != nil {
		return "", nil, err
	}

	s.repo.UpdateStatus(user.ID, model.UserStatusOnline)
	return token, user, nil
}

func (s *UserService) GetProfile(userID uint) (*model.User, error) {
	return s.repo.FindByID(userID)
}

func (s *UserService) UpdateProfile(userID uint, nickname, avatarURL string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}
	if nickname != "" {
		user.Nickname = nickname
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}
	return s.repo.Update(user)
}

func (s *UserService) UpdateELO(userID uint, delta int) {
	user, _ := s.repo.FindByID(userID)
	if user != nil {
		user.EloRating += delta
		if user.EloRating < 0 {
			user.EloRating = 0
		}
		s.repo.Update(user)
	}
}

func (s *UserService) UpdateStats(userID uint, isWin bool) {
	s.repo.IncrementGames(userID, isWin)
}

// RecordGameResult updates each human player's cumulative stats after a game ends.
// AI players are skipped. The winner gains ELO; losers drop.
func (s *UserService) RecordGameResult(winnerID uint, players []*game.Player) {
	for _, p := range players {
		if p.IsAI {
			continue
		}
		isWin := p.ID == winnerID
		if err := s.repo.IncrementGames(p.ID, isWin); err != nil {
			log.Printf("Failed to increment games for user %d: %v", p.ID, err)
		}
		if err := s.repo.IncrementStats(p.ID, p.LieCount, p.ChallengeCount, p.ChallengeSuccess); err != nil {
			log.Printf("Failed to increment stats for user %d: %v", p.ID, err)
		}
		if isWin {
			s.UpdateELO(p.ID, 20)
		} else {
			s.UpdateELO(p.ID, -15)
		}
	}
}

func (s *UserService) SetOnline(userID uint) {
	utils.Rdb.Set(context.Background(), userOnlineKey(userID), true, 0)
}

func (s *UserService) SetOffline(userID uint) {
	utils.Rdb.Del(context.Background(), userOnlineKey(userID))
	s.repo.UpdateStatus(userID, model.UserStatusOffline)
}

func (s *UserService) IsOnline(userID uint) bool {
	_, err := utils.Rdb.Get(context.Background(), userOnlineKey(userID)).Result()
	return err == nil
}

func (s *UserService) OnlineCount() int64 {
	keys, _ := utils.Rdb.Keys(context.Background(), "user:online:*").Result()
	return int64(len(keys))
}

func userOnlineKey(userID uint) string {
	return fmt.Sprintf("user:online:%d", userID)
}

type UserStats struct {
	TotalGames                int     `json:"total_games"`
	TotalWins                 int     `json:"total_wins"`
	TotalLosses               int     `json:"total_losses"`
	WinRate                   float64 `json:"win_rate"`
	EloRating                 int     `json:"elo_rating"`
	TotalLies                 int     `json:"total_lies"`
	TotalChallenges           int     `json:"total_challenges"`
	TotalSuccessfulChallenges int     `json:"total_successful_challenges"`
	ChallengeSuccessRate      float64 `json:"challenge_success_rate"`
	AvgRank                   float64 `json:"avg_rank"`
}

func (s *UserService) GetUserStats(userID uint) (*UserStats, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	stats := &UserStats{
		TotalGames:                user.TotalGames,
		TotalWins:                 user.TotalWins,
		TotalLosses:               user.TotalLosses,
		EloRating:                 user.EloRating,
		TotalLies:                 user.TotalLies,
		TotalChallenges:           user.TotalChallenges,
		TotalSuccessfulChallenges: user.TotalSuccessfulChallenges,
	}

	// Calculate win rate
	if stats.TotalGames > 0 {
		stats.WinRate = float64(stats.TotalWins) / float64(stats.TotalGames)
	}

	// Calculate challenge success rate
	if stats.TotalChallenges > 0 {
		stats.ChallengeSuccessRate = float64(stats.TotalSuccessfulChallenges) / float64(stats.TotalChallenges)
	}

	// Calculate average rank from game_players table
	avgRank, err := s.repo.GetAvgRank(userID)
	if err == nil {
		stats.AvgRank = avgRank
	}

	return stats, nil
}
