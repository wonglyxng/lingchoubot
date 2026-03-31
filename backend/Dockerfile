FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /api ./cmd/api
RUN CGO_ENABLED=0 go build -o /migrate ./cmd/migrate
RUN CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /api /usr/local/bin/api
COPY --from=builder /migrate /usr/local/bin/migrate
COPY --from=builder /worker /usr/local/bin/worker
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

CMD ["api"]
