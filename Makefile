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

# gRPC bonus phase (requires protoc, protoc-gen-go, protoc-gen-go-grpc).
# PROTOC defaults to the local tools/ copy; override if protoc is on PATH.
PROTOC ?= tools/protoc/bin/protoc

gen-proto:
	rm -rf gen
	$(PROTOC) --go_out=. --go_opt=module=github.com/oat431/7-solution-interview \
	       --go-grpc_out=. --go-grpc_opt=module=github.com/oat431/7-solution-interview \
	       proto/user_service/v1/user_service.proto
