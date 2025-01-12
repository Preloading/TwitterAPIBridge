package main

import (
	"fmt"

	"github.com/Preloading/TwitterAPIBridge/config"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/Preloading/TwitterAPIBridge/twitterv1"
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
