package utility

import (
	"fmt"
	"os"
)

type Env struct {
	DatabaseHost       string
	DatabasePort       string
	DatabaseUser       string
	DatabasePassword   string
	DatabaseSchema     string
	JWTSecret          string
	BasePath           string
	BindBasePath       string
	ContainerPath      string
	ProblemBasePath    string
	SubmissionBasePath string
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

	var basePath, bindBasePath, containerPath, problemBasePath, submissionBasePath string
	if inDocker {
		basePath = "/app/data"
		bindBasePath = "server_data"
		containerPath = "/app/containers"
		problemBasePath = "/app/data/problems"
		submissionBasePath = "/app/data/submissions"
	} else {
		basePath = "../.."
		bindBasePath = "../.."
		containerPath = "../containers"
		problemBasePath = "../../problems"
		submissionBasePath = "../../submissions"
	}

	EnvData = Env{
		DatabaseHost:       lookupEnv("DB_HOST"),
		DatabasePort:       lookupEnv("DB_PORT"),
		DatabaseUser:       lookupEnv("DB_USER"),
		DatabasePassword:   lookupEnv("DB_PASS"),
		DatabaseSchema:     lookupEnv("DB_SCHEMA"),
		JWTSecret:          lookupEnv("JWT_SECRET"),
		BasePath:           basePath,
		BindBasePath:       bindBasePath,
		ContainerPath:      containerPath,
		ProblemBasePath:    problemBasePath,
		SubmissionBasePath: submissionBasePath,
	}
}
