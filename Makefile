VERSION := `git describe --tags | cut -d '-' -f 1`.`git rev-parse --short HEAD`

all: dep test build-all

dep:
	go get -a ./client/
	go get -a ./event/
	go get -a ./launcher/
	go get -a ./listener/
	go get -a ./cli/
	go get -a ./server/...
	go get -a ./redis-proxy/

test:
	go test -v -bench . ./...

build-all: _cli _server _redis-proxy

_cli:
	go build -a -ldflags "-X main.version='$(VERSION)'" -o bin/events-cli ./cli/

_server:
	go build -a -ldflags "-X main.version='$(VERSION)'" -o bin/events-server ./server/

_redis-proxy:
	go build -a -ldflags "-X main.version='$(VERSION)'" -o bin/redis-proxy ./redis-proxy/
