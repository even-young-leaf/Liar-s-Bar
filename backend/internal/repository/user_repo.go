package repository

import (
	"liars-bar/internal/database"
	"liars-bar/internal/model"

	"gorm.io/gorm"
)

type UserRepo struct{}

func NewUserRepo() *UserRepo { return &UserRepo{} }

func (r *UserRepo) Create(user *model.User) error {
	return database.DB.Create(user).Error
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := database.DB.First(&user, id).Error
	return &user, err
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := database.DB.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *UserRepo) Update(user *model.User) error {
	return database.DB.Save(user).Error
}

func (r *UserRepo) UpdateStatus(id uint, status model.UserStatus) error {
	return database.DB.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

func (r *UserRepo) IncrementGames(userID uint, isWin bool) error {
	updates := map[string]interface{}{
		"total_games": gorm.Expr("total_games + 1"),
	}
	if isWin {
		updates["total_wins"] = gorm.Expr("total_wins + 1")
	} else {
		updates["total_losses"] = gorm.Expr("total_losses + 1")
	}
	return database.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *UserRepo) IncrementStats(userID uint, lies, challenges, successfulChallenges int) error {
	updates := map[string]interface{}{}
	if lies != 0 {
		updates["total_lies"] = gorm.Expr("total_lies + ?", lies)
	}
	if challenges != 0 {
		updates["total_challenges"] = gorm.Expr("total_challenges + ?", challenges)
	}
	if successfulChallenges != 0 {
		updates["total_successful_challenges"] = gorm.Expr("total_successful_challenges + ?", successfulChallenges)
	}
	if len(updates) == 0 {
		return nil
	}
	return database.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *UserRepo) TopByElo(limit int) ([]model.User, error) {
	var users []model.User
	err := database.DB.Order("elo_rating DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *UserRepo) GetAvgRank(userID uint) (float64, error) {
	var result struct {
		AvgRank float64
	}
	err := database.DB.Table("game_players").
		Select("AVG(final_rank) as avg_rank").
		Where("user_id = ? AND is_ai = false", userID).
		Scan(&result).Error
	return result.AvgRank, err
}
