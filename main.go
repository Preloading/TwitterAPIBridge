package main

import (
	"fmt"
	_ "net/http/pprof"

	"github.com/Preloading/TwitterAPIBridge/config"
	"github.com/Preloading/TwitterAPIBridge/db_controller"
	"github.com/Preloading/TwitterAPIBridge/twitterv1"
)

var (
	configData *config.Config
)

func main() {
	// Enable pprof for debugging purposes
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	var err error
	configData, err = config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	if configData.SecretKey == "" {
		fmt.Println("The JWT Secret key must be set in config.yaml.")
		return
	} else if len(configData.SecretKey) < 32 {
		fmt.Println("The JWT Secret key must be 32 bytes long")
		return
	}

	db_controller.InitDB(*configData)
	twitterv1.InitServer(configData)
}
