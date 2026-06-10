package routes

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"oj/server/database"
	"oj/server/sandbox"
	"oj/server/sandbox/resources"

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

	submissionId := submission.Id.String()
	err = resources.SaveUploadedSubmission(submissionId, body.File, func() error { return database.DeleteSubmission(submissionId) })
	if err != nil {
		ctx.String(http.StatusInternalServerError, "failed to write file")
		return
	}

	go func() {
		score, status, output, err := sandbox.CreateWorker(submissionId, body.ProblemId, problem.Settings, problem.Answer)
		if err != nil {
			fmt.Println(score, status, output, err)
			return
		}
		database.UpdateSubmissionByWorkerOutput(submissionId, score, status, output)
	}()

	ctx.JSON(http.StatusCreated, gin.H{"id": submissionId})
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

	_, err := database.GetSubmissionById(submissionId)
	if err != nil {
		ctx.String(http.StatusNotFound, "")
		return
	}

	submissionManager := resources.SubmissionManager{Id: submissionId}
	path := submissionManager.GetChildPath(resources.SubmissionZip)
	ctx.FileAttachment(path, "source.zip")
}
