dev-up:
	docker compose -f docker-compose.dev.yml up -d --build

dev-down:
	docker compose -f docker-compose.dev.yml down

dev-restart: dev-down dev-up

dev-logs:
	docker logs $(shell docker ps -q -f "name=lorem-backend")

dev-clean:
	docker compose -f docker-compose.dev.yml down --rmi all -v --remove-orphans

prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f docker-compose.prod.yml down

prod-restart: prod-down prod-up

prod-logs:
	docker logs $(shell docker ps -q -f "name=lorem-backend")

prod-clean:
	docker compose -f docker-compose.prod.yml down --rmi all -v --remove-orphans

db-up:
	docker compose -f docker-compose.db.yml up -d

db-down:
	docker compose -f docker-compose.db.yml down

db-restart: db-down db-up

db-clean:
	docker compose -f docker-compose.db.yml down --rmi all -v --remove-orphans

lint:
	go fmt ./...
	go vet ./...

test:
	go test -v ./internal/...

pre-commit: lint test