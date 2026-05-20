package database

import "github.com/google/uuid"

type Problem struct {
	Id          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Description string    `gorm:"not null"`
}
