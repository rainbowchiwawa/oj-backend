package database

import (
	"oj/server/sandbox"
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
	Id         uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProblemId  uuid.UUID        `gorm:"type:uuid;not null" json:"problem_id"`
	UserId     uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
	Status     SubmissionStatus `gorm:"type:varchar(16);default:'pending';index;not null" json:"status"`
	ConfigLog  *string          `gorm:"type:text"`
	CompileLog *string          `gorm:"type:text"`
	OutputLog  *string          `gorm:"type:text"`
	CreatedAt  time.Time        `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"not null;default:now()" json:"updated_at"`
}

type SubmissionAggration struct {
	Status SubmissionStatus
	Count  int64
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

func UpdateSubmissionByWorkerOutput(id string, output *sandbox.WorkerOutput) (Submission, error) {
	var newSubmission Submission
	if output.Compiler == nil {
		newSubmission = Submission{
			ConfigLog:  nil,
			CompileLog: nil,
			OutputLog:  nil,
			Status:     StatusPending,
		}
	} else if output.Runner == nil {
		newSubmission = Submission{
			ConfigLog:  output.Compiler.ConfigLog,
			CompileLog: output.Compiler.CompileLog,
			OutputLog:  nil,
		}
		if output.Compiler.CompileLog == nil {
			newSubmission.Status = StatusSE
		} else {
			newSubmission.Status = StatusCE
		}
	} else {
		newSubmission = Submission{
			ConfigLog:  output.Compiler.ConfigLog,
			CompileLog: output.Compiler.CompileLog,
			OutputLog:  output.Runner.OutputLog,
		}
		if output.Runner.ExitCode != 0 {
			newSubmission.Status = StatusRE
		} else {
			newSubmission.Status = StatusWA
		}
	}
	if res := db.Table("submissions").Where("id = ?", id).Updates(&newSubmission); res.Error != nil {
		return Submission{}, res.Error
	}
	return newSubmission, nil
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
	if res := db.Table("submissions").Select("Id", "ProblemId", "UserId", "Status").Where("user_id = ?", userId).Find(&submissions).Order("created_at desc"); res.Error != nil {
		return nil, res.Error
	}
	return submissions, nil
}

func GetStatisticByProblemId(problemId string) (map[string]int64, error) {
	var submissions []SubmissionAggration
	if res := db.Table("submissions").Select("Status, COUNT(*) as Count").Where("problemId = ?", problemId).Where("status <> ?", StatusPending).Group("status").Find(&submissions); res.Error != nil {
		return nil, res.Error
	}
	statistic := make(map[string]int64)
	for _, submission := range submissions {
		statistic[submission.Status] = submission.Count
	}
	return statistic, nil
}

func GetStatisticByUserId(userId string) (map[string]int64, error) {
	var submissions []SubmissionAggration
	if res := db.Table("submissions").Select("Status, COUNT(*) as Count").Where("userId = ?", userId).Where("status <> ?", StatusPending).Group("status").Find(&submissions); res.Error != nil {
		return nil, res.Error
	}
	statistic := make(map[string]int64)
	for _, submission := range submissions {
		statistic[submission.Status] = submission.Count
	}
	return statistic, nil
}
