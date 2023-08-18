VERSION = $(shell git describe --tags | head -1)

let-rds-sleep: go.mod go.sum *.go
	go build -ldflags "-s -w -X main.version=${VERSION}" -trimpath ./cmd/$@

test:
	go test -v ./...

.PHONY: clean
clean:
	rm -f let-rds-sleep
