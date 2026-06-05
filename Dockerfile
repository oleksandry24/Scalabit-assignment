FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download 
RUN go mod tidy

COPY . .

RUN go build -o /api ./cmd/api/main.go

FROM alpine:latest

WORKDIR /home/appuser/

COPY --from=builder /api ./api

EXPOSE 8080

CMD ["./api"]