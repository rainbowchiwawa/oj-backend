package routes

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"

	"oj/server/database"
	"oj/server/parser"
	"oj/server/sandbox/resources"

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

// @Summary Create or edit a problem
// @Description Create a new problem or edit an existing one
// @Tags problems
// @Security Bearer
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "Problem Title"
// @Param description formData string true "Problem Description"
// @Param file formData file true "Problem Zip File"
// @Success 200 {object} map[string]interface{}
// @Success 201 {object} map[string]interface{}
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /problems [put]
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
	err = resources.SaveUploadedProblem(
		problemId,
		isNew,
		body.File,
		func(r *parser.TestResults, s *parser.ProblemSettings) error {
			_, err := database.UpdateProblem(problemId, r, s)
			return err
		},
		func() error { return database.DeleteProblem(problemId) },
	)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	var code int
	if isNew {
		code = http.StatusCreated
	} else {
		code = http.StatusOK
	}

	ctx.JSON(code, gin.H{"id": problemId})
}

// @Summary Delete a problem
// @Description Delete a specific problem by ID
// @Tags problems
// @Security Bearer
// @Produce plain
// @Param id path string true "Problem ID"
// @Success 200 {string} string "OK"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /problems/{id} [delete]
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

	if err = resources.DeleteUploadedProblem(id, func() error { return database.DeleteProblem(id) }); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.String(http.StatusOK, "")
}

// @Summary Get a problem
// @Description Get problem details by ID
// @Tags problems
// @Produce json
// @Param id path string true "Problem ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {string} string "Not Found"
// @Router /problems/{id} [get]
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

// @Summary List problems
// @Description Get all problems
// @Tags problems
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 500 {string} string "Internal Server Error"
// @Router /problems [get]
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

// @Summary Get problem template
// @Description Download problem template zip file
// @Tags problems
// @Produce application/zip
// @Param id path string true "Problem ID"
// @Success 200 {file} file
// @Failure 404 {string} string "Not Found"
// @Router /problems/{id}/template [get]
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

	zipPath, ok, err := resources.GetProblemFilePath(id, resources.ProblemTemplateZip)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "failed to check template zip: "+err.Error())
		return
	}

	if !ok {
		ctx.String(http.StatusNotFound, "template zip not found")
		return
	}

	ctx.FileAttachment(zipPath, fmt.Sprintf("%s_template.zip", problem.Title))
}

// @Summary Get problem test cases
// @Description Download problem test cases zip file
// @Tags problems
// @Security Bearer
// @Produce application/zip
// @Param id path string true "Problem ID"
// @Success 200 {file} file
// @Failure 404 {string} string "Not Found"
// @Router /problems/{id}/testcases [get]
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

	zipPath, ok, err := resources.GetProblemFilePath(id, resources.ProblemZip)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "failed to check template zip: "+err.Error())
		return
	}

	if !ok {
		ctx.String(http.StatusNotFound, "template zip not found")
		return
	}

	ctx.FileAttachment(zipPath, fmt.Sprintf("%s.zip", problem.Title))
}
