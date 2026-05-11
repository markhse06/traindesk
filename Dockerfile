FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/main .
COPY .env .

ARG APP_PORT=8080

EXPOSE ${APP_PORT}

ENV HTTP_PORT=${APP_PORT}

CMD ["./main"]