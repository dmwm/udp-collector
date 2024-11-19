package main

import (
	"flag"
	"log"
	"os"
	"fmt"
	"time"
	"runtime"

	"udp-collector/udpserver"
	"udp-collector/udpservermonitor"
)

// version of the code
var version string

func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now().Format("2006-02-01")
	return fmt.Sprintf("UDPServer git=%s go=%s date=%s", version, goVersion, tstamp)
}

func main() {
	var config string
	flag.StringVar(&config, "config", "", "configuration file")
	var version bool
	flag.BoolVar(&version, "version", false, "version")
	flag.Parse()

	if version {
		log.Println(info())
		os.Exit(0)
	}

	if config == "" {
		log.Println("Usage: udp_collector -config=/path/to/config.json [-version]")
		os.Exit(1)
	}

	// Start the udp server
	go func() {
		udpserver.StartServer(config)
	}()

	// Start the udp server monitor
	go func() {
		udpservermonitor.StartMonitor(config)
	}()

	select {}
}