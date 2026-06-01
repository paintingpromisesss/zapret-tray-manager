.PHONY: fmt lint test vuln check build generate installer

VERSION ?= dev

build:
	go build -ldflags="-H windowsgui -s -w -X main.version=$(VERSION)" -o "build/Zapret Tray Manager.exe" ./cmd

generate:
	go generate ./cmd

ISCC ?= "C:/Program Files (x86)/Inno Setup 6/iscc.exe"

installer: build
	$(ISCC) /DAppVersion=$(VERSION) installer.iss

fmt:
	gofmt -w .
	goimports -w .

lint:
	golangci-lint run ./...

test:
	go test -race ./...

vuln:
	govulncheck ./...

check: fmt lint test vuln