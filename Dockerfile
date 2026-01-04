# Multi-stage build for minimal image size
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o edgecore ./cmd/edgecore

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and config
COPY --from=builder /build/edgecore .
COPY config.json .

EXPOSE 8080

CMD ["./edgecore"]
