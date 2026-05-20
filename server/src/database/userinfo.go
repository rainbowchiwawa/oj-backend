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

func IsUserValid(id string, target UserType) bool {
	user, err := GetUserById(id)
	if err != nil {
		return false
	}
	if user.Type == target {
		return true
	}
	if target == TypeUser && user.Type == TypeAdmin {
		return true
	}
	return false
}

func CreateUser(userType UserType, name string, passwordHash string) error {
	res := db.Table("user_infos").Create(&UserInfo{Type: userType, Name: name, PasswordHash: passwordHash})
	return res.Error
}

func GetUserById(_id string) (UserInfo, error) {
	id, err := uuid.Parse(_id)
	if err != nil {
		return UserInfo{}, err
	}

	var user UserInfo
	if res := db.Table("user_infos").Where(&UserInfo{Id: id}).Take(&user); res.Error != nil {
		return UserInfo{}, res.Error
	}
	return user, nil
}

func GetUserByName(name string) (UserInfo, error) {
	var user UserInfo
	if res := db.Table("user_infos").Where(&UserInfo{Name: name}).Take(&user); res.Error != nil {
		return UserInfo{}, res.Error
	}
	return user, nil
}
