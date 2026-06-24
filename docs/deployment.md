# Deployment Guide — Lorem E-Commerce Backend

## Table of Contents

1. [Local Development](#1-local-development)
2. [Local Production Simulation](#2-local-production-simulation)

---

## 1. Local Development

Start all services (backend + PostgreSQL + Redis) using the development Makefile target:

```bash
make dev-up
```

Verify the backend is running:

```bash
curl http://localhost:5000/health
# Expected: {"status":"ok"}
```

Stop the stack:

```bash
make dev-down
```

---

## 2. Local Production Simulation

Build and start the production Docker Compose stack (uses `docker-compose.prod.yml`, which includes
`docker-compose.db.yml` for PostgreSQL + Redis):

```bash
# Using the Makefile shortcut
make prod-up

# Or directly with Docker Compose
docker compose -f docker-compose.prod.yml up -d
```

Verify the backend is healthy:

```bash
curl http://localhost:5000/health
# Expected: {"status":"ok"}
```

Tail logs:

```bash
docker compose -f docker-compose.prod.yml logs -f backend
```

Stop the stack:

```bash
docker compose -f docker-compose.prod.yml down
# To also remove volumes (database data):
docker compose -f docker-compose.prod.yml down -v
```
