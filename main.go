package main

import (
	"flag"
	"log"
	"os"

	"udp-collector/udpserver"
	"udp-collector/udpservermonitor"
)

func main() {
	var config string
	flag.StringVar(&config, "config", "", "configuration file")
	var version bool
	flag.BoolVar(&version, "version", false, "version")
	flag.Parse()

	if config == "" {
		log.Println("Usage: udp_collector -config=/path/to/config.json [-version]")
		os.Exit(1)
	}

	// Start the udp server
	go func() {
		udpserver.StartServer(config, version)
	}()

	// Start the udp server monitor
	go func() {
		udpservermonitor.StartMonitor(config)
	}()

	select {}
}