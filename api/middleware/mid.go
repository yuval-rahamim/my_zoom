package middleware

import (
	"net/http"
	"yuval/controllers"
	"yuval/inits"
	"yuval/models"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware ensures the user is authenticated.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := controllers.GetUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Set the user ID in the context for later use in the handler
		c.Set("userID", userID)

		// Call the next handler in the chain
		c.Next()
	}
}

// IsUserManager ensures the user is authenticated AND a manager.
func ManagerMiddlewar() gin.HandlerFunc {
	return func(c *gin.Context) {
		// If the request was aborted in AuthMiddleware, return early
		if c.IsAborted() {
			return
		}

		// Retrieve user ID from context (set by AuthMiddleware)
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
			c.Abort()
			return
		}

		// Convert userID to the expected type (assuming it's a string)
		userIDStr, ok := userID.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
			c.Abort()
			return
		}

		// Fetch user details from the database
		var currentUser models.User
		if err := inits.DB.Where("id = ?", userIDStr).First(&currentUser).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized: User not found"})
			c.Abort()
			return
		}

		// Check if the user is a manager
		if !currentUser.Manager {
			c.JSON(http.StatusForbidden, gin.H{"message": "Forbidden: User is not a manager"})
			c.Abort()
			return
		}

		// Proceed to the next handler
		c.Next()
	}
}
