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
	BasePath         string
	BindBasePath     string
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

	inDocker := os.Getenv("RUN_ENV") == "docker"

	var basePath string
	var bindBasePath string
	if inDocker {
		basePath = "/app/data"
		bindBasePath = "server_data"
	} else {
		basePath = "../.."
		bindBasePath = "../.."
	}

	EnvData = Env{
		DatabaseHost:     lookupEnv("DB_HOST"),
		DatabasePort:     lookupEnv("DB_PORT"),
		DatabaseUser:     lookupEnv("DB_USER"),
		DatabasePassword: lookupEnv("DB_PASS"),
		DatabaseSchema:   lookupEnv("DB_SCHEMA"),
		JWTSecret:        lookupEnv("JWT_SECRET"),
		BasePath:         basePath,
		BindBasePath:     bindBasePath,
	}
}
