package main

import (
	"github.com/Preloading/MastodonTwitterAPI/db_controller"
	"github.com/Preloading/MastodonTwitterAPI/twitterv1"
)

func main() {
	db_controller.InitDB()
	twitterv1.InitServer()
}
