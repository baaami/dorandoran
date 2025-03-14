GATEWAY_BINARY=gatewayApp
USER_BINARY=userApp
CHAT_BINARY=chatApp
AUTH_BINARY=authApp
MATCH_BINARY=matchApp
GAME_BINARY=gameApp
PUSH_BINARY=pushApp
SERVICES=doran-gateway doran-user doran-chat doran-auth doran-match doran-game doran-push
INFRAS=mysql mongo redis

## up: starts all containers in the background without forcing build
up:
	@echo "Starting Docker images..."
	docker-compose up -d
	@echo "Docker images started!"

## up_build: stops docker-compose (if running), builds all projects and starts docker compose
up_build: build_gateway build_user build_chat build_auth build_match build_game build_push build_push
	@echo "Stopping docker images (if running...)"
	docker-compose down
	@echo "Building (when required) and starting docker images..."
	docker-compose up --build -d
	@echo "Docker images built and started!"

## up_service: stops all services except MySQL, MongoDB, RabbitMQ, builds and restarts them
up_service: build_gateway build_user build_chat build_auth build_match build_game build_push
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
	cd services/gateway && env GOOS=linux CGO_ENABLED=0 go build -o ${GATEWAY_BINARY} ./cmd
	@echo "Done!"

## build_user: builds the user binary as a linux executable
build_user:
	@echo "Building user binary..."
	cd services/user && env GOOS=linux CGO_ENABLED=0 go build -o ${USER_BINARY} ./cmd
	@echo "Done!"

## build_chat: builds the chat binary as a linux executable
build_chat:
	@echo "Building chat binary..."
	cd services/chat && env GOOS=linux CGO_ENABLED=0 go build -o ${CHAT_BINARY} ./cmd
	@echo "Done!"

## build_auth: builds the auth binary as a linux executable
build_auth:
	@echo "Building auth binary..."
	cd services/auth && env GOOS=linux CGO_ENABLED=0 go build -o ${AUTH_BINARY} ./cmd
	@echo "Done!"

## build_match: builds the auth binary as a linux executable
build_match:
	@echo "Building match socket binary..."
	cd services/match && env GOOS=linux CGO_ENABLED=0 go build -o ${MATCH_BINARY} ./cmd
	@echo "Done!"

## build_game: builds the auth binary as a linux executable
build_game:
	@echo "Building game binary..."
	cd services/game && env GOOS=linux CGO_ENABLED=0 go build -o ${GAME_BINARY} ./cmd
	@echo "Done!"	

## build_push: builds the push binary as a linux executable
build_push:
	@echo "Building push binary..."
	cd services/push && env GOOS=linux CGO_ENABLED=0 go build -o ${PUSH_BINARY} ./cmd
	@echo "Done!"	

