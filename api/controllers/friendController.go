package controllers

import (
	"net/http"
	"strconv"
	"yuval/inits"
	"yuval/models"

	"github.com/gin-gonic/gin"
)

// AddFriend creates a mutual friendship between the authenticated user and another user
func AddFriend(c *gin.Context) {
	var req struct {
		FriendName string `json:"friendName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	// Get authenticated user ID
	isAuth, userID := IsUserAuthenticatedGetId(c)
	if !isAuth {
		return
	}

	var user models.User
	if err := inits.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Find the friend user
	var friend models.User
	if err := inits.DB.Where("name = ?", req.FriendName).First(&friend).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Friend user not found"})
		return
	}

	// Prevent user from adding themselves
	if user.ID == friend.ID {
		c.JSON(http.StatusBadRequest, gin.H{"message": "You cannot add yourself as a friend"})
		return
	}

	// Check if already friends (in either direction)
	var existing models.Friend
	if err := inits.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		user.ID, friend.ID, friend.ID, user.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Already friends"})
		return
	}

	// Create friendship
	inits.DB.Create(&models.Friend{UserID: user.ID, FriendID: friend.ID})

	c.JSON(http.StatusOK, gin.H{"message": "Friend added successfully"})
}

// DeleteFriend removes the friendship from both sides
func DeleteFriend(c *gin.Context) {
	var req struct {
		FriendName string `json:"friendName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	isAuth, userID := IsUserAuthenticatedGetId(c)
	if !isAuth {
		return
	}

	var user models.User
	if err := inits.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	var friend models.User
	if err := inits.DB.Where("name = ?", req.FriendName).First(&friend).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Friend user not found"})
		return
	}

	inits.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		user.ID, friend.ID, friend.ID, user.ID).Delete(&models.Friend{})

	c.JSON(http.StatusOK, gin.H{"message": "Friendship deleted"})
}

// CheckFriendship checks if the authenticated user is friends with another user (in either direction)
func CheckFriendship(c *gin.Context) {
	friendName := c.Param("name")

	// Authenticate user
	isAuth, userIDStr := IsUserAuthenticatedGetId(c)
	if !isAuth {
		return
	}

	// Convert user ID to uint
	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Invalid user ID"})
		return
	}

	// Find the friend user by name
	var friend models.User
	if err := inits.DB.Where("name = ?", friendName).First(&friend).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Check if friendship exists in either direction
	var existing models.Friend
	if err := inits.DB.
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userIDUint, friend.ID, friend.ID, userIDUint).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"friends": true})
		return
	}

	// No friendship found
	c.JSON(http.StatusOK, gin.H{"friends": false})
}

// GetFriends returns a list of all friends for the authenticated user
func GetFriends(c *gin.Context) {
	isAuth, userIDStr := IsUserAuthenticatedGetId(c)
	if !isAuth {
		return
	}

	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Invalid user ID"})
		return
	}

	// Find all friendships involving the user
	var connections []models.Friend
	if err := inits.DB.Where("user_id = ? OR friend_id = ?", userIDUint, userIDUint).Find(&connections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching friendships"})
		return
	}

	// Extract the other user's ID from each connection
	friendIDs := make([]uint, 0)
	for _, conn := range connections {
		if conn.UserID == uint(userIDUint) {
			friendIDs = append(friendIDs, conn.FriendID)
		} else {
			friendIDs = append(friendIDs, conn.UserID)
		}
	}

	// Get user details for all friend IDs
	var friends []models.User
	if len(friendIDs) > 0 {
		if err := inits.DB.Where("id IN ?", friendIDs).Find(&friends).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching friend details"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"friends": friends})
}
