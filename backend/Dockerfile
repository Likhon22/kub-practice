FROM golang:1.25-alpine AS base

WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

FROM base AS dev
COPY . .
EXPOSE 8080
CMD ["go", "run", "main.go"]

FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/main ./main.go

FROM alpine:latest AS prod
COPY --from=builder /app/bin/main .
EXPOSE 8080
CMD ["./main"]
