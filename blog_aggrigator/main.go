package main

import (
	"fmt"
	"log"

	"gator/internal/config"
)

func main() {
	// 1. Read config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Set the current user
	if err := cfg.SetUser("Devin"); err != nil {
		log.Fatal(err)
	}

	// 3. Read it again
	updatedCfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	// 4. Print to terminal
	fmt.Printf("%+v\n", updatedCfg)
}
