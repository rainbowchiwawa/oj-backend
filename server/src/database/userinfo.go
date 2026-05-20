package database

import (
	"github.com/google/uuid"
)

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

func CreateUser(name string, passwordHash string) error {
	if res := db.Table("user_infos").Create(&UserInfo{Name: name, PasswordHash: passwordHash}); res.Error != nil {
		return res.Error
	}
	return nil
}

func GetUserByName(name string) (UserInfo, error) {
	var user UserInfo
	if res := db.Table("user_infos").Where(&UserInfo{Name: name}).Take(&user); res.Error != nil {
		return UserInfo{}, res.Error
	}
	return user, nil
}
