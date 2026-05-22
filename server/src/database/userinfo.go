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
	Id           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Type         UserType  `gorm:"not null" json:"type"`
	Name         string    `gorm:"not null;unique" json:"name"`
	PasswordHash string    `gorm:"not null" json:"password_hash"`
}

func IsUserValid(id string, target UserType) (bool, error) {
	user, err := GetUserById(id)
	if err != nil {
		return false, err
	}
	if user.Type == target {
		return true, nil
	}
	if target == TypeUser && user.Type == TypeAdmin {
		return true, nil
	}
	return false, nil
}

func CreateUser(userType UserType, name string, passwordHash string) error {
	res := db.Table("user_infos").Create(&UserInfo{Type: userType, Name: name, PasswordHash: passwordHash})
	return res.Error
}

func GetUserById(id string) (UserInfo, error) {
	var user UserInfo
	if res := db.Table("user_infos").Where("id = ?", id).Take(&user); res.Error != nil {
		return UserInfo{}, res.Error
	}
	return user, nil
}

func GetUserByName(name string) (UserInfo, error) {
	var user UserInfo
	if res := db.Table("user_infos").Where("name = ?", name).Take(&user); res.Error != nil {
		return UserInfo{}, res.Error
	}
	return user, nil
}
