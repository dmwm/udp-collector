### CMSSW UDP collector service

[![Go CI build](https://github.com/dmwm/udp-collector/actions/workflows/go-ci.yml/badge.svg)](https://github.com/dmwm/udp-collector/actions/workflows/go-ci.yml)

CMSSW UDP collector service consist of UDP server `udp_server`
application deployed on Kubernetes cluster. To compile it you
need a [Go-lang](http://golang.org/) to be installed on your system.
Then use the following instructions:
```
# build executables
go build udp_server.go
go build udp_client.go
```

### Service maintenance
To deploy the service please refer to [CMSKubernetes](https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/monitoring/services) description of the `udp-collector`.

### Production procedure
You can test the udp server with provided udp client code.
```
# start server as following
udp_server -config udp_server.json

# start client as following
udp_client
```
The `udp_client` provides options to specify host, port and number of
documents to be used.

### Running on a virtual machine
You can check the history of this repository to see the old instructions if you decide to run the code on a virtual machine.