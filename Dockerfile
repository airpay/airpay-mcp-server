# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /airpay-mcp-server ./cmd/airpay-mcp-server

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /airpay-mcp-server /usr/local/bin/airpay-mcp-server

# Streamable HTTP transport (MCP spec 2025-03-26)
ENV TRANSPORT=http
ENV PORT=8888

EXPOSE 8888

ENTRYPOINT ["airpay-mcp-server"]
