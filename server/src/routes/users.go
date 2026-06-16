package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"oj/server/database"
)

type UserRegisterRequest struct {
	Type     string `json:"type" binding:"required,oneof=admin user"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLoginRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// @Summary Register a new user
// @Description Register a user with name, password, and type
// @Tags users
// @Accept json
// @Produce plain
// @Param request body UserRegisterRequest true "User Registration Info"
// @Success 200 {string} string "OK"
// @Failure 400 {string} string "Bad Request"
// @Failure 409 {string} string "Conflict"
// @Router /users/register [post]
func UserRegisterHandler(ctx *gin.Context) {
	var body UserRegisterRequest
	if err := ctx.Bind(&body); err != nil {
		return
	}

	hash := sha256.Sum256([]byte(body.Password))
	if err := database.CreateUser(body.Type, body.Name, hex.EncodeToString(hash[:])); err != nil {
		fmt.Println(err)
		ctx.String(http.StatusConflict, "")
		return
	}

	ctx.String(http.StatusOK, "")
}

// @Summary Get current user info
// @Description Get the details of the currently logged-in user
// @Tags users
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /users/me [get]
func UserMeHandler(ctx *gin.Context) {
	user := GetUser(ctx)
	ctx.JSON(http.StatusOK, gin.H{
		"id":   user.UserId,
		"name": user.UserName,
		"type": user.UserType,
	})
}

// @Summary User login
// @Description Login with name and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body UserLoginRequest true "User Login Request"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {string} string "Unauthorized"
// @Router /users/login [post]
func _dummyLogin() { /* Login handled by gin-jwt LoginHandler */ }

// @Summary User logout
// @Description Logout current user
// @Tags users
// @Security Bearer
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /users/logout [post]
func _dummyLogout() { /* Logout handled by gin-jwt LogoutHandler */ }
