FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/worker-service ./cmd

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /bin/worker-service /app/worker-service
CMD ["/app/worker-service"]
