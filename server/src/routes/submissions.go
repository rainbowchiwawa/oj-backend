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

// @Summary Create a submission
// @Description Submit code for a problem
// @Tags submissions
// @Security Bearer
// @Accept multipart/form-data
// @Produce json
// @Param problem_id formData string true "Problem ID"
// @Param file formData file true "Submission zip file"
// @Success 201 {object} map[string]interface{} "id of the submission"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /submissions [post]
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

// @Summary List submissions by user
// @Description Get all submissions of a specific user
// @Tags submissions
// @Security Bearer
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {array} map[string]interface{}
// @Failure 500 {string} string "Internal Server Error"
// @Router /submissions/user/{id} [get]
func SubmissionGetAllByUserIdHandler(ctx *gin.Context) {
	userId := ctx.Param("id")

	submissions, err := database.GetAllSubmissionByUserId(userId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "database error")
		return
	}

	ctx.JSON(http.StatusOK, submissions)
}

// @Summary List user submissions
// @Description Get all submissions of the currently logged in user
// @Tags submissions
// @Security Bearer
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 500 {string} string "Internal Server Error"
// @Router /submissions [get]
func SubmissionGetAllHandler(ctx *gin.Context) {
	user := GetUser(ctx)

	submissions, err := database.GetAllSubmissionByUserId(user.UserId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "")
		return
	}

	ctx.JSON(http.StatusOK, submissions)
}

// @Summary Get a submission
// @Description Get a specific submission by its ID
// @Tags submissions
// @Security Bearer
// @Produce json
// @Param submissionId path string true "Submission ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not Found"
// @Router /submissions/{submissionId} [get]
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

// @Summary Rerun a submission
// @Description Rerun a specific submission by its ID
// @Tags submissions
// @Security Bearer
// @Produce plain
// @Param submissionId path string true "Submission ID"
// @Success 200 {string} string "OK"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not Found"
// @Failure 409 {string} string "Conflict"
// @Failure 500 {string} string "Internal Server Error"
// @Failure 503 {string} string "Service Unavailable"
// @Router /submissions/{submissionId} [post]
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

// @Summary Get submission source
// @Description Get the source zip file of a submission
// @Tags submissions
// @Security Bearer
// @Produce application/zip
// @Param submissionId path string true "Submission ID"
// @Success 200 {file} file
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not Found"
// @Router /submissions/{submissionId}/source [get]
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
