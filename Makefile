build: build-mac build-linux
build-mac:
	go build -o ./bin/mac/mysql_tool -ldflags="-w"

build-linux:
	GOOS=linux GOARCH=amd64 go build -o ./bin/linux/mysql_tool -ldflags="-w"

dep-re-init:
	rm -rf vendor
	rm -f Gopkg.*
	dep init
