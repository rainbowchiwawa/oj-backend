package routes

import (
	"github.com/gin-gonic/gin"
)

func Init() {
	router := gin.Default()
	api := router.Group("/api")
	{
		users := api.Group("/users")
		{
			users.POST("/login", UserLoginHandler)
			users.POST("/register", UserRegisterHandler)
		}
	}
	router.Run()
}
