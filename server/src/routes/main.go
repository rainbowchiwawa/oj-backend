package routes

import (
	"fmt"
	"oj/server/database"
	"os"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
)

func Init() {
	userAuth, userErr := jwt.New(JWTInitParams(database.TypeUser))
	adminAuth, adminErr := jwt.New(JWTInitParams(database.TypeAdmin))
	if userErr != nil || adminErr != nil {
		fmt.Println(userErr, adminErr)
		os.Exit(-1)
	}

	router := gin.Default()
	router.MaxMultipartMemory = 50 << 20 // 50 MB
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

		problems := api.Group("/problems")
		{
			problems.GET("/:id", ProblemGetHandler)
			problems.GET("/:id/template", ProblemTemplateGetHandler)
			problems.GET("", ProblemsGetHandler)
		}
		problemsAuthed := api.Group("/problems", adminAuth.MiddlewareFunc())
		{
			problemsAuthed.PUT("", ProblemCreateOrEditHandler)
			problemsAuthed.DELETE("/:id", ProblemDeleteHandler)
			problemsAuthed.GET("/:id/testcases", ProblemTestCasesGetHandler)
		}

		submissionsAuthed := api.Group("/submissions", userAuth.MiddlewareFunc())
		{
			submissionsAuthed.POST("", SubmissionCreateHandler)
			submissionsAuthed.GET("", SubmissionGetAllHandler)
			submissionsAuthed.GET("/:submissionId", SubmissionGetHandler)
			submissionsAuthed.GET("/:submissionId/source", SubmissionGetSourceHandler)
		}
	}
	router.Run(":8080")
}
