package database

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Problem struct {
	Id          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string    `gorm:"not null;uniqueIndex"`
	Description string    `gorm:"not null"`
}

// CreateOrEditProblem creates a new problem or updates an existing one by title.
// Returns the problem ID, whether a new record was created, and any error.
func CreateOrEditProblem(title string, description string) (string, bool, error) {
	problem, err := GetProblemByTitle(title)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// Real database error — not just "record not found".
		return "", false, err
	}

	if problem.Id != uuid.Nil {
		// Existing problem — update description.
		if err := db.Table("problems").Where("id = ?", problem.Id).Update("description", description).Error; err != nil {
			return "", false, err
		}
		return problem.Id.String(), false, nil
	}

	// New problem — create.
	newProblem := Problem{Title: title, Description: description}
	if err := db.Table("problems").Create(&newProblem).Error; err != nil {
		return "", false, err
	}
	return newProblem.Id.String(), true, nil
}

func DeleteProblem(id string) error {
	if err := db.Table("problems").Where("id = ?", id).Delete(&Problem{}).Error; err != nil {
		return err
	}
	return nil
}

func GetProblemById(id string) (Problem, error) {
	var problem Problem
	if res := db.Table("problems").Where("id = ?", id).Take(&problem); res.Error != nil {
		return Problem{}, res.Error
	}
	return problem, nil
}

func GetProblemByTitle(title string) (Problem, error) {
	var problem Problem
	if res := db.Table("problems").Where("title = ?", title).Take(&problem); res.Error != nil {
		return Problem{}, res.Error
	}
	return problem, nil
}

func GetProblems() ([]Problem, error) {
	var problems []Problem
	if res := db.Table("problems").Find(&problems); res.Error != nil {
		return nil, res.Error
	}
	return problems, nil
}

