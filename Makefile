GATEWAY_BINARY=gatewayApp
USER_BINARY=userApp
CHAT_BINARY=chatApp
CONSUMER_BINARY=consumerApp
AUTH_BINARY=authApp
SERVICES=gateway-service user-service chat-service consumer-service auth-service

## up: starts all containers in the background without forcing build
up:
	@echo "Starting Docker images..."
	docker-compose up -d
	@echo "Docker images started!"

## up_build: stops docker-compose (if running), builds all projects and starts docker compose
up_build: build_gateway build_user build_chat build_consumer build_auth
	@echo "Stopping docker images (if running...)"
	docker-compose down
	@echo "Building (when required) and starting docker images..."
	docker-compose up --build -d
	@echo "Docker images built and started!"

## up_service: stops all services except MySQL, MongoDB, RabbitMQ, builds and restarts them
up_service: build_gateway build_user build_chat build_consumer build_auth
	@echo "Stopping services except for MySQL, MongoDB, RabbitMQ..."
	docker-compose stop ${SERVICES}
	docker-compose rm -f ${SERVICES}
	@echo "Building and starting services..."
	docker-compose up --build -d ${SERVICES}
	@echo "Services have been rebuilt and started!"

down_service:
	@echo "Stopping services except for MySQL, MongoDB, RabbitMQ..."
	docker-compose stop ${SERVICES}
	docker-compose rm -f ${SERVICES}

## down: stop docker compose
down:
	@echo "Stopping docker compose..."
	docker-compose down
	@echo "Done!"

## build_gateway: builds the gateway biary as a linux executable
build_gateway:
	@echo "Building gateway binary..."
	cd gateway-service && env GOOS=linux CGO_ENABLED=0 go build -o ${GATEWAY_BINARY} ./cmd/api
	@echo "Done!"

## build_user: builds the user binary as a linux executable
build_user:
	@echo "Building user binary..."
	cd user-service && env GOOS=linux CGO_ENABLED=0 go build -o ${USER_BINARY} ./cmd/api
	@echo "Done!"

## build_chat: builds the chat binary as a linux executable
build_chat:
	@echo "Building chat binary..."
	cd chat-service && env GOOS=linux CGO_ENABLED=0 go build -o ${CHAT_BINARY} ./cmd/api
	@echo "Done!"

## build_auth: builds the auth binary as a linux executable
build_auth:
	@echo "Building auth binary..."
	cd auth-service && env GOOS=linux CGO_ENABLED=0 go build -o ${AUTH_BINARY} ./cmd/api
	@echo "Done!"

## build_consumer: builds the consumer binary as a linux executable
build_consumer:
	@echo "Building consumer binary..."
	cd consumer-service && env GOOS=linux CGO_ENABLED=0 go build -o ${CONSUMER_BINARY} .
	@echo "Done!"

