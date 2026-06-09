package routes

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"oj/server/database"
	"oj/server/utility"

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

	data, err := utility.ToFileData(body.File)
	if err != nil {
		ctx.String(http.StatusBadRequest, "cannot open file")
		return
	}

	problem, isNew, err := database.CreateOrEditProblem(body.Title, body.Description)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create problem")
		return
	}

	problemId := problem.Id.String()
	success := false
	defer func() {
		if success {
			if !isNew {
				utility.CleanupProblemBackup(problemId)
			}
			return
		}
		// Rollback: remove partially-created new directory.
		if cleanErr := utility.DeleteProblemDirectory(problemId); cleanErr != nil {
			log.Printf("rollback: failed to delete problem directory %s: %v", problemId, cleanErr)
		}
		if isNew {
			// New problem — remove the DB record too.
			if cleanErr := database.DeleteProblem(problemId); cleanErr != nil {
				log.Printf("rollback: failed to delete problem record %s: %v", problemId, cleanErr)
			}
		} else {
			// Edit — restore old directory from backup.
			if cleanErr := utility.RestoreProblemDirectory(problemId); cleanErr != nil {
				log.Printf("rollback: failed to restore old problem directory %s: %v", problemId, cleanErr)
			}
		}
	}()

	if !isNew {
		if err := utility.BackupProblemDirectory(problemId); err != nil {
			ctx.String(http.StatusInternalServerError, "cannot backup old problem directory")
			return
		}
	}

	if err := utility.CreateProblemDirectory(problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create problem directory")
		return
	}

	if err := utility.ExtractProblemFile(body.File, problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot extract problem file: "+err.Error())
		return
	}

	if err := utility.SaveProblemZip(data, problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot save problem zip")
		return
	}

	if err := utility.CreateProblemTemplateZip(problemId); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot create template zip")
		return
	}

	success = true
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

	// Backup directory first so we can restore if DB delete fails.
	if err := utility.BackupProblemDirectory(id); err != nil {
		ctx.String(http.StatusInternalServerError, "cannot backup problem directory: "+err.Error())
		return
	}

	if err := database.DeleteProblem(id); err != nil {
		// DB delete failed — restore directory from backup.
		if restoreErr := utility.RestoreProblemDirectory(id); restoreErr != nil {
			log.Printf("rollback: failed to restore problem directory %s: %v", id, restoreErr)
		}
		ctx.String(http.StatusInternalServerError, "cannot delete problem from database: "+err.Error())
		return
	}

	// Both succeeded — remove the backup.
	utility.CleanupProblemBackup(id)
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

	zipPath := utility.GetProblemFilePath(id, "template.zip")

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

	zipPath := utility.GetProblemFilePath(id, "problem.zip")

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
