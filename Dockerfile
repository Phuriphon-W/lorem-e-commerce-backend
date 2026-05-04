# --- Base Stage ---
FROM golang:1.25-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# --- Development Stage ---
FROM base AS dev
# Install air for hot reload
RUN go install github.com/air-verse/air@latest
CMD ["air", "-c", ".air.toml"]

# --- Builder Stage for Production ---
FROM base AS builder
# Copy the rest of the source code
COPY . .
# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server/main.go

# --- Production Stage ---
FROM alpine:latest AS prod
WORKDIR /root/
# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata
# Copy the pre-built binary file from the previous stage
COPY --from=builder /server .
CMD ["./server"]