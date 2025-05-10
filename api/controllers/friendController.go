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
		if existing.Accepted {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Already friends"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": "Friend request already sent, waiting for acceptance"})
		return
	}

	// Create pending friendship (Accepted = false)
	inits.DB.Create(&models.Friend{
		UserID:   user.ID,
		FriendID: friend.ID,
		Accepted: false,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Friend request sent"})
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

func AcceptFriendship(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	// Authenticate user
	isAuth, userIDStr := IsUserAuthenticatedGetId(c)
	if !isAuth {
		return
	}

	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Invalid user ID"})
		return
	}

	// Find the user who sent the friend request
	var sender models.User
	if err := inits.DB.Where("name = ?", body.Name).First(&sender).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Only accept if the current user is the recipient of a pending request
	var request models.Friend
	if err := inits.DB.
		Where("user_id = ? AND friend_id = ? AND accepted = false", sender.ID, userIDUint).
		First(&request).Error; err == nil {
		request.Accepted = true
		inits.DB.Save(&request)
		c.JSON(http.StatusOK, gin.H{"message": "Friendship accepted"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"message": "No pending friendship request from this user"})
}

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

	var connections []models.Friend
	if err := inits.DB.Where("user_id = ? OR friend_id = ?", userIDUint, userIDUint).Find(&connections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching friendships"})
		return
	}

	type FriendInfo struct {
		User                 models.User `json:"user"`
		Accepted             bool        `json:"accepted"`
		ThisUserNeedToAccept bool        `json:"thisUserNeedToAccept"`
	}

	var friendInfos []FriendInfo
	for _, conn := range connections {
		var friendID uint
		if conn.UserID == uint(userIDUint) {
			friendID = conn.FriendID
		} else {
			friendID = conn.UserID
		}

		var friend models.User
		if err := inits.DB.First(&friend, friendID).Error; err != nil {
			continue
		}

		thisUserNeedsToAccept := !conn.Accepted && conn.FriendID == uint(userIDUint)

		friendInfos = append(friendInfos, FriendInfo{
			User:                 friend,
			Accepted:             conn.Accepted,
			ThisUserNeedToAccept: thisUserNeedsToAccept,
		})
	}

	c.JSON(http.StatusOK, gin.H{"friends": friendInfos})
}
