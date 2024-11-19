VERSION=`git rev-parse --short HEAD`
flags=-ldflags="-s -w -X main.version=${VERSION}"

all: build

build:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o udp_collector ${flags}

build_debug:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o udp_collector ${flags} -gcflags="-m -m"

build_all: build_osx build_linux build

build_osx:
	go clean; rm -rf pkg udp_collector_osx; GOOS=darwin CGO_ENABLED=0 go build -o udp_collector ${flags}

build_linux:
	go clean; rm -rf pkg udp_collector_linux; GOOS=linux CGO_ENABLED=0 go build -o udp_collector ${flags}

build_power8:
	go clean; rm -rf pkg udp_collector_power8; GOARCH=ppc64le GOOS=linux CGO_ENABLED=0 go build -o udp_collector ${flags}

build_arm64:
	go clean; rm -rf pkg udp_collector_arm64; GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o udp_collector ${flags}

build_windows:
	go clean; rm -rf pkg udp_collector.exe; GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -o udp_collector ${flags}

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	go test -v -race -bench=.