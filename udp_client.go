package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
)

func record(seed, user, host string) map[string]interface{} {
	data := make(map[string]interface{})
	data["read_vector_bytes"] = 133206046
	data["site_name"] = "T3_US_Cornell"
	data["read_vector_ndocs_average"] = 21.411799999999999
	data["user_dn"] = fmt.Sprintf("/DC=ch/DC=cern/OU=Organic Units/OU=Users/CN=%s%s", user, seed)
	data["file_lfn"] = fmt.Sprintf("/store/fake/file_%s.root", seed)
	data["read_bytes"] = 148607872
	data["file_size"] = 27502730289
	data["read_single_average"] = 3793.5500000000002
	data["client_host"] = "rossmann-a251"
	data["read_vector_average"] = 7835650.0
	data["read_vector_sigma"] = 7081190.0
	data["server_host"] = "cmshdp-d019"
	data["read_vector_operations"] = 17
	data["read_single_bytes"] = 15401826
	data["app_info"] = "something"
	data["client_domain"] = host
	data["start_time"] = 1395960729
	data["read_vector_ndocs_sigma"] = 56.063099999999999
	data["read_single_sigma"] = 84703.899999999994
	data["server_domain"] = host
	data["read_single_operations"] = 4060
	data["read_bytes_at_close"] = 148607872
	data["end_time"] = 1395960959
	data["fallback"] = false
	data["unique_id"] = fmt.Sprintf("60DC3A6D-02B6-E311-B2BD-0002C90B73D8-0%s", seed)
	return data
}

func send(host string, port, ndocs int) {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	user := "test"

	for i := 0; i < ndocs; i++ {
		doc := record(fmt.Sprintf("%d", i), user, host)
		data, err := json.Marshal(doc)
		if err == nil {
			conn.Write(data)
		} else {
			log.Println(err)
		}
	}
}

func main() {
	var host string
	flag.StringVar(&host, "host", "localhost", "host name")
	var port int
	flag.IntVar(&port, "port", 9331, "port number")
	var ndocs int
	flag.IntVar(&ndocs, "ndocs", 10, "number of docs to generate and send")
	flag.Parse()
	// log time, filename, and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	send(host, port, ndocs)
}
