package routes

import (
	"mime/multipart"
	"net/http"
	"oj/server/database"
	"oj/server/sandbox"
	"oj/server/sandbox/resources"
	"os"

	"github.com/gin-gonic/gin"
)

type SubmissionCreateRequest struct {
	ProblemId string                `form:"problem_id" binding:"required"`
	File      *multipart.FileHeader `form:"file" binding:"required"`
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
	err = resources.SaveUploadedSubmission(submissionId, body.File, func() error {
		database.UpdateSubmissionWithWorkerOutput(submissionId, 0, sandbox.StatusError, &sandbox.WorkerLogs{})
		submissionManager := resources.SubmissionManager{Id: submissionId}
		submissionManager.ClearFiles()
		os.Remove(submissionManager.GetZipPath())
		return nil
	})
	if err != nil {
		ctx.String(http.StatusInternalServerError, "failed to write file")
		return
	}

	err = sandbox.PushJob(sandbox.WorkerInput{
		ProblemId:    problem.Id.String(),
		SubmissionId: submissionId,
		Settings:     problem.Settings,
		Answer:       problem.Answer,
	})
	if err != nil {
		database.UpdateSubmissionWithWorkerOutput(submissionId, 0, sandbox.StatusError, &sandbox.WorkerLogs{})
		submissionManager := resources.SubmissionManager{Id: submissionId}
		submissionManager.ClearFiles()
		os.Remove(submissionManager.GetZipPath())
		ctx.String(http.StatusServiceUnavailable, "Job queue is full")
		return
	}
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

	submission, err = database.ResetSubmission(submissionId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "failed to reset submission")
		return
	}

	err = sandbox.PushJob(sandbox.WorkerInput{
		ProblemId:    problem.Id.String(),
		SubmissionId: submissionId,
		Settings:     problem.Settings,
		Answer:       problem.Answer,
	})
	if err != nil {
		database.UpdateSubmissionWithWorkerOutput(submissionId, 0, sandbox.StatusError, &sandbox.WorkerLogs{})
		ctx.String(http.StatusServiceUnavailable, "Job queue is full")
		return
	}
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
