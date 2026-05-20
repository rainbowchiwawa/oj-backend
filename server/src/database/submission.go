package database

import "github.com/google/uuid"

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
	Id     uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Status SubmissionStatus `gorm:"type:varchar(16);default:'pending';index;not null"`
}
