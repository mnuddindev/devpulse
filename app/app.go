package main

import (
	"fmt"

	"github.com/mnuddindev/devpulse/config"
)

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
}
