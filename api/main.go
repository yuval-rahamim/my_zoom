package main

import (
	"fmt"
	"log"
	"net/http"
	"yuval/controllers"
	"yuval/dasher"
	"yuval/inits"
	"yuval/middleware"
	"yuval/models"
	"yuval/websocket2" // Import WebSocket package

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func init() {
	inits.InitConfig()
	inits.ConnectToDB()
	inits.DB.AutoMigrate(&models.User{}, &models.Session{}, &models.UserSession{}, &models.Friend{}) // Ensure you migrate all relevant models
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Enable CORS for frontend
	r.SetTrustedProxies([]string{"myzoom.co.il"})
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true // Allow all origins dynamically
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Initialize WebSocket Hub and start handling messages
	go websocket2.HandleMessages()

	// WebSocket2 route
	r.GET("/ws", middleware.AuthMiddleware(), websocket2.HandleConnections)

	go func() {
		http.HandleFunc("/b", dasher.HandleWebsocket)
		log.Fatal(http.ListenAndServeTLS(":8080", "keys/myzoom.crt", "keys/myzoom.key", nil))
	}()

	// Public routes
	r.POST("/users/login", controllers.Login)
	r.POST("/users/signup", controllers.UsersCreate)

	// Protected user routes (Require authentication)
	r.GET("/users", middleware.AuthMiddleware(), controllers.UsersIndex)
	r.PUT("/users/update", middleware.AuthMiddleware(), controllers.UserUpdate)
	r.GET("/users/cookie", middleware.AuthMiddleware(), controllers.User)
	r.POST("/users/logout", middleware.AuthMiddleware(), controllers.LogOut)
	r.GET("/users/:name", middleware.AuthMiddleware(), controllers.GetUserByName)

	r.GET("/user/meetings", middleware.AuthMiddleware(), controllers.GetUserMeetings)
	r.DELETE("/user/meetings/delete", middleware.AuthMiddleware(), controllers.DeleteUserMeetingVideos)

	r.GET("/friends/all", middleware.AuthMiddleware(), controllers.GetFriends)
	r.POST("/friends/add", middleware.AuthMiddleware(), controllers.AddFriend)
	r.POST("/friends/accept", middleware.AuthMiddleware(), controllers.AcceptFriendship)
	r.DELETE("/friends/delete", middleware.AuthMiddleware(), controllers.DeleteFriend)

	// Session routes (Require authentication)
	r.POST("/sessions/create", middleware.AuthMiddleware(), controllers.CreateSession)
	r.POST("/sessions/join", middleware.AuthMiddleware(), controllers.JoinSession)
	r.GET("/sessions/:id", middleware.AuthMiddleware(), controllers.GetSessionDetails) // Fetch session details and participants

	r.Static("/uploads", "./uploads")
	r.Static("/videos", "./videos")

	// r.POST("/video/upload", middleware.AuthMiddleware(), controllers.ConvertToMPEGTS)
	r.POST("/video/stream", middleware.AuthMiddleware(), dasher.ServeDashFile)

	// Admin routes (Require both authentication & manager check)
	r.DELETE("/users/delete", middleware.AuthMiddleware(), middleware.ManagerMiddlewar(), controllers.UsersDelete)
	r.PUT("/users/manager", middleware.AuthMiddleware(), middleware.ManagerMiddlewar(), controllers.UserMakeManager)

	// Start server
	port := viper.GetInt("server.port")
	err := r.RunTLS(fmt.Sprintf(":%d", port), "keys/myzoom.crt", "keys/myzoom.key")
	if err != nil {
		log.Fatalf("Failed to run HTTPS server: %v", err)
	}

}
