VERSION=`git rev-parse --short HEAD`
flags=-ldflags="-s -w -X main.version=${VERSION}"

all: build

build:
	go clean; rm -rf pkg; go build -o udp_server ${flags} udp_server.go

build_debug:
	go clean; rm -rf pkg; go build -o udp_server ${flags} -gcflags="-m -m" udp_server.go

build_all: build_osx build_linux build

build_osx:
	go clean; rm -rf pkg udp_server_osx; GOOS=darwin go build -o udp_server ${flags} udp_server.go

build_linux:
	go clean; rm -rf pkg udp_server_linux; GOOS=linux go build -o udp_server ${flags} udp_server.go

build_power8:
	go clean; rm -rf pkg udp_server_power8; GOARCH=ppc64le GOOS=linux go build -o udp_server ${flags} udp_server.go

build_arm64:
	go clean; rm -rf pkg udp_server_arm64; GOARCH=arm64 GOOS=linux go build -o udp_server ${flags} udp_server.go

build_windows:
	go clean; rm -rf pkg udp_server.exe; GOARCH=amd64 GOOS=windows go build -o udp_server ${flags} udp_server.go

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	go test
