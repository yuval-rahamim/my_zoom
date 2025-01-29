package controllers

import (
	"example/yuval/inits"
	"example/yuval/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

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
	expirationTime := time.Now().Add(30 * time.Minute)
	c.SetCookie("JWT", cookieValue, int(expirationTime.Unix()), "/", "", false, true)

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
	Manager  bool
}

// User login function
func Login(c *gin.Context) {

	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	var user models.User
	result := inits.DB.Where("name = ?", account.Name).First(&user)

	if result.Error != nil {
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

// Update user details
func UsersUpdate(c *gin.Context) {
	if !IsUserManager(c) {
		return
	}

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

	inits.DB.Model(&user).Updates(models.User{
		Name: account.Name,
	})
	if account.Manager != user.Manager {
		inits.DB.Model(&user).Update("Manager", account.Manager)
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
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
	user := models.User{Name: account.Name, Password: password, Manager: false}
	result = inits.DB.Create(&user)
	if result.Error != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
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
	var user struct {
		Name string
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	result := inits.DB.Unscoped().Where("Name = ?", user.Name).Delete(&models.User{})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.Status(http.StatusOK)
}

// Make a user a manager
func UserMakeManager(c *gin.Context) {
	if !IsUserManager(c) {
		return
	}

	var account struct {
		Name string
	}
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

// Get user details
func User(c *gin.Context) {
	cookieValue, err := c.Cookie("JWT")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	token, err := jwt.Parse(cookieValue, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("secret")), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := claims["iss"]
	var user models.User
	if err := inits.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
