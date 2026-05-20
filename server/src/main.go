package main

import (
	"oj/server/database"
	"oj/server/routes"
)

func main() {
	database.Init()
	routes.Init()
}
