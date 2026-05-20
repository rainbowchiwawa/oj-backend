package routes

import (
	"fmt"
	"oj/server/database"
	"os"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
)

func Init() {
	userAuth, err := jwt.New(JWTInitParams(database.TypeUser))
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	router := gin.Default()
	api := router.Group("/api")
	{
		users := api.Group("/users")
		{
			users.POST("/register", UserRegisterHandler)
			users.POST("/login", userAuth.LoginHandler)
		}
		usersAuthed := api.Group("/users", userAuth.MiddlewareFunc())
		{
			usersAuthed.POST("/logout", userAuth.LogoutHandler)
			usersAuthed.GET("/me", UserMeHandler)
		}
	}
	router.Run(":8080")
}
