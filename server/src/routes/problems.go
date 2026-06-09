package routes

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"

	"oj/server/database"
	"oj/server/sandbox"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// maxUploadSize is the maximum allowed size for uploaded zip files (50 MB).
const maxUploadSize = 50 << 20

type ProblemCreateRequest struct {
	Title       string                `form:"title" binding:"required"`
	Description string                `form:"description" binding:"required"`
	File        *multipart.FileHeader `form:"file" binding:"required"`
}

func ProblemCreateOrEditHandler(ctx *gin.Context) {
	var body ProblemCreateRequest
	if err := ctx.ShouldBind(&body); err != nil {
		ctx.String(http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if body.File.Size > maxUploadSize {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("file too large: %d bytes (max %d)", body.File.Size, maxUploadSize))
		return
	}

	problem, isNew, err := database.CreateOrEditProblem(body.Title, body.Description)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create problem")
		return
	}

	problemId := problem.Id.String()
	err = sandbox.SaveProblemFile(problemId, isNew, body.File, func() error { return database.DeleteProblem(problemId) })
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id": problemId,
	})
}

func ProblemDeleteHandler(ctx *gin.Context) {
	id := ctx.Param("id")

	_, err := database.GetProblemById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, "problem not found")
		} else {
			ctx.String(http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}

	if err = sandbox.DeleteProblemFile(id, func() error { return database.DeleteProblem(id) }); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.String(http.StatusOK, "")
}

func ProblemGetHandler(ctx *gin.Context) {
	id := ctx.Param("id")
	problem, err := database.GetProblemById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, "problem not found")
		} else {
			ctx.String(http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"id":          problem.Id,
		"title":       problem.Title,
		"description": problem.Description,
	})
}

func ProblemsGetHandler(ctx *gin.Context) {
	problems, err := database.GetProblems()
	if err != nil {
		ctx.String(http.StatusInternalServerError, "cannot fetch problems: "+err.Error())
		return
	}

	type problemResponse struct {
		Id    string `json:"id"`
		Title string `json:"title"`
	}

	var problemsList []problemResponse
	for _, problem := range problems {
		problemsList = append(problemsList, problemResponse{
			Id:    problem.Id.String(),
			Title: problem.Title,
		})
	}
	ctx.JSON(http.StatusOK, problemsList)
}

func ProblemTemplateGetHandler(ctx *gin.Context) {
	id := ctx.Param("id")

	problem, err := database.GetProblemById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, "problem not found")
		} else {
			ctx.String(http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}

	zipPath := sandbox.GetProblemFilePath(id, "template.zip")

	if _, err := os.Stat(zipPath); err != nil {
		if os.IsNotExist(err) {
			ctx.String(http.StatusNotFound, "template zip not found")
		} else {
			ctx.String(http.StatusInternalServerError, "failed to check template zip: "+err.Error())
		}
		return
	}

	ctx.FileAttachment(zipPath, fmt.Sprintf("%s_template.zip", problem.Title))
}

func ProblemTestCasesGetHandler(ctx *gin.Context) {
	id := ctx.Param("id")

	problem, err := database.GetProblemById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, "problem not found")
		} else {
			ctx.String(http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}

	zipPath := sandbox.GetProblemFilePath(id, "problem.zip")

	if _, err := os.Stat(zipPath); err != nil {
		if os.IsNotExist(err) {
			ctx.String(http.StatusNotFound, "problem zip not found")
		} else {
			ctx.String(http.StatusInternalServerError, "failed to check problem zip: "+err.Error())
		}
		return
	}

	ctx.FileAttachment(zipPath, fmt.Sprintf("%s.zip", problem.Title))
}
