package main

import (
	"oj/server/database"
	"oj/server/routes"
	"oj/server/sandbox"
	"oj/server/utility"
)

func main() {
	utility.InitEnv()
	database.Init()
	sandbox.Init()
	routes.Init()
}
