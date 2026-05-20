package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/google/uuid"
)

var DB *gorm.DB

type Config struct {
	Host   string
	Port   string
	User   string
	Pass   string
	Schema string
}

type UserType = string

const (
	TypeUser  UserType = "user"
	TypeAdmin UserType = "admin"
)

type UserInfo struct {
	Id           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Type         UserType  `gorm:"not null"`
	Name         string    `gorm:"not null;unique"`
	PasswordHash string    `gorm:"not null"`
}

type Problem struct {
	Id          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Description string    `gorm:"not null"`
}

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

func Init() {
	config := Config{
		Host:   os.Getenv("DB_HOST"),
		Port:   os.Getenv("DB_PORT"),
		User:   os.Getenv("DB_USER"),
		Pass:   os.Getenv("DB_PASS"),
		Schema: os.Getenv("DB_SCHEMA"),
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", config.Host, config.User, config.Pass, config.Schema, config.Port)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println(err)
		fmt.Println("fuck you postgres!")
		os.Exit(-1)
	}

	DB.AutoMigrate(&UserInfo{}, &Problem{}, &Submission{})
	fmt.Println("db: Im done")
}
