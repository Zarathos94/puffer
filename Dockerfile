# Build stage
FROM golang:1.23 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o puffer main.go

# Run stage
FROM gcr.io/distroless/base-debian11

WORKDIR /app

COPY --from=builder /app/puffer .
COPY --from=builder /app/. /app/

# Set environment variables (override in deployment)
ENV ETH_RPC_URL=""
ENV REDIS_ADDR=""
ENV ETHERSCAN_API_KEY=""

EXPOSE 8080

ENTRYPOINT ["/app/puffer"]