.PHONY: build run test cover vet fmt gen-proto docker-up docker-down smoke tidy

build:
	go build -o bin/api ./cmd/api

run:
	set -a && . ./.env && set +a && go run ./cmd/api

test:
	go test -race -cover ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

smoke:
	bash scripts/smoke.sh

# gRPC bonus phase (requires protoc, protoc-gen-go, protoc-gen-go-grpc)
gen-proto:
	protoc --go_out=gen --go_opt=paths=source_relative \
	       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
	       proto/user_service/v1/user_service.proto
