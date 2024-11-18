package udpserver

// udp_server - UDP Server implementation with optional support to send UDP messages
//              to StompAMQ endpoint
//
// Copyright (c) 2020 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-stomp/stomp"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// version of the code
var version string

// global pointer to Stomp connection
var stompConn *stomp.Conn

// Configuration stores server configuration parameters
type Configuration struct {
	Port                 int     `json:"port"`                 // server port number
	IPAddr               string  `json:"ipAddr"`               // server ip address to bind
	MonitorPort          int     `json:"monitorPort"`          // server monitor port number
	BufSize              int     `json:"bufSize"`              // buffer size
	StompURI             string  `json:"stompURI"`             // StompAMQ URI
	StompLogin           string  `json:"stompLogin"`           // StompAQM login name
	StompPassword        string  `json:"stompPassword"`        // StompAQM password
	StompIterations      int     `json:"stompIterations"`      // Stomp iterations
	SendTimeout          int     `json:"sendTimeout"`          // heartbeat send timeout in seconds
	RecvTimeout          int     `json:"recvTimeout"`          // heartbeat recv timeout in seconds
	HeartBeatGracePeriod float64 `json:"heartBeatGracePeriod"` // is used to calculate the read heart-beat timeout
	Endpoint             string  `json:"endpoint"`             // StompAMQ endpoint
	ContentType          string  `json:"contentType"`          // ContentType of UDP packet
	LogFile              string  `json:"logFile"`              // log file name
	Verbose              bool    `json:"verbose"`              // verbose output
}

// custom rotate logger
type rotateLogWriter struct {
	RotateLogs *rotatelogs.RotateLogs
}

func (w rotateLogWriter) Write(data []byte) (int, error) {
	return w.RotateLogs.Write([]byte(utcMsg(data)))
}

// helper function to use proper UTC message in a logger
func utcMsg(data []byte) string {
	s := string(data)
	v, e := url.QueryUnescape(s)
	if e == nil {
		return v
	}
	return s
}

var Config Configuration

// parseConfig parse given config file
func parseConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Unable to read", err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		log.Println("Unable to parse", err)
		return err
	}
	// default values
	if Config.Port == 0 {
		Config.Port = 9331 // default port
	}
	if Config.MonitorPort == 0 {
		Config.MonitorPort = 9330 // default port
	}
	if Config.BufSize == 0 {
		Config.BufSize = 1024 // 1 KByte
	}
	if Config.StompIterations == 0 {
		Config.StompIterations = 3 // number of Stomp attempts
	}
	if Config.ContentType == "" {
		Config.ContentType = "application/json"
	}
	if Config.HeartBeatGracePeriod == 0 {
		Config.HeartBeatGracePeriod = 1
	}
	if Config.SendTimeout == 0 {
		Config.SendTimeout = 600 // in seconds
	}
	if Config.RecvTimeout == 0 {
		Config.RecvTimeout = 0 // in seconds
	}
	return nil
}

func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now().Format("2006-02-01")
	return fmt.Sprintf("UDPServer git=%s go=%s date=%s", version, goVersion, tstamp)
}

// StompConnection returns Stomp connection
func StompConnection() (*stomp.Conn, error) {
	if Config.StompURI == "" {
		err := errors.New("Unable to connect to Stomp, not URI")
		return nil, err
	}
	if Config.StompLogin == "" {
		err := errors.New("Unable to connect to Stomp, not login")
		return nil, err
	}
	if Config.StompPassword == "" {
		err := errors.New("Unable to connect to Stomp, not password")
		return nil, err
	}
	conn, err := stomp.Dial("tcp",
		Config.StompURI,
		stomp.ConnOpt.Login(Config.StompLogin, Config.StompPassword),
		stomp.ConnOpt.HeartBeat(time.Duration(Config.SendTimeout)*time.Second, time.Duration(Config.RecvTimeout)*time.Second),
		stomp.ConnOpt.HeartBeatGracePeriodMultiplier(Config.HeartBeatGracePeriod),
	)
	if err != nil {
		log.Printf("Unable to connect to %s, error %v", Config.StompURI, err)
	}
	if Config.Verbose {
		log.Printf("connected to StompAMQ server %s %v", Config.StompURI, conn)
	}
	return conn, err
}

func sendDataToStomp(data []byte) {
	for i := 0; i < Config.StompIterations; i++ {
		err := stompConn.Send(Config.Endpoint, Config.ContentType, data)
		if err != nil {
			if i == Config.StompIterations-1 {
				log.Printf("unable to send data to %s, data %s, error %v, iteration %d", Config.Endpoint, string(data), err, i)
			} else {
				log.Printf("unable to send data to %s, error %v, iteration %d", Config.Endpoint, err, i)
			}
			if stompConn != nil {
				stompConn.Disconnect()
			}
			stompConn, err = StompConnection()
		} else {
			if Config.Verbose {
				log.Printf("send data to StompAMQ endpoint %s", Config.Endpoint)
			}
			return
		}
	}
}

// udp server implementation
func udpServer() {
	const maxFailedPacketLength = 1000 // maximum length of the failed packet to be printed
	udpAddr := &net.UDPAddr{Port: Config.Port}
	// if configuration provides explicitly IPAddr to bind use it here
	if Config.IPAddr != "" {
		udpAddr = &net.UDPAddr{
			Port: Config.Port,
			IP:   net.ParseIP(Config.IPAddr),
		}
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	defer conn.Close()
	log.Printf("UDP server %s\n", conn.LocalAddr().String())

	stompConn, err = StompConnection()
	// defer stomp connection if it exists
	if stompConn != nil {
		defer stompConn.Disconnect()
	}

	// set initial buffer size to handle UDP packets
	bufSize := Config.BufSize
	for {
		// create a buffer we'll use to read the UDP packets
		buffer := make([]byte, bufSize)

		// read UDP packets
		rlen, remote, err := conn.ReadFromUDP(buffer[:])
		if err != nil {
			log.Printf("Unable to read UDP packet, error %v", err)
			// clear-up our buffer
			buffer = buffer[:0]
			continue
		}
		data := buffer[:rlen]

		// if we receive ping message from monitoring server
		// we will send POST HTTP request to it with our pong reply
		if string(data) == "ping" {
			if Config.Verbose {
				log.Println("received monitor", string(data))
			}
			// send POST request to monitoring server, but don't care about response
			s := []byte("pong")
			rurl := fmt.Sprintf("http://localhost:%d", Config.MonitorPort)
			http.Post(rurl, "text/plain", bytes.NewBuffer(s))

			// clean-up our buffer
			buffer = buffer[:0]
			continue
		}

		// try to parse the data, we are expecting JSON
		var packet map[string]interface{}
		err = json.Unmarshal(data, &packet)
		if err != nil {
			log.Printf("unable to unmarshal UDP packet into JSON, error %v\n", err)
			e := string(err.Error())
			if strings.Contains(e, "invalid character") {
				// truncate the malformed JSON if it exceeds the maximum length
				// and dump it
				failedData := string(data)
				if len(failedData) > maxFailedPacketLength {
					failedData = failedData[:maxFailedPacketLength] + "..."
				}
				log.Println(failedData)
			} else if strings.Contains(e, "unexpected end of JSON input") {
				// let's increse buf size to adjust to the packet size
				bufSize = bufSize * 2
				if bufSize > 1024*Config.BufSize {
					log.Fatalf("unable to unmarshal UDP packet into JSON with buffer size %d", bufSize)
				}
			}
			// at this point we already read from UDP connection and our
			// message didn't fit into buffer therefore we may skip the rest
			// clear-up our buffer and continue
			buffer = buffer[:0]
			continue
		}

		// dump message to our log
		if Config.Verbose {
			sdata := strings.TrimSpace(string(data))
			log.Printf("received: %s from %s\n", sdata, remote)
		}

		// check if the message has a key named "type" and rename it to "read_type"
		if val, ok := packet["type"]; ok {
			packet["read_type"] = val
			delete(packet, "type")
		}

		// send data to Stomp endpoint
		if Config.Endpoint != "" && stompConn != nil {
			newData, err := json.Marshal(packet)
			if err != nil {
				log.Printf("unable to marshal UDP packet into JSON, error %v\n", err)
				// truncate the failed packet if it exceeds the maximum length
				failedPacket := fmt.Sprint(packet)
				if len(failedPacket) > maxFailedPacketLength {
					failedPacket = failedPacket[:maxFailedPacketLength] + "..."
				}
				log.Println(failedPacket) // dump the truncated message to the log
				// clear-up our buffer
				buffer = buffer[:0]
				continue
			}
			if Config.Verbose {
				sNewData := strings.TrimSpace(string(newData))
				log.Printf("sent to AMQ: %s\n", sNewData)
			}
			sendDataToStomp(newData)
		}

		// clear-up our buffer
		buffer = buffer[:0]
	}
}

func StartServer(config string, version bool) {
	if version {
		log.Println(info())
		os.Exit(0)
	}
	err := parseConfig(config)
	// set log file or log output
	if Config.LogFile != "" {
		logName := Config.LogFile + "-%Y%m%d"
		hostname, err := os.Hostname()
		if err == nil {
			logName = Config.LogFile + "-" + hostname + "-%Y%m%d"
		}
		rl, err := rotatelogs.New(logName)
		if err == nil {
			rotlogs := rotateLogWriter{RotateLogs: rl}
			log.SetOutput(rotlogs)
		}
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		// log time, filename, and line number
		if Config.Verbose {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		} else {
			log.SetFlags(log.LstdFlags)
		}
	}

	if err == nil {
		udpServer()
	}
	log.Fatal(err)
}
