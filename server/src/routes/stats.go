package routes

import (
	"net/http"
	"oj/server/database"

	"github.com/gin-gonic/gin"
)

func StatsGetProblemHandler(ctx *gin.Context) {
	problemId := ctx.Param("problemId")

	statistic, err := database.GetStatisticByProblemId(problemId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "")
		return
	}

	ctx.JSON(http.StatusOK, statistic)
}

func StatsGetUserHandler(ctx *gin.Context) {
	userId := ctx.Param("userId")

	statistic, err := database.GetStatisticByUserId(userId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "")
		return
	}

	ctx.JSON(http.StatusOK, statistic)
}
