# Phase 10: Docker Compose Improvements - Implementation Plan

This execution plan outlines the concrete steps required to implement Phase 10 of Track 2 (DevOps & Deployment), based on the decisions clarified.

## 1. Add `/health` Endpoint
- **File:** `internal/api/routers.go`
- **Action:** Add a native `echo.Context` HTTP handler in the `NewRouter` function before initializing Huma. This ensures a lightweight endpoint that remains hidden from the generated OpenAPI documentation.
- **Snippet:**
  ```go
  router.GET("/health", func(c echo.Context) error {
      return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
  })
  ```

## 2. Harden `Dockerfile` for Production
- **File:** `Dockerfile`
- **Action:**
  - Install `curl` in the `prod` Alpine stage.
  - Create a non-root user relying on Alpine's default UID/GID assignment (`appuser` / `appgroup`) for security.
  - Add a `HEALTHCHECK` instruction to continuously monitor the application via the new `/health` endpoint.
- **Changes:**
  ```dockerfile
  # In the prod stage:
  RUN apk --no-cache add ca-certificates tzdata curl
  RUN addgroup -S appgroup && adduser -S appuser -G appgroup
  USER appuser
  HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:5000/health || exit 1
  ```

## 3. Configure PostgreSQL Health Check
- **File:** `docker-compose.db.yml`
- **Action:** Define a `healthcheck` on the `db` service. To avoid hardcoding credentials, standard `$${POSTGRES_USER}` and `$${POSTGRES_DB}` environment variables will be used.
- **Changes:**
  ```yaml
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5
  ```

## 4. Update Development Compose File
- **File:** `docker-compose.dev.yml`
- **Action:**
  - Add a `healthcheck` block to the `backend` service.
  - Modify `depends_on: db` to strictly wait for the DB to be healthy before starting.
- **Changes:**
  ```yaml
    depends_on:
      db:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:5000/health"]
      interval: 10s
      timeout: 5s
      retries: 3
  ```

## 5. Update Production Compose File
- **File:** `docker-compose.prod.yml`
- **Action:**
  - Similar to dev, enforce `depends_on: db: condition: service_healthy`.
  - Ensure the backend service has the `restart: unless-stopped` policy applied.

## 6. Overhaul Project Documentation
- **File:** `README.md`
- **Action:** Rewrite the README to cover all essential developer onboarding steps:
  - Project Overview & Architecture Summary
  - Tech Stack
  - Prerequisites (Go 1.25, Docker, Make)
  - Setup Instructions (Docker Compose Dev, Prod, and Bare-metal)
  - Environment Variables Reference
  - Available Make Targets
  - API Documentation (Huma OpenAPI link)
