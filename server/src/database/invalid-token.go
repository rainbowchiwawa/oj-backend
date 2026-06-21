package database

import (
	"time"

	"github.com/google/uuid"
)

type InvalidToken struct {
	Id        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ExpiredAt time.Time `gorm:"not null" json:"expired_at"`
}

func IsTokenValid(_id string) bool {
	id, err := uuid.Parse(_id)
	if err != nil {
		return false
	}

	var invalidToken InvalidToken
	if res := db.Table("invalid_tokens").Where(&InvalidToken{Id: id}).Take(&invalidToken); res.Error != nil {
		return true
	}
	return false
}

func CreateInvalidToken(_id string, expiredAt float64) error {
	id, err := uuid.Parse(_id)
	if err != nil {
		return err
	}

	res := db.Table("invalid_tokens").Create(&InvalidToken{Id: id, ExpiredAt: time.Unix(int64(expiredAt), 0)})
	return res.Error
}

func ClearExpiredToken() error {
	res := db.Table("invalid_tokens").Where("expired_at < ?", time.Now()).Delete(&InvalidToken{})
	return res.Error
}
