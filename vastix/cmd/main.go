package main

import (
	"fmt"
	"vastix/internal"
	log "vastix/internal/logging"

	_ "github.com/joho/godotenv/autoload"
)

var version string = "0.1.0-beta" // Default to 0.1.0-beta, will be set by build script

func main() {
	if err := internal.Run(version); err != nil {
		log.Fatal(fmt.Sprintf("Failed to run Vastix application: %v", err))
		return
	}
}
