.PHONY: lint
lint:
	gofmt -s -w .
	golangci-lint run

.PHONY: install
install:
	go mod tidy

.PHONY: docs
docs:
	make install
	make lint
	go tool swag fmt -d ./
	go tool swag init -g cmd/main.go -o api --v3.1 --parseInternal --parseDependency

# Executa somente testes unitários
.PHONY: test
test:
	go test ./internal... -v -count=1 -race -cover -coverprofile=./tmp/coverage.unit.out

# Executa somente testes de integração
.PHONY: test-integration
test-integration:
	go test -tags=integration ./internal... -v -count=1 -race -cover -coverprofile=./tmp/coverage.integration.out

.PHONY: dev
dev:
	go tool air

.PHONY: run
run:
	make docs
	go run -race cmd/main.go

.PHONY: up
up:
	docker compose -f deployments/payment-processor/docker-compose.yaml up -d
	docker compose -f deployments/docker-compose.yaml up --build

.PHONY: down
down:
	docker compose -f deployments/docker-compose.yaml down
	docker compose -f deployments/payment-processor/docker-compose.yaml down
	