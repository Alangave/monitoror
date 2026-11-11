BINARY := upmonitor
UI_DIR := web

.PHONY: all build-ui build run test vet docker clean

all: build

build-ui:
	cd $(UI_DIR) && npm ci && npm run build

build: build-ui
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY) ./cmd/upmonitor

run:
	UPMONITOR_CONFIG=config-example.json go run ./cmd/upmonitor

test:
	go test ./...

vet:
	go vet ./...

docker:
	docker build -t upmonitor .

clean:
	rm -rf bin $(UI_DIR)/dist
