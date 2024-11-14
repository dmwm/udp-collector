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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

type Exporter struct {
	memoryPercent    prometheus.Gauge
	memoryTotal      prometheus.Gauge
	memoryFree       prometheus.Gauge
	swapPercent      prometheus.Gauge
	swapTotal        prometheus.Gauge
	swapFree         prometheus.Gauge
	cpuPercent       prometheus.Gauge
	load1            prometheus.Gauge
	load5            prometheus.Gauge
	load15           prometheus.Gauge
	cpuTotal         prometheus.Gauge
	vSize            prometheus.Gauge
	rss              prometheus.Gauge
	openFDs          prometheus.Gauge
	maxFDs           prometheus.Gauge
	maxVSize         prometheus.Gauge
	processCPU       prometheus.Gauge
	processMemory    prometheus.Gauge
	numberThreads    prometheus.Gauge
	openFiles        prometheus.Gauge
}

func NewExporter() *Exporter {
	const metricPrefix = "udp_server_"
	return &Exporter{
        memoryPercent: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "memory_percent",
            Help: "Memory usage percentage",
        }),
        memoryTotal: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "memory_total",
            Help: "Total memory",
        }),
        memoryFree: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "memory_free",
            Help: "Free memory",
        }),
        swapPercent: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "swap_percent",
            Help: "Swap usage percentage",
        }),
        swapTotal: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "swap_total",
            Help: "Total swap",
        }),
        swapFree: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "swap_free",
            Help: "Free swap",
        }),
        cpuPercent: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "cpu_percent",
            Help: "CPU usage percentage",
        }),
        load1: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "load1",
            Help: "1-minute load average",
        }),
        load5: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "load5",
            Help: "5-minute load average",
        }),
        load15: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "load15",
            Help: "15-minute load average",
        }),
        cpuTotal: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "cpu_total",
            Help: "Total CPU time",
        }),
        vSize: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "vsize",
            Help: "Virtual memory size",
        }),
        rss: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "rss",
            Help: "Resident set size",
        }),
        openFDs: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "open_fds",
            Help: "Number of open file descriptors",
        }),
        maxFDs: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "max_fds",
            Help: "Maximum number of open file descriptors",
        }),
        maxVSize: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "max_vsize",
            Help: "Maximum virtual memory size",
        }),
        processCPU: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "process_cpu",
            Help: "Process CPU usage percentage",
        }),
        processMemory: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "process_memory",
            Help: "Process memory usage percentage",
        }),
        numberThreads: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "number_threads",
            Help: "Number of threads for the process",
        }),
        openFiles: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: metricPrefix + "open_files",
            Help: "Number of open files for the process",
        }),
    }
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
    e.memoryPercent.Describe(ch)
    e.memoryTotal.Describe(ch)
    e.memoryFree.Describe(ch)
    e.swapPercent.Describe(ch)
    e.swapTotal.Describe(ch)
    e.swapFree.Describe(ch)
    e.cpuPercent.Describe(ch)
    e.load1.Describe(ch)
    e.load5.Describe(ch)
    e.load15.Describe(ch)
    e.cpuTotal.Describe(ch)
    e.vSize.Describe(ch)
    e.rss.Describe(ch)
    e.openFDs.Describe(ch)
    e.maxFDs.Describe(ch)
    e.maxVSize.Describe(ch)
    e.processCPU.Describe(ch)
    e.processMemory.Describe(ch)
    e.numberThreads.Describe(ch)
    e.openFiles.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	pat := fmt.Sprintf("udp_server -config")
	pid := udpServerPID(pat)

	// Collect data
	if memInfo, err := mem.VirtualMemory(); err == nil {
		e.memoryPercent.Set(memInfo.UsedPercent)
		e.memoryTotal.Set(float64(memInfo.Total))
		e.memoryFree.Set(float64(memInfo.Free))
	} else if verbose {
		log.Printf("Failed to collect memory metrics: %v", err)
	}

	if swapInfo, err := mem.SwapMemory(); err == nil {
		e.swapPercent.Set(swapInfo.UsedPercent)
		e.swapTotal.Set(float64(swapInfo.Total))
		e.swapFree.Set(float64(swapInfo.Free))
	} else if verbose {
		log.Printf("Failed to collect swap metrics: %v", err)
	}

	if cpuPercent, err := cpu.Percent(time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		e.cpuPercent.Set(cpuPercent[0])
	} else if verbose {
		log.Printf("Failed to collect CPU percent: %v", err)
	}

	
	if loadAvg, err := load.Avg(); err == nil {
		e.load1.Set(loadAvg.Load1)
		e.load5.Set(loadAvg.Load5)
		e.load15.Set(loadAvg.Load15)
	} else if verbose {
		log.Printf("Failed to collect load metrics: %v", err)
	}

	if proc, err := procfs.NewProc(pid); err == nil {
		if stat, err := proc.Stat(); err == nil {
			e.cpuTotal.Set(float64(stat.CPUTime()))
			e.vSize.Set(float64(stat.VirtualMemory()))
			e.rss.Set(float64(stat.ResidentMemory()))
		}
		if fds, err := proc.FileDescriptorsLen(); err == nil {
			e.openFDs.Set(float64(fds))
		}
		if limits, err := proc.Limits(); err == nil {
			e.maxFDs.Set(float64(limits.OpenFiles))
			e.maxVSize.Set(float64(limits.AddressSpace))
		}
	} else if verbose {
		log.Printf("Failed to collect process metrics: %v", err)
	}

	if proc, err := process.NewProcess(int32(pid)); err == nil {
		if cpuPercent, err := proc.CPUPercent(); err == nil {
			e.processCPU.Set(cpuPercent)
		}
		if memPercent, err := proc.MemoryPercent(); err == nil {
			e.processMemory.Set(float64(memPercent))
		}
		if numThreads, err := proc.NumThreads(); err == nil {
			e.numberThreads.Set(float64(numThreads))
		}
		if openFiles, err := proc.OpenFiles(); err == nil {
			e.openFiles.Set(float64(len(openFiles)))
		}
	} else if verbose {
		log.Printf("Failed to collect process metrics: %v", err)
	}

	// Send collected metrics to Prometheus
    e.memoryPercent.Collect(ch)
    e.memoryTotal.Collect(ch)
    e.memoryFree.Collect(ch)
    e.swapPercent.Collect(ch)
    e.swapTotal.Collect(ch)
    e.swapFree.Collect(ch)
    e.cpuPercent.Collect(ch)
    e.load1.Collect(ch)
    e.load5.Collect(ch)
    e.load15.Collect(ch)
    e.cpuTotal.Collect(ch)
    e.vSize.Collect(ch)
    e.rss.Collect(ch)
    e.openFDs.Collect(ch)
    e.maxFDs.Collect(ch)
    e.maxVSize.Collect(ch)
    e.processCPU.Collect(ch)
    e.processMemory.Collect(ch)
    e.numberThreads.Collect(ch)
    e.openFiles.Collect(ch)
}

func udpPing(hostPort string) {
	// Connect to udp server
	conn, err := net.Dial("udp", hostPort)
	if err != nil {
		if verbose {
			log.Printf("Unable to contact UDP server at %s: %v", hostPort, err)
		}
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

// func metricsHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "GET" {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}
// 	pat := fmt.Sprintf("udp_server -config")
// 	pid := udpServerPID(pat)
// 	metrics := make(map[string]interface{})
// 	metrics["lastUpdate"] = lastUpdate.Unix()
// 	if v, e := mem.VirtualMemory(); e == nil {
// 		metrics["memory_percent"] = v.UsedPercent
// 		metrics["memory_total"] = float64(v.Total)
// 		metrics["memory_free"] = float64(v.Free)
// 	}
// 	if v, e := mem.SwapMemory(); e == nil {
// 		metrics["swap_percent"] = v.UsedPercent
// 		metrics["swap_total"] = float64(v.Total)
// 		metrics["swap_free"] = float64(v.Free)
// 	}
// 	if c, e := cpu.Percent(time.Millisecond, false); e == nil {
// 		metrics["cpu_percent"] = c[0] // one value since we didn't ask per cpu
// 	}
// 	if l, e := load.Avg(); e == nil {
// 		metrics["load1"] = l.Load1
// 		metrics["load5"] = l.Load5
// 		metrics["load15"] = l.Load15
// 	}
// 	if proc, err := procfs.NewProc(pid); err == nil {
// 		if stat, err := proc.Stat(); err == nil {
// 			metrics["cpu_total"] = float64(stat.CPUTime())
// 			metrics["vsize"] = float64(stat.VirtualMemory())
// 			metrics["rss"] = float64(stat.ResidentMemory())
// 		}
// 		if fds, err := proc.FileDescriptorsLen(); err == nil {
// 			metrics["open_fds"] = float64(fds)
// 		}
// 		if limits, err := proc.Limits(); err == nil {
// 			metrics["max_fds"] = float64(limits.OpenFiles)
// 			metrics["max_vsize"] = float64(limits.AddressSpace)
// 		}
// 	}
// 	if proc, err := process.NewProcess(int32(pid)); err == nil {
// 		if v, e := proc.CPUPercent(); e == nil {
// 			metrics["proccess_cpu"] = float64(v)
// 		}
// 		if v, e := proc.MemoryPercent(); e == nil {
// 			metrics["process_memory"] = float64(v)
// 		}

// 		if v, e := proc.NumThreads(); e == nil {
// 			metrics["number_threads"] = float64(v)
// 		}
// 		if oFiles, e := proc.OpenFiles(); e == nil {
// 			metrics["open_files"] = float64(len(oFiles))
// 		}
// 	}
// 	data, err := json.Marshal(metrics)
// 	log.Println("metrics", string(data), err)
// 	if err == nil {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte(data))
// 		return
// 	}
// 	log.Println(err)
// 	w.WriteHeader(http.StatusInternalServerError)
// }

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
			time.Sleep(5 * time.Second)
			udpPing(hostPort)
		}
	}()

	exporter := NewExporter()
	prometheus.MustRegister(exporter)

	// start our monitoring server
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthCheckHandler)
	http.HandleFunc("/", requestHandler)

	log.Printf("Starting monitoring server at %s", monHostPort)
	if err := http.ListenAndServe(monHostPort, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}