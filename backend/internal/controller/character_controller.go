package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type CharacterController struct{}

func NewCharacterController() *CharacterController {
	return &CharacterController{}
}

type CharacterInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Skill       string `json:"skill"`
	AvatarURL   string `json:"avatar_url"`
}

func (c *CharacterController) GetCharacters(ctx *gin.Context) {
	characters := []CharacterInfo{
		{
			ID:          "scubby",
			Name:        "Scubby",
			Description: "起手多一张万能牌",
			Skill:       "开局额外获得1张WILD牌，可以代替任何牌型",
			AvatarURL:   "/assets/characters/scubby.png",
		},
		{
			ID:          "foxy",
			Name:        "Foxy",
			Description: "可以偷看其他玩家手牌",
			Skill:       "每局游戏可使用一次偷看技能，查看目标玩家的所有手牌3秒",
			AvatarURL:   "/assets/characters/foxy.png",
		},
		{
			ID:          "bristle",
			Name:        "Bristle",
			Description: "每轮可以质疑两次",
			Skill:       "不受每轮一次质疑的限制，每轮可以质疑最多2次",
			AvatarURL:   "/assets/characters/bristle.png",
		},
		{
			ID:          "tor",
			Name:        "Tor",
			Description: "减少惩罚或免疫",
			Skill:       "失败惩罚时有50%概率减少1发子弹或完全免疫惩罚",
			AvatarURL:   "/assets/characters/tor.png",
		},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": characters,
	})
}
