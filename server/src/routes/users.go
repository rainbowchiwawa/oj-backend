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

func UserMeHandler(ctx *gin.Context) {
	user := GetUser(ctx)
	ctx.JSON(http.StatusOK, gin.H{
		"id":   user.UserId,
		"name": user.UserName,
		"type": user.UserType,
	})
}
