package middleware

import (
	"net/http"
	"yuval/controllers"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware ensures the user is authenticated.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := controllers.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Step 2: Set the user ID in the context for later use in the handler
		c.Set("userID", userID)

		// Step 3: Call the next handler in the chain
		c.Next()
	}
}
