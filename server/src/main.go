package main

import (
	"oj/server/database"
	"oj/server/routes"
	"oj/server/utility"
)

func main() {
	utility.InitEnv()
	database.Init()
	routes.Init()
}
