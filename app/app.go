package main

import (
	"fmt"

	"github.com/mnuddindev/devpulse/config"
)

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error while loading config: %v\n", err)
		return
	}
	fmt.Println(config.ServerConfig.App)
}
