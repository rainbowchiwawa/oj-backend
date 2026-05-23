package routes

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"oj/server/database"
	"oj/server/sandbox"
	"oj/server/utility"

	"github.com/gin-gonic/gin"
)

type SubmissionCreateRequest struct {
	ProblemId string                `form:"problem_id"`
	File      *multipart.FileHeader `form:"file"`
}

func SubmissionCreateHandler(ctx *gin.Context) {
	user := GetUser(ctx)

	var body SubmissionCreateRequest
	if err := ctx.Bind(&body); err != nil {
		return
	}

	problem, err := database.GetProblemById(body.ProblemId)
	if err != nil {
		ctx.String(http.StatusNotFound, "problem not found")
		return
	}

	submission, err := database.CreateSubmission(problem.Id.String(), user.UserId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "database error")
		return
	}

	path := sandbox.GetSubmissionPath(submission.Id.String()) + "/source.zip"
	if err := utility.WriteFileToPath(body.File, path); err != nil {
		fmt.Println(err)
		ctx.String(http.StatusInternalServerError, "failed to write file")
		return
	}

	go func() {
		err := sandbox.CreateWorker(&submission)
		if err != nil {
			return
		}
		database.UpdateSubmission(&submission)
	}()

	ctx.JSON(http.StatusCreated, submission.Id.String())
}

func SubmissionGetAllHandler(ctx *gin.Context) {
	user := GetUser(ctx)

	submissions, err := database.GetAllSubmissionByUserId(user.UserId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "")
		return
	}

	ctx.JSON(http.StatusOK, submissions)
}

func SubmissionGetHandler(ctx *gin.Context) {
	user := GetUser(ctx)
	submissionId := ctx.Param("submissionId")

	submission, err := database.GetSubmissionById(submissionId)
	if err != nil {
		ctx.String(http.StatusNotFound, "")
		return
	}

	if submission.UserId.String() != user.UserId {
		ctx.String(http.StatusForbidden, "")
		return
	}

	ctx.JSON(http.StatusOK, submission)
}

func SubmissionGetSourceHandler(ctx *gin.Context) {
	submissionId := ctx.Param("submissionId")

	submission, err := database.GetSubmissionById(submissionId)
	if err != nil {
		ctx.String(http.StatusNotFound, "")
		return
	}

	path := sandbox.GetSubmissionPath(submission.Id.String()) + "/source.zip"
	ctx.FileAttachment(path, "source.zip")
}
