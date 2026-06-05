dev-up:
	docker compose -f docker-compose.dev.yml up -d --build

dev-down:
	docker compose -f docker-compose.dev.yml down

dev-restart: dev-down dev-up

dev-logs:
	docker logs -f $(shell docker ps -q -f "name=lorem-backend")

dev-clean:
	docker compose -f docker-compose.dev.yml down --rmi all -v --remove-orphans

prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f docker-compose.prod.yml down

prod-restart: prod-down prod-up

prod-logs:
	docker logs -f $(shell docker ps -q -f "name=lorem-backend")

prod-clean:
	docker compose -f docker-compose.prod.yml down --rmi all -v --remove-orphans

db-up:
	docker compose -f docker-compose.db.yml up -d

db-down:
	docker compose -f docker-compose.db.yml down

db-restart: db-down db-up

db-clean:
	docker compose -f docker-compose.db.yml down --rmi all -v --remove-orphans

seed-db:
	powershell -Command "$$env:DB_HOST='localhost'; $$env:DB_PORT=5433; go run ./cmd/seed/main.go"

lint:
	go fmt ./...
	go vet ./...

test:
	go test -v -cover ./internal/...

coverage-check:
	go test -coverprofile coverage.out ./internal/modules/... ./internal/api/middleware/... ./internal/utils/...
	@COVERAGE=$$(go tool cover -func coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $$COVERAGE%"; \
	awk "BEGIN{if ($$COVERAGE + 0 < 80) {print \"FAIL: Coverage \" $$COVERAGE \"% is below 80% threshold\"; exit 1}}"
	@rm -f coverage.out

pre-commit: lint test