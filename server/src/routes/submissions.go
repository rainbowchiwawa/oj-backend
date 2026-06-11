package routes

import (
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

	sandbox.PushJob(sandbox.WorkerInput{
		ProblemId:    problem.Id.String(),
		SubmissionId: submissionId,
		Settings:     problem.Settings,
		Answer:       problem.Answer,
	})
	ctx.JSON(http.StatusCreated, gin.H{"id": submissionId})
}

func SubmissionGetAllByUserIdHandler(ctx *gin.Context) {
	userId := ctx.Param("id")

	submissions, err := database.GetAllSubmissionByUserId(userId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "database error")
		return
	}

	ctx.JSON(http.StatusOK, submissions)
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

func SubmissionRerunHandler(ctx *gin.Context) {
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

	if submission.Status == sandbox.StatusPending {
		ctx.String(http.StatusConflict, "the test is still running")
		return
	}

	problem, err := database.GetProblemById(submission.ProblemId.String())
	if err != nil {
		ctx.String(http.StatusNotFound, "")
		return
	}

	sandbox.PushJob(sandbox.WorkerInput{
		ProblemId:    problem.Id.String(),
		SubmissionId: submissionId,
		Settings:     problem.Settings,
		Answer:       problem.Answer,
	})
	ctx.String(http.StatusOK, "")
}

func SubmissionGetSourceHandler(ctx *gin.Context) {
	user := GetUser(ctx)
	submissionId := ctx.Param("submissionId")

	submission, err := database.GetSubmissionById(submissionId)
	if err != nil {
		ctx.String(http.StatusNotFound, "")
		return
	}

	if submission.UserId.String() != user.UserId && user.UserType != database.TypeAdmin {
		ctx.String(http.StatusForbidden, "Permission denied")
		return
	}

	submissionManager := resources.SubmissionManager{Id: submissionId}
	path := submissionManager.GetZipPath()
	ctx.FileAttachment(path, "source.zip")
}
