package controller

import (
	"net/http"

	"liars-bar/internal/game"
	"liars-bar/internal/match"
	"liars-bar/internal/websocket"

	"github.com/gin-gonic/gin"
)

type MatchController struct {
	matchService *match.MatchService
	hub          *websocket.Hub
}

func NewMatchController(hub *websocket.Hub, matchService *match.MatchService) *MatchController {
	return &MatchController{hub: hub, matchService: matchService}
}

func (c *MatchController) StartMatch(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	nickname := ctx.GetString("nickname")
	if nickname == "" {
		nickname = ctx.GetString("username")
	}
	var req struct {
		CharacterID string `json:"character_id"`
	}
	_ = ctx.ShouldBindJSON(&req)
	characterID := game.NormalizeCharacterID(req.CharacterID)
	c.matchService.JoinQueue(userID, nickname, characterID)
	ctx.JSON(http.StatusOK, gin.H{
		"code":           0,
		"status":         "WAITING",
		"character_id":   characterID,
		"character_name": game.CharacterName(characterID),
	})
}

func (c *MatchController) CancelMatch(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	c.matchService.LeaveQueue(userID)
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "cancelled"})
}

func (c *MatchController) MatchStatus(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	status := c.matchService.GetQueueStatus(userID)
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "status": status})
}

type LobbyController struct {
	hub          *websocket.Hub
	matchService *match.MatchService
}

func NewLobbyController(hub *websocket.Hub, matchService *match.MatchService) *LobbyController {
	return &LobbyController{
		hub:          hub,
		matchService: matchService,
	}
}

func (c *LobbyController) GetLobby(ctx *gin.Context) {
	queueLength := 0
	if c.matchService != nil {
		queueLength = c.matchService.QueueLength()
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"online_count": c.hub.OnlineCount(),
			"queue_length": queueLength,
			"active_rooms": c.hub.ActiveRooms(),
		},
	})
}

type AdminController struct {
	hub *websocket.Hub
}

func NewAdminController(hub *websocket.Hub) *AdminController {
	return &AdminController{hub: hub}
}

func (c *AdminController) GetOnline(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"online_count": c.hub.OnlineCount(),
		},
	})
}

func (c *AdminController) GetRooms(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": c.hub.ActiveRooms(),
	})
}
