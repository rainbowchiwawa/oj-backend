FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./src ./src/
RUN go build -o main ./src/main.go

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache docker-cli
COPY --from=builder /app/main .
CMD ["./main"]