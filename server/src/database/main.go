package database

import (
	"fmt"
	"oj/server/utility"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		utility.EnvData.DatabaseHost,
		utility.EnvData.DatabaseUser,
		utility.EnvData.DatabasePassword,
		utility.EnvData.DatabaseSchema,
		utility.EnvData.DatabasePort,
	)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	err = db.AutoMigrate(&UserInfo{}, &Problem{}, &Submission{}, &InvalidToken{})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	fmt.Println("db ok")
}
