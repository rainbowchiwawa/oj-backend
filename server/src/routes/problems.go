package routes

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"oj/server/database"
	"oj/server/utility"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProblemCreateRequest struct {
	Title       string                `form:"title" binding:"required"`
	Description string                `form:"description" binding:"required"`
	File        *multipart.FileHeader `form:"file" binding:"required"`
}

func ProblemCreateHandler(ctx *gin.Context) {
	var body ProblemCreateRequest
	if err := ctx.ShouldBind(&body); err != nil {
		ctx.String(http.StatusBadRequest, "invalid request: " + err.Error())
		return
	}

	file, err := body.File.Open()
	if err != nil {
		ctx.String(http.StatusBadRequest, "cannot open file")
		return
	}
	defer file.Close()
	
	problemId, err := database.CreateOrEditProblem(body.Title, body.Description)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create problem")
		return
	}

	if err := utility.CreateProblemDirectory(problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create problem directory")
		return
	}
	
	if err := utility.ExtractProblemFile(file, body.File.Size, problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot extract problem file")
		return
	}

	if err := utility.SaveProblemZip(file, problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot save problem zip")
		return
	}

	if err := utility.CreateProblemTemplateZip(problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create template zip")
		return
	}

	ctx.String(http.StatusOK, "")
}

func ProblemDeleteHandler(ctx *gin.Context) {
	id := ctx.Param("id")

	// 1. Check if the problem exists
	_, err := database.GetProblemById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, "problem not found")
		} else {
			ctx.String(http.StatusInternalServerError, "database error: "+err.Error())
		}
		return
	}
	
	

	// 2. Delete database record
	if err := database.DeleteProblem(id); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot delete problem from database: "+err.Error())
		return
	}
	
	// 3. Delete directory
	if err := utility.DeleteProblemDirectory(id); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot delete problem directory")
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
	ctx.JSON(http.StatusOK, problems)
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

	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems"
	}
	zipPath := filepath.Join(problemsDir, id, "template.zip")

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

	problemsDir := os.Getenv("PROBLEMS_DIR")
	if problemsDir == "" {
		problemsDir = "../../problems"
	}
	zipPath := filepath.Join(problemsDir, id, "problem.zip")

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
