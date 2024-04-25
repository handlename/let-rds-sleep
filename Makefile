export CGO_ENABLED=0

let-rds-sleep: go.mod go.sum *.go
	go build -ldflags "-s -w" -trimpath ./cmd/$@

test:
	go test -v ./...

.PHONY: clean
clean:
	rm -f let-rds-sleep
