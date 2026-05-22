package utility

import (
	"fmt"
	"os"
)

type Env struct {
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSchema   string
	JWTSecret        string
}

var EnvData Env

func lookupEnv(key string) string {
	value, exist := os.LookupEnv(key)
	if !exist {
		fmt.Println("env key ${" + key + "} is unset, exit")
		os.Exit(-1)
	}
	return value
}

func InitEnv() {
	EnvData = Env{
		DatabaseHost:     lookupEnv("DB_HOST"),
		DatabasePort:     lookupEnv("DB_PORT"),
		DatabaseUser:     lookupEnv("DB_USER"),
		DatabasePassword: lookupEnv("DB_PASS"),
		DatabaseSchema:   lookupEnv("DB_SCHEMA"),
		JWTSecret:        lookupEnv("JWT_SECRET"),
	}
}
