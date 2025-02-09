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

	inits.DB.AutoMigrate(&models.User{}, &models.Session{}) // Ensure you migrate all relevant models
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Enable CORS for frontend
	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5174"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// User routes (no authentication required)
	r.PUT("/users/update", controllers.UsersUpdate)
	r.POST("/users/signup", controllers.UsersCreate)
	r.POST("/users/login", controllers.Login)
	r.GET("/users/cookie", controllers.User)
	r.POST("/users/logout", controllers.LogOut)
	r.GET("/users", controllers.UsersIndex)
	r.GET("/users/:id", controllers.UsersShow)
	r.DELETE("/users/delete", controllers.UsersDelete)

	// Protected routes (authentication required)
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware()) // Apply the authentication middleware to all routes in this group

	auth.POST("/sessions/create", controllers.CreateSession) // Session creation (requires authentication)
	auth.POST("/sessions/join", controllers.JoinSession)     // Join session (requires authentication)

	port := viper.GetInt("server.port")
	r.Run(fmt.Sprintf(":%d", port))
}
