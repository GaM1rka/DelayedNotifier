FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /bin/sender-service ./cmd

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /bin/sender-service /app/sender-service
EXPOSE 8004
CMD ["/app/sender-service"]
