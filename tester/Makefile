PWD ?= `pwd`

all: server/server listener/listener launcher/launcher 

listener/listener: listener/main.go
	go build -o listener/listener ./listener/

launcher/launcher: launcher/main.go
	go build -o launcher/launcher ./launcher/

server/server: ../server/*.go
	go build -ldflags "-X main.version=test-server" -o server/server ../server

install: install-server install-listener install-launcher

install-server:
	cat ./events-server.service | sed 's@{PWD}@'$(PWD)'@g' > /lib/systemd/system/events-server.service
	systemctl daemon-reload
	systemctl start events-server
	systemctl status events-server

install-listener:
	cat ./events-listener.service | sed 's@{PWD}@'$(PWD)'@g' > /lib/systemd/system/events-listener.service
	systemctl daemon-reload
	systemctl start events-listener
	systemctl status events-listener

install-launcher:
	cat ./events-launcher.service | sed 's@{PWD}@'$(PWD)'@g' > /lib/systemd/system/events-launcher.service
	systemctl daemon-reload
	systemctl start events-launcher
	systemctl status events-launcher

uninstall: uninstall-server uninstall-listener uninstall-launcher

uninstall-server:
	systemctl stop events-server
	rm /lib/systemd/system/events-server.service
	systemctl daemon-reload

uninstall-listener:
	systemctl stop events-listener
	rm /lib/systemd/system/events-listener.service
	systemctl daemon-reload

uninstall-launcher:
	systemctl stop events-launcher
	rm /lib/systemd/system/events-launcher.service
	systemctl daemon-reload

run-server:
	./server/server -env ../server/conf/.env

run-listener:
	./listener/listener

run-launcher:
	./launcher/launcher
