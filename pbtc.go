package main

import (
	"log"
	"os"
)

func main() {

	// Initialize basic logger
	pbtcLog := log.New(os.Stdout, "[PBTC]", log.LstdFlags)

	pbtcLog.Println("Passive Bitcoin by CIRL")
}
