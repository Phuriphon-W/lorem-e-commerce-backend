db-up:
	docker compose up -d

db-down:
	docker compose down

db-restart: db-up db-down

db-clean:
	docker compose down --rmi all -v --remove-orphans

test:
	go test -v ./internal/...