package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"oj/server/database"
)

type UserLoginRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserRegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required,oneof:admin user"`
	Password string `json:"password" binding:"required"`
}

func UserLoginHandler(ctx *gin.Context) {
	var body UserLoginRequest
	if err := ctx.Bind(&body); err != nil {
		return
	}

	user, err := database.GetUserByName(body.Name)
	if err != nil {
		fmt.Println(err)
		ctx.String(http.StatusUnauthorized, "")
		return
	}

	hash := sha256.Sum256([]byte(body.Password))
	if hex.EncodeToString(hash[:]) != user.PasswordHash {
		ctx.String(http.StatusUnauthorized, "")
		return
	}

	ctx.String(http.StatusOK, "")
}

func UserRegisterHandler(ctx *gin.Context) {
	var body UserRegisterRequest
	if err := ctx.Bind(&body); err != nil {
		return
	}

	hash := sha256.Sum256([]byte(body.Password))
	if err := database.CreateUser(body.Name, hex.EncodeToString(hash[:])); err != nil {
		fmt.Println(err)
		ctx.String(http.StatusConflict, "")
		return
	}

	ctx.String(http.StatusOK, "")
}
