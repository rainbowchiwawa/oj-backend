package main

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/google/uuid"
)

var db *gorm.DB

type Config struct {
	Host   string
	Port   string
	User   string
	Pass   string
	Schema string
}

type UserInfo struct {
	Id           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name         string    `gorm:"not null"`
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

func initDB() {
	config := Config{
		Host:   os.Getenv("DB_HOST"),
		Port:   os.Getenv("DB_PORT"),
		User:   os.Getenv("DB_USER"),
		Pass:   os.Getenv("DB_PASS"),
		Schema: os.Getenv("DB_SCHEMA"),
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", config.Host, config.User, config.Pass, config.Schema, config.Port)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println(err)
		fmt.Println("fuck you postgres!")
		os.Exit(-1)
	}

	db.AutoMigrate(&UserInfo{}, &Problem{}, &Submission{})
	fmt.Println("db: Im done")
}
