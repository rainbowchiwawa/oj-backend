package routes

import (
	"net/http"
	"oj/server/database"

	"github.com/gin-gonic/gin"
)

// @Summary Get problem statistics
// @Description Get the statistics for a specific problem
// @Tags stats
// @Produce json
// @Param problemId path string true "Problem ID"
// @Success 200 {object} map[string]int
// @Failure 500 {string} string "Internal Server Error"
// @Router /stats/problems/{problemId} [get]
func StatsGetProblemHandler(ctx *gin.Context) {
	problemId := ctx.Param("problemId")

	statistic, err := database.GetStatisticByProblemId(problemId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "")
		return
	}

	ctx.JSON(http.StatusOK, statistic)
}

// @Summary Get user statistics
// @Description Get the statistics for a specific user
// @Tags stats
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]int
// @Failure 500 {string} string "Internal Server Error"
// @Router /stats/users/{userId} [get]
func StatsGetUserHandler(ctx *gin.Context) {
	userId := ctx.Param("userId")

	statistic, err := database.GetStatisticByUserId(userId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "")
		return
	}

	ctx.JSON(http.StatusOK, statistic)
}
