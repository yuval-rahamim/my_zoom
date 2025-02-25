package main

import (
	"fmt"
	"yuval/controllers"
	"yuval/inits"
	"yuval/middleware"
	"yuval/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func init() {
	inits.InitConfig()
	inits.ConnectToDB()
	inits.DB.AutoMigrate(&models.User{}, &models.Session{}, &models.UserSession{}) // Ensure you migrate all relevant models
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Enable CORS for frontend
	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true // Allow all origins dynamically
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Public routes
	r.POST("/users/login", controllers.Login)
	r.POST("/users/signup", controllers.UsersCreate)

	// Protected user routes (Require authentication)
	r.GET("/users", middleware.AuthMiddleware(), controllers.UsersIndex)
	r.PUT("/users/update", middleware.AuthMiddleware(), controllers.UserUpdate)
	r.GET("/users/cookie", middleware.AuthMiddleware(), controllers.User)
	r.POST("/users/logout", middleware.AuthMiddleware(), controllers.LogOut)
	r.GET("/users/:name", middleware.AuthMiddleware(), controllers.GetUserByName)

	// Session routes (Require authentication)
	r.POST("/sessions/create", middleware.AuthMiddleware(), controllers.CreateSession)
	r.POST("/sessions/join", middleware.AuthMiddleware(), controllers.JoinSession)
	r.GET("/sessions/:id", middleware.AuthMiddleware(), controllers.GetSessionDetails) // Fetch session details and participants

	r.Static("/uploads", "./uploads")
	r.POST("/video/upload", middleware.AuthMiddleware(), controllers.ConvertToMPEGTS)
	r.POST("/video/stream", middleware.AuthMiddleware(), controllers.ServeDashFile)

	// Admin routes (Require both authentication & manager check)
	r.DELETE("/users/delete", middleware.AuthMiddleware(), middleware.ManagerMiddlewar(), controllers.UsersDelete)
	r.PUT("/users/manager", middleware.AuthMiddleware(), middleware.ManagerMiddlewar(), controllers.UserMakeManager)

	// Start server
	port := viper.GetInt("server.port")
	r.Run(fmt.Sprintf(":%d", port))
}
