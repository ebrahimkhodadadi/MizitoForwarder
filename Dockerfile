# Stage 1: Build the Go binary
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Stage 2: Minimal runtime image
FROM debian:bookworm-slim

# Install certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/main .

# Expose port if needed (مثلاً 8080)
EXPOSE 8080

# Run the binary
CMD ["./main"]
