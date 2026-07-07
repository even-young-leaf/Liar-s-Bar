package controller

import (
	"net/http"
	"strconv"

	"liars-bar/internal/config"
	"liars-bar/internal/model"
	"liars-bar/internal/service"
	"liars-bar/internal/websocket"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService *service.UserService
}

func NewAuthController(cfg *config.Config) *AuthController {
	return &AuthController{userService: service.NewUserService(cfg)}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	Nickname string `json:"nickname" binding:"required,min=1,max=50"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (c *AuthController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	_, err := c.userService.Register(req.Username, req.Password, req.Nickname)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	token, user, err := c.userService.Login(req.Username, req.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":  0,
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"nickname": user.Nickname,
			"username": user.Username,
		},
	})
}

type UserController struct {
	userService  *service.UserService
	hub          *websocket.Hub
	matchService interface {
		GetQueueStatus(userID uint) string
	}
}

func NewUserController(cfg *config.Config) *UserController {
	return &UserController{userService: service.NewUserService(cfg)}
}

func (c *UserController) SetDependencies(hub *websocket.Hub, matchService interface{ GetQueueStatus(userID uint) string }) {
	c.hub = hub
	c.matchService = matchService
}

func (c *UserController) GetProfile(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	user, err := c.userService.GetProfile(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "user not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": user,
	})
}

func (c *UserController) UpdateProfile(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	var req struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := c.userService.UpdateProfile(userID, req.Nickname, req.AvatarURL); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func (c *UserController) GetStatus(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	status := "ONLINE"
	var roomID uint = 0
	canReconnect := false

	// Check if in match queue
	if c.matchService != nil {
		queueStatus := c.matchService.GetQueueStatus(userID)
		if queueStatus == "WAITING" {
			status = "IN_QUEUE"
			ctx.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": gin.H{
					"status":        status,
					"room_id":       0,
					"can_reconnect": false,
				},
			})
			return
		}
	}

	// Check if in a room
	if c.hub != nil {
		if client := c.hub.GetClient(userID); client != nil && client.RoomID > 0 {
			roomID = client.RoomID
			room := c.hub.GetRoom(roomID)
			if room != nil {
				status = "IN_GAME"
				canReconnect = true
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"status":        status,
			"room_id":       roomID,
			"can_reconnect": canReconnect,
		},
	})
}

func (c *UserController) GetStats(ctx *gin.Context) {
	userID := ctx.GetUint("userID")

	stats, err := c.userService.GetUserStats(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "user not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
	})
}

type RoomController struct {
	roomService *service.RoomService
	hub         *websocket.Hub
}

func NewRoomController(roomService *service.RoomService, hub *websocket.Hub) *RoomController {
	return &RoomController{roomService: roomService, hub: hub}
}

func (c *RoomController) ensureHubRoom(room *model.Room) {
	if c.hub == nil || room == nil {
		return
	}
	if c.hub.GetRoom(room.ID) != nil {
		return
	}
	c.hub.RegisterRoom(websocket.NewGameRoom(room.ID, room.RoomName, c.hub))
}

func (c *RoomController) CreateRoom(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	var req struct {
		Name string `json:"name"`
	}
	ctx.ShouldBindJSON(&req)
	if req.Name == "" {
		req.Name = "New Room"
	}

	room, err := c.roomService.CreateRoom(userID, req.Name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.ensureHubRoom(room)

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": room})
}

func (c *RoomController) ListRooms(ctx *gin.Context) {
	rooms, err := c.roomService.ListActiveRooms()
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": []interface{}{}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": rooms})
}

func (c *RoomController) GetRoom(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid room id"})
		return
	}
	room, err := c.roomService.GetRoom(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "room not found"})
		return
	}
	players, _ := c.roomService.GetPlayers(uint(id))
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"room": room, "players": players}})
}

func (c *RoomController) JoinRoom(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid room id"})
		return
	}
	room, err := c.roomService.GetRoom(uint(id))
	if err != nil {
		// DB miss: fall back to the in-memory hub so that rooms which exist
		// only in memory (e.g. a matchmade room whose DB row was cleaned up)
		// can still be joined instead of silently 404-ing. The actual player
		// add happens later via the WebSocket PLAYER_JOIN handshake.
		if c.hub != nil && c.hub.GetRoom(uint(id)) != nil {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
			return
		}
		ctx.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "room not found"})
		return
	}
	if err := c.roomService.JoinRoom(room.ID, userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	c.ensureHubRoom(room)
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func (c *RoomController) LeaveRoom(ctx *gin.Context) {
	userID := ctx.GetUint("userID")
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid room id"})
		return
	}
	if err := c.roomService.LeaveRoom(uint(id), userID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if c.hub != nil {
		if room := c.hub.GetRoom(uint(id)); room != nil {
			room.HandleEvent(websocket.GameEvent{Type: "PLAYER_LEAVE", PlayerID: userID})
		}
		if client := c.hub.GetClient(userID); client != nil && client.RoomID == uint(id) {
			client.RoomID = 0
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
