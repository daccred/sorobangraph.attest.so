package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

// MockUser represents mock user data
type MockUser struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u UserController) Retrieve(c *gin.Context) {
	userID := c.Param("id")
	if userID != "" {
		// Return mock user data
		mockUser := MockUser{
			ID:        userID,
			Username:  "user_" + userID,
			Email:     "user" + userID + "@example.com",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now(),
		}
		c.JSON(http.StatusOK, gin.H{"message": "User found!", "user": mockUser})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
	c.Abort()
}
