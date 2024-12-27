package main

import (
	"fmt"

	"github.com/Preloading/MastodonTwitterAPI/config"
	"github.com/Preloading/MastodonTwitterAPI/db_controller"
	"github.com/Preloading/MastodonTwitterAPI/twitterv1"
)

var (
	configData *config.Config
)

func main() {
	var err error
	configData, err = config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	db_controller.InitDB(*configData)
	twitterv1.InitServer(configData)
}
