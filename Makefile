default: build

build: build-client build-gateway
.PHONY: build

build-client:
	go build -o bin/client ./client
.PHONY: build-client

build-gateway:
	go build -o bin/gateway ./server/gateway
.PHONY: build-gateway

run-client: build-client
	./bin/client
.PHONY: run-client

run-gateway: build-gateway
	./bin/gateway
.PHONY: run-gateway
