package main

import (
	"fmt"
	"vastix"
	"vastix/internal"
	log "vastix/internal/logging"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Set up panic logging - panic will still crash the program but will be saved to ~/.vastix/logs/panic.log
	defer log.LogPanic()

	if err := internal.Run(vastix.AppVersion()); err != nil {
		log.Fatal(fmt.Sprintf("Failed to run Vastix application: %v", err))
		return
	}
}
