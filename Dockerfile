# Dockerfile (repo root)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# deps
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/mwork-api ./cmd/api

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/mwork-api /app/mwork-api

EXPOSE 8080
CMD ["/app/mwork-api"]
