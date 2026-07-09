package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"liars-bar/internal/config"
	"liars-bar/internal/controller"
	"liars-bar/internal/database"
	"liars-bar/internal/game"
	"liars-bar/internal/match"
	"liars-bar/internal/middleware"
	"liars-bar/internal/model"
	"liars-bar/internal/service"
	"liars-bar/internal/utils"
	"liars-bar/internal/websocket"

	"github.com/gin-gonic/gin"
	gorillaWs "github.com/gorilla/websocket"
)

func main() {
	cfg := config.Load()

	// Initialize MySQL
	if err := database.Init(&cfg.Database); err != nil {
		log.Printf("Warning: MySQL init failed: %v (running without DB)", err)
	}

	// Initialize Redis
	if err := utils.InitRedis(&cfg.Redis); err != nil {
		log.Printf("Warning: Redis init failed: %v (running without Redis)", err)
	}

	// Initialize WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize services
	authCtrl := controller.NewAuthController(cfg)
	userCtrl := controller.NewUserController(cfg)
	roomService := service.NewRoomService()
	userSvc := service.NewUserService(cfg)
	roomCtrl := controller.NewRoomController(roomService, hub)
	gameService := service.NewGameService()
	gameCtrl := controller.NewGameController(gameService)

	// Wire room service to hub for database cleanup
	hub.RoomService = roomService

	// Start room cleanup task: check every 20 minutes, remove rooms older than 20 minutes
	hub.StartCleanupTask(20*time.Minute, 20*time.Minute)

	// Wire game-over stat recording (invoked exactly once per finished game)
	hub.OnGameOver = func(roomID, winnerID uint, players []*game.Player) {
		userSvc.RecordGameResult(winnerID, players)
		// Mark as finished first, then delete from database
		roomService.UpdateStatus(roomID, model.RoomStatusFinished)
		roomService.DeleteRoom(roomID)
	}

	// Initialize matchmaking
	matchService := match.NewMatchService(hub, &cfg.Game, roomService)
	matchCtrl := controller.NewMatchController(hub, matchService)

	// Set dependencies for user controller
	userCtrl.SetDependencies(hub, matchService)

	lobbyCtrl := controller.NewLobbyController(hub, matchService)
	adminCtrl := controller.NewAdminController(hub)
	characterCtrl := controller.NewCharacterController()

	// Setup Gin router
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// WebSocket endpoint
	upgrader := gorillaWs.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	r.GET("/ws", middleware.AuthMiddleware(&cfg.JWT), func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		userID := c.GetUint("userID")
		username := c.GetString("username")
		nickname := c.GetString("nickname")
		if nickname == "" {
			nickname = username
		}

		client := websocket.NewClient(userID, username, nickname, conn, hub)
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	// API routes
	api := r.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/login", authCtrl.Login)
	}

	// Character routes (public)
	api.GET("/characters", characterCtrl.GetCharacters)

	// Protected routes
	protected := api.Group("", middleware.AuthMiddleware(&cfg.JWT))

	// User routes
	user := protected.Group("/user")
	{
		user.GET("/profile", userCtrl.GetProfile)
		user.PUT("/profile", userCtrl.UpdateProfile)
		user.GET("/status", userCtrl.GetStatus)
		user.GET("/stats", userCtrl.GetStats)
	}

	// Match routes
	matchRoutes := protected.Group("/match")
	{
		matchRoutes.POST("/start", matchCtrl.StartMatch)
		matchRoutes.POST("/cancel", matchCtrl.CancelMatch)
		matchRoutes.GET("/status", matchCtrl.MatchStatus)
	}

	// Room routes
	roomRoutes := protected.Group("/rooms")
	{
		roomRoutes.POST("", roomCtrl.CreateRoom)
		roomRoutes.GET("", roomCtrl.ListRooms)
		roomRoutes.GET("/:id", roomCtrl.GetRoom)
		roomRoutes.POST("/:id/join", roomCtrl.JoinRoom)
		roomRoutes.POST("/:id/leave", roomCtrl.LeaveRoom)
	}

	// Lobby route
	protected.GET("/lobby", lobbyCtrl.GetLobby)

	// History routes
	protected.GET("/history", gameCtrl.GetHistory)
	protected.GET("/games/:id", gameCtrl.GetGameDetail)

	// Admin routes
	admin := api.Group("/admin")
	{
		admin.GET("/online", adminCtrl.GetOnline)
		admin.GET("/rooms", adminCtrl.GetRooms)
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
