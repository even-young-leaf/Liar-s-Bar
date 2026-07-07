package controller

import (
	"net/http"
	"strconv"

	"liars-bar/internal/service"

	"github.com/gin-gonic/gin"
)

type GameController struct {
	gameService *service.GameService
}

func NewGameController(gameService *service.GameService) *GameController {
	return &GameController{gameService: gameService}
}

func (c *GameController) GetHistory(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	games, total, err := c.gameService.GetUserHistory(userID, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get history",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"games":     games,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (c *GameController) GetGameDetail(ctx *gin.Context) {
	gameIDStr := ctx.Param("id")
	gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid game id",
		})
		return
	}

	detail, err := c.gameService.GetGameDetail(uint(gameID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "game not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": detail,
	})
}
