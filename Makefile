VERSION := `git describe --tags | cut -d '-' -f 1`.`git rev-parse --short HEAD`

all: dep test _events-cli _server _redis-proxy

dep:
	go get -a ./client/
	go get -a ./event/
	go get -a ./eventd/
	go get -a ./launcher/
	go get -a ./listener/
	go get -a ./redis-proxy/
	go get -a ./server/...

test:
	go test -v -bench . ./...

_events-cli:
	go build -ldflags "-X main.version='$(VERSION)'" -o bin/events-cli ./eventd/

_redis-proxy:
	go build -ldflags "-X main.version='$(VERSION)'" -o bin/redis-proxy ./redis-proxy

_server:
	go build -ldflags "-X main.version='$(VERSION)'" -o bin/events-driver ./server/
