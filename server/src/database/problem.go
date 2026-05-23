package database

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Problem struct {
	Id          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title       string    `gorm:"not null;uniqueIndex" json:"title"`
	Description string    `gorm:"not null" json:"description"`
	Answer      string    `gorm:"not null"`
}

func CreateOrEditProblem(title string, description string) (Problem, bool, error) {
	problem, err := GetProblemByTitle(title)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return Problem{}, false, err
	}

	if problem.Id != uuid.Nil {
		if res := db.Table("problems").Where("id = ?", problem.Id).Update("description", description); res.Error != nil {
			return Problem{}, false, res.Error
		}
		return problem, false, nil
	}

	newProblem := Problem{Title: title, Description: description}
	if res := db.Table("problems").Create(&newProblem); res.Error != nil {
		return Problem{}, false, res.Error
	}
	return newProblem, true, nil
}

func DeleteProblem(id string) error {
	if res := db.Table("problems").Where("id = ?", id).Delete(&Problem{}); res.Error != nil {
		return res.Error
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
