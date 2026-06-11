package database

import (
	"oj/server/sandbox"
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	Id        uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProblemId uuid.UUID           `gorm:"type:uuid;not null" json:"problem_id"`
	UserId    uuid.UUID           `gorm:"type:uuid;not null" json:"user_id"`
	Score     int                 `gorm:"not null;default:0" json:"score"`
	Status    sandbox.TestStatus  `gorm:"type:varchar(16);default:'pending';index;not null" json:"status"`
	Result    *sandbox.WorkerLogs `gorm:"type:jsonb" json:"result"`
	CreatedAt time.Time           `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time           `gorm:"not null;default:now()" json:"updated_at"`
}

type SubmissionAggration struct {
	Status string
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

func UpdateSubmissionWithWorkerOutput(id string, score int, status sandbox.TestStatus, output *sandbox.WorkerLogs) (Submission, error) {
	newSubmission := Submission{Score: score, Status: status, Result: output}
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
	if res := db.Table("submissions").Select("Status, COUNT(*) as Count").Where("problemId = ?", problemId).Where("status <> ?", sandbox.StatusPending).Group("status").Find(&submissions); res.Error != nil {
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
	if res := db.Table("submissions").Select("Status, COUNT(*) as Count").Where("userId = ?", userId).Where("status <> ?", sandbox.StatusPending).Group("status").Find(&submissions); res.Error != nil {
		return nil, res.Error
	}
	statistic := make(map[string]int64)
	for _, submission := range submissions {
		statistic[submission.Status] = submission.Count
	}
	return statistic, nil
}

func DeleteSubmission(id string) error {
	if res := db.Table("submissions").Where("id = ?", id).Delete(&Submission{}); res.Error != nil {
		return res.Error
	}
	return nil
}
