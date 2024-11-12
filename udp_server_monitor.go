package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/procfs"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/
)

// global variables
var monitorInterval time.Duration
var lastUpdate time.Time
var verbose bool

func udpPing(host_port string) {
	// Connect to udp server
	conn, err := net.Dial("udp", host_port)
	if err != nil {
		fmt.Printf("Unable to contact: %s", host_port)
		return
	}
	defer conn.Close()

	// write ping message
	conn.Write([]byte("ping"))
}

func udpServerPID(pat string) int {
	cmd := fmt.Sprintf("ps -eo pid,args | grep \"%s\" | grep -v grep | awk '{print $1}'", pat)
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		log.Printf("Unable to find process pattern: %v, error: %v\n", pat, err)
		return 0
	}
	outStr := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(outStr)
	if err != nil {
		log.Printf("Error parsing PID from command output: %v", outStr)
		return 0
	}
	log.Printf("Found PID: %d for pattern %s", pid, pat)
	return pid
}

// requestHandler helper function for our monitoring server
// we should only received POST request from udp_server with pong data message
func requestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if verbose {
		log.Println("received", string(data), r.Method, r.RemoteAddr)
	}
	if err == nil {
		if string(data) == "pong" {
			lastUpdate = time.Now()
		}
	}
	w.WriteHeader(http.StatusOK)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	pat := fmt.Sprintf("udp_server -config")
	pid := udpServerPID(pat)
	metrics := make(map[string]interface{})
	metrics["lastUpdate"] = lastUpdate.Unix()
	if v, e := mem.VirtualMemory(); e == nil {
		metrics["memory_percent"] = v.UsedPercent
		metrics["memory_total"] = float64(v.Total)
		metrics["memory_free"] = float64(v.Free)
	}
	if v, e := mem.SwapMemory(); e == nil {
		metrics["swap_percent"] = v.UsedPercent
		metrics["swap_total"] = float64(v.Total)
		metrics["swap_free"] = float64(v.Free)
	}
	if c, e := cpu.Percent(time.Millisecond, false); e == nil {
		metrics["cpu_percent"] = c[0] // one value since we didn't ask per cpu
	}
	if l, e := load.Avg(); e == nil {
		metrics["load1"] = l.Load1
		metrics["load5"] = l.Load5
		metrics["load15"] = l.Load15
	}
	if proc, err := procfs.NewProc(pid); err == nil {
		if stat, err := proc.Stat(); err == nil {
			metrics["cpu_total"] = float64(stat.CPUTime())
			metrics["vsize"] = float64(stat.VirtualMemory())
			metrics["rss"] = float64(stat.ResidentMemory())
		}
		if fds, err := proc.FileDescriptorsLen(); err == nil {
			metrics["open_fds"] = float64(fds)
		}
		if limits, err := proc.Limits(); err == nil {
			metrics["max_fds"] = float64(limits.OpenFiles)
			metrics["max_vsize"] = float64(limits.AddressSpace)
		}
	}
	if proc, err := process.NewProcess(int32(pid)); err == nil {
		if v, e := proc.CPUPercent(); e == nil {
			metrics["proccess_cpu"] = float64(v)
		}
		if v, e := proc.MemoryPercent(); e == nil {
			metrics["process_memory"] = float64(v)
		}

		if v, e := proc.NumThreads(); e == nil {
			metrics["number_threads"] = float64(v)
		}
		if oFiles, e := proc.OpenFiles(); e == nil {
			metrics["open_files"] = float64(len(oFiles))
		}
	}
	data, err := json.Marshal(metrics)
	log.Println("metrics", string(data), err)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	}
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
}

// healthCheckHandler responds to health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    // Calculate the threshold time
    thresholdTime := time.Now().Add(-3 * monitorInterval)

    if lastUpdate.After(thresholdTime) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, "OK")
    } else {
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintln(w, "Error: last update is too old")
    }
}

func main() {
	var config string
	flag.StringVar(&config, "config", "udp_server.json", "UDP server config")
	flag.Parse()

	// parse config file
	data, e := os.ReadFile(config)
	if e != nil {
		log.Fatalf("Unable to read config file: %v\n", config)
		os.Exit(1)
	}
	var c map[string]interface{}
	e = json.Unmarshal(data, &c)
	if e != nil {
		log.Fatalf("Unable to unmarshal data: %v\n", data)
	}

	// setup variables from config parameters
	hostPort := fmt.Sprintf(":%d", int64(c["port"].(float64)))
	monHostPort := fmt.Sprintf(":%d", int64(c["monitorPort"].(float64)))
	monitorInterval = time.Duration(int64(c["monitorInterval"].(float64))) * time.Second
	verbose = c["verbose"].(bool)
	if verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	lastUpdate = time.Now()

	// create goroutine with running UDP ping
	go func() {
		for {
			time.Sleep(1 * time.Second)
			udpPing(hostPort)
		}
	}()

	// start our monitoring server
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthCheckHandler)
	http.HandleFunc("/", requestHandler)
	http.ListenAndServe(monHostPort, nil)
}