package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type Config struct {
	Host   string
	Port   string
	User   string
	Pass   string
	Schema string
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
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println(err)
		fmt.Println("fuck you postgres!")
		os.Exit(-1)
	}

	db.AutoMigrate(&UserInfo{}, &Problem{}, &Submission{})
	fmt.Println("db: Im done")
}
