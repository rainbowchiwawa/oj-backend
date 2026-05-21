package database

import (
	"github.com/google/uuid"
)

type Problem struct {
	Id          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string    `gorm:"not null;uniqueIndex"`
	Description string    `gorm:"not null"`
}

func CreateOrEditProblem(title string, description string) (string, error) {
	problem, _ := GetProblemByTitle(title)
	if problem.Id != uuid.Nil {
		if err := db.Table("problems").Where("id = ?", problem.Id).Update("description", description).Error; err != nil {
			return "", err
		}
		return problem.Id.String(), nil
	}

	newProblem := Problem{Title: title, Description: description}
	if err := db.Table("problems").Create(&newProblem).Error; err != nil {
		return "", err
	}
	return newProblem.Id.String(), nil
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

