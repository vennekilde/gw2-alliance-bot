BIN_NAME = gw2-alliance-bot

gen-api:
	go-raml client --ramlfile api/api.raml --dir internal/api --package api --import-path github.com/vennekilde/gw2-alliance-bot/internal/api 

build: export CGO_ENABLED=0
build: 
	go build -installsuffix 'static' -o ./bin/$(BIN_NAME) ./cmd/gw2-alliance-bot/main.go

build_windows: export GOOS=windows
build_windows: export GOARCH=amd64
build_windows: BIN_NAME:=$(BIN_NAME).exe
build_windows: build

build_linux: export GOOS=linux
build_linux: export GOARCH=amd64
build_linux: build

package:
	docker build . -t vennekilde/gw2-alliance-bot

scan:
	gosec ./...