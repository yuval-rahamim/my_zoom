package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"yuval/inits"
	"yuval/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

// Get user details
func User(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userID)
	var user models.User
	if err := inits.DB.Where("id = ?", userIDStr).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Extracts user ID from JWT cookie and checks if the user is authenticated
func GetUserIDFromToken(c *gin.Context) (string, error) {
	// Step 1: Retrieve the JWT cookie
	cookieValue, err := c.Cookie("JWT")
	if err != nil {
		return "", err
	}

	// Step 2: Parse the token
	token, err := jwt.Parse(cookieValue, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("secret")), nil
	})
	if err != nil || !token.Valid {
		return "", err
	}

	// Step 3: Extract user ID from claims
	claims := token.Claims.(jwt.MapClaims)
	userID, ok := claims["iss"].(string)
	if !ok {
		return "", err
	}

	// Step 4: Extend the cookie's expiration time
	maxAge := 30 * 60 // 30 minutes
	c.SetCookie("JWT", cookieValue, maxAge, "/", "", false, true)

	return userID, nil
}

// Checks if the user is authenticated and returns the user ID
func IsUserAuthenticatedGetId(c *gin.Context) (bool, string) {
	userID, err := GetUserIDFromToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return false, ""
	}
	return true, userID
}

// Checks if the user is a manager
func IsUserManager(c *gin.Context) bool {
	isAuthenticated, userID := IsUserAuthenticatedGetId(c)
	if !isAuthenticated {
		return false
	}

	var currentUser models.User
	if err := inits.DB.Where("id = ?", userID).First(&currentUser).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized: User not found"})
		return false
	}

	return currentUser.Manager
}

var account struct {
	Name     string
	Password string
	ImgPath  string
	Manager  bool
}

// Create a new user
func UsersCreate(c *gin.Context) {

	// Bind JSON input
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	// Check if the user already exists
	var existingUser models.User
	result := inits.DB.Where("name = ?", account.Name).First(&existingUser)
	if result.Error == nil {
		// User already exists
		c.JSON(http.StatusBadRequest, gin.H{"message": "User already signed up, try signing in"})
		return
	}

	// Hash the password
	password, err := bcrypt.GenerateFromPassword([]byte(account.Password), 14)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error hashing password"})
		return
	}

	// Create the new user
	user := models.User{
		Name:     account.Name,
		Password: password,
		ImgPath:  account.ImgPath,
		Manager:  false,
	}

	result = inits.DB.Create(&user)
	if result.Error != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// User login function
func Login(c *gin.Context) {

	if err := c.ShouldBindJSON(&account); err != nil {
		log.Printf("Database error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	var user models.User
	result := inits.DB.Where("name = ?", account.Name).First(&user)

	if result.Error != nil {
		log.Printf("Database error: %v", result.Error)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(account.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Incorrect password"})
		return
	}

	secret := viper.GetString("secret")
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Secret key is missing"})
		return
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    strconv.Itoa(int(user.ID)),
		ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
	})

	token, err := claims.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Error creating token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create token"})
		return
	}

	// Set cookie to expire in 30 minutes
	c.SetCookie("JWT", token, 30*60, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Log out user
func LogOut(c *gin.Context) {
	c.SetCookie("JWT", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Unified Update function for user
func UserUpdate(c *gin.Context) {
	// Get userID from middleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Convert userID to string (if stored as int in context)
	userIDStr := fmt.Sprintf("%v", userID)

	// Check if the user is a manager
	isManager := IsUserManager(c)

	// Parse request body for the update
	var input struct {
		Name     string `json:"Name"`
		ImgPath  string `json:"ImgPath"`
		UserName string `json:"userName"` // Target user to update, if different from the logged-in user
		Manager  bool   `json:"Manager"`  // To update manager status (only for managers)
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	// Check if we are updating the logged-in user or someone else
	var targetUser models.User
	if input.UserName == "" {
		// If no UserName is provided, update the current user
		if err := inits.DB.First(&targetUser, userIDStr).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
			return
		}
	} else {
		// If updating another user, ensure the logged-in user is a manager
		if !isManager {
			c.JSON(http.StatusForbidden, gin.H{"message": "You must be a manager to update another user"})
			return
		}

		// If updating another user, find the target user
		if err := inits.DB.Where("name = ?", input.UserName).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
			return
		}
	}

	// Prepare updates
	updates := map[string]interface{}{}
	if len(input.Name) >= 3 && input.Name != targetUser.Name {
		// Check if the new name is already taken (except for the current user)
		var existingUser models.User
		if err := inits.DB.Where("name = ? AND id != ?", input.Name, targetUser.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "User name already exists"})
			return
		}
		updates["Name"] = input.Name
	}

	if input.ImgPath != "" && input.ImgPath != targetUser.ImgPath {
		updates["ImgPath"] = input.ImgPath
	}

	// Only allow managers to update "Manager" status
	if isManager && targetUser.Manager != input.Manager {
		updates["Manager"] = input.Manager
	}

	// If the user is not a manager, they cannot change the "Manager" field
	if !isManager && input.Manager != targetUser.Manager {
		// Make sure users cannot change their "Manager" status
		c.JSON(http.StatusForbidden, gin.H{"message": "You cannot change your Manager status"})
		return
	}

	// If there are no changes, return early
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No changes detected"})
		return
	}

	// Apply updates
	if err := inits.DB.Model(&targetUser).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully", "user": targetUser})
}

// List all users
func UsersIndex(c *gin.Context) {
	var users []models.User
	inits.DB.Find(&users)

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func UsersShow(c *gin.Context) {
	// Log the received ID
	idParam := c.Param("id")
	log.Printf("Received ID: %s", idParam)

	// Convert ID to integer
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID"})
		return
	}

	var user models.User
	// Debug query to log actual SQL
	if err := inits.DB.Debug().First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Delete a user
func UsersDelete(c *gin.Context) {
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	result := inits.DB.Unscoped().Where("Name = ?", account.Name).Delete(&models.User{})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.Status(http.StatusOK)
}

// Make a user a manager
func UserMakeManager(c *gin.Context) {
	if err := c.BindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	var user models.User
	result := inits.DB.Where("name = ?", account.Name).First(&user)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	inits.DB.Model(&user).Update("Manager", true)

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// GetUserByName fetches a user by their name from the database
func GetUserByName(c *gin.Context) {
	name := c.Param("name") // Extract the username from the URL

	var user models.User
	if err := inits.DB.Where("name = ?", name).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
