package database

import (
	"time"

	"github.com/google/uuid"
)

type SubmissionStatus = string

const (
	StatusPending SubmissionStatus = "pending"
	StatusAC      SubmissionStatus = "AC"
	StatusWA      SubmissionStatus = "WA"
	StatusCE      SubmissionStatus = "CE"
	StatusSE      SubmissionStatus = "SE"
	StatusRE      SubmissionStatus = "RE"
	StatusTLE     SubmissionStatus = "TLE"
	StatusMLE     SubmissionStatus = "MLE"
)

type Submission struct {
	Id        uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProblemId uuid.UUID        `gorm:"type:uuid;not null"`
	UserId    uuid.UUID        `gorm:"type:uuid;not null"`
	Status    SubmissionStatus `gorm:"type:varchar(16);default:'pending';index;not null"`
	CreatedAt time.Time        `gorm:"not null;default:now()"`
	UpdatedAt time.Time        `gorm:"not null;default:now()"`
}

func CreateSubmission(problemId string, userId string) (Submission, error) {
	_problemId, err := uuid.Parse(problemId)
	if err != nil {
		return Submission{}, err
	}

	_userId, err := uuid.Parse(userId)
	if err != nil {
		return Submission{}, err
	}

	newSubmission := Submission{ProblemId: _problemId, UserId: _userId}
	if res := db.Table("submissions").Create(&newSubmission); res.Error != nil {
		return Submission{}, res.Error
	}
	return newSubmission, nil
}

func UpdateSubmissionStatus(id string, status SubmissionStatus) error {
	return db.Table("submission").Where("id = ?", id).Update("status", status).Update("updated_at", time.Now()).Error
}

func GetSubmissionById(id string) (Submission, error) {
	var submission Submission
	if res := db.Table("submissions").Where("id = ?", id).Take(&submission); res.Error != nil {
		return Submission{}, res.Error
	}
	return submission, nil
}

func GetAllSubmissionByUserId(userId string) ([]Submission, error) {
	var submissions []Submission
	if res := db.Table("submissions").Where("user_id = ?", userId).Find(&submissions).Order("created_at desc"); res.Error != nil {
		return nil, res.Error
	}
	return submissions, nil
}
