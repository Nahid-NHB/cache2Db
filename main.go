package main

import (
	"flag"
	"log"

	"github.com/nahid12105080/cacheDB/config"
	"github.com/nahid12105080/cacheDB/core"
	"github.com/nahid12105080/cacheDB/server"
)

func setUpFlags() {
	flag.StringVar(&config.Host, "host", config.Host, "host of the cacheDB server")
	flag.IntVar(&config.Port, "port", config.Port, "port of the cacheDB server")
	flag.IntVar(&config.MaxKeys, "maxkeys", config.MaxKeys, "maximum number of keys to retain before evicting (0 = unlimited)")
	flag.Parse()
}

func main() {
	setUpFlags()

	if err := core.Load(); err != nil {
		log.Printf("failed to load snapshot: %v", err)
	}

	log.Printf("Server starting on %s:%d\n", config.Host, config.Port)

	server.RunSyncTCPServer()
}
