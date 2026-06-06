FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download 
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 go build -o /api ./cmd/api/main.go

FROM alpine:latest

RUN addgroup -S appgroup -g 10000 && \
    adduser -S appuser -G appgroup -u 10000

USER 10000

WORKDIR /home/appuser/

COPY --from=builder --chown=appuser:appgroup /api ./api

EXPOSE 8080

CMD ["./api"]