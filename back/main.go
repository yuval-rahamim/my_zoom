package main

import (
	"fmt"
	"yuval/controllers"
	"yuval/inits"
	"yuval/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func init() {
	inits.InitConfig()
	inits.ConnectToDB()

	inits.DB.AutoMigrate(&models.User{})
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	r.SetTrustedProxies([]string{"127.0.0.1"})
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// User routes
	r.PUT("/users/update", controllers.UsersUpdate)
	r.POST("/users/signup", controllers.UsersCreate)
	r.POST("/users/login", controllers.Login)
	r.GET("/users/cookie", controllers.User)
	r.POST("/users/logout", controllers.LogOut)
	r.GET("/users", controllers.UsersIndex)
	r.GET("/users/:id", controllers.UsersShow)
	r.DELETE("/users/delete", controllers.UsersDelete)

	port := viper.GetInt("server.port")
	r.Run(fmt.Sprintf(":%d", port))
}
