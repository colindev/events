VERSION := `git describe --tags | cut -d '-' -f 1`.`git rev-parse --short HEAD`
SERVERNAME ?= events-server

all: dep test build-all

dep:
	go get -a ./client/
	go get -a ./event/
	go get -a ./launcher/
	go get -a ./listener/
	go get -a ./cli/
	go get -a ./store/
	go get -a ./server/
	go get -a ./redis-proxy/...

test:
	go test -v -race ./...
	go test -run x -bench . ./...

build-all: bin/events-cli bin/store bin/events-server bin/redis-proxy bin/watch

bin/events-cli: cli/*.go 
	go build -a -ldflags "-X main.version=$(VERSION)" -o bin/events-cli ./cli/

bin/store: store/cli/*.go
	go build -a -ldflags "-X main.version=$(VERSION)" -o bin/store ./store/cli/

bin/events-server: server/*.go
	go build -a -ldflags "-X main.version=$(VERSION) -X main.appName=$(SERVERNAME)" -o bin/events-server ./server/

bin/redis-proxy: redis-proxy/*.go
	go build -a -ldflags "-X main.version=$(VERSION)" -o bin/redis-proxy ./redis-proxy/

bin/watch: redis-proxy/watch/*.go
	go build -a -ldflags "-X main.version=$(VERSION)" -o bin/watch ./redis-proxy/watch/
