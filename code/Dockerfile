# Stage 1: Build the binary
FROM golang:1.23 AS builder

WORKDIR /app

# Enable CGO
ENV CGO_ENABLED=1

# Install dependencies needed to build sqlite3
RUN apt update && apt install -y gcc libc6-dev

# Copy and build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

# Stage 2: Create a smaller final image
FROM debian:bullseye-slim

WORKDIR /app

# Install runtime dependencies for sqlite3
RUN apt update && apt install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

# Copy the binary from the builder
COPY --from=builder /app/main .

CMD ["./main"]
