package main

import (
	"oj/server/database"
	"oj/server/routes"
	"oj/server/utility"
)

// @title OJ Backend API
// @version 1.0
// @description This is the backend API for the Online Judge.
// @host localhost:8080
// @BasePath /api
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	utility.InitEnv()
	database.Init()
	routes.Init()
}
