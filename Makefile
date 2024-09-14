GATEWAY_BINARY=gatewayApp
USER_BINARY=userApp

## up: starts all containers in the background without forcing build
up:
	@echo "Starting Docker images..."
	docker-compose up -d
	@echo "Docker images started!"

## up_build: stops docker-compose (if running), builds all projects and starts docker compose
up_build: build_gateway
	@echo "Stopping docker images (if running...)"
	docker-compose down
	@echo "Building (when required) and starting docker images..."
	docker-compose up --build -d
	@echo "Docker images built and started!"

## down: stop docker compose
down:
	@echo "Stopping docker compose..."
	docker-compose down
	@echo "Done!"

## build_gateway: builds the gateway binary as a linux executable
build_gateway:
	@echo "Building gateway binary..."
	cd gateway-service && go build -o ${GATEWAY_BINARY} ./cmd/api
	@echo "Done!"

## build_gateway: builds the user binary as a linux executable
build_user:
	@echo "Building user binary..."
	cd user-service && go build -o ${USER_BINARY} ./cmd/api
	@echo "Done!"