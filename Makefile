# mic-off build & test entry points.
.PHONY: build build-mac build-win test-unit vet run clean

VERSION ?= 0.1.0
LDFLAGS  = -s -w -X main.version=$(VERSION)

build: build-mac build-win        ## Build every release artifact into dist/

build-mac:                        ## dist/mic-off.app + dist/mic-off-macos.zip (universal)
	VERSION=$(VERSION) sh packaging/macos/build.sh

build-win:                        ## dist/mic-off-windows-x64.exe and -arm64.exe (cross-compiled, no cgo)
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS) -H windowsgui" -o dist/mic-off-windows-x64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS) -H windowsgui" -o dist/mic-off-windows-arm64.exe .

test-unit:                        ## Run unit tests
	go test ./...

vet:                              ## Type-check both targets
	go vet ./...
	CGO_ENABLED=0 GOOS=windows go vet ./...

run:                              ## Run the tray app from source (macOS dev loop)
	go run .

clean:
	rm -rf dist
